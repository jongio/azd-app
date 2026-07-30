// Package startuptime records how long each service takes to become ready on
// every `azd app run` and compares the current run against previous ones so a
// startup slowdown surfaces without digging through logs.
//
// The store is intentionally decoupled from the CLI and the orchestrator: it
// works with plain time and duration values and reads and writes a JSON file at
// a caller-supplied path. That keeps the history model, the run cap, and the
// regression math easy to test in isolation.
package startuptime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jongio/azd-core/fileutil"
)

// DefaultMaxRuns is the number of runs kept per project. Older runs are dropped
// when the history grows past this size.
const DefaultMaxRuns = 20

// DefaultRegressionRatio is the fractional slowdown (0.25 = 25 percent slower
// than the previous run) at which a service is flagged as regressed.
const DefaultRegressionRatio = 0.25

// DefaultRegressionFloor is the smallest absolute slowdown that can be flagged.
// Deltas below this are ignored so sub-second jitter on fast services does not
// produce noisy warnings.
const DefaultRegressionFloor = 500 * time.Millisecond

// RunTiming is the recorded time-to-ready for one run.
type RunTiming struct {
	Timestamp time.Time `json:"timestamp"`
	// Services maps a service name to its time-to-ready for this run.
	Services map[string]time.Duration `json:"services"`
}

// History is the persisted, chronological list of runs for a single project.
// The newest run is the last element.
type History struct {
	Runs []RunTiming `json:"runs"`
}

// Load reads history from path. A missing file or unparseable content is
// treated as empty history with no error, so a first run or a corrupted file
// never breaks `azd app run`.
func Load(path string) History {
	// path is derived from the azd config dir and a hashed project path.
	data, err := os.ReadFile(path)
	if err != nil {
		return History{}
	}

	var h History
	if err := json.Unmarshal(data, &h); err != nil {
		return History{}
	}

	return h
}

// Save writes the given history to path after capping it to the newest maxRuns
// entries. A maxRuns <= 0 falls back to DefaultMaxRuns.
func Save(path string, h History, maxRuns int) error {
	if maxRuns <= 0 {
		maxRuns = DefaultMaxRuns
	}

	if len(h.Runs) > maxRuns {
		h.Runs = h.Runs[len(h.Runs)-maxRuns:]
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create timing directory: %w", err)
	}

	if err := fileutil.AtomicWriteJSON(path, h); err != nil {
		return fmt.Errorf("failed to write timing history: %w", err)
	}

	return nil
}

// Append adds a run to the history and returns the updated history. It does not
// persist; call Save to write it out (Save applies the run cap).
func (h History) Append(run RunTiming) History {
	h.Runs = append(h.Runs, run)
	return h
}

// Last returns the most recent run and true, or a zero run and false when the
// history is empty.
func (h History) Last() (RunTiming, bool) {
	if len(h.Runs) == 0 {
		return RunTiming{}, false
	}
	return h.Runs[len(h.Runs)-1], true
}

// HistoryPath returns the JSON history file path for projectDir inside baseDir
// (typically the azd config directory). The file name is derived from a hash of
// the absolute project path so different projects never collide and the path is
// filesystem safe regardless of the project location.
func HistoryPath(baseDir, projectDir string) string {
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		abs = projectDir
	}

	sum := sha256.Sum256([]byte(filepath.Clean(abs)))
	name := hex.EncodeToString(sum[:])[:16] + ".json"

	return filepath.Join(baseDir, "app", "timing", name)
}

// ServiceDelta describes one service's time-to-ready for the current run and
// how it compares to the previous run.
type ServiceDelta struct {
	Service string
	// Duration is the time-to-ready in the current run.
	Duration time.Duration
	// PrevDuration is the time-to-ready in the previous run. Only meaningful
	// when HasPrev is true.
	PrevDuration time.Duration
	// HasPrev is true when the service was present in the previous run.
	HasPrev bool
	// Delta is Duration minus PrevDuration. Zero when HasPrev is false.
	Delta time.Duration
	// Regressed is true when the service got slower than the previous run by
	// more than the regression ratio and above the absolute floor.
	Regressed bool
}

// Compare computes a per-service delta for current against the most recent run
// in h, using the default regression thresholds. Results are sorted by service
// name for stable output.
func Compare(h History, current RunTiming) []ServiceDelta {
	return CompareWithThresholds(h, current, DefaultRegressionRatio, DefaultRegressionFloor)
}

// CompareWithThresholds is Compare with explicit regression thresholds, exposed
// for testing and future configuration.
func CompareWithThresholds(h History, current RunTiming, ratio float64, floor time.Duration) []ServiceDelta {
	prev, hasPrev := h.Last()

	deltas := make([]ServiceDelta, 0, len(current.Services))
	for name, dur := range current.Services {
		d := ServiceDelta{Service: name, Duration: dur}

		if hasPrev {
			if prevDur, ok := prev.Services[name]; ok {
				d.HasPrev = true
				d.PrevDuration = prevDur
				d.Delta = dur - prevDur
				d.Regressed = isRegression(prevDur, dur, ratio, floor)
			}
		}

		deltas = append(deltas, d)
	}

	sort.Slice(deltas, func(i, j int) bool {
		return deltas[i].Service < deltas[j].Service
	})

	return deltas
}

// Summary returns a single human-readable line describing the service timing
// and its change versus the previous run. It does not include any regression
// marker or color; the caller decides how to highlight a regressed service.
func (d ServiceDelta) Summary() string {
	cur := d.Duration.Round(time.Millisecond)

	if !d.HasPrev {
		return fmt.Sprintf("%-20s %8s  (first run)", d.Service, cur)
	}

	delta := d.Delta.Round(time.Millisecond)
	sign := "+"
	if delta < 0 {
		sign = "-"
		delta = -delta
	}

	return fmt.Sprintf("%-20s %8s  (%s%s vs previous)", d.Service, cur, sign, delta)
}

// isRegression reports whether cur is slower than prev by more than ratio and
// by at least floor. A non-positive prev cannot regress (nothing to compare).
func isRegression(prev, cur time.Duration, ratio float64, floor time.Duration) bool {
	if prev <= 0 {
		return false
	}

	delta := cur - prev
	if delta < floor {
		return false
	}

	return float64(delta)/float64(prev) > ratio
}
