// Package trust manages workspace trust for azd-app, preventing untrusted
// azure.yaml files from executing arbitrary host commands without user consent.
//
// Security context: azure.yaml hooks (prerun, postrun, prestop, poststop) and
// service run commands are user-defined shell strings executed on the host
// (CWE-78, CWE-94). Cloning an untrusted repo and running "azd app run" without
// this gate would give the template author silent code execution.
//
// Trust is recorded in ~/.azd/trusted-workspaces.json (mode 0o600).  Each
// record stores the absolute project root path and a SHA-256 hash of azure.yaml.
// When the file changes the record is invalidated and the user is re-prompted.
package trust

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// storeFileName is the basename of the trust-store file under ~/.azd/.
	storeFileName = "trusted-workspaces.json"

	// storeFileMode is the permission bits applied to the trust-store file.
	// Owner-read/write only, trust records contain local paths that should
	// not be world-readable.
	storeFileMode = 0o600

	// storeDirMode is applied to ~/.azd/ when the directory is created.
	storeDirMode = 0o700
)

// ErrHashChanged is returned by IsWorkspaceTrusted when the workspace is
// present in the store but azure.yaml has been modified since it was trusted.
// Callers should re-prompt the user for consent rather than blocking silently.
var ErrHashChanged = errors.New("azure.yaml changed since workspace was trusted")

// trustedEntry is the per-workspace record persisted to the store.
type trustedEntry struct {
	ProjectRoot   string    `json:"projectRoot"`
	AzureYamlHash string    `json:"azureYamlHash"`
	TrustedAt     time.Time `json:"trustedAt"`
}

// trustStoreData is the top-level JSON document written to disk.
type trustStoreData struct {
	Workspaces []trustedEntry `json:"workspaces"`
}

// TrustStore manages workspace trust records persisted at ~/.azd/trusted-workspaces.json.
type TrustStore struct {
	storePath string
}

// NewTrustStore returns a TrustStore backed by the default path
// (~/.azd/trusted-workspaces.json).
func NewTrustStore() (*TrustStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine home directory: %w", err)
	}
	return &TrustStore{
		storePath: filepath.Join(home, ".azd", storeFileName),
	}, nil
}

// newTrustStoreAt creates a TrustStore backed by an explicit file path.
// Used by tests to avoid touching the real home directory.
func newTrustStoreAt(storePath string) *TrustStore {
	return &TrustStore{storePath: storePath}
}

// IsWorkspaceTrusted reports whether projectRoot is in the trust store and its
// azure.yaml content matches the stored hash.
//
// Return semantics:
//   - (true,  nil):           trusted and azure.yaml unchanged
//   - (false, nil):           workspace not in the store (never trusted)
//   - (false, ErrHashChanged): in the store but azure.yaml has changed
//   - (false, other):         I/O or parse error
func (ts *TrustStore) IsWorkspaceTrusted(projectRoot string) (bool, error) {
	root, err := normalizePath(projectRoot)
	if err != nil {
		return false, err
	}

	data, err := ts.load()
	if err != nil {
		return false, err
	}

	for _, entry := range data.Workspaces {
		if entry.ProjectRoot == root {
			currentHash, err := computeAzureYamlHash(root)
			if err != nil {
				return false, fmt.Errorf("failed to hash azure.yaml: %w", err)
			}
			if entry.AzureYamlHash != currentHash {
				return false, ErrHashChanged
			}
			return true, nil
		}
	}

	// Workspace not yet in store.
	return false, nil
}

// TrustWorkspace records projectRoot as trusted by computing and storing the
// current SHA-256 hash of azure.yaml.  If a record already exists it is
// updated in place, so azure.yaml changes are re-acknowledged.
func (ts *TrustStore) TrustWorkspace(projectRoot string) error {
	root, err := normalizePath(projectRoot)
	if err != nil {
		return err
	}

	hash, err := computeAzureYamlHash(root)
	if err != nil {
		return fmt.Errorf("failed to hash azure.yaml: %w", err)
	}

	data, err := ts.load()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for i, entry := range data.Workspaces {
		if entry.ProjectRoot == root {
			data.Workspaces[i].AzureYamlHash = hash
			data.Workspaces[i].TrustedAt = now
			return ts.save(data)
		}
	}

	data.Workspaces = append(data.Workspaces, trustedEntry{
		ProjectRoot:   root,
		AzureYamlHash: hash,
		TrustedAt:     now,
	})
	return ts.save(data)
}

// RevokeWorkspace removes the trust record for projectRoot.
// Revoking an unknown workspace is a no-op (not an error).
func (ts *TrustStore) RevokeWorkspace(projectRoot string) error {
	root, err := normalizePath(projectRoot)
	if err != nil {
		return err
	}

	data, err := ts.load()
	if err != nil {
		return err
	}

	// Filter out the matching entry; reuse the backing array to avoid an
	// allocation when nothing matches.
	filtered := data.Workspaces[:0]
	for _, entry := range data.Workspaces {
		if entry.ProjectRoot != root {
			filtered = append(filtered, entry)
		}
	}
	data.Workspaces = filtered

	return ts.save(data)
}

// computeAzureYamlHash returns the hex-encoded SHA-256 of the azure.yaml file
// located at filepath.Join(projectRoot, "azure.yaml").
func computeAzureYamlHash(projectRoot string) (string, error) {
	azureYamlPath := filepath.Join(projectRoot, "azure.yaml")
	content, err := os.ReadFile(azureYamlPath)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", azureYamlPath, err)
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}

// load reads and parses the trust store from disk.
// Returns an empty store (no error) when the file does not yet exist.
func (ts *TrustStore) load() (*trustStoreData, error) {
	raw, err := os.ReadFile(ts.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &trustStoreData{}, nil
		}
		return nil, fmt.Errorf("failed to read trust store at %s: %w", ts.storePath, err)
	}

	var data trustStoreData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("failed to parse trust store at %s: %w", ts.storePath, err)
	}
	return &data, nil
}

// save writes the trust store to disk atomically using a write-then-rename
// pattern, with 0o600 permissions on the final file.
func (ts *TrustStore) save(data *trustStoreData) error {
	dir := filepath.Dir(ts.storePath)
	if err := os.MkdirAll(dir, storeDirMode); err != nil {
		return fmt.Errorf("failed to create trust store directory %s: %w", dir, err)
	}

	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize trust store: %w", err)
	}

	// Write to a temp file first so the final rename is as atomic as the
	// OS permits (avoids a truncated file if the process is killed mid-write).
	tmpPath := ts.storePath + ".tmp"
	if err := os.WriteFile(tmpPath, raw, storeFileMode); err != nil {
		return fmt.Errorf("failed to write trust store: %w", err)
	}
	if err := os.Rename(tmpPath, ts.storePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize trust store: %w", err)
	}
	return nil
}

// normalizePath returns an absolute, cleaned path for use as a store key.
func normalizePath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path %q: %w", p, err)
	}
	return filepath.Clean(abs), nil
}
