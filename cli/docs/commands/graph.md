# azd app graph

Show the service dependency graph.

## Synopsis

```
azd app graph [flags]
```

## Description

Show services, resources, dependency edges, and startup levels from `azure.yaml`.

A startup level groups the services that can start in parallel. Level 0 has no dependencies, level 1 depends only on level 0, and so on. This is the same ordering `azd app run` uses, so the graph explains why a service waits.

`text`, `json`, and `markdown` print to stdout. `mermaid`, `dot`, and `d2` emit a diagram you can drop into a README or an architecture doc. Any format can be redirected to a file with `--output-file`.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--format` | `-f` | string | `text` | Output format: `text`, `json`, `markdown`, `mermaid`, `dot`, or `d2` |
| `--output-file` | | string | | Write output to this file instead of stdout |
| `--focus` | | string | | Limit the graph to a service, its dependencies, and its dependents |
| `--services-only` | | bool | `false` | Show only services and service-to-service edges |

## Output Formats

| Format | Use |
|--------|-----|
| `text` | Human-readable summary in the terminal |
| `json` | Scripting and further processing |
| `markdown` | Tables for docs or issue comments |
| `mermaid` | Flowchart for a README or any Mermaid-aware renderer |
| `dot` | Graphviz source for `dot -Tpng` and friends |
| `d2` | D2 source for the D2 diagram toolchain |

## Filtering

`--focus <service>` narrows the graph to one service, everything it depends on, and everything that depends on it. Use it when a large project makes the full graph unreadable.

`--services-only` omits resource nodes such as databases and caches, leaving only service-to-service edges. Use it for architecture diagrams that need the app shape without managed dependencies.

The two flags compose, and both work with every output format.

## Examples

### Human-readable text

```bash
azd app graph
```

### Mermaid flowchart written to a file

```bash
azd app graph --format mermaid --output-file docs/services.mmd
```

### Graphviz DOT to stdout

```bash
azd app graph --format dot
```

Render it:

```bash
azd app graph --format dot | dot -Tpng -o services.png
```

### D2 diagram written to a file

```bash
azd app graph --format d2 --output-file docs/services.d2
```

### Markdown tables for docs or issue comments

```bash
azd app graph --format markdown
```

### Just one service and its connected nodes

```bash
azd app graph --focus api
```

### Service-only Mermaid diagram

```bash
azd app graph --services-only --format mermaid
```

## Exit Codes

| Code | Description |
|------|-------------|
| `0` | The graph was produced |
| `1` | `azure.yaml` could not be read, the format was unknown, `--focus` named an unknown service, or the output file could not be written |

## Related Commands

- [azd app run](run.md) - Run the development environment
- [azd app info](info.md) - Show information about running services
- [azd app env](env.md) - Print the resolved environment for a service
