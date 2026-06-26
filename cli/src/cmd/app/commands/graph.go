package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/spf13/cobra"
)

const (
	graphOutputText = "text"
	graphOutputJSON = "json"
)

type graphOptions struct {
	output string
	writer io.Writer
}

type graphResult struct {
	Project string      `json:"project"`
	Nodes   []graphNode `json:"nodes"`
	Edges   []graphEdge `json:"edges"`
	Levels  [][]string  `json:"levels"`
}

type graphNode struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Level    int    `json:"level"`
	Host     string `json:"host,omitempty"`
	Language string `json:"language,omitempty"`
	Project  string `json:"project,omitempty"`
}

type graphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// NewGraphCommand creates the graph command.
func NewGraphCommand() *cobra.Command {
	opts := &graphOptions{writer: os.Stdout}
	cmd := &cobra.Command{
		Use:          "graph",
		Short:        "Show the service dependency graph",
		Long:         "Show services, resources, dependency edges, and startup levels from azure.yaml.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGraph(opts)
		},
	}
	cmd.Flags().StringVarP(&opts.output, "output", "o", graphOutputText, "Output format: text or json")
	return cmd
}

func runGraph(opts *graphOptions) error {
	if opts == nil {
		opts = &graphOptions{writer: os.Stdout}
	}
	if opts.writer == nil {
		opts.writer = os.Stdout
	}
	if opts.output == "" {
		opts.output = graphOutputText
	}
	if opts.output != graphOutputText && opts.output != graphOutputJSON {
		return fmt.Errorf("invalid output format: %s (must be text or json)", opts.output)
	}

	azureYamlPath, err := findAzureYaml()
	if err != nil {
		return err
	}
	projectDir := filepath.Dir(azureYamlPath)
	azureYaml, err := service.ParseAzureYaml(projectDir)
	if err != nil {
		return fmt.Errorf("failed to parse azure.yaml: %w", err)
	}
	if len(azureYaml.Services) == 0 && len(azureYaml.Resources) == 0 {
		return fmt.Errorf("azure.yaml has no services or resources to graph")
	}

	graph, err := service.BuildDependencyGraph(azureYaml.Services, azureYaml.Resources)
	if err != nil {
		return err
	}
	result := buildGraphResult(projectDir, graph)

	if opts.output == graphOutputJSON {
		enc := json.NewEncoder(opts.writer)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	printGraphText(opts.writer, result)
	return nil
}

func buildGraphResult(projectDir string, graph *service.DependencyGraph) graphResult {
	nodeNames := make([]string, 0, len(graph.Nodes))
	for name := range graph.Nodes {
		nodeNames = append(nodeNames, name)
	}
	sort.Strings(nodeNames)

	nodes := make([]graphNode, 0, len(nodeNames))
	for _, name := range nodeNames {
		node := graph.Nodes[name]
		out := graphNode{
			Name:  name,
			Type:  "service",
			Level: node.Level,
		}
		if node.IsResource {
			out.Type = "resource"
		}
		if node.Service != nil {
			out.Host = node.Service.Host
			out.Language = node.Service.Language
			out.Project = node.Service.Project
		}
		nodes = append(nodes, out)
	}

	edges := make([]graphEdge, 0)
	for from, deps := range graph.Edges {
		for _, dep := range deps {
			edges = append(edges, graphEdge{From: from, To: dep})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From == edges[j].From {
			return edges[i].To < edges[j].To
		}
		return edges[i].From < edges[j].From
	})

	return graphResult{
		Project: projectDir,
		Nodes:   nodes,
		Edges:   edges,
		Levels:  service.TopologicalSort(graph),
	}
}

func printGraphText(w io.Writer, result graphResult) {
	_, _ = fmt.Fprintf(w, "Dependency graph for %s\n\n", result.Project)

	if len(result.Levels) == 0 {
		_, _ = fmt.Fprintln(w, "No service startup levels found.")
	} else {
		_, _ = fmt.Fprintln(w, "Startup levels:")
		for i, level := range result.Levels {
			_, _ = fmt.Fprintf(w, "  Level %d: %s\n", i, joinGraphNames(level))
		}
	}

	_, _ = fmt.Fprintln(w, "\nNodes:")
	for _, node := range result.Nodes {
		detail := node.Type
		if node.Language != "" {
			detail += ", " + node.Language
		}
		if node.Host != "" {
			detail += ", " + node.Host
		}
		_, _ = fmt.Fprintf(w, "  - %s (%s, level %d)\n", node.Name, detail, node.Level)
	}

	_, _ = fmt.Fprintln(w, "\nEdges:")
	if len(result.Edges) == 0 {
		_, _ = fmt.Fprintln(w, "  No dependency edges.")
		return
	}
	for _, edge := range result.Edges {
		_, _ = fmt.Fprintf(w, "  %s -> %s\n", edge.From, edge.To)
	}
}

func joinGraphNames(names []string) string {
	if len(names) == 0 {
		return "(none)"
	}
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	return strings.Join(sorted, ", ")
}
