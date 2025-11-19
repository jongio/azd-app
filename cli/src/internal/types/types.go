package types

// PythonProject represents a detected Python project.
type PythonProject struct {
	Dir            string
	PackageManager string // "uv", "poetry", or "pip"
	Entrypoint     string // Optional: entry point file specified in azure.yaml
}

// NodeProject represents a detected Node.js project.
type NodeProject struct {
	Dir            string
	PackageManager string // "npm", "pnpm", or "yarn"
}

// DotnetProject represents a detected .NET project.
type DotnetProject struct {
	Path string // Path to .csproj or .sln file
}

// AspireProject represents a detected Aspire project.
type AspireProject struct {
	Dir         string
	ProjectFile string // Path to AppHost.csproj
}

// LogicAppProject represents a detected Logic Apps Standard project.
type LogicAppProject struct {
	Dir string // Directory containing workflows folder
}

// FunctionAppProject represents a detected Azure Functions project.
type FunctionAppProject struct {
	Dir      string // Directory containing host.json
	Variant  string // Type of Functions app: "logicapps", "nodejs", "python", "dotnet", "java"
	Language string // Programming language detected for the project
}
