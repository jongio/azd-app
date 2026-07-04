package commands

import (
	"os"
	"path/filepath"
	"time"

	"github.com/jongio/azd-core/cliout"

	"github.com/jongio/azd-app/cli/src/internal/config"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/startuptime"
)

// timingBaseDir returns the directory that holds startup timing history. It
// reuses the azd config directory so history lives alongside other azd state
// and honors the same test overrides as the config package.
func timingBaseDir() string {
	if path, err := config.GetConfigPath(); err == nil {
		return filepath.Dir(path)
	}

	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".azd")
	}

	return ".azd"
}

// buildRunTiming derives per-service time-to-ready from an orchestration
// result. Each service's duration is measured from when its process started to
// when orchestration reported every service ready. Services without a start
// time, or a run with no ready time, produce no entry. It returns false when
// there is nothing worth recording.
func buildRunTiming(result *service.OrchestrationResult) (startuptime.RunTiming, bool) {
	if result == nil || result.ReadyTime.IsZero() {
		return startuptime.RunTiming{}, false
	}

	services := make(map[string]time.Duration)
	for name, proc := range result.Processes {
		if proc == nil || proc.StartTime.IsZero() {
			continue
		}

		dur := result.ReadyTime.Sub(proc.StartTime)
		if dur < 0 {
			dur = 0
		}
		services[name] = dur
	}

	if len(services) == 0 {
		return startuptime.RunTiming{}, false
	}

	return startuptime.RunTiming{
		Timestamp: result.ReadyTime,
		Services:  services,
	}, true
}

// recordStartupTimings prints a per-service time-to-ready summary with a delta
// versus the previous run, flags services that got notably slower, and appends
// this run to the per-project history. All failures are soft: timing must never
// interfere with a run. The --no-timing flag skips it entirely.
func recordStartupTimings(projectDir string, result *service.OrchestrationResult) {
	if runNoTiming {
		return
	}

	current, ok := buildRunTiming(result)
	if !ok {
		return
	}

	path := startuptime.HistoryPath(timingBaseDir(), projectDir)
	history := startuptime.Load(path)
	deltas := startuptime.Compare(history, current)

	regressions := 0
	cliout.Info("Startup timing (time to ready):")
	for _, d := range deltas {
		if d.Regressed {
			regressions++
			cliout.Warning("  %s  SLOWER", d.Summary())
			continue
		}
		cliout.Info("  %s", d.Summary())
	}

	if regressions > 0 {
		cliout.Warning("%d service(s) started notably slower than the previous run.", regressions)
	}

	if err := startuptime.Save(path, history.Append(current), startuptime.DefaultMaxRuns); err != nil {
		cliout.Warning("Could not save startup timing history: %v", err)
	}
}
