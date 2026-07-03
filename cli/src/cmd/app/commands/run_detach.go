package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/runstate"
	"github.com/jongio/azd-core/cliout"
)

const (
	detachedRunEnvVar = "AZD_APP_DETACHED"
	runLogFileName    = "run.log"
)

type detachedRunResult struct {
	Detached bool   `json:"detached"`
	PID      int    `json:"pid"`
	LogFile  string `json:"logFile"`
}

func maybeStartDetachedRun(projectDir string) (*detachedRunResult, bool, error) {
	if !runDetach || os.Getenv(detachedRunEnvVar) == "1" {
		return nil, false, nil
	}

	result, err := startDetachedRun(projectDir)
	if err != nil {
		return nil, false, err
	}

	return result, true, nil
}

func startDetachedRun(projectDir string) (*detachedRunResult, error) {
	statePath, err := runstate.Path(projectDir)
	if err != nil {
		return nil, fmt.Errorf("resolve run state path: %w", err)
	}

	stateDir := filepath.Dir(statePath)
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return nil, fmt.Errorf("create run state directory: %w", err)
	}

	logPath := filepath.Join(stateDir, runLogFileName)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open detached run log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable path: %w", err)
	}

	cmd := exec.Command(exePath, stripDetachFlag(os.Args[1:])...)
	cmd.Env = append(os.Environ(), detachedRunEnvVar+"=1")
	cmd.SysProcAttr = detachSysProcAttr()
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start detached run process: %w", err)
	}

	return &detachedRunResult{
		Detached: true,
		PID:      cmd.Process.Pid,
		LogFile:  logPath,
	}, nil
}

func printDetachedStartResult(result *detachedRunResult) error {
	if cliout.IsJSON() {
		return cliout.PrintJSON(result)
	}

	cliout.Success("Started azd app run in background")
	cliout.Item("PID: %d", result.PID)
	cliout.Item("Status: azd app status")
	cliout.Item("Stop: azd app stop")
	cliout.Item("Logs: %s", result.LogFile)
	return nil
}

func stripDetachFlag(args []string) []string {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--detach" || strings.HasPrefix(arg, "--detach=") {
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered
}
