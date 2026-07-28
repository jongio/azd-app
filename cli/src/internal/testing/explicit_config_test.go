package testing

import (
	"fmt"
	"strings"
	"testing"
)

func TestHasExplicitCommand(t *testing.T) {
	cases := []struct {
		name string
		cfg  *ServiceTestConfig
		want bool
	}{
		{"nil config", nil, false},
		{"empty config", &ServiceTestConfig{}, false},
		{"blank command", &ServiceTestConfig{Unit: &TestTypeConfig{Command: "   "}}, false},
		{"unit command", &ServiceTestConfig{Unit: &TestTypeConfig{Command: "npm test"}}, true},
		{"integration command", &ServiceTestConfig{Integration: &TestTypeConfig{Command: "npm run it"}}, true},
		{"e2e command", &ServiceTestConfig{E2E: &TestTypeConfig{Command: "npm run smoke"}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cfg.HasExplicitCommand(); got != tc.want {
				t.Errorf("HasExplicitCommand() = %v, want %v", got, tc.want)
			}
		})
	}
}

// A service whose language is not a recognized test language (e.g. `docker`) is
// still testable when it declares an explicit test command; the language gate is
// bypassed and the configured framework is reported.
func TestValidateService_ExplicitCommand_UnsupportedLanguage(t *testing.T) {
	service := ServiceInfo{
		Name:     "web",
		Language: "docker",
		Dir:      t.TempDir(),
		Config: &ServiceTestConfig{
			Framework: "vitest",
			Unit:      &TestTypeConfig{Command: "mise exec -- npm run test"},
		},
	}

	v := ValidateService(service)

	if !v.CanTest {
		t.Fatalf("expected CanTest=true for explicit-command docker service, got false (skip: %s)", v.SkipReason)
	}
	if v.Framework != "vitest" {
		t.Errorf("expected framework 'vitest', got '%s'", v.Framework)
	}
}

func TestValidateService_ExplicitCommand_DefaultsFrameworkToCustom(t *testing.T) {
	service := ServiceInfo{
		Name:     "svc",
		Language: "", // no language at all
		Dir:      t.TempDir(),
		Config:   &ServiceTestConfig{E2E: &TestTypeConfig{Command: "./run-e2e.sh"}},
	}

	v := ValidateService(service)

	if !v.CanTest {
		t.Fatalf("expected CanTest=true, got false (skip: %s)", v.SkipReason)
	}
	if v.Framework != "custom" {
		t.Errorf("expected framework 'custom' when unset, got '%s'", v.Framework)
	}
}

// A container/emulator service without an explicit test command is still
// skipped (unchanged behaviour).
func TestValidateService_DockerNoExplicitCommand_Skipped(t *testing.T) {
	service := ServiceInfo{
		Name:     "postgres",
		Language: "docker",
		Dir:      t.TempDir(),
	}

	v := ValidateService(service)

	if v.CanTest {
		t.Errorf("expected CanTest=false for docker service without an explicit test command")
	}
}

func TestNewRunnerForService_ExplicitConfig_FrameworkDispatch(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name      string
		framework string
		wantType  string
	}{
		{"vitest -> node", "vitest", "*testing.NodeTestRunner"},
		{"jest -> node", "jest", "*testing.NodeTestRunner"},
		{"pytest -> python", "pytest", "*testing.PythonTestRunner"},
		{"xunit -> dotnet", "xunit", "*testing.DotnetTestRunner"},
		{"gotest -> go", "gotest", "*testing.GoTestRunner"},
		{"unspecified -> node", "", "*testing.NodeTestRunner"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &ServiceTestConfig{Framework: tc.framework, Unit: &TestTypeConfig{Command: "echo hi"}}
			svc := ServiceInfo{Name: "s", Language: "docker", Dir: dir, Config: cfg}

			runner, err := newRunnerForService(svc, cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := fmt.Sprintf("%T", runner); got != tc.wantType {
				t.Errorf("runner type = %s, want %s", got, tc.wantType)
			}
		})
	}
}

// Language dispatch still wins when the language IS a recognized test language,
// regardless of any configured framework.
func TestNewRunnerForService_LanguageWins(t *testing.T) {
	cfg := &ServiceTestConfig{Framework: "pytest", Unit: &TestTypeConfig{Command: "echo hi"}}
	svc := ServiceInfo{Name: "s", Language: "typescript", Dir: t.TempDir(), Config: cfg}

	runner, err := newRunnerForService(svc, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := fmt.Sprintf("%T", runner); got != "*testing.NodeTestRunner" {
		t.Errorf("runner type = %s, want *testing.NodeTestRunner", got)
	}
}

func TestNewRunnerForService_UnsupportedLanguage_NoExplicitCommand(t *testing.T) {
	svc := ServiceInfo{Name: "s", Language: "docker", Dir: t.TempDir()}

	if _, err := newRunnerForService(svc, &ServiceTestConfig{}); err == nil {
		t.Errorf("expected an error for a docker service without an explicit test command")
	}
}

// `--type all` on an explicit-command service expands to run each configured
// type (rather than falling back to the framework's default command). `go
// version` stands in for a real, always-available test command that exits 0.
func TestExecuteServiceTests_All_ExpandsExplicitTypes(t *testing.T) {
	o := NewTestOrchestrator(&TestConfig{})
	svc := ServiceInfo{
		Name:     "web",
		Language: "docker",
		Dir:      t.TempDir(),
		Config: &ServiceTestConfig{
			Framework: "vitest",
			Unit:      &TestTypeConfig{Command: "go version"},
		},
	}

	res, err := o.executeServiceTests(svc, "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Errorf("expected success, got failure: %s", res.Error)
	}
	if res.TestType != testFilterAll {
		t.Errorf("expected aggregate TestType %q, got %q", testFilterAll, res.TestType)
	}
}

// A non-test-language service configured for only one type must NOT run the
// framework's default command when a different, unconfigured type is requested;
// it is skipped (nothing to run) instead.
func TestExecuteServiceTests_UnconfiguredType_NonTestLanguage_Skipped(t *testing.T) {
	o := NewTestOrchestrator(&TestConfig{})
	svc := ServiceInfo{
		Name:     "web",
		Language: "docker",
		Dir:      t.TempDir(),
		Config: &ServiceTestConfig{
			Framework: "vitest",
			Unit:      &TestTypeConfig{Command: "go version"}, // only unit configured
		},
	}

	res, err := o.executeServiceTests(svc, "e2e")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Errorf("expected success (nothing to run), got failure: %s", res.Error)
	}
	if res.Total != 0 {
		t.Errorf("expected 0 tests for an unconfigured type, got %d (framework default likely ran)", res.Total)
	}
}

func TestTypeHasExplicitCommand(t *testing.T) {
	cfg := &ServiceTestConfig{
		Unit: &TestTypeConfig{Command: "npm run test"},
		E2E:  &TestTypeConfig{Command: "  "},
	}
	if !typeHasExplicitCommand(cfg, "unit") {
		t.Error("expected unit to have an explicit command")
	}
	if typeHasExplicitCommand(cfg, "integration") {
		t.Error("expected integration (nil) to have no explicit command")
	}
	if typeHasExplicitCommand(cfg, "e2e") {
		t.Error("expected e2e (blank) to have no explicit command")
	}
	if typeHasExplicitCommand(nil, "unit") {
		t.Error("expected nil config to have no explicit command")
	}
}

func TestIsRecognizedTestLanguage(t *testing.T) {
	for _, lang := range []string{"js", "javascript", "ts", "typescript", "python", "py", "go", "golang", "csharp", "dotnet"} {
		if !isRecognizedTestLanguage(lang) {
			t.Errorf("expected %q to be a recognized test language", lang)
		}
	}
	for _, lang := range []string{"docker", "", "rust", "php"} {
		if isRecognizedTestLanguage(lang) {
			t.Errorf("expected %q to NOT be a recognized test language", lang)
		}
	}
}

// A service whose language has no framework runner must not be reported as
// testable for a type it never configured. executeServiceTests skips that
// combination, so claiming it is testable produces a bogus "passed" summary
// for a run that never happened.
func TestValidateServiceForType_UnconfiguredTypeOnUnrecognizedLanguage_Skipped(t *testing.T) {
	service := ServiceInfo{
		Name:     "postgres",
		Language: "docker",
		Dir:      t.TempDir(),
		Config:   &ServiceTestConfig{Unit: &TestTypeConfig{Command: "./run-unit.sh"}},
	}

	v := ValidateServiceForType(service, "e2e")

	if v.CanTest {
		t.Fatalf("expected CanTest=false for e2e on a docker service that only configures unit")
	}
	if !strings.Contains(v.SkipReason, "e2e") {
		t.Errorf("expected SkipReason to name the missing type, got %q", v.SkipReason)
	}
}

func TestValidateServiceForType_ConfiguredTypeOnUnrecognizedLanguage_Testable(t *testing.T) {
	service := ServiceInfo{
		Name:     "postgres",
		Language: "docker",
		Dir:      t.TempDir(),
		Config:   &ServiceTestConfig{Unit: &TestTypeConfig{Command: "./run-unit.sh"}},
	}

	v := ValidateServiceForType(service, "unit")

	if !v.CanTest {
		t.Fatalf("expected CanTest=true for the configured type, got false (skip: %s)", v.SkipReason)
	}
	if v.Command != "./run-unit.sh" {
		t.Errorf("expected the explicit unit command, got %q", v.Command)
	}
}

// A recognized language keeps its framework runner for types it did not
// configure explicitly, so it stays testable and reports no explicit command.
func TestValidateServiceForType_UnconfiguredTypeOnRecognizedLanguage_StillTestable(t *testing.T) {
	service := ServiceInfo{
		Name:     "web",
		Language: "typescript",
		Dir:      t.TempDir(),
		Config:   &ServiceTestConfig{Framework: "vitest", Unit: &TestTypeConfig{Command: "npm run unit"}},
	}

	v := ValidateServiceForType(service, "e2e")

	if !v.CanTest {
		t.Fatalf("expected CanTest=true for a recognized language, got false (skip: %s)", v.SkipReason)
	}
	if v.Command != "" {
		t.Errorf("expected no explicit command for the framework-driven type, got %q", v.Command)
	}
}

// Validation and execution must agree on which service/type pairs run.
func TestValidateServiceForTypeMatchesExecutionGuard(t *testing.T) {
	dir := t.TempDir()
	cfg := &ServiceTestConfig{Unit: &TestTypeConfig{Command: "./run-unit.sh"}}

	for _, testType := range orderedTestTypes {
		t.Run(testType, func(t *testing.T) {
			service := ServiceInfo{Name: "svc", Language: "docker", Dir: dir, Config: cfg}

			canTest := ValidateServiceForType(service, testType).CanTest
			willRun := typeHasExplicitCommand(cfg, testType)

			if canTest != willRun {
				t.Errorf("validation reports CanTest=%v but execution would run=%v for type %q", canTest, willRun, testType)
			}
		})
	}
}

// AC10: the requested type's explicit command is what displayValidationSummary
// reports, so validation must surface it even when a framework is also set.
func TestValidateServiceForType_ReportsExplicitCommandForRequestedType(t *testing.T) {
	service := ServiceInfo{
		Name:     "web",
		Language: "typescript",
		Dir:      t.TempDir(),
		Config: &ServiceTestConfig{
			Framework: "vitest",
			Unit:      &TestTypeConfig{Command: "npm run unit"},
			E2E:       &TestTypeConfig{Command: "npx tsx scripts/smoke.ts"},
		},
	}

	v := ValidateServiceForType(service, "e2e")

	if v.Command != "npx tsx scripts/smoke.ts" {
		t.Errorf("expected the e2e command, got %q", v.Command)
	}
	if v.Framework != "vitest" {
		t.Errorf("Framework should keep its configured meaning, got %q", v.Framework)
	}
}

// Requesting every type lists each configured command so the summary does not
// silently pick one.
func TestValidateServiceForType_AllListsEveryConfiguredCommand(t *testing.T) {
	service := ServiceInfo{
		Name:     "web",
		Language: "docker",
		Dir:      t.TempDir(),
		Config: &ServiceTestConfig{
			Unit: &TestTypeConfig{Command: "run-unit"},
			E2E:  &TestTypeConfig{Command: "run-e2e"},
		},
	}

	v := ValidateServiceForType(service, testFilterAll)

	if !strings.Contains(v.Command, "unit: run-unit") || !strings.Contains(v.Command, "e2e: run-e2e") {
		t.Errorf("expected both commands listed, got %q", v.Command)
	}
}

// A single configured command needs no type prefix.
func TestValidateServiceForType_AllWithOneCommandOmitsPrefix(t *testing.T) {
	service := ServiceInfo{
		Name:     "web",
		Language: "docker",
		Dir:      t.TempDir(),
		Config:   &ServiceTestConfig{E2E: &TestTypeConfig{Command: "run-e2e"}},
	}

	if got := ValidateServiceForType(service, testFilterAll).Command; got != "run-e2e" {
		t.Errorf("expected the bare command, got %q", got)
	}
}

// ValidateService must stay equivalent to requesting every type.
func TestValidateServiceDelegatesToAllTypes(t *testing.T) {
	service := ServiceInfo{
		Name:     "web",
		Language: "docker",
		Dir:      t.TempDir(),
		Config:   &ServiceTestConfig{Unit: &TestTypeConfig{Command: "run-unit"}},
	}

	if ValidateService(service).Command != ValidateServiceForType(service, testFilterAll).Command {
		t.Error("ValidateService should match ValidateServiceForType with the all filter")
	}
}

// Framework-driven services report no explicit command, preserving the original
// "<framework> detected" summary line.
func TestValidateServiceForType_FrameworkDrivenReportsNoCommand(t *testing.T) {
	service := ServiceInfo{
		Name:     "web",
		Language: "typescript",
		Dir:      t.TempDir(),
		Config:   &ServiceTestConfig{Framework: "vitest", Unit: &TestTypeConfig{}},
	}

	if got := ValidateServiceForType(service, "unit").Command; got != "" {
		t.Errorf("expected no command for a framework-driven service, got %q", got)
	}
}
