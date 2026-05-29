// Package commands provides the command-line interface for the azd-app CLI.
package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-core/cliout"
	types "github.com/jongio/azd-core/projecttype"
	"github.com/jongio/azd-core/security"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewInitCommand creates the init command for setting up azd app in a project.
func NewInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize azure.yaml for local development with azd app",
		Long: `Scan your project directory to detect services, languages, frameworks, and
dependencies, then generate or enrich an azure.yaml configuration file with
all the settings needed for 'azd app run'.

If azure.yaml already exists, init enriches it with azd app extensions
(ports, commands, healthchecks) without overwriting existing configuration.

If no azure.yaml exists, init creates one from scratch based on detected
project structure.

Examples:
  # Initialize in current directory
  azd app init

  # Preview what would be generated (no file changes)
  azd app init --dry-run

  # Force overwrite of existing services section
  azd app init --force`,
		SilenceUsage: true,
		RunE:         runInit,
	}

	cmd.Flags().Bool("dry-run", false, "Show what would be generated without writing files")
	cmd.Flags().Bool("force", false, "Overwrite existing services section in azure.yaml")

	return cmd
}

// DetectedService represents a service discovered during project scanning.
type DetectedService struct {
	Name           string            `json:"name"`
	Language       string            `json:"language"`
	Framework      string            `json:"framework,omitempty"`
	Project        string            `json:"project"`
	Ports          []string          `json:"ports,omitempty"`
	Command        string            `json:"command,omitempty"`
	PackageManager string            `json:"packageManager,omitempty"`
	Environment    map[string]string `json:"environment,omitempty"`
	Type           string            `json:"type,omitempty"`
	Mode           string            `json:"mode,omitempty"`
	Uses           []string          `json:"uses,omitempty"`
	IsWorkspace    bool              `json:"isWorkspace,omitempty"`
}

// InitResult contains the outcome of the init command.
type InitResult struct {
	Services      []DetectedService `json:"services"`
	AzureYamlPath string            `json:"azureYamlPath"`
	Created       bool              `json:"created"`
	Enriched      bool              `json:"enriched"`
}

func runInit(cmd *cobra.Command, _ []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	cliout.CommandHeader("init", "Initialize project for azd app run")

	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Check for existing azure.yaml
	existingPath, _ := detector.FindAzureYaml(workingDir)
	hasExisting := existingPath != ""

	if hasExisting && !force {
		cliout.Info("Found existing azure.yaml: %s", existingPath)
		cliout.Section("🔍", "Scanning project to enrich configuration")
	} else {
		cliout.Section("🔍", "Scanning project structure")
	}

	// Detect all services in the project
	services, err := detectServices(workingDir)
	if err != nil {
		return fmt.Errorf("failed to detect services: %w", err)
	}

	if len(services) == 0 {
		cliout.Warning("No services detected in current directory")
		cliout.Newline()
		cliout.Item("Supported project types:")
		cliout.Item("  • Node.js / TypeScript (package.json)")
		cliout.Item("  • Python (requirements.txt, pyproject.toml)")
		cliout.Item("  • .NET (*.csproj, *.sln)")
		cliout.Item("  • Azure Functions / Logic Apps (host.json)")
		cliout.Item("  • Go (go.mod)")
		cliout.Newline()
		cliout.Item("Make sure you're in the root of your project directory.")
		return fmt.Errorf("no services detected")
	}

	// Display detected services
	displayDetectedServices(services)

	if dryRun {
		cliout.Newline()
		cliout.Info("Dry run — no files modified")
		cliout.Newline()
		cliout.Item("Generated azure.yaml would contain:")
		cliout.Newline()
		yamlContent := generateAzureYamlContent(workingDir, services)
		fmt.Println(yamlContent)
		return nil
	}

	// Generate or enrich azure.yaml
	var result InitResult
	result.Services = services

	if hasExisting && !force {
		result.AzureYamlPath = existingPath
		result.Enriched = true
		if err := enrichAzureYaml(existingPath, services); err != nil {
			return fmt.Errorf("failed to enrich azure.yaml: %w", err)
		}
		cliout.Newline()
		cliout.Success("Enriched azure.yaml with %d service(s)", len(services))
	} else {
		azureYamlPath := filepath.Join(workingDir, "azure.yaml")
		result.AzureYamlPath = azureYamlPath
		result.Created = true

		yamlContent := generateAzureYamlContent(workingDir, services)
		// #nosec G306 -- azure.yaml is a config file, 0644 is appropriate
		if err := os.WriteFile(azureYamlPath, []byte(yamlContent), 0o644); err != nil {
			return fmt.Errorf("failed to write azure.yaml: %w", err)
		}
		cliout.Newline()
		cliout.Success("Created azure.yaml with %d service(s)", len(services))
	}

	cliout.Label("Path", result.AzureYamlPath)
	cliout.Newline()
	cliout.Item("Next steps:")
	cliout.Item("  1. Review azure.yaml and adjust ports/commands if needed")
	cliout.Item("  2. Run 'azd app reqs' to check prerequisites")
	cliout.Item("  3. Run 'azd app run' to start all services")

	if cliout.IsJSON() {
		return cliout.PrintJSON(result)
	}

	return nil
}

// detectServices scans the project directory and returns detected services.
func detectServices(rootDir string) ([]DetectedService, error) {
	var services []DetectedService
	// Track directories already claimed by more specific detectors
	claimedDirs := make(map[string]bool)

	// Detect Azure Functions / Logic Apps first (most specific)
	functionApps, _ := detector.FindFunctionApps(rootDir)
	for _, fa := range functionApps {
		svc := buildFunctionService(fa, rootDir)
		if svc != nil {
			services = append(services, *svc)
			claimedDirs[fa.Dir] = true
		}
	}

	// Detect Node.js projects
	nodeProjects, _ := detector.FindNodeProjects(rootDir)
	for _, np := range nodeProjects {
		if claimedDirs[np.Dir] {
			continue
		}
		// Skip workspace roots that are the project root itself (they contain child services)
		if np.IsWorkspaceRoot && np.Dir == rootDir && len(nodeProjects) > 1 {
			continue
		}

		svc := buildNodeService(np, rootDir)
		if svc != nil {
			services = append(services, *svc)
			claimedDirs[np.Dir] = true
		}
	}

	// Detect Python projects
	pythonProjects, _ := detector.FindPythonProjects(rootDir)
	for _, pp := range pythonProjects {
		if claimedDirs[pp.Dir] {
			continue
		}
		svc := buildPythonService(pp, rootDir)
		if svc != nil {
			services = append(services, *svc)
			claimedDirs[pp.Dir] = true
		}
	}

	// Detect .NET projects
	dotnetProjects, _ := detector.FindDotnetProjects(rootDir)
	for _, dp := range dotnetProjects {
		projectDir := filepath.Dir(dp.Path)
		if claimedDirs[projectDir] {
			continue
		}
		svc := buildDotnetService(dp, rootDir)
		if svc != nil {
			services = append(services, *svc)
			claimedDirs[projectDir] = true
		}
	}

	// Detect Go projects
	goProjects, _ := detector.FindGoProjects(rootDir)
	for _, gp := range goProjects {
		if claimedDirs[gp.Dir] {
			continue
		}
		svc := buildGoService(gp, rootDir)
		if svc != nil {
			services = append(services, *svc)
			claimedDirs[gp.Dir] = true
		}
	}

	// Detect infrastructure dependencies (databases, caches, queues) for each service
	for i := range services {
		services[i].Uses = detectServiceDependencies(services[i], rootDir)
	}

	return services, nil
}

// buildNodeService creates a DetectedService from a Node.js project.
func buildNodeService(np types.NodeProject, rootDir string) *DetectedService {
	relPath := getRelativePath(np.Dir, rootDir)
	name := inferServiceName(relPath, np.Dir)

	svc := &DetectedService{
		Name:           name,
		Language:       "js",
		Project:        relPath,
		PackageManager: np.PackageManager,
		IsWorkspace:    np.IsWorkspaceRoot,
	}

	// Detect TypeScript
	if fileExistsInProject(np.Dir, "tsconfig.json") {
		svc.Language = "ts"
	}

	// Detect framework and default port
	svc.Framework, svc.Ports, svc.Command = detectNodeFrameworkAndConfig(np.Dir, np.PackageManager)

	// If we have ports, it's an HTTP service
	if len(svc.Ports) > 0 {
		svc.Type = "http"
	}

	return svc
}

// buildPythonService creates a DetectedService from a Python project.
func buildPythonService(pp types.PythonProject, rootDir string) *DetectedService {
	relPath := getRelativePath(pp.Dir, rootDir)
	name := inferServiceName(relPath, pp.Dir)

	svc := &DetectedService{
		Name:           name,
		Language:       "python",
		Project:        relPath,
		PackageManager: pp.PackageManager,
	}

	// Detect framework and default port
	svc.Framework, svc.Ports, svc.Command = detectPythonFrameworkAndConfig(pp.Dir)

	if len(svc.Ports) > 0 {
		svc.Type = "http"
	}

	return svc
}

// buildDotnetService creates a DetectedService from a .NET project.
func buildDotnetService(dp types.DotnetProject, rootDir string) *DetectedService {
	// DotnetProject.Path is the path to .csproj or .sln
	projectDir := filepath.Dir(dp.Path)
	relPath := getRelativePath(projectDir, rootDir)
	name := inferServiceName(relPath, projectDir)

	svc := &DetectedService{
		Name:     name,
		Language: "dotnet",
		Project:  relPath,
		Ports:    []string{"5000"},
		Command:  "dotnet watch run",
		Type:     "http",
	}

	return svc
}

// buildFunctionService creates a DetectedService from an Azure Functions project.
func buildFunctionService(fa types.FunctionAppProject, rootDir string) *DetectedService {
	relPath := getRelativePath(fa.Dir, rootDir)
	name := inferServiceName(relPath, fa.Dir)
	if name == "." || name == "" {
		name = "functions"
	}

	svc := &DetectedService{
		Name:     name,
		Language: fa.Language,
		Project:  relPath,
		Ports:    []string{"7071"},
		Type:     "http",
	}

	return svc
}

// buildGoService creates a DetectedService from a Go project.
func buildGoService(gp detector.GoProject, rootDir string) *DetectedService {
	relPath := getRelativePath(gp.Dir, rootDir)
	name := inferServiceName(relPath, gp.Dir)

	svc := &DetectedService{
		Name:     name,
		Language: "go",
		Project:  relPath,
	}

	// Check if it has a main.go (likely a server)
	if fileExistsInProject(gp.Dir, "main.go") || fileExistsInProject(gp.Dir, "cmd") {
		// Check for common web frameworks in go.mod
		goModPath := filepath.Join(gp.Dir, "go.mod")
		if err := security.ValidatePath(goModPath); err == nil {
			// #nosec G304 -- Path validated
			if data, err := os.ReadFile(goModPath); err == nil {
				content := string(data)
				if strings.Contains(content, "github.com/gin-gonic/gin") ||
					strings.Contains(content, "github.com/labstack/echo") ||
					strings.Contains(content, "github.com/gofiber/fiber") ||
					strings.Contains(content, "github.com/gorilla/mux") ||
					strings.Contains(content, "connectrpc.com/connect") {
					svc.Framework = "http-server"
					svc.Ports = []string{"8080"}
					svc.Command = "go run ."
					svc.Type = "http"
				}
			}
		}

		// Default: assume it's a server if no framework detected but has main.go
		if svc.Type == "" {
			svc.Ports = []string{"8080"}
			svc.Command = "go run ."
			svc.Type = "http"
		}
	}

	return svc
}

// detectNodeFrameworkAndConfig detects Node.js framework and infers port/command.
func detectNodeFrameworkAndConfig(projectDir string, packageManager string) (framework string, ports []string, command string) {
	packageJSONPath := filepath.Join(projectDir, "package.json")
	if err := security.ValidatePath(packageJSONPath); err != nil {
		return "", nil, ""
	}

	// #nosec G304 -- Path validated
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return "", nil, ""
	}

	var pkg struct {
		Scripts      map[string]string `json:"scripts"`
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "", nil, ""
	}

	runCmd := packageManager + " run dev"
	if _, ok := pkg.Scripts["dev"]; !ok {
		if _, ok := pkg.Scripts["start"]; ok {
			runCmd = packageManager + " run start"
		} else {
			runCmd = packageManager + " start"
		}
	}

	// Detect framework from dependencies
	if pkg.Dependencies != nil {
		switch {
		case pkg.Dependencies["next"] != "":
			return "Next.js", []string{"3000"}, runCmd
		case pkg.Dependencies["nuxt"] != "":
			return "Nuxt", []string{"3000"}, runCmd
		case pkg.Dependencies["@angular/core"] != "":
			return "Angular", []string{"4200"}, runCmd
		case pkg.Dependencies["react"] != "" && pkg.Dependencies["react-scripts"] != "":
			return "Create React App", []string{"3000"}, runCmd
		case pkg.Dependencies["react"] != "" && pkg.Dependencies["vite"] != "":
			return "Vite+React", []string{"5173"}, runCmd
		case pkg.Dependencies["vite"] != "":
			return "Vite", []string{"5173"}, runCmd
		case pkg.Dependencies["express"] != "":
			return "Express", []string{"3000"}, runCmd
		case pkg.Dependencies["fastify"] != "":
			return "Fastify", []string{"3000"}, runCmd
		case pkg.Dependencies["hono"] != "":
			return "Hono", []string{"3000"}, runCmd
		case pkg.Dependencies["koa"] != "":
			return "Koa", []string{"3000"}, runCmd
		case pkg.Dependencies["@nestjs/core"] != "":
			return "NestJS", []string{"3000"}, runCmd
		case pkg.Dependencies["astro"] != "":
			return "Astro", []string{"4321"}, runCmd
		case pkg.Dependencies["svelte"] != "" || pkg.Dependencies["@sveltejs/kit"] != "":
			return "SvelteKit", []string{"5173"}, runCmd
		}
	}

	// Default: if has scripts, assume it's a service
	if len(pkg.Scripts) > 0 {
		if _, ok := pkg.Scripts["dev"]; ok {
			return "", []string{"3000"}, runCmd
		}
		if _, ok := pkg.Scripts["start"]; ok {
			return "", []string{"3000"}, runCmd
		}
	}

	return "", nil, ""
}

// detectPythonFrameworkAndConfig detects Python framework and infers port/command.
func detectPythonFrameworkAndConfig(projectDir string) (framework string, ports []string, command string) {
	// Check for common frameworks in requirements or pyproject.toml
	reqFiles := []string{"requirements.txt", "pyproject.toml", "Pipfile"}

	for _, reqFile := range reqFiles {
		filePath := filepath.Join(projectDir, reqFile)
		if err := security.ValidatePath(filePath); err != nil {
			continue
		}
		// #nosec G304 -- Path validated
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		content := strings.ToLower(string(data))

		switch {
		case strings.Contains(content, "fastapi"):
			// Look for the entry point
			entryModule := findPythonEntryModule(projectDir)
			if entryModule != "" {
				return "FastAPI", []string{"8000"}, fmt.Sprintf("uvicorn %s:app --reload --port 8000", entryModule)
			}
			return "FastAPI", []string{"8000"}, "uvicorn main:app --reload --port 8000"
		case strings.Contains(content, "flask"):
			return "Flask", []string{"5000"}, "flask run --reload --port 5000"
		case strings.Contains(content, "django"):
			return "Django", []string{"8000"}, "python manage.py runserver 8000"
		case strings.Contains(content, "streamlit"):
			return "Streamlit", []string{"8501"}, "streamlit run app.py --server.port 8501"
		case strings.Contains(content, "uvicorn") || strings.Contains(content, "starlette"):
			entryModule := findPythonEntryModule(projectDir)
			if entryModule != "" {
				return "ASGI", []string{"8000"}, fmt.Sprintf("uvicorn %s:app --reload --port 8000", entryModule)
			}
			return "ASGI", []string{"8000"}, "uvicorn main:app --reload --port 8000"
		}
	}

	return "", nil, ""
}

// findPythonEntryModule finds the Python entry point module name.
func findPythonEntryModule(projectDir string) string {
	candidates := []string{"main.py", "app.py", "server.py", "api.py"}
	for _, candidate := range candidates {
		if fileExistsInProject(projectDir, candidate) {
			return strings.TrimSuffix(candidate, ".py")
		}
	}
	// Check src/ directory
	srcDir := filepath.Join(projectDir, "src")
	if info, err := os.Stat(srcDir); err == nil && info.IsDir() {
		for _, candidate := range candidates {
			if fileExistsInProject(srcDir, candidate) {
				return "src." + strings.TrimSuffix(candidate, ".py")
			}
		}
	}
	return ""
}

// dependencyPattern maps package/library names to infrastructure dependency names.
type dependencyPattern struct {
	Packages []string // Package names to look for in dependency files
	Dep      string   // The dependency name for the uses field
}

// knownDependencies defines patterns for detecting infrastructure dependencies.
var knownDependencies = []dependencyPattern{
	{
		Packages: []string{"pg", "postgres", "postgresql", "node-postgres", "pgx", "psycopg", "psycopg2", "asyncpg", "Npgsql", "lib/pq", "jackc/pgx", "gorm.io/driver/postgres"},
		Dep:      "postgres",
	},
	{
		Packages: []string{"redis", "ioredis", "aioredis", "redis-py", "StackExchange.Redis", "go-redis", "redigo"},
		Dep:      "redis",
	},
	{
		Packages: []string{"mongodb", "mongoose", "pymongo", "motor", "MongoDB.Driver", "go.mongodb.org/mongo-driver"},
		Dep:      "mongodb",
	},
	{
		Packages: []string{"mysql", "mysql2", "mysqlclient", "aiomysql", "PyMySQL", "MySql.Data", "Pomelo.EntityFrameworkCore.MySql", "go-sql-driver/mysql"},
		Dep:      "mysql",
	},
	{
		Packages: []string{"amqplib", "amqp", "pika", "aio-pika", "RabbitMQ.Client", "streadway/amqp", "rabbitmq/amqp091-go"},
		Dep:      "rabbitmq",
	},
	{
		Packages: []string{"kafkajs", "kafka-python", "aiokafka", "confluent-kafka", "Confluent.Kafka", "segmentio/kafka-go", "IBM/sarama"},
		Dep:      "kafka",
	},
	{
		Packages: []string{"@azure/service-bus", "azure-servicebus", "Azure.Messaging.ServiceBus"},
		Dep:      "servicebus",
	},
	{
		Packages: []string{"@azure/storage-blob", "azure-storage-blob", "Azure.Storage.Blobs"},
		Dep:      "storage",
	},
	{
		Packages: []string{"@azure/cosmos", "azure-cosmos", "Microsoft.Azure.Cosmos"},
		Dep:      "cosmosdb",
	},
}

// detectServiceDependencies scans a service's package files for infrastructure dependencies.
func detectServiceDependencies(svc DetectedService, rootDir string) []string {
	projectDir := filepath.Join(rootDir, svc.Project)
	if svc.Project == "." || svc.Project == "" {
		projectDir = rootDir
	}

	var deps []string
	seen := make(map[string]bool)

	// Read all relevant dependency files
	var content string
	lang := strings.ToLower(svc.Language)
	switch lang {
	case "js", "ts", "javascript", "typescript":
		content = readFileContent(filepath.Join(projectDir, "package.json"))
	case "python":
		content = readFileContent(filepath.Join(projectDir, "requirements.txt"))
		content += "\n" + readFileContent(filepath.Join(projectDir, "pyproject.toml"))
		content += "\n" + readFileContent(filepath.Join(projectDir, "Pipfile"))
	case "dotnet", "csharp", "c#", "fsharp":
		content = readFilesWithExt(projectDir, ".csproj")
	case "go":
		content = readFileContent(filepath.Join(projectDir, "go.mod"))
	default:
		// For unknown languages (e.g., Functions with detected language), try all common files
		content = readFileContent(filepath.Join(projectDir, "package.json"))
		content += "\n" + readFileContent(filepath.Join(projectDir, "requirements.txt"))
		content += "\n" + readFileContent(filepath.Join(projectDir, "pyproject.toml"))
		content += "\n" + readFilesWithExt(projectDir, ".csproj")
		content += "\n" + readFileContent(filepath.Join(projectDir, "go.mod"))
	}

	if content == "" {
		return nil
	}

	contentLower := strings.ToLower(content)

	for _, dep := range knownDependencies {
		for _, pkg := range dep.Packages {
			if strings.Contains(contentLower, strings.ToLower(pkg)) {
				if !seen[dep.Dep] {
					deps = append(deps, dep.Dep)
					seen[dep.Dep] = true
				}
				break
			}
		}
	}

	return deps
}

// readFileContent reads a file and returns its content as a string.
func readFileContent(path string) string {
	if err := security.ValidatePath(path); err != nil {
		return ""
	}
	// #nosec G304 -- Path validated
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// readFilesWithExt reads all files with the given extension in a directory.
func readFilesWithExt(dir string, ext string) string {
	var content strings.Builder
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ext) {
			path := filepath.Join(dir, entry.Name())
			if err := security.ValidatePath(path); err != nil {
				continue
			}
			// #nosec G304 -- Path validated
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			content.WriteString(string(data))
			content.WriteString("\n")
		}
	}
	return content.String()
}

// generateAzureYamlContent produces the full azure.yaml content.
func generateAzureYamlContent(rootDir string, services []DetectedService) string {
	var b strings.Builder

	dirName := filepath.Base(rootDir)
	b.WriteString("# Generated by azd app init\n")
	b.WriteString("# yaml-language-server: $schema=https://raw.githubusercontent.com/jongio/azd-app/main/schemas/v1.1/azure.yaml.json\n")
	b.WriteString("\n")
	fmt.Fprintf(&b, "name: %s\n", sanitizeName(dirName))
	b.WriteString("\n")

	// Generate reqs section based on detected languages
	reqs := inferReqs(services)
	if len(reqs) > 0 {
		b.WriteString("reqs:\n")
		for _, req := range reqs {
			fmt.Fprintf(&b, "  - name: %s\n", req)
		}
		b.WriteString("\n")
	}

	// Generate services section
	b.WriteString("services:\n")
	for _, svc := range services {
		fmt.Fprintf(&b, "  %s:\n", svc.Name)

		if svc.Language != "" {
			fmt.Fprintf(&b, "    language: %s\n", svc.Language)
		}
		if svc.Project != "" && svc.Project != "." {
			fmt.Fprintf(&b, "    project: ./%s\n", filepath.ToSlash(svc.Project))
		} else if svc.Project == "." {
			b.WriteString("    project: .\n")
		}
		if svc.Command != "" {
			fmt.Fprintf(&b, "    command: \"%s\"\n", svc.Command)
		}
		if len(svc.Ports) > 0 {
			b.WriteString("    ports:\n")
			for _, port := range svc.Ports {
				fmt.Fprintf(&b, "      - \"%s\"\n", port)
			}
		}
		if svc.Type != "" && svc.Type != "http" {
			fmt.Fprintf(&b, "    type: %s\n", svc.Type)
		}
		if svc.Mode != "" {
			fmt.Fprintf(&b, "    mode: %s\n", svc.Mode)
		}
		if len(svc.Uses) > 0 {
			b.WriteString("    uses:\n")
			for _, use := range svc.Uses {
				fmt.Fprintf(&b, "      - \"%s\"\n", use)
			}
		}
		if len(svc.Environment) > 0 {
			b.WriteString("    environment:\n")
			for k, v := range svc.Environment {
				fmt.Fprintf(&b, "      %s: %s\n", k, v)
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

// enrichAzureYaml adds azd-app extensions to an existing azure.yaml.
func enrichAzureYaml(azureYamlPath string, services []DetectedService) error {
	if err := security.ValidatePath(azureYamlPath); err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// #nosec G304 -- Path validated
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		return fmt.Errorf("failed to read azure.yaml: %w", err)
	}

	// Parse existing YAML
	var existing map[string]any
	if err := yaml.Unmarshal(data, &existing); err != nil {
		return fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	// Get existing services
	existingServices, _ := existing["services"].(map[string]any)
	if existingServices == nil {
		existingServices = make(map[string]any)
	}

	// Enrich each detected service
	modified := false
	for _, svc := range services {
		existingSvc, exists := existingServices[svc.Name]
		if !exists {
			// Add new service
			newSvc := buildServiceMap(svc)
			existingServices[svc.Name] = newSvc
			modified = true
			cliout.Item("  + Added service: %s (%s)", svc.Name, svc.Language)
			continue
		}

		// Enrich existing service with missing fields
		svcMap, ok := existingSvc.(map[string]any)
		if !ok {
			continue
		}

		enriched := false
		if _, hasCommand := svcMap["command"]; !hasCommand && svc.Command != "" {
			svcMap["command"] = svc.Command
			enriched = true
		}
		if _, hasPorts := svcMap["ports"]; !hasPorts && len(svc.Ports) > 0 {
			svcMap["ports"] = svc.Ports
			enriched = true
		}
		if _, hasLang := svcMap["language"]; !hasLang && svc.Language != "" {
			svcMap["language"] = svc.Language
			enriched = true
		}

		if enriched {
			existingServices[svc.Name] = svcMap
			modified = true
			cliout.Item("  ~ Enriched service: %s", svc.Name)
		} else {
			cliout.Item("  = Service unchanged: %s", svc.Name)
		}
	}

	if !modified {
		cliout.Info("No changes needed — azure.yaml already has complete service configuration")
		return nil
	}

	existing["services"] = existingServices

	// Write back
	output, err := yaml.Marshal(existing)
	if err != nil {
		return fmt.Errorf("failed to marshal azure.yaml: %w", err)
	}

	// #nosec G306 -- azure.yaml is a config file, 0644 is appropriate
	if err := os.WriteFile(azureYamlPath, output, 0o644); err != nil {
		return fmt.Errorf("failed to write azure.yaml: %w", err)
	}

	return nil
}

// buildServiceMap converts a DetectedService to a map for YAML serialization.
func buildServiceMap(svc DetectedService) map[string]any {
	m := make(map[string]any)
	if svc.Language != "" {
		m["language"] = svc.Language
	}
	if svc.Project != "" {
		m["project"] = "./" + filepath.ToSlash(svc.Project)
	}
	if svc.Command != "" {
		m["command"] = svc.Command
	}
	if len(svc.Ports) > 0 {
		m["ports"] = svc.Ports
	}
	if svc.Type != "" && svc.Type != "http" {
		m["type"] = svc.Type
	}
	if svc.Mode != "" {
		m["mode"] = svc.Mode
	}
	if len(svc.Uses) > 0 {
		m["uses"] = svc.Uses
	}
	return m
}

// Helper functions

func getRelativePath(dir string, rootDir string) string {
	rel, err := filepath.Rel(rootDir, dir)
	if err != nil {
		return filepath.Base(dir)
	}
	return rel
}

func inferServiceName(relPath string, dir string) string {
	if relPath == "." || relPath == "" {
		return sanitizeName(filepath.Base(dir))
	}
	// Use the last path component as the service name
	name := filepath.Base(relPath)
	return sanitizeName(name)
}

func sanitizeName(name string) string {
	// Replace spaces and special chars with hyphens
	name = strings.ToLower(name)
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, name)
	// Remove leading/trailing hyphens
	name = strings.Trim(name, "-")
	if name == "" {
		name = "app"
	}
	return name
}

func inferReqs(services []DetectedService) []string {
	seen := make(map[string]bool)
	var reqs []string

	for _, svc := range services {
		switch svc.Language {
		case "js", "ts":
			if !seen["node"] {
				reqs = append(reqs, "node")
				seen["node"] = true
			}
			if svc.PackageManager != "" && !seen[svc.PackageManager] {
				reqs = append(reqs, svc.PackageManager)
				seen[svc.PackageManager] = true
			}
		case "python", "Python":
			if !seen["python"] {
				reqs = append(reqs, "python")
				seen["python"] = true
			}
		case "dotnet", "C#":
			if !seen["dotnet"] {
				reqs = append(reqs, "dotnet")
				seen["dotnet"] = true
			}
		case "go", "Go":
			if !seen["go"] {
				reqs = append(reqs, "go")
				seen["go"] = true
			}
		}
	}

	return reqs
}

func fileExistsInProject(dir string, filename string) bool {
	path := filepath.Join(dir, filename)
	if err := security.ValidatePath(path); err != nil {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func displayDetectedServices(services []DetectedService) {
	cliout.Newline()
	cliout.Section("📦", "Detected services")
	for _, svc := range services {
		langInfo := svc.Language
		if svc.Framework != "" {
			langInfo = fmt.Sprintf("%s/%s", svc.Language, svc.Framework)
		}
		portInfo := ""
		if len(svc.Ports) > 0 {
			portInfo = fmt.Sprintf(" → port %s", svc.Ports[0])
		}
		cliout.Item("  %s (%s) at ./%s%s", svc.Name, langInfo, filepath.ToSlash(svc.Project), portInfo)
	}
}
