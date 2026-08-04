package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/invopop/jsonschema"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/spf13/cobra"
)

// newMetadataTestRoot builds a minimal root command so the metadata generator
// has something to introspect without pulling in the real command tree.
func newMetadataTestRoot() *cobra.Command {
	root := &cobra.Command{Use: "app"}
	child := &cobra.Command{Use: "run", Short: "Run services"}
	child.Flags().Bool("detach", false, "run in the background")
	root.AddCommand(child)
	return root
}

func runMetadataCommand(t *testing.T) map[string]any {
	t.Helper()

	cmd := NewMetadataCommand(newMetadataTestRoot)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(nil)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("metadata command failed: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatalf("metadata output is not valid JSON: %v\noutput: %s", err, out.String())
	}
	return doc
}

// TestMetadataCommandEmitsConfiguration is the core regression for this
// command. azdext.NewMetadataCommand cannot populate Configuration, so if
// anyone reverts NewMetadataCommand to the SDK helper the configuration block
// silently disappears from the document azd consumes. This fails when that
// happens.
func TestMetadataCommandEmitsConfiguration(t *testing.T) {
	doc := runMetadataCommand(t)

	if got := doc["schemaVersion"]; got != metadataSchemaVersion {
		t.Errorf("schemaVersion = %v, want %q", got, metadataSchemaVersion)
	}
	if got := doc["id"]; got != metadataExtensionID {
		t.Errorf("id = %v, want %q", got, metadataExtensionID)
	}
	if cmds, ok := doc["commands"].([]any); !ok || len(cmds) == 0 {
		t.Error("commands is missing or empty")
	}

	cfg, ok := doc["configuration"].(map[string]any)
	if !ok {
		t.Fatal("configuration block is missing from the metadata document")
	}
	for _, key := range []string{"global", "project", "service", "environmentVariables"} {
		if _, present := cfg[key]; !present {
			t.Errorf("configuration.%s is missing", key)
		}
	}
}

// TestMetadataExtensionIDMatchesManifest keeps the id reported to azd in sync
// with extension.yaml. A mismatch makes azd attribute the metadata to an
// extension that does not exist.
func TestMetadataExtensionIDMatchesManifest(t *testing.T) {
	manifest := manifestPath(t)
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatalf("read extension.yaml: %v", err)
	}

	want := "id: " + metadataExtensionID
	if !strings.Contains(string(data), want) {
		t.Errorf("extension.yaml does not contain %q; metadataExtensionID is stale", want)
	}
}

// TestProjectSchemaDropsOnlyAzdKeys pins the split between azd-owned and
// azd-app-owned azure.yaml keys. The project schema is reflected from
// service.AzureYaml, so a field added there flows through automatically. This
// test makes the removal list the only thing that needs a human decision, and
// fails if that decision was skipped.
func TestProjectSchemaDropsOnlyAzdKeys(t *testing.T) {
	full := newConfigReflector().Reflect(&service.AzureYaml{})
	published := ExtensionConfiguration().Project

	all := schemaPropertyNames(t, full)
	kept := schemaPropertyNames(t, published)

	if len(all) == 0 {
		t.Fatal("reflected AzureYaml schema has no properties")
	}

	dropped := map[string]bool{}
	for _, name := range all {
		dropped[name] = true
	}
	for _, name := range kept {
		delete(dropped, name)
	}

	var got []string
	for name := range dropped {
		got = append(got, name)
	}
	sort.Strings(got)

	want := append([]string(nil), azdOwnedProjectKeys...)
	sort.Strings(want)

	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("project schema dropped %v, want exactly %v.\n"+
			"A new top-level azure.yaml key appeared. Decide whether azd or azd-app owns it "+
			"and update azdOwnedProjectKeys accordingly.", got, want)
	}
}

// TestServiceSchemaMatchesServiceStruct asserts the published service schema
// is the whole service.Service shape, unfiltered. Service scope has no
// azd-owned keys to strip, so any divergence means the schema stopped being
// generated from the struct.
func TestServiceSchemaMatchesServiceStruct(t *testing.T) {
	want := schemaPropertyNames(t, newConfigReflector().Reflect(&service.Service{}))
	got := schemaPropertyNames(t, ExtensionConfiguration().Service)

	if len(want) == 0 {
		t.Fatal("reflected Service schema has no properties")
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("service schema properties = %v, want %v", got, want)
	}
}

// TestSchemasUseYamlKeyNames guards the FieldNameTag setting. Most azure.yaml
// structs carry no json tag, so with the reflector's default the schema would
// publish Go field names ("Command") instead of real config keys ("command").
// That failure is invisible without an explicit check because the schema still
// generates and still validates, just against the wrong key names.
func TestSchemasUseYamlKeyNames(t *testing.T) {
	svc := ExtensionConfiguration().Service
	for _, name := range schemaPropertyNames(t, svc) {
		if name == "" {
			t.Fatal("service schema has an empty property name")
		}
		if r := rune(name[0]); r >= 'A' && r <= 'Z' {
			t.Errorf("service schema property %q is capitalized, which means the reflector "+
				"fell back to Go field names; FieldNameTag must stay set to \"yaml\"", name)
		}
	}
}

// TestGlobalSchemaCoversPersistedConfigKeys checks the user configuration
// schema against the paths azdconfig actually writes. The schema is
// hand-modeled (no Go type describes the stored tree), so it is the one part
// of this file that can silently drift from reality.
func TestGlobalSchemaCoversPersistedConfigKeys(t *testing.T) {
	global := ExtensionConfiguration().Global

	encoded, err := json.Marshal(global)
	if err != nil {
		t.Fatalf("marshal global schema: %v", err)
	}
	text := string(encoded)

	for _, key := range []string{
		"app",
		"preferences",
		"projects",
		"alwaysKillPortConflicts",
		"logs",
		"dashboardPort",
		"ports",
	} {
		if !strings.Contains(text, `"`+key+`"`) {
			t.Errorf("global schema does not describe %q", key)
		}
	}
}

// TestGlobalSchemaIsNotRequired makes sure the schema does not demand config
// that a first-run user has never written. Marking "app" required would make
// every fresh install fail validation.
func TestGlobalSchemaIsNotRequired(t *testing.T) {
	if req := ExtensionConfiguration().Global.Required; len(req) != 0 {
		t.Errorf("global schema requires %v; all azd-app user configuration is optional", req)
	}
}

// azdAppEnvVarPattern matches the environment variable names azd-app defines.
// PROJECT_DIR is matched separately because it does not carry the prefix.
var azdAppEnvVarPattern = regexp.MustCompile(`AZD_APP_[A-Z0-9_]+`)

// TestEveryAzdAppEnvVarIsDocumented is the rot-proof half of this task. It
// scans non-test source for every AZD_APP_* name and fails if one is not in
// the published metadata, so adding a variable without documenting it breaks
// the build instead of quietly shipping an incomplete document.
//
// It found AZD_APP_INSTANCE and AZD_APP_DETACHED missing on its first run.
func TestEveryAzdAppEnvVarIsDocumented(t *testing.T) {
	documented := map[string]bool{}
	for _, v := range ExtensionEnvironmentVariables() {
		documented[v.Name] = true
	}

	found := map[string][]string{}
	srcDir, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve src dir: %v", err)
	}

	err = filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		// The metadata file is the documentation itself; scanning it would
		// make every name trivially self-satisfying.
		if filepath.Base(path) == "metadata.go" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, name := range azdAppEnvVarPattern.FindAllString(string(data), -1) {
			rel, _ := filepath.Rel(srcDir, path)
			found[name] = append(found[name], rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk source tree: %v", err)
	}

	if len(found) == 0 {
		t.Fatal("scan found no AZD_APP_* names; the walk is broken and this test proves nothing")
	}

	var missing []string
	for name, files := range found {
		if !documented[name] {
			missing = append(missing, name+" (used in "+strings.Join(dedupe(files), ", ")+")")
		}
	}
	sort.Strings(missing)

	if len(missing) > 0 {
		t.Errorf("environment variables used in source but not documented in "+
			"ExtensionEnvironmentVariables:\n  %s", strings.Join(missing, "\n  "))
	}
}

// TestDocumentedEnvVarsAreComplete checks the metadata entries themselves are
// usable: a name with no description tells a consumer nothing.
func TestDocumentedEnvVarsAreComplete(t *testing.T) {
	vars := ExtensionEnvironmentVariables()
	if len(vars) == 0 {
		t.Fatal("no environment variables documented")
	}

	seen := map[string]bool{}
	for _, v := range vars {
		if v.Name == "" {
			t.Error("environment variable with an empty name")
			continue
		}
		if seen[v.Name] {
			t.Errorf("environment variable %q is listed twice", v.Name)
		}
		seen[v.Name] = true

		if strings.TrimSpace(v.Description) == "" {
			t.Errorf("environment variable %q has no description", v.Name)
		}
	}

	// PROJECT_DIR carries no AZD_APP_ prefix so the scanning test cannot see
	// it. mcp.go reads it as the legacy fallback, so pin it explicitly.
	if !seen["PROJECT_DIR"] {
		t.Error("PROJECT_DIR is read by getProjectDir but is not documented")
	}
}

func schemaPropertyNames(t *testing.T, schema *jsonschema.Schema) []string {
	t.Helper()
	if schema == nil || schema.Properties == nil {
		return nil
	}
	var names []string
	for p := schema.Properties.Oldest(); p != nil; p = p.Next() {
		names = append(names, p.Key)
	}
	sort.Strings(names)
	return names
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range in {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// failingWriter always fails, so the metadata command's write error branch gets
// exercised. Left uncovered it is unreachable in tests, which is how an
// unchecked write survived there in the first place.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestMetadataCommandReportsWriteFailure(t *testing.T) {
	command := NewMetadataCommand(newMetadataTestRoot)
	command.SetOut(failingWriter{})
	command.SetErr(failingWriter{})

	err := command.Execute()
	if err == nil {
		t.Fatal("Execute() = nil, want a write failure")
	}
	if !strings.Contains(err.Error(), "failed to write metadata") {
		t.Errorf("error = %v, want it to mention the metadata write", err)
	}
}
