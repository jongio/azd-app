package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jongio/azd-core/cliout"
	types "github.com/jongio/azd-core/projecttype"
)

// depsCheckStatus reports whether one detected project has its dependencies installed.
type depsCheckStatus struct {
	Type      string `json:"type"`
	Dir       string `json:"dir,omitempty"`
	Path      string `json:"path,omitempty"`
	Manager   string `json:"manager,omitempty"`
	Marker    string `json:"marker"`
	Installed bool   `json:"installed"`
}

// depsCheckResult is the machine-readable summary for deps --check.
type depsCheckResult struct {
	Projects     []depsCheckStatus `json:"projects"`
	TotalChecked int               `json:"totalChecked"`
	Missing      int               `json:"missing"`
	AllInstalled bool              `json:"allInstalled"`
}

// depsPathIsDir reports whether path exists and is a directory.
func depsPathIsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// depsPathIsFile reports whether path exists and is a regular file.
func depsPathIsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// nodeDepsMarker returns the path checked for installed Node dependencies and whether it exists.
// Workspace children may share a node_modules at the workspace root, so that is checked too.
func nodeDepsMarker(p types.NodeProject) (string, bool) {
	primary := filepath.Join(p.Dir, "node_modules")
	if depsPathIsDir(primary) {
		return primary, true
	}
	if p.WorkspaceRoot != "" {
		shared := filepath.Join(p.WorkspaceRoot, "node_modules")
		if depsPathIsDir(shared) {
			return shared, true
		}
	}
	return primary, false
}

// pythonDepsMarker returns the path checked for an installed Python environment and whether it exists.
func pythonDepsMarker(p types.PythonProject) (string, bool) {
	venv := filepath.Join(p.Dir, ".venv")
	if depsPathIsDir(venv) {
		return venv, true
	}
	if alt := filepath.Join(p.Dir, "venv"); depsPathIsDir(alt) {
		return alt, true
	}
	return venv, false
}

// dotnetDepsMarker returns the restore marker checked for a .NET project and whether it exists.
// A successful restore writes obj/project.assets.json next to the project file.
func dotnetDepsMarker(p types.DotnetProject) (string, bool) {
	marker := filepath.Join(filepath.Dir(p.Path), "obj", "project.assets.json")
	return marker, depsPathIsFile(marker)
}

// buildDepsCheckResult inspects each detected project for its installed dependency marker.
func buildDepsCheckResult(projects DetectedProjects) depsCheckResult {
	result := depsCheckResult{}

	for _, p := range projects.Node {
		marker, installed := nodeDepsMarker(p)
		result.Projects = append(result.Projects, depsCheckStatus{
			Type:      "node",
			Dir:       p.Dir,
			Manager:   p.PackageManager,
			Marker:    marker,
			Installed: installed,
		})
	}
	for _, p := range projects.Python {
		marker, installed := pythonDepsMarker(p)
		result.Projects = append(result.Projects, depsCheckStatus{
			Type:      "python",
			Dir:       p.Dir,
			Manager:   p.PackageManager,
			Marker:    marker,
			Installed: installed,
		})
	}
	for _, p := range projects.Dotnet {
		marker, installed := dotnetDepsMarker(p)
		result.Projects = append(result.Projects, depsCheckStatus{
			Type:      "dotnet",
			Path:      p.Path,
			Marker:    marker,
			Installed: installed,
		})
	}

	result.TotalChecked = len(result.Projects)
	for _, s := range result.Projects {
		if !s.Installed {
			result.Missing++
		}
	}
	result.AllInstalled = result.Missing == 0
	return result
}

// runDepsCheck verifies that dependencies are installed for each detected service
// without installing anything. It returns an error when any service is missing its
// dependencies, which surfaces as a non-zero exit code for CI gating.
func runDepsCheck(opts *DepsOptions, w io.Writer) error {
	if opts == nil {
		opts = &DepsOptions{}
	}
	if w == nil {
		w = os.Stdout
	}

	searchRoot, err := getSearchRoot()
	if err != nil {
		return handleDepsError(err, "failed to determine search root")
	}

	nodeProjects, pythonProjects, dotnetProjects, err := detectProjectsFromAzureYaml(searchRoot)
	if err != nil {
		return handleDepsError(err, "failed to detect projects from azure.yaml")
	}

	projects := DetectedProjects{Node: nodeProjects, Python: pythonProjects, Dotnet: dotnetProjects}
	if len(opts.Services) > 0 {
		projects = filterDetectedProjectsByService(projects, opts.Services, searchRoot)
	}

	if projects.Total() == 0 {
		if cliout.IsJSON() {
			return cliout.PrintJSON(depsCheckResult{Projects: []depsCheckStatus{}, AllInstalled: true})
		}
		if len(opts.Services) > 0 {
			_, _ = fmt.Fprintf(w, "No projects found matching services: %v\n", opts.Services)
		} else {
			_, _ = fmt.Fprintln(w, msgNoProjectsDetected)
		}
		return nil
	}

	result := buildDepsCheckResult(projects)

	if cliout.IsJSON() {
		if err := cliout.PrintJSON(result); err != nil {
			return err
		}
	} else {
		renderDepsCheckText(w, result, searchRoot)
	}

	if result.Missing > 0 {
		return fmt.Errorf("%d of %d service(s) missing dependencies; run 'azd app deps' to install", result.Missing, result.TotalChecked)
	}
	return nil
}

// renderDepsCheckText prints a human-readable dependency check report.
func renderDepsCheckText(w io.Writer, result depsCheckResult, searchRoot string) {
	_, _ = fmt.Fprintln(w, "Dependency check")
	for _, s := range result.Projects {
		location := s.Dir
		if location == "" {
			location = s.Path
		}
		if rel, err := filepath.Rel(searchRoot, location); err == nil && rel != "." {
			location = rel
		}
		label := s.Type
		if s.Manager != "" {
			label = fmt.Sprintf("%s/%s", s.Type, s.Manager)
		}
		status := "installed"
		if !s.Installed {
			status = "missing"
		}
		_, _ = fmt.Fprintf(w, "  [%s] %s (%s)\n", status, location, label)
	}
	if result.Missing > 0 {
		_, _ = fmt.Fprintf(w, "%d of %d service(s) missing dependencies. Run 'azd app deps' to install.\n", result.Missing, result.TotalChecked)
	} else {
		_, _ = fmt.Fprintf(w, "All %d service(s) have dependencies installed.\n", result.TotalChecked)
	}
}
