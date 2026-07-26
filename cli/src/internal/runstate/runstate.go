// Package runstate persists azd app run metadata for cross-process status checks.
package runstate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/azdconfig"
)

// RunState stores process and endpoint details for a running azd app session.
type RunState struct {
	PID          int            `json:"pid"`
	DashboardURL string         `json:"dashboardUrl"`
	Services     []ServiceState `json:"services"`
	StartTime    time.Time      `json:"startTime"`
}

// ServiceState stores lightweight service endpoint details for status output.
type ServiceState struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
	Port int    `json:"port,omitempty"`
}

// baseDir is the root directory for run state files.
// Defaults to ~/.azd/azd-app. Tests can override this value.
var baseDir string

func stateDir(projectDir string) (string, error) {
	base := baseDir
	if base == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home directory: %w", err)
		}
		base = filepath.Join(homeDir, ".azd", "azd-app")
	}

	return filepath.Join(base, azdconfig.ProjectHash(projectDir)), nil
}

// stateFileName is the run state file name within a project's state directory.
const stateFileName = "run.json"

// Path returns the run-state path for the given project.
func Path(projectDir string) (string, error) {
	dir, err := stateDir(projectDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, stateFileName), nil
}

// Write persists the run state for a project.
//
// A detached run has the parent seed this file and the child rewrite it seconds
// later, so writes are ordered in practice rather than concurrent. Replacing the
// file atomically is deliberately not attempted: Windows refuses to rename over
// a file a reader currently has open, which would trade a rare short read window
// for a writer that can fail outright.
func Write(projectDir string, st RunState) error {
	path, err := Path(projectDir)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create run state directory: %w", err)
	}

	payload, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run state: %w", err)
	}

	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return fmt.Errorf("write run state file: %w", err)
	}

	return nil
}

// Read loads the run state for a project.
// It returns (nil, false, nil) when the state file does not exist.
func Read(projectDir string) (*RunState, bool, error) {
	path, err := Path(projectDir)
	if err != nil {
		return nil, false, err
	}

	payload, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read run state file: %w", err)
	}

	var st RunState
	if err := json.Unmarshal(payload, &st); err != nil {
		return nil, false, fmt.Errorf("unmarshal run state file: %w", err)
	}

	return &st, true, nil
}

// Remove deletes the run state file for a project.
func Remove(projectDir string) error {
	path, err := Path(projectDir)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove run state file: %w", err)
	}

	return nil
}

// IsRunning reports whether the recorded process is still alive.
func IsRunning(st *RunState) bool {
	if st == nil || st.PID <= 0 {
		return false
	}

	return pidAlive(st.PID)
}
