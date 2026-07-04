package startuptime

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMissingFileReturnsEmpty(t *testing.T) {
	h := Load(filepath.Join(t.TempDir(), "does-not-exist.json"))
	assert.Empty(t, h.Runs)
}

func TestLoadCorruptFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	require.NoError(t, os.WriteFile(path, []byte("{ not valid json"), 0o600))

	h := Load(path)
	assert.Empty(t, h.Runs)
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "history.json")

	want := History{Runs: []RunTiming{
		{
			Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			Services:  map[string]time.Duration{"api": 2 * time.Second},
		},
	}}

	require.NoError(t, Save(path, want, DefaultMaxRuns))

	got := Load(path)
	require.Len(t, got.Runs, 1)
	assert.Equal(t, want.Runs[0].Services["api"], got.Runs[0].Services["api"])
	assert.True(t, want.Runs[0].Timestamp.Equal(got.Runs[0].Timestamp))
}

func TestSaveCapsToNewestRuns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")

	var h History
	for i := 0; i < 10; i++ {
		h = h.Append(RunTiming{
			Timestamp: time.Unix(int64(i), 0),
			Services:  map[string]time.Duration{"api": time.Duration(i) * time.Second},
		})
	}

	require.NoError(t, Save(path, h, 3))

	got := Load(path)
	require.Len(t, got.Runs, 3)
	// Oldest kept run is index 7, newest is 9.
	assert.Equal(t, 7*time.Second, got.Runs[0].Services["api"])
	assert.Equal(t, 9*time.Second, got.Runs[2].Services["api"])
}

func TestSaveNonPositiveMaxRunsUsesDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")

	var h History
	for i := 0; i < DefaultMaxRuns+5; i++ {
		h = h.Append(RunTiming{Timestamp: time.Unix(int64(i), 0)})
	}

	require.NoError(t, Save(path, h, 0))

	got := Load(path)
	assert.Len(t, got.Runs, DefaultMaxRuns)
}

func TestLastEmptyHistory(t *testing.T) {
	_, ok := History{}.Last()
	assert.False(t, ok)
}

func TestLastReturnsNewest(t *testing.T) {
	h := History{Runs: []RunTiming{
		{Services: map[string]time.Duration{"api": time.Second}},
		{Services: map[string]time.Duration{"api": 2 * time.Second}},
	}}

	last, ok := h.Last()
	require.True(t, ok)
	assert.Equal(t, 2*time.Second, last.Services["api"])
}

func TestHistoryPathIsStableAndProjectScoped(t *testing.T) {
	base := t.TempDir()

	a1 := HistoryPath(base, filepath.Join("proj", "a"))
	a2 := HistoryPath(base, filepath.Join("proj", "a"))
	b := HistoryPath(base, filepath.Join("proj", "b"))

	assert.Equal(t, a1, a2, "same project resolves to the same path")
	assert.NotEqual(t, a1, b, "different projects resolve to different paths")
	assert.Equal(t, ".json", filepath.Ext(a1))
	assert.Contains(t, a1, filepath.Join(base, "app", "timing"))
}

func TestCompareNoPreviousRun(t *testing.T) {
	current := RunTiming{Services: map[string]time.Duration{"api": 3 * time.Second}}

	deltas := Compare(History{}, current)
	require.Len(t, deltas, 1)
	assert.Equal(t, "api", deltas[0].Service)
	assert.False(t, deltas[0].HasPrev)
	assert.False(t, deltas[0].Regressed)
	assert.Zero(t, deltas[0].Delta)
}

func TestCompareSortedByService(t *testing.T) {
	current := RunTiming{Services: map[string]time.Duration{
		"web": time.Second,
		"api": time.Second,
		"db":  time.Second,
	}}

	deltas := Compare(History{}, current)
	require.Len(t, deltas, 3)
	assert.Equal(t, "api", deltas[0].Service)
	assert.Equal(t, "db", deltas[1].Service)
	assert.Equal(t, "web", deltas[2].Service)
}

func TestCompareDetectsRegression(t *testing.T) {
	prev := History{Runs: []RunTiming{{Services: map[string]time.Duration{
		"api": 4 * time.Second,
	}}}}
	// 4s -> 6s is a 50 percent slowdown, well above the 25 percent ratio and
	// the 500ms floor.
	current := RunTiming{Services: map[string]time.Duration{"api": 6 * time.Second}}

	deltas := Compare(prev, current)
	require.Len(t, deltas, 1)
	assert.True(t, deltas[0].HasPrev)
	assert.Equal(t, 2*time.Second, deltas[0].Delta)
	assert.True(t, deltas[0].Regressed)
}

func TestCompareIgnoresSmallAbsoluteDelta(t *testing.T) {
	// 100ms -> 300ms is a 200 percent slowdown by ratio, but the 200ms delta
	// is below the 500ms floor, so it must not be flagged.
	prev := History{Runs: []RunTiming{{Services: map[string]time.Duration{
		"api": 100 * time.Millisecond,
	}}}}
	current := RunTiming{Services: map[string]time.Duration{"api": 300 * time.Millisecond}}

	deltas := Compare(prev, current)
	require.Len(t, deltas, 1)
	assert.False(t, deltas[0].Regressed)
}

func TestCompareFasterRunNotRegressed(t *testing.T) {
	prev := History{Runs: []RunTiming{{Services: map[string]time.Duration{
		"api": 5 * time.Second,
	}}}}
	current := RunTiming{Services: map[string]time.Duration{"api": 2 * time.Second}}

	deltas := Compare(prev, current)
	require.Len(t, deltas, 1)
	assert.Equal(t, -3*time.Second, deltas[0].Delta)
	assert.False(t, deltas[0].Regressed)
}

func TestCompareNewServiceHasNoPrev(t *testing.T) {
	prev := History{Runs: []RunTiming{{Services: map[string]time.Duration{
		"api": 2 * time.Second,
	}}}}
	current := RunTiming{Services: map[string]time.Duration{
		"api": 2 * time.Second,
		"web": 3 * time.Second, // new this run
	}}

	deltas := Compare(prev, current)
	require.Len(t, deltas, 2)

	byName := map[string]ServiceDelta{}
	for _, d := range deltas {
		byName[d.Service] = d
	}
	assert.True(t, byName["api"].HasPrev)
	assert.False(t, byName["web"].HasPrev)
}

func TestIsRegressionZeroPrev(t *testing.T) {
	assert.False(t, isRegression(0, 10*time.Second, DefaultRegressionRatio, DefaultRegressionFloor))
}

func TestServiceDeltaSummary(t *testing.T) {
	firstRun := ServiceDelta{Service: "api", Duration: 2 * time.Second}
	assert.Contains(t, firstRun.Summary(), "api")
	assert.Contains(t, firstRun.Summary(), "first run")

	slower := ServiceDelta{
		Service:  "web",
		Duration: 3 * time.Second,
		HasPrev:  true,
		Delta:    time.Second,
	}
	assert.Contains(t, slower.Summary(), "+1s vs previous")

	faster := ServiceDelta{
		Service:  "db",
		Duration: 2 * time.Second,
		HasPrev:  true,
		Delta:    -time.Second,
	}
	assert.Contains(t, faster.Summary(), "-1s vs previous")
}

func TestCompareWithThresholdsCustom(t *testing.T) {
	prev := History{Runs: []RunTiming{{Services: map[string]time.Duration{
		"api": 10 * time.Second,
	}}}}
	// 10s -> 11s is a 10 percent slowdown: below the default 25 percent, but
	// above a custom 5 percent ratio.
	current := RunTiming{Services: map[string]time.Duration{"api": 11 * time.Second}}

	assert.False(t, Compare(prev, current)[0].Regressed)
	assert.True(t, CompareWithThresholds(prev, current, 0.05, DefaultRegressionFloor)[0].Regressed)
}
