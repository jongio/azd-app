package commands

import (
	"testing"

	"github.com/jongio/azd-core/cliout"
)

func setTestOutputFormat(t *testing.T, format string) {
	t.Helper()
	previous := cliout.GetFormat()
	if err := cliout.SetFormat(format); err != nil {
		t.Fatalf("set output format: %v", err)
	}
	t.Cleanup(func() {
		if err := cliout.SetFormat(string(previous)); err != nil {
			t.Errorf("restore output format: %v", err)
		}
	})
}
