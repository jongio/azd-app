package commands

import types "github.com/jongio/azd-core/projecttype"

// DetectedProjects groups detected project types that are typically processed together.
type DetectedProjects struct {
	Node   []types.NodeProject
	Python []types.PythonProject
	Dotnet []types.DotnetProject
	// ServicesByDir maps a project directory to the names of the azure.yaml
	// services that share it. Used to label a single (deduped) install with the
	// services it covers when several services point at one directory. Nil/empty
	// entries fall back to the directory name.
	ServicesByDir map[string][]string
}

// Total returns the total number of detected projects.
func (p DetectedProjects) Total() int {
	return len(p.Node) + len(p.Python) + len(p.Dotnet)
}
