package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// namedVolumeRegex matches a Docker named-volume identifier (no path separators).
var namedVolumeRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

// GetCommandArgs returns the container command as a token list. Array-form
// `command:` is returned verbatim; string-form is tokenized with shell-style
// quoting. Returns nil when no command is configured.
func (s *Service) GetCommandArgs() []string {
	if len(s.CommandArgs) > 0 {
		return s.CommandArgs
	}
	if strings.TrimSpace(s.Command) != "" {
		return parseCommandLine(s.Command)
	}
	return nil
}

// parseCommandLine splits a command string into tokens, honoring single and
// double quotes so arguments containing spaces are preserved. It is intentionally
// minimal (no variable expansion or escape processing) and sufficient for
// container command overrides.
func parseCommandLine(s string) []string {
	var args []string
	var cur strings.Builder
	inSingle, inDouble, hasToken := false, false, false

	flush := func() {
		if hasToken {
			args = append(args, cur.String())
			cur.Reset()
			hasToken = false
		}
	}

	for _, r := range s {
		switch {
		case inSingle:
			if r == '\'' {
				inSingle = false
			} else {
				cur.WriteRune(r)
			}
		case inDouble:
			if r == '"' {
				inDouble = false
			} else {
				cur.WriteRune(r)
			}
		case r == '\'':
			inSingle = true
			hasToken = true
		case r == '"':
			inDouble = true
			hasToken = true
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			flush()
		default:
			cur.WriteRune(r)
			hasToken = true
		}
	}
	flush()
	return args
}

// DeriveNetworkName returns a stable, per-project Docker network name so that
// container services in the same project can resolve each other by service name.
// The name is derived purely from the (absolute) project directory, making it
// identical across the run, restart, and dashboard code paths without threading
// a parameter. A short hash of the full path keeps it unique across projects
// that share a directory basename.
func DeriveNetworkName(projectDir string) string {
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		abs = projectDir
	}
	abs = filepath.Clean(abs)

	sum := sha256.Sum256([]byte(abs))
	hash := hex.EncodeToString(sum[:])[:8]

	base := sanitizeNetworkComponent(filepath.Base(abs))
	if base == "" {
		base = "app"
	}
	return fmt.Sprintf("azd-app-%s-%s", base, hash)
}

// sanitizeNetworkComponent lowercases and strips characters not allowed in a
// Docker network name component.
func sanitizeNetworkComponent(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-._")
}

// resolveVolumeSpec normalizes a Docker Compose-style volume spec for a container
// service. Named volumes and anonymous volumes pass through unchanged. Bind
// mounts have their host source resolved to an absolute path relative to
// projectDir; a relative bind mount that escapes the project directory is
// rejected to prevent directory traversal.
func resolveVolumeSpec(spec, projectDir string) (string, error) {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return "", fmt.Errorf("empty volume spec")
	}

	source, rest, found := splitVolumeSource(trimmed)
	if !found {
		// Anonymous volume (container path only), pass through.
		return trimmed, nil
	}

	if isNamedVolume(source) {
		// Docker-managed named volume, pass through.
		return trimmed, nil
	}

	// Bind mount: resolve the host source to an absolute path.
	sourceWasRelative := !filepath.IsAbs(source)
	resolved := source

	// For relative binds, canonicalize the project directory first so that
	// symlinks in path prefixes (e.g., /var -> /private/var on macOS) don't
	// cause false containment rejections. Then resolve the source under the
	// canonical project directory so the containment check compares canonical
	// paths on both sides (CWE-59 mitigation).
	if sourceWasRelative {
		absProject, err := filepath.Abs(projectDir)
		if err != nil {
			absProject = filepath.Clean(projectDir)
		}
		if canonicalProject, err := filepath.EvalSymlinks(absProject); err == nil {
			absProject = canonicalProject
		}

		resolved = filepath.Clean(filepath.Join(absProject, source))

		// If the resolved path itself is a symlink, canonicalize it so a
		// symlink that points outside the project tree is correctly rejected.
		if canonical, err := filepath.EvalSymlinks(resolved); err == nil {
			resolved = canonical
		}

		rel, err := filepath.Rel(absProject, resolved)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("volume %q escapes the project directory", spec)
		}
	} else {
		resolved = filepath.Clean(resolved)
	}

	return resolved + ":" + rest, nil
}

// splitVolumeSource splits a volume spec into its host source and the remainder
// (container target plus optional mode), correctly handling Windows drive-letter
// sources such as `C:\data:/container`. found is false for anonymous volumes
// (a single container path with no host source).
func splitVolumeSource(spec string) (source, rest string, found bool) {
	// Windows drive path source: `C:\...` or `C:/...`.
	if len(spec) >= 3 && isDriveLetter(spec[0]) && spec[1] == ':' && (spec[2] == '\\' || spec[2] == '/') {
		if idx := strings.IndexByte(spec[2:], ':'); idx >= 0 {
			cut := 2 + idx
			return spec[:cut], spec[cut+1:], true
		}
		return "", "", false
	}

	idx := strings.IndexByte(spec, ':')
	if idx < 0 {
		return "", "", false
	}
	return spec[:idx], spec[idx+1:], true
}

// isNamedVolume reports whether a volume source refers to a Docker named volume
// (no path separators and no drive colon), as opposed to a bind-mount path.
func isNamedVolume(source string) bool {
	if strings.ContainsAny(source, `/\`) {
		return false
	}
	return namedVolumeRegex.MatchString(source)
}

// isDriveLetter reports whether b is an ASCII drive letter (A-Z or a-z).
func isDriveLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
