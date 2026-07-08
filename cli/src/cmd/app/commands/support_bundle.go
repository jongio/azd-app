package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/healthcheck"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
	"github.com/jongio/azd-app/cli/src/internal/version"
	"github.com/jongio/azd-core/cliout"
	"github.com/jongio/azd-core/security"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const defaultSupportBundleTail = 200

var secretValuePattern = regexp.MustCompile(
	`(?i)((?:secret|password|token|key|credential|connection[_-]?string|auth)[_\-]?\w*\s*[=:]\s*)"?([^\s"]{6,})"?` +
		`|` +
		`(eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,})`,
)

type supportBundleOptions struct {
	output  string
	tail    int
	service string
}

type supportBundleManifest struct {
	CreatedAt  string   `json:"createdAt"`
	ProjectDir string   `json:"projectDir"`
	Version    string   `json:"version"`
	BuildTime  string   `json:"buildTime"`
	Commit     string   `json:"commit"`
	OS         string   `json:"os"`
	Arch       string   `json:"arch"`
	Files      []string `json:"files"`
	Warnings   []string `json:"warnings,omitempty"`
}

// NewSupportBundleCommand creates the support-bundle command.
func NewSupportBundleCommand() *cobra.Command {
	opts := &supportBundleOptions{tail: defaultSupportBundleTail}
	cmd := &cobra.Command{
		Use:          "support-bundle",
		Short:        "Collect local diagnostics for support",
		Long:         "Collect sanitized project, service, health, and log diagnostics into a local folder for issue reports.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSupportBundle(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "Output folder path")
	cmd.Flags().IntVar(&opts.tail, "tail", defaultSupportBundleTail, "Recent log lines per service to include")
	cmd.Flags().StringVarP(&opts.service, "service", "s", "", "Include logs and health for specific service(s), comma-separated")
	return cmd
}

func runSupportBundle(ctx context.Context, opts *supportBundleOptions) error {
	if opts == nil {
		opts = &supportBundleOptions{tail: defaultSupportBundleTail}
	}
	if opts.tail < 0 {
		return fmt.Errorf("--tail must be zero or greater")
	}

	azureYamlPath, err := findAzureYaml()
	if err != nil {
		return err
	}
	projectDir := filepath.Dir(azureYamlPath)
	outputDir, err := resolveSupportBundleOutput(projectDir, opts.output)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outputDir, 0o750); err != nil {
		return fmt.Errorf("failed to create support bundle folder: %w", err)
	}

	manifest := supportBundleManifest{
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		ProjectDir: projectDir,
		Version:    version.Version,
		BuildTime:  BuildTime,
		Commit:     Commit,
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}

	addFile := func(name string, write func(string) error) {
		if err := write(filepath.Join(outputDir, name)); err != nil {
			manifest.Warnings = append(manifest.Warnings, fmt.Sprintf("%s: %v", name, err))
			return
		}
		manifest.Files = append(manifest.Files, name)
	}

	addFile("azure.yaml.redacted", func(path string) error {
		return writeRedactedAzureYaml(azureYamlPath, path)
	})
	addFile("services.json", func(path string) error {
		services, err := serviceinfo.GetServiceInfo(projectDir)
		if err != nil {
			return err
		}
		return writeJSONRedacted(path, services)
	})
	addFile("requirements.json", func(path string) error {
		return writeRequirementsReport(path, azureYamlPath)
	})
	addFile("health.json", func(path string) error {
		return writeHealthReport(ctx, path, projectDir, opts.service)
	})
	addFile("logs.json", func(path string) error {
		return writeLogSnapshot(ctx, path, projectDir, opts)
	})

	if err := writeJSON(filepath.Join(outputDir, "manifest.json"), manifest); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	cliout.Success("Support bundle created: %s", outputDir)
	cliout.Info("Included %d file(s)", len(manifest.Files)+1)
	if len(manifest.Warnings) > 0 {
		cliout.Warning("Completed with %d warning(s). See manifest.json.", len(manifest.Warnings))
	}
	return nil
}

func resolveSupportBundleOutput(projectDir, output string) (string, error) {
	if output == "" {
		name := "support-bundle-" + time.Now().UTC().Format("20060102-150405")
		return filepath.Join(projectDir, ".azd", "support-bundles", name), nil
	}
	if err := security.ValidatePath(output); err != nil {
		return "", fmt.Errorf("invalid output path: %w", err)
	}
	if filepath.IsAbs(output) {
		return filepath.Clean(output), nil
	}
	return filepath.Join(projectDir, output), nil
}

func writeRedactedAzureYaml(sourcePath, targetPath string) error {
	data, err := os.ReadFile(sourcePath) // #nosec G304 -- source path is discovered by findAzureYaml.
	if err != nil {
		return err
	}
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		masked := redactSecretText(string(data))
		return os.WriteFile(targetPath, []byte(masked), 0o600)
	}
	redactYAMLNode(&node, "")
	out, err := yaml.Marshal(&node)
	if err != nil {
		return err
	}
	return os.WriteFile(targetPath, out, 0o600)
}

func redactYAMLNode(node *yaml.Node, parentKey string) {
	if node == nil {
		return
	}
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			value := node.Content[i+1]
			if isSensitiveKey(key) {
				replaceYAMLValue(value)
				continue
			}
			redactYAMLNode(value, key)
		}
	case yaml.SequenceNode, yaml.DocumentNode:
		for _, child := range node.Content {
			redactYAMLNode(child, parentKey)
		}
	case yaml.ScalarNode:
		if isSensitiveKey(parentKey) {
			node.Value = "***"
			return
		}
		node.Value = redactSecretText(node.Value)
	}
}

func replaceYAMLValue(node *yaml.Node) {
	if node == nil {
		return
	}
	node.Kind = yaml.ScalarNode
	node.Tag = "!!str"
	node.Value = "***"
	node.Content = nil
}

func writeHealthReport(ctx context.Context, path, projectDir, serviceFilter string) error {
	monitor, err := healthcheck.NewHealthMonitor(healthcheck.MonitorConfig{
		ProjectDir:      projectDir,
		DefaultEndpoint: defaultHealthEndpoint,
		Timeout:         2 * time.Second,
		LogLevel:        "error",
		LogFormat:       "text",
	})
	if err != nil {
		return err
	}
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	report, err := monitor.Check(checkCtx, parseServiceFilter(serviceFilter))
	if err != nil {
		return err
	}
	return writeJSON(path, report)
}

func writeLogSnapshot(ctx context.Context, path, projectDir string, opts *supportBundleOptions) error {
	logOpts := &logsOptions{
		tail:    opts.tail,
		level:   "all",
		output:  jsonOutputVal,
		source:  string(LogSourceLocal),
		service: opts.service,
	}
	executor := newLogsExecutorForMCP(logOpts, projectDir)
	logCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	collected, err := executor.collect(logCtx, nil)
	if err != nil {
		return err
	}
	for i := range collected.Entries {
		collected.Entries[i].Message = redactSecretText(collected.Entries[i].Message)
	}
	for i := range collected.EntriesWithContext {
		collected.EntriesWithContext[i].Message = redactSecretText(collected.EntriesWithContext[i].Message)
	}
	return writeJSONRedacted(path, collected)
}

func writeRequirementsReport(path, azureYamlPath string) error {
	azureYaml, err := loadSupportBundleReqs(azureYamlPath)
	if err != nil {
		return err
	}

	reqs := effectiveSupportBundleReqs(azureYaml)
	if len(reqs) == 0 {
		return writeJSON(path, ReqsResult{Satisfied: true, Reqs: []ReqResult{}})
	}

	results, allSatisfied := checkSupportBundleReqs(reqs)
	return writeJSON(path, ReqsResult{Satisfied: allSatisfied, Reqs: results})
}

func loadSupportBundleReqs(azureYamlPath string) (*AzureYaml, error) {
	data, err := os.ReadFile(azureYamlPath) // #nosec G304 -- path is discovered by findAzureYaml.
	if err != nil {
		return nil, err
	}

	var azureYaml AzureYaml
	if err := yaml.Unmarshal(data, &azureYaml); err != nil {
		return nil, err
	}
	return &azureYaml, nil
}

func effectiveSupportBundleReqs(azureYaml *AzureYaml) []Prerequisite {
	if azureYaml == nil {
		return nil
	}

	reqs := append([]Prerequisite(nil), azureYaml.Reqs...)
	if azureYaml.hasContainerServices() && !azureYaml.hasDockerReq() {
		reqs = append(reqs, Prerequisite{Name: toolDocker, MinVersion: "20.0.0", CheckRunning: true})
	}
	return reqs
}

func checkSupportBundleReqs(reqs []Prerequisite) ([]ReqResult, bool) {
	originalFormat := cliout.GetFormat()
	_ = cliout.SetFormat("json")
	defer func() {
		_ = cliout.SetFormat(string(originalFormat))
	}()

	return performReqsCheck(reqs)
}

func writeJSONRedacted(path string, value any) error {
	var decoded any
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	return writeJSON(path, redactJSONValue(decoded, ""))
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func redactJSONValue(value any, key string) any {
	if isSensitiveKey(key) {
		return "***"
	}
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, v := range typed {
			out[k] = redactJSONValue(v, k)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, v := range typed {
			out[i] = redactJSONValue(v, key)
		}
		return out
	case string:
		return redactSecretText(typed)
	default:
		return value
	}
}

func isSensitiveKey(key string) bool {
	upper := strings.ToUpper(key)
	for _, part := range []string{"PASSWORD", "SECRET", "TOKEN", "KEY", "CREDENTIAL", "CONNECTION_STRING", "CONNECTIONSTRING", "AUTH"} {
		if strings.Contains(upper, part) {
			return true
		}
	}
	return false
}

func redactSecretText(text string) string {
	return secretValuePattern.ReplaceAllStringFunc(text, func(match string) string {
		if idx := strings.IndexAny(match, "=:"); idx >= 0 {
			return match[:idx+1] + "***"
		}
		return "***"
	})
}
