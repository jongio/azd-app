package commands

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

// writeAzureYamlForCompletion writes a minimal azure.yaml with the given service
// names into dir.
func writeAzureYamlForCompletion(t *testing.T, dir string, names ...string) {
	t.Helper()

	content := "name: completion-test\nservices:\n"
	for _, n := range names {
		content += "  " + n + ":\n    language: node\n    host: local\n    project: ./" + n + "\n"
	}

	path := filepath.Join(dir, "azure.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write azure.yaml: %v", err)
	}
}

func TestServiceNamesFromAzureYaml_Sorted(t *testing.T) {
	dir := t.TempDir()
	writeAzureYamlForCompletion(t, dir, "web", "api", "worker")
	t.Chdir(dir)

	got := serviceNamesFromAzureYaml()
	want := []string{"api", "web", "worker"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("serviceNamesFromAzureYaml() = %v, want %v", got, want)
	}
}

func TestServiceNamesFromAzureYaml_MissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if got := serviceNamesFromAzureYaml(); got != nil {
		t.Errorf("serviceNamesFromAzureYaml() = %v, want nil when azure.yaml is missing", got)
	}
}

func TestCompleteServiceNames(t *testing.T) {
	dir := t.TempDir()
	writeAzureYamlForCompletion(t, dir, "api", "web", "worker")
	t.Chdir(dir)

	tests := []struct {
		name       string
		toComplete string
		want       []string
	}{
		{
			name:       "empty prefix suggests all",
			toComplete: "",
			want:       []string{"api", "web", "worker"},
		},
		{
			name:       "prefix filters",
			toComplete: "w",
			want:       []string{"web", "worker"},
		},
		{
			name:       "comma keeps chosen and completes trailing segment",
			toComplete: "api,w",
			want:       []string{"api,web", "api,worker"},
		},
		{
			name:       "comma excludes already chosen name",
			toComplete: "web,w",
			want:       []string{"web,worker"},
		},
		{
			name:       "no match returns empty",
			toComplete: "zzz",
			want:       []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, directive := completeServiceNames(nil, nil, tt.toComplete)
			if directive != cobra.ShellCompDirectiveNoFileComp {
				t.Errorf("directive = %v, want ShellCompDirectiveNoFileComp", directive)
			}
			if !reflect.DeepEqual(normalizeCompletion(got), normalizeCompletion(tt.want)) {
				t.Errorf("completeServiceNames(%q) = %v, want %v", tt.toComplete, got, tt.want)
			}
		})
	}
}

func TestCompleteServiceNames_NoAzureYaml(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	got, directive := completeServiceNames(nil, nil, "")
	if got != nil {
		t.Errorf("completeServiceNames() = %v, want nil when azure.yaml is missing", got)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want ShellCompDirectiveNoFileComp", directive)
	}
}

func TestCompleteServiceArgs(t *testing.T) {
	dir := t.TempDir()
	writeAzureYamlForCompletion(t, dir, "api", "web", "worker")
	t.Chdir(dir)

	tests := []struct {
		name       string
		args       []string
		toComplete string
		want       []string
	}{
		{
			name:       "empty prefix suggests all",
			args:       nil,
			toComplete: "",
			want:       []string{"api", "web", "worker"},
		},
		{
			name:       "prefix filters",
			args:       nil,
			toComplete: "w",
			want:       []string{"web", "worker"},
		},
		{
			name:       "already chosen names are skipped",
			args:       []string{"api"},
			toComplete: "",
			want:       []string{"web", "worker"},
		},
		{
			name:       "chosen skip combines with prefix",
			args:       []string{"web"},
			toComplete: "w",
			want:       []string{"worker"},
		},
		{
			name:       "no match returns empty",
			args:       nil,
			toComplete: "zzz",
			want:       []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, directive := completeServiceArgs(nil, tt.args, tt.toComplete)
			if directive != cobra.ShellCompDirectiveNoFileComp {
				t.Errorf("directive = %v, want ShellCompDirectiveNoFileComp", directive)
			}
			if !reflect.DeepEqual(normalizeCompletion(got), normalizeCompletion(tt.want)) {
				t.Errorf("completeServiceArgs(%v, %q) = %v, want %v", tt.args, tt.toComplete, got, tt.want)
			}
		})
	}
}

func TestCompleteServiceArgs_NoAzureYaml(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	got, directive := completeServiceArgs(nil, nil, "")
	if got != nil {
		t.Errorf("completeServiceArgs() = %v, want nil when azure.yaml is missing", got)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want ShellCompDirectiveNoFileComp", directive)
	}
}

func TestRegisterServiceFlagCompletion(t *testing.T) {
	cmd := &cobra.Command{Use: "demo"}
	cmd.Flags().String("service", "", "service filter")

	registerServiceFlagCompletion(cmd, "service")

	// Registering an unknown flag must not panic.
	registerServiceFlagCompletion(cmd, "does-not-exist")
}

// normalizeCompletion treats nil and empty slices as equal for comparison.
func normalizeCompletion(s []string) []string {
	if len(s) == 0 {
		return []string{}
	}
	return s
}
