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
