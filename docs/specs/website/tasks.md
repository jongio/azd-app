# Website Tasks

## Progress: 20/20 complete ✅

---

## Phase 1: Foundation

### Task 1: Initialize Astro Project
- **Status**: DONE
- **Agent**: Developer
- **Description**: Create Astro project in `/web` with Tailwind, MDX, content collections
- **Acceptance**: `pnpm dev` runs, builds successfully

### Task 2: GitHub Actions - Website Deploy
- **Status**: DONE
- **Agent**: DevOps
- **Description**: Single workflow for production deploy + PR previews
- **Acceptance**: Main deploys to Pages, PRs get preview URLs

### Task 3: GitHub Actions - Demo Template Sync
- **Status**: DONE
- **Agent**: DevOps
- **Description**: Workflow to sync cli/demo/ to jongio/azd-app-demo repo on merge
- **Acceptance**: Demo repo updates automatically, includes `.vscode/mcp.json`

### Task 4: Command Validation Script
- **Status**: DONE
- **Agent**: Developer
- **Description**: Build script that fails if any command in cli/docs/commands/ lacks documentation
- **Acceptance**: Build fails with clear error when command missing

---

## Phase 2: Layout & Components

### Task 5: Layout + Navigation
- **Status**: DONE
- **Agent**: Designer then Developer
- **Description**: Header with MCP Server prominent, sidebar, footer, dark/light toggle, mobile menu
- **Acceptance**: Responsive layout, theme toggle works

### Task 6: Code Block + Terminal Components
- **Status**: DONE
- **Agent**: Designer then Developer
- **Description**: Syntax highlighting, copy button, language indicator, terminal styling
- **Acceptance**: Works for bash, yaml, json, ts, python, go, csharp

### Task 7: Screenshot Component
- **Status**: DONE
- **Agent**: Designer then Developer
- **Description**: Display screenshots with lightbox, captions, dark/light variants
- **Acceptance**: Click to enlarge, loads appropriate theme variant

---

## Phase 3: Core Pages

### Task 8: Landing Page
- **Status**: DONE
- **Agent**: Designer then Developer
- **Description**: Hero featuring GitHub Copilot AI debugging, feature cards, demo template CTA
- **Acceptance**: Copilot is primary AI shown, `azd init -t jongio/azd-app-demo` prominent

### Task 9: Quick Start Page
- **Status**: DONE
- **Agent**: Designer then Developer
- **Description**: Streamlined install flow using demo template with "fix the bug" challenge
- **Acceptance**: 4-step install (azd, extension, template, debug with AI), completable in 5 minutes

### Task 10: Guided Tour (8 steps)
- **Status**: DONE
- **Agent**: Designer then Developer
- **Description**: Progressive tutorial ending with Copilot MCP setup step
- **Acceptance**: Progress persists in localStorage, step 8 covers MCP server setup

---

## Phase 4: MCP Server Section

### Task 11: MCP Overview Page
- **Status**: DONE
- **Agent**: Designer then Developer
- **Description**: Landing page explaining MCP server capabilities (AI debugging, service management, log analysis)
- **Acceptance**: Clear explanation of MCP, use cases, supported clients (Copilot, Cursor, Claude)

### Task 12: MCP Setup Guides
- **Status**: DONE
- **Agent**: Developer
- **Description**: Step-by-step setup for GitHub Copilot (first), Cursor, Claude Desktop
- **Acceptance**: Copilot guide most detailed, each has copy-paste `.vscode/mcp.json` config

### Task 13: AI Debugging Page
- **Status**: DONE
- **Agent**: Designer then Developer
- **Description**: How to use MCP with AI assistants for debugging (check health, view logs, etc.)
- **Acceptance**: Shows Copilot conversation example and outcome, links to prompts

### Task 14: MCP Tools + Prompt Library
- **Status**: DONE
- **Agent**: Designer then Developer
- **Description**: Tools reference page + copy-paste prompts for AI debugging + interactive demo
- **Acceptance**: All 10 MCP tools documented, 10+ prompts categorized, AI chat demo works

---

## Phase 5: Reference & Polish

### Task 15: CLI Reference Generator
- **Status**: DONE
- **Agent**: Developer
- **Description**: Script to auto-generate reference pages from cli/docs/ at build time
- **Acceptance**: All commands, flags, examples displayed, includes MCP command

### Task 16: Changelog Generator
- **Status**: DONE
- **Agent**: Developer
- **Description**: Script to parse cli/CHANGELOG.md and generate version history pages
- **Acceptance**: `/reference/changelog` shows full history, `/reference/whats-new` shows latest 3

### Task 17: Examples Gallery
- **Status**: DONE
- **Agent**: Designer then Developer
- **Description**: Node.js, Python, .NET, fullstack examples with Copilot debugging tips
- **Acceptance**: Each example shows how to debug with Copilot

### Task 18: Search Integration
- **Status**: DONE
- **Agent**: Developer
- **Description**: Add Pagefind for client-side search with keyboard shortcut
- **Acceptance**: Searches commands, AI prompts, content; `/` or `Cmd+K` opens

### Task 19: Playwright Screenshot Tests
- **Status**: DONE
- **Agent**: Developer
- **Description**: Extend dashboard tests to capture screenshots for website
- **Acceptance**: Dashboard + CLI screenshots in light/dark modes

---

## Phase 6: Demo Template

### Task 20: Create Demo Project
- **Status**: DONE
- **Agent**: Developer
- **Description**: Create cli/demo/ with API bug, `.vscode/mcp.json`, `.gitignore`, `.gitattributes`, LICENSE
- **Acceptance**: Running shows error in logs, mcp.json enables Copilot MCP, smoke test passes
