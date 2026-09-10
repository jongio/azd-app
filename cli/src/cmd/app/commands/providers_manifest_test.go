package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/stretchr/testify/require"
)

// manifestPath resolves cli/extension.yaml from this package's directory.
func manifestPath(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "extension.yaml"))
	require.NoError(t, err)
	require.FileExists(t, path, "extension manifest must live at cli/extension.yaml")
	return path
}

// TestProvidersMatchManifest keeps extension.yaml and the runtime registrations
// from drifting apart.
//
// azd reads the manifest to decide which hosts this extension claims, then asks
// the running extension for the matching provider. A provider declared but not
// registered makes `azd deploy` fail at the point of use with a confusing error,
// and one registered but not declared is never reached at all. Neither shows up
// in a normal build or in any other test.
func TestProvidersMatchManifest(t *testing.T) {
	require.NoError(t, azdext.VerifyProvidersMatchManifest(ConfigureExtensionHost, manifestPath(t)))
}

// TestProvidersMatchManifestDetectsDrift proves the check above can actually
// fail. A test that only ever passes is indistinguishable from one that does
// nothing, so this feeds the verifier a manifest declaring a provider the
// extension never registers and requires it to complain.
func TestProvidersMatchManifestDetectsDrift(t *testing.T) {
	drifted := filepath.Join(t.TempDir(), "extension.yaml")
	require.NoError(t, os.WriteFile(drifted, []byte(
		"providers:\n"+
			"  - name: local\n"+
			"    type: service-target\n"+
			"  - name: nonexistent\n"+
			"    type: service-target\n"), 0o600))

	err := azdext.VerifyProvidersMatchManifest(ConfigureExtensionHost, drifted)
	require.Error(t, err, "a provider declared in the manifest but never registered must be reported")
	require.Contains(t, err.Error(), "nonexistent")
}

// TestProvidersMatchManifestDetectsUndeclared covers the other direction: a
// provider the extension registers but the manifest never declares. azd would
// never route a service to it, so the registration is dead code.
func TestProvidersMatchManifestDetectsUndeclared(t *testing.T) {
	empty := filepath.Join(t.TempDir(), "extension.yaml")
	require.NoError(t, os.WriteFile(empty, []byte("providers: []\n"), 0o600))

	err := azdext.VerifyProvidersMatchManifest(ConfigureExtensionHost, empty)
	require.Error(t, err, "a registered provider missing from the manifest must be reported")
	require.Contains(t, err.Error(), "local")
}

// TestConfigureExtensionHostRegistrationIsLazy pins the property the verifier
// depends on. VerifyProvidersMatchManifest runs ConfigureExtensionHost against a
// host with no azd connection, so registration must record names without
// constructing providers. If a future change eagerly builds a provider here, it
// would dereference a nil client and this test catches it before CI does.
func TestConfigureExtensionHostRegistrationIsLazy(t *testing.T) {
	host := azdext.NewExtensionHost(nil)
	require.NotPanics(t, func() { ConfigureExtensionHost(host) })

	targets := host.ServiceTargets()
	require.Len(t, targets, 1)
	require.Equal(t, "local", targets[0].Host)
}
