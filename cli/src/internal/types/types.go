// Package types provides project type definitions.
// This is a thin wrapper re-exporting from azd-core/projecttype.
package types

import core "github.com/jongio/azd-core/projecttype"

// Re-export types from azd-core/projecttype.
type PythonProject = core.PythonProject
type NodeProject = core.NodeProject
type DotnetProject = core.DotnetProject
type AspireProject = core.AspireProject
type LogicAppProject = core.LogicAppProject
type FunctionAppProject = core.FunctionAppProject
