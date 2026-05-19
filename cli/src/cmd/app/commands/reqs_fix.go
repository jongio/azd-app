package commands

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/jongio/azd-app/cli/src/internal/cache"
	"github.com/jongio/azd-core/cliout"
	"github.com/jongio/azd-core/pathutil"
)

// runClearCache clears the reqs cache.
func runClearCache() error {
	cacheManager, err := cache.NewCacheManager()
	if err != nil {
		return fmt.Errorf("failed to initialize cache manager: %w", err)
	}

	if err := cacheManager.ClearCache(); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	if cliout.IsJSON() {
		return cliout.PrintJSON(map[string]any{
			"success": true,
			"message": "Reqs cache cleared successfully",
		})
	}

	cliout.Success("Reqs cache cleared successfully")
	return nil
}

// FixResult represents the result of attempting to fix a requirement.
type FixResult struct {
	Name      string `json:"name"`
	Fixed     bool   `json:"fixed"`
	Found     bool   `json:"found"`
	Path      string `json:"path,omitempty"`
	Message   string `json:"message"`
	Satisfied bool   `json:"satisfied"`
}

type reqsFixRunner struct {
	checker *PrerequisiteChecker
}

func newReqsFixRunner() *reqsFixRunner {
	return &reqsFixRunner{checker: NewPrerequisiteChecker()}
}

func (r *reqsFixRunner) parseReqs() (string, []Prerequisite, error) {
	azureYamlPath, azureYaml, err := loadAzureYaml()
	if err != nil {
		return "", nil, err
	}

	reqs := append([]Prerequisite{}, azureYaml.Reqs...)
	reqs = r.ensureDockerReq(azureYaml, reqs)
	if len(reqs) == 0 {
		return "", nil, fmt.Errorf("no reqs defined in azure.yaml - run 'azd app reqs --generate' to add them")
	}

	return azureYamlPath, reqs, nil
}

func (r *reqsFixRunner) ensureDockerReq(azureYaml *AzureYaml, reqs []Prerequisite) []Prerequisite {
	if azureYaml.hasContainerServices() && !azureYaml.hasDockerReq() {
		return append(reqs, Prerequisite{Name: "docker", MinVersion: "20.0.0", CheckRunning: true})
	}
	return reqs
}

func (r *reqsFixRunner) checkPrerequisite(prereq Prerequisite) FixResult {
	fixResult := FixResult{Name: prereq.Name}
	config := r.checker.getToolConfig(prereq)
	toolCommand := config.Command

	toolPath := pathutil.FindToolInPath(toolCommand)
	if toolPath != "" {
		fixResult.Found = true
		fixResult.Path = toolPath
		result := r.checker.Check(prereq)
		if result.Satisfied {
			fixResult.Fixed = true
			fixResult.Satisfied = true
			fixResult.Message = fmt.Sprintf("Found and verified: %s", toolPath)
			if !cliout.IsJSON() {
				cliout.ItemSuccess("Found: %s", toolPath)
				cliout.ItemSuccess("Version verified successfully")
			}
			return fixResult
		}
		fixResult.Message = fmt.Sprintf("Found at %s but version check failed: %s", toolPath, result.Message)
		if !cliout.IsJSON() {
			cliout.ItemWarning("Found: %s", toolPath)
			cliout.ItemWarning("Version check failed: %s", result.Message)
		}
		return fixResult
	}

	toolPath = pathutil.SearchToolInSystemPath(toolCommand)
	if toolPath != "" {
		fixResult.Found = true
		fixResult.Path = toolPath
		fixResult.Message = fmt.Sprintf("Found at %s but not in PATH - restart terminal may be needed", toolPath)
		if !cliout.IsJSON() {
			cliout.ItemWarning("Found: %s", toolPath)
			cliout.ItemWarning("Tool is installed but not in current PATH")
			cliout.Info("   %s Restart your terminal to update PATH", cliout.IconBulb)
		}
		return fixResult
	}

	suggestion := pathutil.GetInstallSuggestion(toolCommand)
	fixResult.Message = fmt.Sprintf("Not found - %s", suggestion)
	if !cliout.IsJSON() {
		cliout.ItemError("Not found in system PATH")
		cliout.Info("   %s %s", cliout.IconBulb, suggestion)
	}
	return fixResult
}

// runReqsFix attempts to fix PATH issues for missing tools.
func runReqsFix() error {
	cliout.CommandHeader("reqs --fix", "Fix PATH issues for missing tools")
	if !cliout.IsJSON() {
		cliout.Section(cliout.IconTool, "Attempting to fix requirement issues...")
	}

	runner := newReqsFixRunner()
	azureYamlPath, reqs, err := runner.parseReqs()
	if err != nil {
		return err
	}

	var failedReqs []Prerequisite
	for _, prereq := range reqs {
		if result := runner.checker.Check(prereq); !result.Satisfied {
			failedReqs = append(failedReqs, prereq)
		}
	}

	if len(failedReqs) == 0 {
		if cliout.IsJSON() {
			return cliout.PrintJSON(map[string]any{"success": true, "message": "All requirements already satisfied"})
		}
		cliout.Success("All requirements already satisfied!")
		return nil
	}

	if !cliout.IsJSON() {
		cliout.Newline()
		cliout.Step(cliout.IconRefresh, "Refreshing environment PATH...")
	}
	if _, err = pathutil.RefreshPATH(); err != nil {
		if !cliout.IsJSON() {
			cliout.Warning("Failed to refresh PATH: %v", err)
		}
	} else if !cliout.IsJSON() {
		cliout.ItemSuccess("PATH refreshed successfully")
	}

	fixResults := make([]FixResult, 0, len(failedReqs))
	fixedCount := 0
	for _, prereq := range failedReqs {
		if !cliout.IsJSON() {
			cliout.Newline()
			cliout.Step(cliout.IconSearch, "Searching for %s...", prereq.Name)
		}

		fixResult := runner.checkPrerequisite(prereq)
		if fixResult.Fixed && fixResult.Satisfied {
			fixedCount++
		}
		fixResults = append(fixResults, fixResult)
	}

	if fixedCount > 0 {
		cacheDir := filepath.Join(filepath.Dir(azureYamlPath), ".azure", "cache")
		cacheManager, cacheErr := cache.NewCacheManagerWithOptions(cache.CacheOptions{Enabled: true, CacheDir: cacheDir})
		if cacheErr == nil {
			if err := cacheManager.ClearCache(); err != nil && !cliout.IsJSON() {
				cliout.Warning("Failed to clear cache: %v", err)
			}
		}
	}

	if !cliout.IsJSON() {
		cliout.Newline()
		cliout.Section(cliout.IconCheck, "Re-checking requirements...")
	}

	allResults := make([]ReqResult, 0, len(reqs))
	allSatisfied := true
	for _, prereq := range reqs {
		result := runner.checker.Check(prereq)
		allResults = append(allResults, result)
		if !result.Satisfied {
			allSatisfied = false
		}
	}

	if cliout.IsJSON() {
		return cliout.PrintJSON(map[string]any{
			"success":      fixedCount > 0,
			"fixed":        fixedCount,
			"total":        len(failedReqs),
			"allSatisfied": allSatisfied,
			"fixes":        fixResults,
			"results":      allResults,
		})
	}

	cliout.Newline()
	if fixedCount > 0 {
		cliout.Success("Fixed %d of %d issues!", fixedCount, len(failedReqs))
	} else {
		cliout.Warning("Could not automatically fix any issues")
	}

	if !allSatisfied {
		cliout.Newline()
		cliout.Info("%s Next steps:", cliout.IconBulb)
		cliout.Item("1. Run suggested install commands above")
		cliout.Item("2. Restart your terminal to refresh PATH")
		cliout.Item("3. Run 'azd app reqs' again to verify")
		return fmt.Errorf("not all requirements satisfied")
	}

	cliout.Newline()
	cliout.Success("All requirements now satisfied!")
	cliout.Newline()
	cliout.Info("ℹ️  Note: Tools may not be available in THIS terminal session")
	if runtime.GOOS == osWindows {
		cliout.Info("   To refresh PATH in your current PowerShell session, run:")
		cliout.Info("   %s$env:PATH = [System.Environment]::GetEnvironmentVariable(\"Path\",\"Machine\") + \";\" + [System.Environment]::GetEnvironmentVariable(\"Path\",\"User\")%s", cliout.Dim, cliout.Reset)
		cliout.Info("   Or simply restart your terminal")
	} else {
		cliout.Info("   To use the tools immediately, restart your terminal or source your shell profile")
	}

	return nil
}
