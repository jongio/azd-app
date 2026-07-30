package trust

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestNewTrustStoreUsesHomeDirectory verifies the default constructor anchors
// the store under ~/.azd rather than the working directory. It only inspects
// the computed path; nothing is created, so the real home dir is untouched.
func TestNewTrustStoreUsesHomeDirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory available: %v", err)
	}

	store, err := NewTrustStore()
	if err != nil {
		t.Fatalf("NewTrustStore returned error: %v", err)
	}

	want := filepath.Join(home, ".azd", storeFileName)
	if store.storePath != want {
		t.Errorf("storePath = %q, want %q", store.storePath, want)
	}
}

// TestTrustWorkspaceUpdatesExistingEntryInPlace covers the re-trust path: a
// workspace that is already recorded must have its hash refreshed rather than
// gaining a second entry. A duplicate would leave the stale hash first in the
// slice, so IsWorkspaceTrusted would keep reporting ErrHashChanged forever.
func TestTrustWorkspaceUpdatesExistingEntryInPlace(t *testing.T) {
	dir := t.TempDir()
	writeAzureYaml(t, dir, "name: original\n")

	store := newStore(t)
	if err := store.TrustWorkspace(dir); err != nil {
		t.Fatalf("first TrustWorkspace: %v", err)
	}

	// Change azure.yaml so the stored hash goes stale.
	writeAzureYaml(t, dir, "name: modified\n")
	if _, err := store.IsWorkspaceTrusted(dir); !errors.Is(err, ErrHashChanged) {
		t.Fatalf("after edit, err = %v, want ErrHashChanged", err)
	}

	// Re-trust should acknowledge the new content in place.
	if err := store.TrustWorkspace(dir); err != nil {
		t.Fatalf("second TrustWorkspace: %v", err)
	}

	trusted, err := store.IsWorkspaceTrusted(dir)
	if err != nil {
		t.Fatalf("after re-trust, unexpected error: %v", err)
	}
	if !trusted {
		t.Error("workspace not trusted after re-trust")
	}

	data, err := store.load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(data.Workspaces) != 1 {
		t.Errorf("len(Workspaces) = %d, want 1 (re-trust must update in place, not append)", len(data.Workspaces))
	}
}

// TestTrustWorkspaceMissingAzureYaml verifies the hash step fails loudly when
// there is no azure.yaml, rather than recording a trust entry for a project
// that cannot be verified later.
func TestTrustWorkspaceMissingAzureYaml(t *testing.T) {
	dir := t.TempDir() // deliberately empty
	store := newStore(t)

	err := store.TrustWorkspace(dir)
	if err == nil {
		t.Fatal("TrustWorkspace succeeded without azure.yaml; want error")
	}
	if !strings.Contains(err.Error(), "failed to hash azure.yaml") {
		t.Errorf("error = %q, want it to mention hashing azure.yaml", err.Error())
	}

	data, err := store.load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(data.Workspaces) != 0 {
		t.Errorf("len(Workspaces) = %d, want 0; a failed trust must not persist an entry", len(data.Workspaces))
	}
}

// TestIsWorkspaceTrustedAzureYamlDeleted covers the hash-error branch of
// IsWorkspaceTrusted: the workspace is in the store but azure.yaml is gone, so
// trust cannot be confirmed. This must not be reported as trusted.
func TestIsWorkspaceTrustedAzureYamlDeleted(t *testing.T) {
	dir := t.TempDir()
	writeAzureYaml(t, dir, "name: test\n")

	store := newStore(t)
	if err := store.TrustWorkspace(dir); err != nil {
		t.Fatalf("TrustWorkspace: %v", err)
	}

	if err := os.Remove(filepath.Join(dir, "azure.yaml")); err != nil {
		t.Fatalf("removing azure.yaml: %v", err)
	}

	trusted, err := store.IsWorkspaceTrusted(dir)
	if trusted {
		t.Error("workspace reported trusted with no azure.yaml to verify against")
	}
	if err == nil {
		t.Fatal("expected an error when azure.yaml is missing")
	}
	if !strings.Contains(err.Error(), "failed to hash azure.yaml") {
		t.Errorf("error = %q, want it to mention hashing azure.yaml", err.Error())
	}
}

// TestLoadRejectsMalformedJSON verifies a corrupt store surfaces a parse error
// instead of being silently treated as "no workspaces trusted", which would
// downgrade the trust gate to a prompt rather than an explicit failure.
func TestLoadRejectsMalformedJSON(t *testing.T) {
	store := newStore(t)
	if err := os.WriteFile(store.storePath, []byte("{not valid json"), 0o600); err != nil {
		t.Fatalf("seeding malformed store: %v", err)
	}

	if _, err := store.load(); err == nil {
		t.Fatal("load accepted malformed JSON; want a parse error")
	}

	dir := t.TempDir()
	writeAzureYaml(t, dir, "name: test\n")

	trusted, err := store.IsWorkspaceTrusted(dir)
	if trusted {
		t.Error("workspace reported trusted despite an unreadable store")
	}
	if err == nil {
		t.Error("IsWorkspaceTrusted swallowed the parse error; want it propagated")
	}
}

// TestLoadTreatsMissingFileAsEmpty pins the documented first-run behaviour:
// a store that has never been written is an empty store, not an error.
func TestLoadTreatsMissingFileAsEmpty(t *testing.T) {
	store := newStore(t)

	data, err := store.load()
	if err != nil {
		t.Fatalf("load on a nonexistent store returned error: %v", err)
	}
	if len(data.Workspaces) != 0 {
		t.Errorf("len(Workspaces) = %d, want 0", len(data.Workspaces))
	}
}

// TestSaveFailsWhenDirectoryCannotBeCreated covers save's MkdirAll error path
// by placing the store beneath a regular file, which can never be a directory.
func TestSaveFailsWhenDirectoryCannotBeCreated(t *testing.T) {
	base := t.TempDir()
	blocker := filepath.Join(base, "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("creating blocker file: %v", err)
	}

	dir := t.TempDir()
	writeAzureYaml(t, dir, "name: test\n")

	store := newTrustStoreAt(filepath.Join(blocker, "sub", storeFileName))

	err := store.TrustWorkspace(dir)
	if err == nil {
		t.Fatal("TrustWorkspace succeeded with an uncreatable store directory; want error")
	}
	if !strings.Contains(err.Error(), "failed to create trust store directory") {
		t.Errorf("error = %q, want it to mention creating the store directory", err.Error())
	}
}

// TestRevokeUnknownWorkspaceIsNoOp verifies revoking a workspace that was
// never trusted neither errors nor disturbs existing entries.
func TestRevokeUnknownWorkspaceIsNoOp(t *testing.T) {
	trustedDir := t.TempDir()
	writeAzureYaml(t, trustedDir, "name: keep\n")

	store := newStore(t)
	if err := store.TrustWorkspace(trustedDir); err != nil {
		t.Fatalf("TrustWorkspace: %v", err)
	}

	unknownDir := t.TempDir()
	if err := store.RevokeWorkspace(unknownDir); err != nil {
		t.Fatalf("RevokeWorkspace on an untrusted workspace returned error: %v", err)
	}

	trusted, err := store.IsWorkspaceTrusted(trustedDir)
	if err != nil {
		t.Fatalf("IsWorkspaceTrusted: %v", err)
	}
	if !trusted {
		t.Error("revoking an unrelated workspace dropped the trusted entry")
	}
}

// TestRevokeOnEmptyStoreSucceeds guards the nil-slice path in RevokeWorkspace,
// where data.Workspaces[:0] operates on a nil slice from a first-run store.
func TestRevokeOnEmptyStoreSucceeds(t *testing.T) {
	store := newStore(t)
	if err := store.RevokeWorkspace(t.TempDir()); err != nil {
		t.Fatalf("RevokeWorkspace on an empty store returned error: %v", err)
	}
}

// TestTrustPersistsAcrossStoreInstances verifies trust survives process
// restarts, which is the entire point of writing the file to disk. A store
// that only worked in-memory would silently re-prompt on every run.
func TestTrustPersistsAcrossStoreInstances(t *testing.T) {
	dir := t.TempDir()
	writeAzureYaml(t, dir, "name: persisted\n")

	storePath := filepath.Join(t.TempDir(), storeFileName)

	first := newTrustStoreAt(storePath)
	if err := first.TrustWorkspace(dir); err != nil {
		t.Fatalf("TrustWorkspace: %v", err)
	}

	second := newTrustStoreAt(storePath)
	trusted, err := second.IsWorkspaceTrusted(dir)
	if err != nil {
		t.Fatalf("IsWorkspaceTrusted on a reopened store: %v", err)
	}
	if !trusted {
		t.Error("trust did not survive reopening the store")
	}
}

// TestSaveLeavesNoTempFile verifies the write-then-rename in save cleans up
// after itself, so the store directory never accumulates .tmp droppings.
func TestSaveLeavesNoTempFile(t *testing.T) {
	dir := t.TempDir()
	writeAzureYaml(t, dir, "name: test\n")

	storeDir := t.TempDir()
	store := newTrustStoreAt(filepath.Join(storeDir, storeFileName))
	if err := store.TrustWorkspace(dir); err != nil {
		t.Fatalf("TrustWorkspace: %v", err)
	}

	entries, err := os.ReadDir(storeDir)
	if err != nil {
		t.Fatalf("reading store dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("leftover temp file %q in store directory", e.Name())
		}
	}
}

// TestStoredPathIsAbsolute verifies normalizePath is applied before persisting,
// so a relative projectRoot and its absolute form resolve to the same entry.
func TestStoredPathIsAbsolute(t *testing.T) {
	dir := t.TempDir()
	writeAzureYaml(t, dir, "name: test\n")

	store := newStore(t)
	if err := store.TrustWorkspace(dir); err != nil {
		t.Fatalf("TrustWorkspace: %v", err)
	}

	raw, err := os.ReadFile(store.storePath)
	if err != nil {
		t.Fatalf("reading store: %v", err)
	}
	var data trustStoreData
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("parsing store: %v", err)
	}
	if len(data.Workspaces) != 1 {
		t.Fatalf("len(Workspaces) = %d, want 1", len(data.Workspaces))
	}
	if !filepath.IsAbs(data.Workspaces[0].ProjectRoot) {
		t.Errorf("ProjectRoot = %q, want an absolute path", data.Workspaces[0].ProjectRoot)
	}
	if data.Workspaces[0].TrustedAt.IsZero() {
		t.Error("TrustedAt is zero; the trust timestamp was not recorded")
	}
}

// TestSaveIgnoresSquattedPredictableTempPath guards the fix that moved save
// off a fixed "<store>.tmp" staging name. Anything squatting that path (a
// leftover directory, or an attacker-planted entry) must no longer affect the
// write, because os.CreateTemp picks an unpredictable name with O_CREATE|O_EXCL.
func TestSaveIgnoresSquattedPredictableTempPath(t *testing.T) {
	dir := t.TempDir()
	writeAzureYaml(t, dir, "name: test\n")

	storeDir := t.TempDir()
	storePath := filepath.Join(storeDir, storeFileName)
	if err := os.Mkdir(storePath+".tmp", 0o700); err != nil {
		t.Fatalf("creating squatting temp dir: %v", err)
	}

	store := newTrustStoreAt(storePath)
	if err := store.TrustWorkspace(dir); err != nil {
		t.Fatalf("TrustWorkspace failed with a squatted temp path: %v", err)
	}

	trusted, err := store.IsWorkspaceTrusted(dir)
	if err != nil {
		t.Fatalf("IsWorkspaceTrusted: %v", err)
	}
	if !trusted {
		t.Error("workspace was not trusted; the squatted temp path broke save")
	}
}

// TestSaveDoesNotFollowSymlinkOnPredictableTempPath is the security half of the
// same fix. The old os.WriteFile on "<store>.tmp" opened with O_CREATE|O_WRONLY
// and followed symlinks, so a link planted at that predictable path redirected
// the trust store write to an arbitrary file the user could write.
func TestSaveDoesNotFollowSymlinkOnPredictableTempPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks on Windows requires elevation or developer mode")
	}

	dir := t.TempDir()
	writeAzureYaml(t, dir, "name: test\n")

	storeDir := t.TempDir()
	storePath := filepath.Join(storeDir, storeFileName)

	victim := filepath.Join(storeDir, "victim.txt")
	if err := os.WriteFile(victim, []byte("original"), 0o600); err != nil {
		t.Fatalf("creating victim file: %v", err)
	}
	if err := os.Symlink(victim, storePath+".tmp"); err != nil {
		t.Fatalf("creating symlink: %v", err)
	}

	store := newTrustStoreAt(storePath)
	if err := store.TrustWorkspace(dir); err != nil {
		t.Fatalf("TrustWorkspace: %v", err)
	}

	contents, err := os.ReadFile(victim)
	if err != nil {
		t.Fatalf("reading victim file: %v", err)
	}
	if string(contents) != "original" {
		t.Error("save followed the planted symlink and overwrote the target file")
	}
}

// TestLoadFailsWhenStorePathIsDirectory covers load's non-IsNotExist read
// error. A directory sitting on the store path is not "no store yet", so it
// must surface as an error instead of being treated as an empty store, which
// would silently downgrade the trust gate.
func TestLoadFailsWhenStorePathIsDirectory(t *testing.T) {
	dir := t.TempDir()
	writeAzureYaml(t, dir, "name: test\n")

	storeDir := t.TempDir()
	storePath := filepath.Join(storeDir, storeFileName)
	if err := os.Mkdir(storePath, 0o700); err != nil {
		t.Fatalf("creating blocking store dir: %v", err)
	}

	store := newTrustStoreAt(storePath)
	err := store.TrustWorkspace(dir)
	if err == nil {
		t.Fatal("TrustWorkspace succeeded with a directory on the store path; want error")
	}
	if !strings.Contains(err.Error(), "failed to read trust store") {
		t.Errorf("error = %q, want it to mention reading the trust store", err.Error())
	}
}

// TestSaveCleansUpTempOnRenameFailure covers save's rename failure path and
// its cleanup. save is called directly because load would reject the blocked
// path first, and the point here is that a failed rename must not leave a
// partial ".tmp" store behind.
func TestSaveCleansUpTempOnRenameFailure(t *testing.T) {
	storeDir := t.TempDir()
	storePath := filepath.Join(storeDir, storeFileName)
	if err := os.Mkdir(storePath, 0o700); err != nil {
		t.Fatalf("creating blocking store dir: %v", err)
	}
	// A non-empty directory cannot be replaced by a rename on any platform.
	if err := os.WriteFile(filepath.Join(storePath, "occupant"), []byte("x"), 0o600); err != nil {
		t.Fatalf("populating blocking dir: %v", err)
	}

	store := newTrustStoreAt(storePath)
	err := store.save(&trustStoreData{Workspaces: []trustedEntry{{ProjectRoot: storeDir}}})
	if err == nil {
		t.Fatal("save succeeded with a blocked rename target; want error")
	}
	if !strings.Contains(err.Error(), "failed to finalize trust store") {
		t.Errorf("error = %q, want it to mention finalizing the trust store", err.Error())
	}

	leftovers, globErr := filepath.Glob(filepath.Join(storeDir, "*.tmp"))
	if globErr != nil {
		t.Fatalf("globbing temp files: %v", globErr)
	}
	if len(leftovers) != 0 {
		t.Errorf("temp files left behind after a failed rename: %v; save must clean up", leftovers)
	}
}
