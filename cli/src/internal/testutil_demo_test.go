package internal

import (
	"fmt"
	"testing"

	"github.com/jongio/azd-core/testutil"
)

// TestUtilDemo demonstrates the use of azd-core/testutil in azd-app.
// This is a simple example showing how testutil can be used in azd-app tests.
func TestUtilDemo(t *testing.T) {
	t.Run("CaptureOutput example", func(t *testing.T) {
		output := testutil.CaptureOutput(t, func() error {
			fmt.Println("Hello from azd-app!")
			fmt.Println("Testing testutil integration")
			return nil
		})

		if !testutil.Contains(output, "Hello from azd-app!") {
			t.Error("Expected output to contain greeting")
		}

		if !testutil.Contains(output, "testutil integration") {
			t.Error("Expected output to contain test message")
		}
	})

	t.Run("Contains helper example", func(t *testing.T) {
		message := "azd-app is using azd-core/testutil"

		if !testutil.Contains(message, "azd-core/testutil") {
			t.Error("Expected message to contain 'azd-core/testutil'")
		}

		if testutil.Contains(message, "not-present") {
			t.Error("Expected message to not contain 'not-present'")
		}
	})
}
