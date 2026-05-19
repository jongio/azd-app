package commands

import types "github.com/jongio/azd-core/projecttype"

// DetectedProjects groups detected project types that are typically processed together.
type DetectedProjects struct {
	Node   []types.NodeProject
	Python []types.PythonProject
	Dotnet []types.DotnetProject
}

// Total returns the total number of detected projects.
func (p DetectedProjects) Total() int {
	return len(p.Node) + len(p.Python) + len(p.Dotnet)
}
