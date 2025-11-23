package executor

// Environment variable names passed to hooks
const (
	// EnvProjectDir is the absolute path to the project directory
	EnvProjectDir = "AZD_APP_PROJECT_DIR"

	// EnvProjectName is the name of the project from azure.yaml
	EnvProjectName = "AZD_APP_PROJECT_NAME"

	// EnvServiceCount is the number of services defined in azure.yaml
	EnvServiceCount = "AZD_APP_SERVICE_COUNT"
)
