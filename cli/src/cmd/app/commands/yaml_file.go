package commands

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func marshalYAMLNode(node *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(node); err != nil {
		_ = enc.Close()
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeFileAtomic(path string, data []byte) error {
	cleanPath := filepath.Clean(path)
	mode := os.FileMode(0o644)
	if info, err := os.Stat(cleanPath); err == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat %s: %w", cleanPath, err)
	}

	// os.CreateTemp opens with O_CREATE|O_EXCL and a random suffix, so it can
	// never follow a pre-existing symlink. A fixed "<path>.tmp" name could:
	// a malicious repo shipping an azure.yaml.tmp symlink would redirect this
	// write to whatever the symlink targeted. The file is created in the
	// destination directory so the final rename stays on one filesystem.
	dir := filepath.Dir(cleanPath)
	tmp, err := os.CreateTemp(dir, filepath.Base(cleanPath)+".*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()

	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}

	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("failed to write temporary file %s: %w", tmpPath, err)
	}
	// CreateTemp always uses 0600; restore the destination's permissions so
	// azure.yaml stays readable by the tools that expect it to be.
	// #nosec G302 -- azure.yaml needs to remain readable when written.
	if err := tmp.Chmod(mode); err != nil {
		cleanup()
		return fmt.Errorf("failed to set permissions on %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close temporary file %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, cleanPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize %s: %w", cleanPath, err)
	}
	return nil
}
