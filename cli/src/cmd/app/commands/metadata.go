package commands

import (
	"encoding/json"
	"fmt"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/azure/azure-dev/cli/azd/pkg/extensions"
	"github.com/invopop/jsonschema"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/spf13/cobra"
)

// metadataSchemaVersion is the version of the extension metadata schema that
// this command emits. It is the schema of the metadata document itself, not
// the version of azd-app.
const metadataSchemaVersion = "1.0"

// metadataExtensionID must match the id in extension.yaml.
const metadataExtensionID = "jongio.azd.app"

// azdOwnedProjectKeys lists the top-level azure.yaml keys that belong to azd
// itself rather than to azd-app.
//
// service.AzureYaml is azd-app's view of the whole file, so reflecting it
// yields azd's keys alongside azd-app's. Publishing azd's own keys in an
// extension's project schema would claim ownership of fields azd validates,
// so they are removed after reflection. Everything not listed here is treated
// as azd-app configuration, which means a new field added to AzureYaml shows
// up in the published schema automatically. TestProjectSchemaDropsOnlyAzdKeys
// pins that split.
var azdOwnedProjectKeys = []string{"name", "services", "resources", "metadata"}

// newConfigReflector builds the reflector used for both the azure.yaml schemas
// and the user configuration schema.
//
// FieldNameTag is "yaml" because azure.yaml is parsed with yaml tags and most
// of those structs carry no json tag. Without this the reflector would fall
// back to Go field names and publish "Command" where the real key is
// "command". ExpandedStruct inlines the root type so the schema describes the
// object directly instead of being a single $ref.
func newConfigReflector() *jsonschema.Reflector {
	return &jsonschema.Reflector{
		FieldNameTag:   "yaml",
		ExpandedStruct: true,
	}
}

// portValue is the stored form of a port assignment in the user config.
//
// azdconfig writes ports as strings but GetAllServicePorts also accepts JSON
// numbers, so both forms are valid on disk and the schema has to say so.
type portValue int

// portStringPattern matches the decimal spelling of a TCP port, 1 through
// 65535. The string branch of the port schema needs its own bound because JSON
// Schema cannot apply minimum and maximum to a string, and a bare ^\d+$ would
// accept "0" and "70000" while rejecting the same values written as numbers.
const portStringPattern = `^([1-9]\d{0,3}|[1-5]\d{4}|6[0-4]\d{3}|65[0-4]\d{2}|655[0-2]\d|6553[0-5])$`

// JSONSchema implements the invopop custom schema hook.
func (portValue) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:        "",
		Description: "TCP port between 1 and 65535. Written as a string; numbers are also accepted when read.",
		OneOf: []*jsonschema.Schema{
			{Type: "string", Pattern: portStringPattern},
			{Type: "integer", Minimum: json.Number("1"), Maximum: json.Number("65535")},
		},
	}
}

// userConfiguration mirrors the "app" subtree that azd-app persists in azd's
// user configuration file through the UserConfig service. Every path written
// by the azdconfig package lands somewhere in this tree, so it is the honest
// answer to "what global configuration does this extension own".
type userConfiguration struct {
	App appUserConfig `yaml:"app,omitempty" jsonschema:"description=Root of all azd-app user configuration."`
}

type appUserConfig struct {
	Preferences appPreferences `yaml:"preferences,omitempty" jsonschema:"description=User preferences that apply to every project."`

	// Projects is keyed by a truncated SHA-256 of the lowercased absolute
	// project path, not by project name, so entries survive a rename and do
	// not collide across two checkouts with the same name.
	Projects map[string]appProjectState `yaml:"projects,omitempty" jsonschema:"description=Per-project state keyed by a 16 character hash of the absolute project path."`
}

type appPreferences struct {
	AlwaysKillPortConflicts string `yaml:"alwaysKillPortConflicts,omitempty" jsonschema:"enum=true,enum=false,description=When true azd app terminates a process holding a required port without prompting. Stored as a string."`

	// Logs is deliberately untyped. The blob is written and read verbatim by
	// the dashboard UI through GetPreferenceSection and SetPreferenceSection;
	// no Go type in this repository describes it, so declaring one here would
	// be an invention that immediately drifts.
	Logs map[string]any `yaml:"logs,omitempty" jsonschema:"description=Opaque log viewer preferences owned by the dashboard UI."`
}

type appProjectState struct {
	DashboardPort portValue            `yaml:"dashboardPort,omitempty" jsonschema:"description=Port the dashboard was last bound to for this project."`
	Ports         map[string]portValue `yaml:"ports,omitempty" jsonschema:"description=Assigned port per service. Keys are percent-encoded service names."`
}

// ExtensionEnvironmentVariables returns the environment variables azd-app
// defines, both the ones it reads as input and the ones it sets on child
// processes.
//
// Only variables azd-app itself defines are listed. Ambient variables it
// merely observes (AZD_ACCESS_TOKEN, AZURE_ENV_NAME, GITHUB_ACTIONS,
// CODESPACES, LOCALAPPDATA and friends) are owned by azd, the CI provider or
// the OS, so documenting them here would claim ownership azd-app does not
// have. TestEveryAzdAppEnvVarIsDocumented enforces that split against the
// source tree, so a new AZD_APP_* variable fails the build until it is
// described here.
func ExtensionEnvironmentVariables() []extensions.EnvironmentVariable {
	return []extensions.EnvironmentVariable{
		{
			Name:        "AZD_APP_PROJECT_DIR",
			Description: "Absolute path to the directory containing azure.yaml. Set by azd when it launches the MCP server; falls back to the current working directory.",
			Example:     "/home/user/myapp",
		},
		{
			Name:        "PROJECT_DIR",
			Description: "Legacy fallback for AZD_APP_PROJECT_DIR, honored only when that variable is unset. Prefer AZD_APP_PROJECT_DIR.",
			Example:     "/home/user/myapp",
		},
		{
			Name:        "AZD_APP_DEBUG",
			Description: "Set to true to emit debug level diagnostics.",
			Default:     "false",
			Example:     "true",
		},
		{
			Name:        "AZD_APP_TRUST",
			Description: "Set to 1 to bypass the interactive workspace trust prompt, equivalent to passing --trust. Intended for CI and other non-interactive runs. Only set this for workspaces you trust; azd app executes commands defined in azure.yaml.",
			Default:     "0",
			Example:     "1",
		},
		{
			Name:        "AZD_APP_PROJECT_NAME",
			Description: "Set by azd app for hook processes. Name of the project from azure.yaml. Not read by azd app.",
			Example:     "myapp",
		},
		{
			Name:        "AZD_APP_SERVICE_COUNT",
			Description: "Set by azd app for hook processes. Number of services defined in azure.yaml. Not read by azd app.",
			Example:     "3",
		},
		{
			Name:        "AZD_APP_INSTANCE",
			Description: "Set by azd app on each service process when a service is scaled with --scale. The 1 based instance number. Not read by azd app.",
			Example:     "2",
		},
		{
			Name:        "AZD_APP_DETACHED",
			Description: "Internal marker set by azd app run --detach on the background child process so it does not detach again. Only the exact value 1 counts. Do not set this by hand.",
			Example:     "1",
		},
	}
}

// ExtensionConfiguration describes every configuration surface azd-app owns:
// its user configuration tree, the azure.yaml keys it reads at project and
// service scope, and the environment variables it defines.
func ExtensionConfiguration() *extensions.ConfigurationMetadata {
	reflector := newConfigReflector()

	global := reflector.Reflect(&userConfiguration{})

	project := newConfigReflector().Reflect(&service.AzureYaml{})
	if project.Properties != nil {
		for _, key := range azdOwnedProjectKeys {
			project.Properties.Delete(key)
		}
	}

	svc := newConfigReflector().Reflect(&service.Service{})

	return &extensions.ConfigurationMetadata{
		Global:               global,
		Project:              project,
		Service:              svc,
		EnvironmentVariables: ExtensionEnvironmentVariables(),
	}
}

// ExtensionMetadata builds the full metadata document for azd-app.
//
// azdext.NewMetadataCommand cannot be used directly: it marshals the result of
// GenerateExtensionMetadata immediately and exposes no hook for setting the
// Configuration field, so the command is rebuilt here around the same
// generator.
func ExtensionMetadata(root *cobra.Command) *extensions.ExtensionCommandMetadata {
	metadata := azdext.GenerateExtensionMetadata(metadataSchemaVersion, metadataExtensionID, root)
	metadata.Configuration = ExtensionConfiguration()
	return metadata
}

// NewMetadataCommand creates the hidden metadata command azd uses to discover
// this extension's commands and configuration. rootCmdProvider returns the
// root command for introspection.
func NewMetadataCommand(rootCmdProvider func() *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:    "metadata",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			metadata := ExtensionMetadata(rootCmdProvider())

			jsonBytes, err := json.MarshalIndent(metadata, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal metadata: %w", err)
			}

			if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(jsonBytes)); err != nil {
				return fmt.Errorf("failed to write metadata: %w", err)
			}

			return nil
		},
	}
}
