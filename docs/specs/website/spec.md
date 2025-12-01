# azd-app Marketing Website

## Overview

A GitHub Pages marketing site with interactive tutorials for azd-app. Auto-generated screenshots from Playwright tests. PR previews for testing changes.

## Goals

1. Marketing landing page showcasing features
2. **MCP server integration** - Enable AI assistants to debug and manage services
3. **Demo template repo** - One-click `azd init` experience with intentional bug
4. Multiple learning paths for different user types
5. Auto-generated CLI reference from source
6. Auto-generated screenshots from test projects
7. PR preview deployments for testing

---

## Site Structure

### Navigation Model

```
┌──────────────────────────────────────────────────────────────────────┐
│  Home  │  Quick Start  │  MCP Server  │  Guided Tour  │  Reference │
└──────────────────────────────────────────────────────────────────────┘
```

### Pages

| Section | Page | Purpose | Audience |
|---------|------|---------|----------|
| **Home** | `/` | Hero, features, social proof, install CTA | Everyone |
| **Quick Start** | `/quickstart` | 5-minute install → first `azd app run` | Impatient devs |
| **Guided Tour** | `/tour/` | Step-by-step learning path | New users |
| | `/tour/1-install` | Install azd + extension | |
| | `/tour/2-reqs` | Check requirements | |
| | `/tour/3-deps` | Install dependencies | |
| | `/tour/4-run` | Run your first app | |
| | `/tour/5-dashboard` | Explore the dashboard | |
| | `/tour/6-logs` | View and filter logs | |
| | `/tour/7-health` | Monitor service health | |
| | `/tour/8-mcp` | MCP server integration | |
| **MCP Server** | `/mcp/` | MCP capabilities & integrations | Developers |
| | `/mcp/overview` | What is MCP, why it matters | |
| | `/mcp/setup` | Configure Copilot, Cursor, Claude | |
| | `/mcp/ai-debugging` | Debug with AI assistants | |
| | `/mcp/tools` | MCP tools reference | |
| | `/mcp/prompts` | Effective prompts for AI | |
| **Reference** | `/reference/` | Complete CLI docs | Power users |
| | `/reference/commands` | All commands (auto-generated) | |
| | `/reference/config` | azure.yaml configuration | |
| | `/reference/env-vars` | Environment variables | |
| | `/reference/troubleshooting` | Common issues & fixes | |
| | `/reference/changelog` | Full version history | |
| | `/reference/whats-new` | Latest release highlights | |
| **Examples** | `/examples/` | Real-world project examples | All users |
| | `/examples/nodejs` | Node.js project setup | |
| | `/examples/python` | Python project setup | |
| | `/examples/dotnet` | .NET / Aspire setup | |
| | `/examples/fullstack` | Multi-service app | |

### User Journeys

```
New User:        Home → Quick Start → Guided Tour → Examples
Evaluating:      Home → Features → Examples → Quick Start  
MCP/AI User:     Home → MCP Server → Setup → AI Debugging
Power User:      Home → Reference → Config
Troubleshooting: Home → MCP Server (ask AI!) → Reference
```

---

## Key Features

### 1. MCP Server (First-Class)
The Model Context Protocol (MCP) server enables programmatic access to azd-app:

**Use Cases:**
- **AI Debugging**: Ask Copilot/Claude to diagnose issues
- **Service Management**: Start, stop, restart services via AI
- **Log Analysis**: Have AI analyze logs and find errors

**Supported Clients:**
- GitHub Copilot (VS Code) - *Recommended*
- Cursor
- Claude Desktop

**MCP Tools Available (10 total):**
| Category | Tool | What it does |
|----------|------|--------------|
| Observe | `get_services` | List all services with status |
| Observe | `get_service_logs` | Stream/filter logs |
| Observe | `get_project_info` | Project configuration |
| Operate | `run_services` | Start services |
| Operate | `stop_services` | Stop services |
| Operate | `restart_service` | Restart a service |
| Operate | `install_dependencies` | Install all deps |
| Config | `check_requirements` | Verify prerequisites |
| Config | `get_environment_variables` | View env vars |
| Config | `set_environment_variable` | Set env var |

### 2. Quick Start (5 minutes)
- Minimal steps: Install → Clone Template → Run → Debug with AI
- **One-click start**: `azd init -t jongio/azd-app-demo`
- Copy-paste commands with explanations
- **Built-in challenge**: "Fix the bug using AI"
- Link to Guided Tour for deeper learning

**Streamlined Installation Flow:**
```bash
# Step 1: Install Azure Developer CLI
# See: https://aka.ms/install-azd
# Quick install:
#   Windows:  winget install microsoft.azd
#   macOS:    brew tap azure/azd && brew install azd
#   Linux:    curl -fsSL https://aka.ms/install-azd.sh | bash

# Step 2: Enable extensions + install azd-app
azd config set alpha.extensions.enabled on
azd extension source add app https://raw.githubusercontent.com/jongio/azd-app/main/registry.json
azd extension install app

# Step 3: Clone demo template & run
azd init -t jongio/azd-app-demo
azd app run

# Step 4: Notice the API error, then ask Copilot to help fix it!
```

### 3. Demo Template Repository
A separate repo (`jongio/azd-app-demo`) auto-synced from this repo:

**Purpose:**
- Provides clean `azd init -t azd-app-demo` experience
- **Ships with an intentional error** for AI debugging demo
- Includes `mcp.json` for GitHub Copilot MCP integration
- Always works with latest azd-app version

**The Demo Error:**
- API service has a bug that causes intermittent 500 errors
- Error appears in logs: `"Error: Connection timeout to database"`
- The fix is simple (missing retry logic or wrong config)
- Perfect for demonstrating: "Ask Copilot why the API is failing"

**Demo Flow:**
1. User runs `azd init -t jongio/azd-app-demo && azd app run`
2. Dashboard shows API service as "unhealthy" or with errors
3. User asks Copilot: "Why is my API service failing?"
4. Copilot uses MCP to check logs, finds the error
5. Copilot suggests the fix
6. User applies fix, service becomes healthy

**Contents (synced from `cli/demo/`):**
```
azd-app-demo/
├── .gitignore
├── .gitattributes
├── LICENSE                 # MIT
├── azure.yaml              # Pre-configured for azd app
├── .vscode/
│   ├── mcp.json            # GitHub Copilot MCP config
│   └── settings.json       # VS Code settings
├── src/
│   ├── api/                # Python Flask API (with intentional bug)
│   └── web/                # Node.js frontend
├── README.md               # Quick start + "Fix the bug with AI" instructions
├── SOLUTION.md             # Spoiler: how to fix the bug (hidden by default)
└── SETUP.md                # Detailed setup guide
```

**.vscode/mcp.json contents:**
```json
{
  "servers": {
    "azd-app": {
      "command": "azd",
      "args": ["app", "mcp", "serve"]
    }
  }
}
```

**Sync Workflow:**
- GitHub Action syncs on merge to main
- Copies demo project to target repo
- Includes `.vscode/mcp.json`, `.gitignore`, `.gitattributes`, `LICENSE`
- Updates version references
- Runs smoke test to verify it works
- Auto-commits to demo repo

### 4. Guided Tour (Interactive)
- Progress tracker (Step 3 of 8)
- "Mark as complete" with localStorage
- Next/Previous navigation
- Estimated time per step
- Screenshots showing expected output
- "Try it yourself" prompts
- Expandable "Learn more" sections
- **Step 8 covers MCP server**: setup and AI debugging

### 5. CLI Reference (Auto-generated)
- **Source**: `cli/docs/commands/*.md` + `cli/docs/cli-reference.md`
- Command index with search/filter
- Each command page shows:
  - Synopsis
  - Flags table (sortable)
  - Examples with copy button
  - Related commands
- Versioned (shows CLI version)

### 6. Examples Gallery
- Filter by language/framework
- Each example shows:
  - Project structure
  - azure.yaml sample
  - Commands to run
  - Expected output screenshots
- Links to actual test projects in repo

### 7. Search
- Client-side search (Pagefind or similar)
- Searches commands, flags, examples
- Keyboard shortcut: `/` or `Cmd+K`

### 8. Helpful Extras
- **Cheat Sheet**: Single page with all commands (web page, not PDF)
- **AI Prompt Library**: Copy-paste prompts for common debugging tasks
- **What's New**: Auto-generated from `cli/CHANGELOG.md` (latest 3 releases)
- **Version History**: Full changelog with all releases, searchable
- **FAQ**: Common questions answered
- **Community**: Links to GitHub Issues, Discussions

---

## Components

### Code Blocks
- Syntax highlighting (bash, yaml, json, ts, python, go, csharp)
- Copy button with "Copied!" feedback
- Language indicator
- Optional filename header
- Optional line highlighting

### Terminal Component
- Simulated terminal look
- Shows command + output
- Optional: typing animation for demos

### Screenshot Component
- Auto-generated from Playwright
- Lightbox on click
- Dark/light mode variants
- Caption with context
- "View full size" option

### Command Card
- Used in reference index
- Shows: name, brief description, key flags
- Click to expand or navigate

### Progress Tracker
- For Guided Tour
- Shows current step, completed steps
- Persists in localStorage

### AI Chat Demo
- Interactive demo showing AI conversation
- Simulated GitHub Copilot responses (primary)
- Shows real MCP tool calls
- "Try this prompt" buttons

### Theme Toggle
- Light / Dark / System
- Persists preference
- Smooth transition

---

## Automation

### Command Sync Validation
- Build script scans `cli/docs/commands/*.md` for all commands
- Validates each command has a tutorial page in `web/src/pages/tour/` OR reference coverage
- **Build fails** if any command is missing
- Clear error message: "Missing documentation for command: notifications"

### CLI Reference Generation
- At build time, reads `cli/docs/cli-reference.md`
- Parses into structured data
- Generates `/reference/commands` pages automatically
- No manual sync needed

### Changelog Generation
- At build time, reads `cli/CHANGELOG.md`
- Parses markdown into structured release data
- Generates `/reference/changelog` (full history)
- Generates `/reference/whats-new` (latest 3 releases)
- Shows version number, date, and categorized changes
- Links to GitHub releases

### Screenshots
- Extend existing Playwright tests in `cli/dashboard/`
- Capture dashboard views (Resources, Console, health states)
- Capture CLI output examples
- Store in `web/public/screenshots/`
- Light + dark mode variants
- Generated in CI alongside tests

### Deployment
- **Production**: Merge to main → deploy to `jongio.github.io/azd-app/`
- **Preview**: PR opened → deploy to `jongio.github.io/azd-app/pr/<number>/`
- Single workflow handles both

### Demo Template Sync
- **Source**: `cli/demo/` (dedicated project with intentional bug)
- **Target**: `jongio/azd-app-demo` repo
- **Trigger**: Merge to main
- **Steps**:
  1. Copy project files to temp directory
  2. Include `.vscode/mcp.json` and `.vscode/settings.json`
  3. Include `.gitignore`, `.gitattributes`, `LICENSE`
  4. Add demo-specific README.md with "Fix the bug" challenge
  5. Update any version references
  6. Run `azd app reqs` and `azd app deps` to verify
  7. Push to demo repo
- **Smoke Test**: CI runs `azd init -t azd-app-demo` + `azd app run` on fresh machine
- **Error Validation**: Verify the intentional error appears in logs

---

## Technical Stack

- **Astro** + Tailwind + MDX
- **Pagefind** for search
- **Playwright** for screenshots (extend existing)
- **GitHub Pages** + Actions

---

## Folder Structure

```
web/
├── src/
│   ├── pages/
│   │   ├── index.astro              # Landing (AI featured prominently)
│   │   ├── quickstart.mdx           # 5-min guide
│   │   ├── mcp/
│   │   │   ├── index.astro          # MCP Server overview
│   │   │   ├── setup.mdx            # Configure Copilot, Cursor, Claude
│   │   │   ├── ai-debugging.mdx     # Debug with AI assistants
│   │   │   ├── tools.mdx            # MCP tools reference
│   │   │   └── prompts.mdx          # Prompt library
│   │   ├── tour/
│   │   │   ├── index.astro          # Tour overview
│   │   │   ├── 1-install.mdx
│   │   │   ├── 2-reqs.mdx
│   │   │   └── ...
│   │   ├── reference/
│   │   │   ├── index.astro          # Reference index
│   │   │   ├── commands/
│   │   │   │   └── [...slug].astro  # Dynamic from CLI docs
│   │   │   ├── config.mdx
│   │   │   ├── env-vars.mdx
│   │   │   ├── changelog.astro      # Full version history (from CHANGELOG.md)
│   │   │   └── whats-new.astro      # Latest 3 releases
│   │   └── examples/
│   │       ├── index.astro
│   │       ├── nodejs.mdx
│   │       └── ...
│   ├── components/
│   │   ├── Layout.astro
│   │   ├── Header.astro
│   │   ├── Sidebar.astro
│   │   ├── CodeBlock.astro
│   │   ├── Terminal.astro
│   │   ├── Screenshot.astro
│   │   ├── CommandCard.astro
│   │   ├── ProgressTracker.astro
│   │   ├── AIChatDemo.astro         # Interactive AI conversation demo
│   │   ├── PromptCard.astro         # Copy-paste prompt cards
│   │   └── Search.astro
│   ├── content/
│   │   └── config.ts               # Content collections
│   └── styles/
│       └── global.css
├── public/
│   └── screenshots/                 # Auto-generated
├── scripts/
│   ├── validate-commands.ts         # Build validation
│   ├── generate-reference.ts        # CLI docs → pages
│   └── generate-changelog.ts        # CHANGELOG.md → pages
├── astro.config.mjs
└── package.json
```

---

## Acceptance Criteria

1. Landing page features AI debugging with GitHub Copilot prominently
2. AI Debugging section has setup guides (Copilot first, then Cursor, Claude)
3. Interactive AI chat demo shows Copilot fixing the demo bug
4. Prompt library has copy-paste prompts for common tasks
5. Quick Start completable in under 5 minutes using demo template
6. `azd init -t jongio/azd-app-demo` works and shows intentional error
7. Demo repo includes `.vscode/mcp.json` for Copilot MCP integration
8. Demo repo auto-syncs from main repo on every merge
9. Guided Tour has 8 steps with progress tracking (including AI step)
10. CLI Reference auto-generated from cli/docs/
11. Search works across all content
12. Screenshots auto-generated in CI
13. PR previews work with URL in PR comment
14. Dark/light mode works
15. Mobile responsive
16. Build fails if command docs missing
