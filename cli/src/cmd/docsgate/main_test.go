package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTemp writes content to a file in a fresh temp dir and returns its path.
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestReadMetadata(t *testing.T) {
	t.Run("from file", func(t *testing.T) {
		path := writeTemp(t, "md.json", `{"commands":[]}`)

		data, err := readMetadata(path, strings.NewReader("ignored"))

		require.NoError(t, err)
		assert.JSONEq(t, `{"commands":[]}`, string(data))
	})

	t.Run("from stdin", func(t *testing.T) {
		data, err := readMetadata("", strings.NewReader(`{"commands":[]}`))

		require.NoError(t, err)
		assert.JSONEq(t, `{"commands":[]}`, string(data))
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := readMetadata(filepath.Join(t.TempDir(), "absent.json"), nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read CLI metadata")
	})

	t.Run("unreadable stdin", func(t *testing.T) {
		_, err := readMetadata("", iotest.ErrReader(errors.New("pipe closed")))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read CLI metadata from stdin")
	})
}

func TestReadChangedFiles(t *testing.T) {
	t.Run("no path disables the change rules", func(t *testing.T) {
		files, err := readChangedFiles("")

		require.NoError(t, err)
		assert.Nil(t, files)
	})

	t.Run("blank lines and padding are dropped", func(t *testing.T) {
		path := writeTemp(t, "changed.txt", "cli/a.go\n\n  cli/b.go  \n\r\n")

		files, err := readChangedFiles(path)

		require.NoError(t, err)
		assert.Equal(t, []string{"cli/a.go", "cli/b.go"}, files)
	})

	t.Run("empty file yields no files", func(t *testing.T) {
		files, err := readChangedFiles(writeTemp(t, "changed.txt", "\n\n"))

		require.NoError(t, err)
		assert.Empty(t, files)
	})

	t.Run("missing file is an error", func(t *testing.T) {
		_, err := readChangedFiles(filepath.Join(t.TempDir(), "absent.txt"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read changed file list")
	})
}

func TestResolveSkipReason(t *testing.T) {
	body := writeTemp(t, "body.txt", "Fixes #1\n\nDocs-Not-Needed: internal rename\n")

	t.Run("explicit reason wins over the body", func(t *testing.T) {
		reason, err := resolveSkipReason("from the flag", body)

		require.NoError(t, err)
		assert.Equal(t, "from the flag", reason)
	})

	t.Run("body supplies the reason", func(t *testing.T) {
		reason, err := resolveSkipReason("", body)

		require.NoError(t, err)
		assert.Equal(t, "internal rename", reason)
	})

	t.Run("bare marker records nothing", func(t *testing.T) {
		reason, err := resolveSkipReason("", writeTemp(t, "body.txt", "Docs-Not-Needed:\n"))

		require.NoError(t, err)
		assert.Empty(t, reason)
	})

	t.Run("no body and no flag", func(t *testing.T) {
		reason, err := resolveSkipReason("", "")

		require.NoError(t, err)
		assert.Empty(t, reason)
	})

	t.Run("whitespace only flag falls back to the body", func(t *testing.T) {
		reason, err := resolveSkipReason("   ", body)

		require.NoError(t, err)
		assert.Equal(t, "internal rename", reason)
	})

	t.Run("whitespace only flag records nothing", func(t *testing.T) {
		reason, err := resolveSkipReason(" \t ", "")

		require.NoError(t, err)
		assert.Empty(t, reason)
	})

	t.Run("missing body file is an error", func(t *testing.T) {
		_, err := resolveSkipReason("", filepath.Join(t.TempDir(), "absent.txt"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read pull request body")
	})
}

// fixtureRepo writes the smallest documentation tree that satisfies the gate
// for a single `run` command and returns its root.
func fixtureRepo(t *testing.T, documented bool) string {
	t.Helper()

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "cli", "docs", "commands"), 0o750))

	reference := "## Commands Overview\n\n| `run` | Run | [→ Full Spec](commands/run.md) |\n"
	if documented {
		reference += "\n## `azd app run`\n\n### Flags\n\n| `--detach` | | bool | | Detach |\n"
		require.NoError(t, os.WriteFile(
			filepath.Join(root, "cli", "docs", "commands", "run.md"), []byte("# azd app run"), 0o600))
	}
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "cli", "docs", "cli-reference.md"), []byte(reference), 0o600))

	return root
}

const fixtureMetadata = `{"commands":[{"name":["run"],"short":"Run","flags":[{"name":"detach","type":"bool"}]}]}`

func TestRun(t *testing.T) {
	t.Run("documented surface exits zero", func(t *testing.T) {
		var out strings.Builder

		code, err := run(
			[]string{"--repo-root", fixtureRepo(t, true)},
			strings.NewReader(fixtureMetadata), &out)

		require.NoError(t, err)
		assert.Equal(t, 0, code)
		assert.Contains(t, out.String(), "Docs gate passed")
	})

	t.Run("undocumented surface exits one without an error", func(t *testing.T) {
		var out strings.Builder

		code, err := run(
			[]string{"--repo-root", fixtureRepo(t, false)},
			strings.NewReader(fixtureMetadata), &out)

		// A gate failure is a verdict, not a crash. Reporting it as an error too
		// would print a second, redundant message on stderr.
		require.NoError(t, err)
		assert.Equal(t, 1, code)
		assert.Contains(t, out.String(), "Docs gate failed")
	})

	t.Run("pull request body clears a change finding", func(t *testing.T) {
		root := fixtureRepo(t, true)
		changed := writeTemp(t, "changed.txt", "cli/src/cmd/app/commands/run.go\n")
		body := writeTemp(t, "body.txt", "Docs-Not-Needed: unexported rename\n")
		var out strings.Builder

		code, err := run([]string{
			"--repo-root", root,
			"--changed-files", changed,
			"--pr-body", body,
		}, strings.NewReader(fixtureMetadata), &out)

		require.NoError(t, err)
		assert.Equal(t, 0, code)
		assert.Contains(t, out.String(), "unexported rename")
	})

	t.Run("metadata read from a file instead of stdin", func(t *testing.T) {
		var out strings.Builder

		code, err := run([]string{
			"--repo-root", fixtureRepo(t, true),
			"--metadata", writeTemp(t, "md.json", fixtureMetadata),
		}, nil, &out)

		require.NoError(t, err)
		assert.Equal(t, 0, code)
	})

	t.Run("errors surface with exit code one", func(t *testing.T) {
		tests := []struct {
			name string
			args []string
			want string
		}{
			{
				name: "unknown flag",
				args: []string{"--nope"},
				want: "flag provided but not defined",
			},
			{
				name: "unreadable metadata",
				args: []string{"--metadata", filepath.Join(t.TempDir(), "absent.json")},
				want: "failed to read CLI metadata",
			},
			{
				name: "unreadable changed files",
				args: []string{"--changed-files", filepath.Join(t.TempDir(), "absent.txt")},
				want: "failed to read changed file list",
			},
			{
				name: "unreadable pull request body",
				args: []string{"--pr-body", filepath.Join(t.TempDir(), "absent.txt")},
				want: "failed to read pull request body",
			},
			{
				name: "empty command tree",
				args: []string{"--repo-root", fixtureRepo(t, true)},
				want: "no commands",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				var out strings.Builder

				code, err := run(tc.args, strings.NewReader(`{"commands":[]}`), &out)

				require.Error(t, err)
				assert.Equal(t, 1, code)
				assert.Contains(t, err.Error(), tc.want)
			})
		}
	})
}
