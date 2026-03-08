# CLI Output Styling Libraries - Go vs JavaScript/TypeScript

Research into how popular CLIs like pnpm and Astro achieve polished terminal output, and equivalent Go libraries for similar results.

**Date**: December 20, 2025  
**Status**: Research Complete

---

## Overview

This research compares the terminal output styling approaches used by pnpm (Node.js) and Astro (Node.js/TypeScript) with available Go libraries to achieve similar polished CLI output in azd-app.

---

## Popular CLI Examples

### pnpm CLI Output Characteristics

- **Background color badges** with padding (e.g., `WARN`, `ERROR`)
- **Colored prefixes** for lifecycle scripts using a color wheel
- **Progress indicators** with repeated characters (`+`, `-`)
- **Table formatting** with Unicode box-drawing characters
- **Text truncation** for long output
- **Columnar layouts** for package lists

**Key Libraries**:
- `chalk` - Terminal colors and styling
- `@zkochan/table` - Table formatting
- `cli-truncate` - Text truncation
- `cli-columns` - Columnar output
- `archy` - Tree-like output

### Astro CLI Output Characteristics

- **Badge formatting** with background colors (`astro`, version badges)
- **Unicode symbols** for visual elements
- **Spinner animations** for loading states
- **Table formatting** with borders
- **Color-coded messages** by severity
- **Box-drawing characters** for borders

**Key Libraries**:
- `piccolore` (modern chalk alternative) - Terminal colors
- `yocto-spinner` - Loading spinners
- Custom table formatting with column width calculations
- Unicode box-drawing characters (`┌`, `─`, `│`, etc.)

---

## Go Equivalent Libraries

### 1. Color & Styling Libraries

#### `fatih/color` ⭐ (Already in azd-app)
- **GitHub**: https://github.com/fatih/color
- **Purpose**: Terminal color and styling (most popular Go color library)
- **Similar to**: chalk, piccolore
- **Features**:
  - Simple API: `color.Red()`, `color.Bold()`, etc.
  - Color mixing: `color.New(color.FgRed, color.Bold).Println()`
  - Auto-detection of color support
  - 16 basic colors + bright variants

**Example**:
```go
color.Red("Error: %s", message)
color.Green("✓ Success")
color.Yellow("⚠ Warning")
```

#### `charmbracelet/lipgloss` ⭐ (Already in azd-app)
- **GitHub**: https://github.com/charmbracelet/lipgloss
- **Purpose**: Advanced styling, layout, and rendering
- **Similar to**: CSS-in-JS for terminal
- **Features**:
  - Style definitions (like CSS classes)
  - Borders and box drawing
  - Alignment and padding
  - Table layouts
  - Color gradients

**Example**:
```go
style := lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("#FAFAFA")).
    Background(lipgloss.Color("#7D56F4")).
    Padding(0, 1)

fmt.Println(style.Render("Hello, World!"))
```

### 2. Table Formatting

#### `jedib0t/go-pretty/v6/table`
- **GitHub**: https://github.com/jedib0t/go-pretty
- **Purpose**: Pretty tables with styling
- **Similar to**: @zkochan/table
- **Features**:
  - Multiple table styles
  - Column alignment
  - Custom colors
  - Auto-sizing
  - CSV/Markdown export

**Example**:
```go
t := table.NewWriter()
t.SetStyle(table.StyleLight)
t.AppendHeader(table.Row{"Package", "Version", "Status"})
t.AppendRow(table.Row{"react", "18.2.0", "✓ installed"})
fmt.Println(t.Render())
```

#### `olekukonko/tablewriter`
- **GitHub**: https://github.com/olekukonko/tablewriter
- **Purpose**: ASCII table generation
- **Features**: Simpler than go-pretty, good for basic tables

### 3. Progress Indicators & Spinners

#### `charmbracelet/bubbles` (Charm ecosystem)
- **GitHub**: https://github.com/charmbracelet/bubbles
- **Purpose**: TUI components (spinner, progress bar, etc.)
- **Similar to**: yocto-spinner
- **Features**:
  - Animated spinners
  - Progress bars
  - Text input
  - Viewport scrolling

#### `schollz/progressbar/v3`
- **GitHub**: https://github.com/schollz/progressbar
- **Purpose**: Simple progress bars
- **Features**: 
  - Customizable progress bars
  - Percentage display
  - ETA calculation

#### `briandowns/spinner`
- **GitHub**: https://github.com/briandowns/spinner
- **Purpose**: CLI spinners
- **Features**:
  - 70+ spinner styles
  - Custom messages
  - Color support

### 4. Text Formatting & Layout

#### `muesli/reflow` ⭐ (Already in azd-app)
- **GitHub**: https://github.com/muesli/reflow
- **Purpose**: Text wrapping, padding, indentation
- **Features**:
  - Word wrap
  - Padding
  - Indentation
  - Truncation

#### `charmbracelet/glamour` ⭐ (Already in azd-app)
- **GitHub**: https://github.com/charmbracelet/glamour
- **Purpose**: Markdown rendering in terminal
- **Features**: Styled markdown output for help text

---

## azd-app Current Implementation

### Already Using (from go.mod)

✅ **`fatih/color`** v1.18.0 - Basic colors  
✅ **`charmbracelet/lipgloss`** v1.1.1 - Advanced styling  
✅ **`charmbracelet/glamour`** v0.10.0 - Markdown rendering  
✅ **`muesli/reflow`** v0.3.0 - Text formatting  
✅ **Custom ANSI codes** - Direct escape sequences in `output/output.go`

### Current Output Features

From `cli/src/internal/output/output.go`:

1. **ANSI Color Constants**: Direct escape sequences for colors
2. **Unicode Symbols**: ✓, ✗, ⚠, ℹ, →, • with ASCII fallbacks
3. **Styled Functions**: 
   - `Success()`, `Error()`, `Warning()`, `Info()`
   - `Header()`, `Section()`, `Divider()`
   - `Label()`, `Highlight()`, `Emphasize()`
4. **Progress Bar**: Simple character-based (`█` filled, `░` empty)
5. **Status Badges**: Color-coded status (healthy, warning, error)
6. **Unicode Detection**: Falls back to ASCII on Windows/non-UTF8

### Service Logger Color Wheel

From `cli/src/internal/service/logger.go`:

```go
var colorCodes = []string{
    "\033[36m", // Cyan
    "\033[33m", // Yellow
    "\033[35m", // Magenta
    "\033[32m", // Green
    "\033[34m", // Blue
    "\033[31m", // Red
    // ... bright variants
}
```

Similar to pnpm's lifecycle script prefixes.

---

## Recommended Enhancements

### 1. Badge Styles (pnpm-style)

Use lipgloss for background color badges:

```go
var (
    AstroBadge = lipgloss.NewStyle().
        Background(lipgloss.Color("#22C55E")).
        Foreground(lipgloss.Color("#000")).
        Bold(true).
        Padding(0, 1).
        Render(" azd ")
    
    WarnBadge = lipgloss.NewStyle().
        Background(lipgloss.Color("#FACC15")).
        Foreground(lipgloss.Color("#000")).
        Bold(true).
        Padding(0, 1).
        Render(" WARN ")
)
```

### 2. Enhanced Progress Bars

Use colored characters with lipgloss:

```go
func ProgressBar(current, total int, width int) string {
    green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E"))
    gray := lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))
    
    percent := float64(current) / float64(total)
    filled := int(percent * float64(width))
    
    bar := green.Render(strings.Repeat("█", filled)) +
           gray.Render(strings.Repeat("░", width-filled))
    
    return fmt.Sprintf("[%s] %d%%", bar, int(percent*100))
}
```

### 3. Styled Tables

Add `go-pretty/table` for better table formatting:

```go
t := table.NewWriter()
t.SetStyle(table.StyleLight)
t.Style().Color.Header = table.ColorOptions{
    Foreground: lipgloss.Color("#60A5FA"),
}
t.AppendHeader(table.Row{"Service", "Status", "Port"})
t.AppendRow(table.Row{"api", "✓ healthy", "3000"})
fmt.Println(t.Render())
```

### 4. Box Messages (Astro-style)

Use lipgloss borders:

```go
func BoxMessage(title, content string) {
    border := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("#3B82F6")).
        Padding(1, 2)
    
    titleStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#60A5FA")).
        Bold(true)
    
    msg := fmt.Sprintf("%s\n\n%s", 
        titleStyle.Render(title), 
        content)
    fmt.Println(border.Render(msg))
}
```

### 5. Spinner for Long Operations

Add spinner for build/install operations:

```go
import "github.com/briandowns/spinner"

s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
s.Suffix = " Installing dependencies..."
s.Start()
// ... do work ...
s.Stop()
```

---

## Libraries to Add

### High Priority

1. **`jedib0t/go-pretty/v6`** - Better table formatting
   ```bash
   go get github.com/jedib0t/go-pretty/v6
   ```

2. **`briandowns/spinner`** - Loading spinners
   ```bash
   go get github.com/briandowns/spinner
   ```

### Optional

3. **`charmbracelet/bubbles`** - Full TUI components (if building interactive features)
   ```bash
   go get github.com/charmbracelet/bubbles
   ```

---

## Key Formatting Patterns from Research

### 1. Badge Formatting
- Use background colors with foreground contrast
- Add padding with thin space (`\u2009`) or lipgloss padding
- Bold text for emphasis

### 2. Color Wheels
- Cycle through colors for multiple services/packages
- Use bright variants for better visibility
- Keep consistent color assignment per service

### 3. Progress Indicators
- Use repeated characters (`█`, `░`, `▓`)
- Color-code by status (green = complete, gray = pending)
- Add percentage or count

### 4. Table Formatting
- Use Unicode box-drawing characters
- Color-code headers
- Auto-size columns
- Separate rows for readability

### 5. Status Indicators
- Map statuses to colors (green=ok, yellow=warn, red=error)
- Use symbols (✓, ✗, ⚠)
- Bold important status text

---

## Examples from pnpm Source

### Badge with Background Color
```typescript
const formatWarn = (message: string) => 
  `${chalk.bgYellow.black('\u2009WARN\u2009')} ${message}`
```

### Color Wheel for Services
```typescript
const colorWheel = ['cyan', 'magenta', 'blue', 'yellow', 'green', 'red']
let currentColor = 0
colorByPkg.set(wd, chalk[colorWheel[currentColor % NUM_COLORS]])
```

### Progress with Repeated Characters
```typescript
const ADDED_CHAR = chalk.green('+')
const REMOVED_CHAR = chalk.red('-')
printPlusesAndMinuses(width, added, removed)
```

---

## Examples from Astro Source

### Badge with Version
```typescript
const badge = bgGreen(bold(` astro `))
const version = green(`v${VERSION}`)
console.log(`${badge} ${version}`)
```

### Table with Unicode Borders
```typescript
const TABLE_OPTIONS = {
  border: {
    topLeft: '┌',
    topRight: '┐',
    horizontal: '─',
    vertical: '│',
    // ...
  }
}
```

---

## Comparison: Current vs Enhanced

### Current azd-app Output
```
azd app run
─────────────────────────────────

✓ Python dependencies installed
✓ Starting services...
  api: http://localhost:3000
  worker: running
```

### Enhanced Output (with lipgloss + go-pretty)
```
┌──────────────────────────────────────────────────────┐
│  azd  v1.0.0  ready in 1.2s                         │
└──────────────────────────────────────────────────────┘

Packages: +5 -1
█████░░░░░

┌─────────┬──────────┬────────────────────┐
│ Service │ Status   │ URL                │
├─────────┼──────────┼────────────────────┤
│ api     │ ✓ ready  │ http://localhost:3000 │
│ worker  │ ✓ ready  │ background         │
└─────────┴──────────┴────────────────────┘
```

---

## Implementation Priority

### Phase 1: Quick Wins (Already Have Libraries)
- [ ] Use lipgloss for badge styles
- [ ] Enhance progress bars with colored blocks
- [ ] Add box messages for important notices

### Phase 2: Add New Libraries
- [ ] Install `jedib0t/go-pretty/v6` for tables
- [ ] Install `briandowns/spinner` for operations
- [ ] Refactor existing output functions to use new styles

### Phase 3: Polish
- [ ] Consistent color palette across all output
- [ ] Add spinners to long-running operations
- [ ] Table output for list commands
- [ ] Box messages for errors/warnings

---

## References

- **pnpm Repository**: https://github.com/pnpm/pnpm
- **Astro Repository**: https://github.com/withastro/astro
- **Charm Libraries**: https://github.com/charmbracelet
- **go-pretty**: https://github.com/jedib0t/go-pretty
- **Awesome Go - Terminal**: https://github.com/avelino/awesome-go#terminal

---

## Conclusion

azd-app already has excellent foundations with `fatih/color` and `charmbracelet/lipgloss`. To achieve pnpm/Astro-level polish:

1. **Use lipgloss** for badges, boxes, and layouts
2. **Add go-pretty** for better tables
3. **Add spinner library** for long operations
4. **Standardize color palette** across all output
5. **Use Unicode box-drawing** for visual structure

The Go ecosystem has mature, well-maintained libraries that can match or exceed the polish of JavaScript CLIs.
