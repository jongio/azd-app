package commands

import (
	"path/filepath"
	"testing"

	"github.com/jongio/azd-core/manifest"
)

// extensionManifestPath and extensionGoModPath locate the extension manifest and module file
// from this package's directory.
var (
	extensionManifestPath = filepath.Join("..", "..", "..", "..", "extension.yaml")
	extensionGoModPath    = filepath.Join("..", "..", "..", "..", "go.mod")
)

// TestRequiredAzdVersionTracksTheSdk keeps the declared azd host floor equal
// to the azure-dev module this extension is compiled against.
//
// The two drift apart silently. Bumping the SDK to pick up a new azdext API
// changes nothing in extension.yaml, so azd keeps offering the extension to
// hosts whose gRPC server does not implement the calls it now makes, and the
// user meets that as a runtime failure instead of a clear message at install
// time.
func TestRequiredAzdVersionTracksTheSdk(t *testing.T) {
	if err := manifest.CheckRequiredAzdVersion(extensionManifestPath, extensionGoModPath); err != nil {
		t.Fatal(err)
	}
}

// TestManifestHasNoInertKeys catches fields azd never reads.
//
// extension.schema.json does not forbid extra properties and azd's loader
// discards anything it does not recognize, so a misspelled or invented key
// passes validation, loads without complaint, and does nothing. That is how
// azd-rest shipped "minAzdVersion" for several releases while advertising no
// version constraint at all.
//
// Keys listed here are known to be inert and kept deliberately.
func TestManifestHasNoInertKeys(t *testing.T) {
	parsed, err := manifest.Load(extensionManifestPath)
	if err != nil {
		t.Fatal(err)
	}

	if unknown := parsed.UnknownKeys(); len(unknown) > 0 {
		t.Errorf("azd reads none of these keys in %s: %v", extensionManifestPath, unknown)
	}
}
