package commands

import (
	"os"
	"testing"
)

// testDetachChildEnv is a sentinel set only by tests that spawn a detached
// child process via startDetachedRun. startDetachedRun re-executes the current
// binary (the test binary during `go test`); when this variable is present the
// spawned child exits immediately from TestMain instead of re-running the whole
// suite, which would otherwise fork-bomb and never terminate.
const testDetachChildEnv = "AZD_APP_TEST_DETACH_CHILD"

func TestMain(m *testing.M) {
	if os.Getenv(testDetachChildEnv) == "1" {
		os.Exit(0)
	}
	os.Exit(m.Run())
}
