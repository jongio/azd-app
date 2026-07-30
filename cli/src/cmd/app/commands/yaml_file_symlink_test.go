package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeFileAtomic used to stage through a predictable "<path>.tmp". A repo
// shipping that name as a symlink redirected the write to the symlink target,
// letting `azd app add/remove` clobber a file outside the project.
func TestWriteFileAtomicDoesNotFollowPredictableTempSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks on Windows requires elevation or developer mode")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "victim.txt")
	require.NoError(t, os.WriteFile(target, []byte("original"), 0o644))

	dest := filepath.Join(dir, "azure.yaml")
	require.NoError(t, os.WriteFile(dest, []byte("services: {}\n"), 0o644))

	// The attack: plant the exact temp path the old implementation used.
	require.NoError(t, os.Symlink(target, dest+".tmp"))

	require.NoError(t, writeFileAtomic(dest, []byte("services: {api: {}}\n")))

	victim, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "original", string(victim),
		"the symlink target must not be overwritten")

	written, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, "services: {api: {}}\n", string(written))
}

func TestWriteFileAtomicPreservesExistingPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not modelled on Windows")
	}

	dir := t.TempDir()
	dest := filepath.Join(dir, "azure.yaml")
	require.NoError(t, os.WriteFile(dest, []byte("a: 1\n"), 0o640))

	require.NoError(t, writeFileAtomic(dest, []byte("a: 2\n")))

	info, err := os.Stat(dest)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o640), info.Mode().Perm())
}

func TestWriteFileAtomicCreatesNewFileAndLeavesNoTempBehind(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "azure.yaml")

	require.NoError(t, writeFileAtomic(dest, []byte("services: {}\n")))

	got, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, "services: {}\n", string(got))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "temp file must be renamed, not left behind")
	assert.Equal(t, "azure.yaml", entries[0].Name())
}
