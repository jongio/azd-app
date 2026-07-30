package commands

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/runstate"
	"github.com/jongio/azd-core/cliout"
)

const (
	detachedRunEnvVar = "AZD_APP_DETACHED"
	// detachedRunEnvValue marks a child as the detached run. Only this exact
	// value counts, so an unrelated truthy setting cannot make a foreground
	// invocation behave like the background one.
	detachedRunEnvValue = "1"
	runLogFileName      = "run.log"
)

type detachedRunResult struct {
	Detached bool   `json:"detached"`
	PID      int    `json:"pid"`
	LogFile  string `json:"logFile"`
}

// detachedChildOnce guards the one-time read of the detached marker. It is a
// pointer so tests can reset it between cases.
var (
	detachedChildOnce = new(sync.Once)
	detachedChild     bool
)

func maybeStartDetachedRun(projectDir string) (*detachedRunResult, bool, error) {
	if !runDetach || IsDetachedChild() {
		return nil, false, nil
	}

	result, err := startDetachedRun(projectDir)
	if err != nil {
		return nil, false, err
	}

	return result, true, nil
}

// IsDetachedChild reports whether this process is the background run spawned by
// `azd app run --detach` rather than the foreground invocation that spawned it.
//
// The marker is read once and then removed from this process's environment.
// Spawned services and lifecycle hooks build their environment from
// os.Environ(), so leaving it set would mark them, and any nested
// `azd app run` they invoke, as detached children. Such a nested invocation
// would skip LoadAzdEnvironment and silently run against the default azd
// environment instead of the one it was given with -e. The result is cached
// because callers run both before and after the unset.
func IsDetachedChild() bool {
	detachedChildOnce.Do(func() {
		detachedChild = os.Getenv(detachedRunEnvVar) == detachedRunEnvValue
		if detachedChild {
			// A failure here only leaves the marker in place, which is the
			// pre-existing behaviour, so there is nothing to recover from.
			_ = os.Unsetenv(detachedRunEnvVar)
		}
	})
	return detachedChild
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

	proc, err := spawnDetached(
		exePath,
		stripDetachFlag(os.Args[1:]),
		append(os.Environ(), detachedRunEnvVar+"="+detachedRunEnvValue),
		logFile,
	)
	if err != nil {
		return nil, err
	}

	recordDetachedRunState(projectDir, proc.Pid)

	return &detachedRunResult{
		Detached: true,
		PID:      proc.Pid,
		LogFile:  logPath,
	}, nil
}

// spawnDetached starts the background process, trying each platform spawn
// attempt in order until one succeeds or an attempt fails for a reason other
// than a rejected job-object breakaway.
func spawnDetached(exePath string, args, env []string, logFile *os.File) (*os.Process, error) {
	return spawnWithAttempts(detachSpawnAttempts(), func(attr *syscall.SysProcAttr) (*os.Process, error) {
		// A command that failed to start cannot be started again, so each
		// attempt needs its own exec.Cmd. Reusing logFile is safe because
		// os/exec passes an *os.File straight through and never closes it.
		//
		// exec.Command (not CommandContext) is deliberate: this process is
		// detached on purpose and must outlive the CLI invocation. Binding it
		// to a context would kill it the moment this command returns.
		cmd := exec.Command(exePath, args...) //nolint:noctx // detached process must outlive this process
		cmd.Env = env
		cmd.SysProcAttr = attr
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		cmd.Stdin = nil

		if err := cmd.Start(); err != nil {
			return nil, err
		}
		return cmd.Process, nil
	})
}

// spawnWithAttempts runs each attempt until one succeeds. It only moves on to
// the next attempt when the failure was a rejected breakaway request; any other
// failure would repeat identically, so it is returned immediately.
func spawnWithAttempts(
	attempts []*syscall.SysProcAttr,
	start func(*syscall.SysProcAttr) (*os.Process, error),
) (*os.Process, error) {
	if len(attempts) == 0 {
		return nil, fmt.Errorf("start detached run process: no spawn attempts configured")
	}

	var lastErr error
	for _, attr := range attempts {
		proc, err := start(attr)
		if err == nil {
			return proc, nil
		}

		lastErr = err
		if !isBreakawayRejected(err) {
			break
		}
	}

	return nil, fmt.Errorf("start detached run process: %w", lastErr)
}

// recordDetachedRunState persists a minimal run state as soon as the background
// process exists so that `azd app status` and `azd app stop` can find it during
// the seconds it takes services and the dashboard to come up. The child
// overwrites this record with full service and dashboard details once ready.
//
// The child normally takes seconds to get that far, but nothing guarantees this
// seed lands first. Any state already recorded against the PID we just spawned
// can only have come from that child, so it is left alone rather than being
// reduced back to a bare PID.
func recordDetachedRunState(projectDir string, pid int) {
	if existing, found, err := runstate.Read(projectDir); err == nil && found && existing.PID == pid {
		return
	}

	err := runstate.Write(projectDir, runstate.RunState{
		PID:       pid,
		StartTime: time.Now(),
	})
	if err != nil {
		// The process is already running, so this is not fatal: only status and
		// stop discovery is degraded until the child writes its own state.
		slog.Warn("failed to write detached run state", "error", err, "pid", pid)
		if !cliout.IsJSON() {
			cliout.Warning("Failed to write run state: %v", err)
		}
	}
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

// stripDetachFlag removes --detach from the argv handed to the child, so the
// child runs in the foreground instead of detaching again.
//
// Everything after a bare "--" is a positional argument rather than a flag, and
// `azd app run` forwards positionals through to service selection. Those are
// copied verbatim so a service whose name happens to look like the flag is not
// silently dropped.
func stripDetachFlag(args []string) []string {
	filtered := make([]string, 0, len(args))
	for i, arg := range args {
		if arg == "--" {
			return append(filtered, args[i:]...)
		}
		if arg == "--detach" || strings.HasPrefix(arg, "--detach=") {
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered
}
