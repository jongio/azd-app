package commands

import (
	"bytes"
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
	graphOutputText     = "text"
	graphOutputJSON     = "json"
	graphOutputMarkdown = "markdown"
	graphOutputMermaid  = "mermaid"
	graphOutputDOT      = "dot"
	graphOutputD2       = "d2"
)

type graphOptions struct {
	output     string
	outputFile string
	focus      string
	writer     io.Writer
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
		Use:   "graph",
		Short: "Show the service dependency graph",
		Long: `Show services, resources, dependency edges, and startup levels from azure.yaml.

Use --output to change the output. text, json, and markdown print to stdout.
mermaid, dot, and d2 emit a diagram you can drop into a README or an architecture doc.
Combine with --output-file to write the result to a file instead of stdout.

Pass --focus <service> to narrow the graph to one service, everything it depends
on, and everything that depends on it. This works with every output format.

Examples:
  # Human-readable text (default)
  azd app graph

  # Mermaid flowchart written to a file
  azd app graph --output mermaid --output-file docs/services.mmd

  # Graphviz DOT to stdout
  azd app graph --output dot

  # D2 diagram written to a file
  azd app graph --output d2 --output-file docs/services.d2

  # Markdown tables for docs or issue comments
  azd app graph --output markdown

  # Just the api service and its connected nodes
  azd app graph --focus api`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGraph(opts)
		},
	}
	cmd.Flags().StringVarP(&opts.output, "output", "o", graphOutputText, "Output format: text, json, markdown, mermaid, dot, or d2")
	cmd.Flags().StringVar(&opts.outputFile, "output-file", "", "Write output to this file instead of stdout")
	cmd.Flags().StringVar(&opts.focus, "focus", "", "Limit the graph to a service, its dependencies, and its dependents")
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
	switch opts.output {
	case graphOutputText, graphOutputJSON, graphOutputMarkdown, graphOutputMermaid, graphOutputDOT, graphOutputD2:
	default:
		return fmt.Errorf("invalid output format: %s (must be text, json, markdown, mermaid, dot, or d2)", opts.output)
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

	if opts.focus != "" {
		result, err = focusGraphResult(result, opts.focus)
		if err != nil {
			return err
		}
	}

	// When --output-file is set, buffer the rendered output and write it to disk.
	writer := opts.writer
	var buf *bytes.Buffer
	if opts.outputFile != "" {
		buf = &bytes.Buffer{}
		writer = buf
	}

	if err := renderGraph(writer, opts.output, result); err != nil {
		return err
	}

	if buf != nil {
		if err := os.WriteFile(opts.outputFile, buf.Bytes(), 0o644); err != nil {
			return fmt.Errorf("failed to write graph to %s: %w", opts.outputFile, err)
		}
		_, _ = fmt.Fprintf(opts.writer, "Wrote %s graph to %s\n", opts.output, opts.outputFile)
	}
	return nil
}

func renderGraph(w io.Writer, format string, result graphResult) error {
	switch format {
	case graphOutputJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	case graphOutputMarkdown:
		renderGraphMarkdown(w, result)
	case graphOutputMermaid:
		renderGraphMermaid(w, result)
	case graphOutputDOT:
		renderGraphDOT(w, result)
	case graphOutputD2:
		renderGraphD2(w, result)
	default:
		printGraphText(w, result)
	}
	return nil
}

// focusGraphResult reduces the graph to the focused node, everything it depends
// on (transitively), and everything that depends on it (transitively). Node
// metadata is preserved; startup levels are filtered to the focused nodes with
// empty levels removed. It returns an error if the focus name is not a node.
func focusGraphResult(result graphResult, focus string) (graphResult, error) {
	known := make(map[string]struct{}, len(result.Nodes))
	for _, n := range result.Nodes {
		known[n.Name] = struct{}{}
	}
	if _, ok := known[focus]; !ok {
		names := make([]string, 0, len(result.Nodes))
		for _, n := range result.Nodes {
			names = append(names, n.Name)
		}
		sort.Strings(names)
		return graphResult{}, fmt.Errorf("service %q not found in the graph. Available: %s",
			focus, strings.Join(names, ", "))
	}

	// deps maps a node to the nodes it depends on; dependents is the reverse.
	deps := make(map[string][]string)
	dependents := make(map[string][]string)
	for _, e := range result.Edges {
		deps[e.From] = append(deps[e.From], e.To)
		dependents[e.To] = append(dependents[e.To], e.From)
	}

	keep := map[string]struct{}{focus: {}}
	collectReachable(focus, deps, keep)
	collectReachable(focus, dependents, keep)

	nodes := make([]graphNode, 0, len(keep))
	for _, n := range result.Nodes {
		if _, ok := keep[n.Name]; ok {
			nodes = append(nodes, n)
		}
	}

	edges := make([]graphEdge, 0, len(result.Edges))
	for _, e := range result.Edges {
		_, okFrom := keep[e.From]
		_, okTo := keep[e.To]
		if okFrom && okTo {
			edges = append(edges, e)
		}
	}

	levels := make([][]string, 0, len(result.Levels))
	for _, level := range result.Levels {
		filtered := make([]string, 0, len(level))
		for _, name := range level {
			if _, ok := keep[name]; ok {
				filtered = append(filtered, name)
			}
		}
		if len(filtered) > 0 {
			levels = append(levels, filtered)
		}
	}

	return graphResult{
		Project: result.Project,
		Nodes:   nodes,
		Edges:   edges,
		Levels:  levels,
	}, nil
}

// collectReachable does a depth-first walk over adj starting at start, adding
// every reachable node to keep. Nodes already in keep are skipped, so cycles
// terminate.
func collectReachable(start string, adj map[string][]string, keep map[string]struct{}) {
	for _, next := range adj[start] {
		if _, ok := keep[next]; ok {
			continue
		}
		keep[next] = struct{}{}
		collectReachable(next, adj, keep)
	}
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

func renderGraphMarkdown(w io.Writer, result graphResult) {
	_, _ = fmt.Fprintf(w, "# Dependency graph for `%s`\n\n", escapeMarkdownCell(result.Project))

	_, _ = fmt.Fprintln(w, "## Startup levels")
	if len(result.Levels) == 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "No service startup levels found.")
	} else {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "| Level | Nodes |")
		_, _ = fmt.Fprintln(w, "| --- | --- |")
		for i, level := range result.Levels {
			_, _ = fmt.Fprintf(w, "| %d | %s |\n", i, escapeMarkdownCell(joinGraphNames(level)))
		}
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "## Nodes")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "| Name | Type | Level | Host | Language | Project |")
	_, _ = fmt.Fprintln(w, "| --- | --- | --- | --- | --- | --- |")
	for _, node := range result.Nodes {
		_, _ = fmt.Fprintf(
			w,
			"| %s | %s | %d | %s | %s | %s |\n",
			escapeMarkdownCell(node.Name),
			escapeMarkdownCell(node.Type),
			node.Level,
			escapeMarkdownCell(markdownValue(node.Host)),
			escapeMarkdownCell(markdownValue(node.Language)),
			escapeMarkdownCell(markdownValue(node.Project)),
		)
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "## Dependencies")
	_, _ = fmt.Fprintln(w)
	if len(result.Edges) == 0 {
		_, _ = fmt.Fprintln(w, "No dependency edges.")
		return
	}
	_, _ = fmt.Fprintln(w, "| From | To |")
	_, _ = fmt.Fprintln(w, "| --- | --- |")
	for _, edge := range result.Edges {
		_, _ = fmt.Fprintf(w, "| %s | %s |\n", escapeMarkdownCell(edge.From), escapeMarkdownCell(edge.To))
	}
}

func markdownValue(value string) string {
	if value == "" {
		return "n/a"
	}
	return value
}

func escapeMarkdownCell(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"|", "\\|",
		"\r\n", " ",
		"\n", " ",
		"\r", " ",
	)
	return replacer.Replace(value)
}

func joinGraphNames(names []string) string {
	if len(names) == 0 {
		return "(none)"
	}
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	return strings.Join(sorted, ", ")
}

// mermaidNodeIDs builds a map from node name to a Mermaid-safe identifier.
// Mermaid node IDs must be alphanumeric or underscore, so unsafe characters are
// replaced and a stable index suffix guarantees uniqueness.
func mermaidNodeIDs(nodes []graphNode) map[string]string {
	ids := make(map[string]string, len(nodes))
	for i, node := range nodes {
		var b strings.Builder
		b.WriteByte('n')
		for _, r := range node.Name {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
				b.WriteRune(r)
			} else {
				b.WriteByte('_')
			}
		}
		fmt.Fprintf(&b, "_%d", i)
		ids[node.Name] = b.String()
	}
	return ids
}

// escapeMermaidLabel escapes a label for use inside a double-quoted Mermaid node.
func escapeMermaidLabel(s string) string {
	replacer := strings.NewReplacer(
		"\"", "#quot;",
		"\n", " ",
		"[", "#91;",
		"]", "#93;",
		"{", "#123;",
		"}", "#125;",
	)
	return replacer.Replace(s)
}

func renderGraphMermaid(w io.Writer, result graphResult) {
	ids := mermaidNodeIDs(result.Nodes)

	_, _ = fmt.Fprintln(w, "flowchart TD")
	for _, node := range result.Nodes {
		label := node.Name
		if node.Type != "" {
			label += " (" + node.Type + ")"
		}
		id := ids[node.Name]
		// Resources use a rounded shape to distinguish them from services.
		if node.Type == "resource" {
			_, _ = fmt.Fprintf(w, "    %s([\"%s\"])\n", id, escapeMermaidLabel(label))
		} else {
			_, _ = fmt.Fprintf(w, "    %s[\"%s\"]\n", id, escapeMermaidLabel(label))
		}
	}
	for _, edge := range result.Edges {
		from, okFrom := ids[edge.From]
		to, okTo := ids[edge.To]
		if !okFrom || !okTo {
			continue
		}
		_, _ = fmt.Fprintf(w, "    %s --> %s\n", from, to)
	}
}

// escapeDOTString escapes a string for use inside a double-quoted Graphviz DOT literal.
func escapeDOTString(s string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"\n", "\\n",
	)
	return replacer.Replace(s)
}

func renderGraphDOT(w io.Writer, result graphResult) {
	_, _ = fmt.Fprintln(w, "digraph services {")
	_, _ = fmt.Fprintln(w, "    rankdir=LR;")
	for _, node := range result.Nodes {
		label := node.Name
		if node.Type != "" {
			label += "\n(" + node.Type + ")"
		}
		shape := "box"
		if node.Type == "resource" {
			shape = "ellipse"
		}
		_, _ = fmt.Fprintf(w, "    \"%s\" [label=\"%s\", shape=%s];\n", escapeDOTString(node.Name), escapeDOTString(label), shape)
	}
	for _, edge := range result.Edges {
		_, _ = fmt.Fprintf(w, "    \"%s\" -> \"%s\";\n", escapeDOTString(edge.From), escapeDOTString(edge.To))
	}
	_, _ = fmt.Fprintln(w, "}")
}

// escapeD2Label makes a string safe to use inside a D2 double-quoted label.
// D2 double-quoted strings cannot contain a literal double quote, so quotes are
// swapped for single quotes and newlines are collapsed to spaces.
func escapeD2Label(s string) string {
	replacer := strings.NewReplacer(
		"\"", "'",
		"\n", " ",
	)
	return replacer.Replace(s)
}

func renderGraphD2(w io.Writer, result graphResult) {
	ids := mermaidNodeIDs(result.Nodes)

	_, _ = fmt.Fprintln(w, "direction: right")
	for _, node := range result.Nodes {
		label := node.Name
		if node.Type != "" {
			label += " (" + node.Type + ")"
		}
		id := ids[node.Name]
		// Resources use a cylinder to read like a datastore, matching how the
		// other diagram formats distinguish resources from services.
		if node.Type == "resource" {
			_, _ = fmt.Fprintf(w, "%s: \"%s\" {\n  shape: cylinder\n}\n", id, escapeD2Label(label))
		} else {
			_, _ = fmt.Fprintf(w, "%s: \"%s\"\n", id, escapeD2Label(label))
		}
	}
	for _, edge := range result.Edges {
		from, okFrom := ids[edge.From]
		to, okTo := ids[edge.To]
		if !okFrom || !okTo {
			continue
		}
		_, _ = fmt.Fprintf(w, "%s -> %s\n", from, to)
	}
}
