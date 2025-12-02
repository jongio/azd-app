# Navigation Structure Specification

## Overview

This document defines the navigation structure for the azd-app marketing website, including main navigation, documentation sidebar, and footer navigation. It establishes the information architecture and content hierarchy.

---

## 1. Main Navigation (Header)

### Structure

| Order | Label        | Path           | Type      | Notes                         |
|-------|--------------|----------------|-----------|-------------------------------|
| 1     | Home         | /              | Internal  | Landing page                  |
| 2     | Quick Start  | /quick-start   | Internal  | Getting started guide         |
| 3     | MCP Server   | /mcp-server    | Internal  | **Prominent** - AI feature    |
| 4     | Guided Tour  | /tour          | Internal  | Interactive walkthrough       |
| 5     | Reference    | /reference     | Internal  | CLI reference documentation   |

### MCP Server Prominence

The MCP Server link is the key differentiator and should be visually prominent:

```typescript
const mainNavigation: NavigationItem[] = [
  { label: 'Home', href: '/' },
  { label: 'Quick Start', href: '/quick-start' },
  { 
    label: 'MCP Server', 
    href: '/mcp-server',
    isProminent: true,
    badge: 'AI',
    icon: 'sparkles'
  },
  { label: 'Guided Tour', href: '/tour', icon: 'map' },
  { label: 'Reference', href: '/reference', icon: 'book' },
];
```

---

## 2. Documentation Sidebar

### Hierarchy

```
📚 Getting Started
├── Quick Start
├── Installation
├── Configuration
└── Upgrading

🤖 MCP Server [AI]
├── Overview
├── Setup & Requirements
├── Available Tools
├── Integration Guide
└── Troubleshooting

📦 Commands
├── run
├── logs
├── health
├── deps
├── info
├── version
├── notifications
├── reqs
└── mcp

🎯 Guided Tour
├── Introduction
├── First Steps
├── Working with Services
├── Environment Variables
└── Advanced Features

📖 Reference
├── CLI Reference
├── Configuration Schema
├── Environment Variables
├── Exit Codes
└── FAQ
```

### Data Structure

```typescript
interface SidebarSection {
  id: string;
  title: string;
  icon: string;
  badge?: { text: string; variant: 'new' | 'beta' | 'deprecated' };
  defaultExpanded: boolean;
  items: SidebarItem[];
}

interface SidebarItem {
  label: string;
  href: string;
  badge?: { text: string; variant: 'new' | 'beta' | 'deprecated' };
}

const sidebarNavigation: SidebarSection[] = [
  {
    id: 'getting-started',
    title: 'Getting Started',
    icon: 'rocket',
    defaultExpanded: true,
    items: [
      { label: 'Quick Start', href: '/docs/quick-start' },
      { label: 'Installation', href: '/docs/installation' },
      { label: 'Configuration', href: '/docs/configuration' },
      { label: 'Upgrading', href: '/docs/upgrading' },
    ],
  },
  {
    id: 'mcp-server',
    title: 'MCP Server',
    icon: 'robot',
    badge: { text: 'AI', variant: 'new' },
    defaultExpanded: true,
    items: [
      { label: 'Overview', href: '/docs/mcp/overview' },
      { label: 'Setup & Requirements', href: '/docs/mcp/setup' },
      { label: 'Available Tools', href: '/docs/mcp/tools' },
      { label: 'Integration Guide', href: '/docs/mcp/integration' },
      { label: 'Troubleshooting', href: '/docs/mcp/troubleshooting' },
    ],
  },
  {
    id: 'commands',
    title: 'Commands',
    icon: 'terminal',
    defaultExpanded: true,
    items: [
      { label: 'run', href: '/docs/commands/run' },
      { label: 'logs', href: '/docs/commands/logs' },
      { label: 'health', href: '/docs/commands/health' },
      { label: 'deps', href: '/docs/commands/deps' },
      { label: 'info', href: '/docs/commands/info' },
      { label: 'version', href: '/docs/commands/version' },
      { label: 'notifications', href: '/docs/commands/notifications' },
      { label: 'reqs', href: '/docs/commands/reqs' },
      { label: 'mcp', href: '/docs/commands/mcp' },
    ],
  },
  {
    id: 'tour',
    title: 'Guided Tour',
    icon: 'map',
    defaultExpanded: false,
    items: [
      { label: 'Introduction', href: '/tour/introduction' },
      { label: 'First Steps', href: '/tour/first-steps' },
      { label: 'Working with Services', href: '/tour/services' },
      { label: 'Environment Variables', href: '/tour/env-vars' },
      { label: 'Advanced Features', href: '/tour/advanced' },
    ],
  },
  {
    id: 'reference',
    title: 'Reference',
    icon: 'book',
    defaultExpanded: false,
    items: [
      { label: 'CLI Reference', href: '/reference/cli' },
      { label: 'Configuration Schema', href: '/reference/schema' },
      { label: 'Environment Variables', href: '/reference/env' },
      { label: 'Exit Codes', href: '/reference/exit-codes' },
      { label: 'FAQ', href: '/reference/faq' },
    ],
  },
];
```

---

## 3. Footer Navigation

### Structure

```typescript
const footerNavigation: FooterSection[] = [
  {
    title: 'Resources',
    links: [
      { label: 'Quick Start', href: '/quick-start' },
      { label: 'Installation', href: '/docs/installation' },
      { label: 'Configuration', href: '/docs/configuration' },
      { label: 'Changelog', href: '/changelog', badge: 'New' },
    ],
  },
  {
    title: 'Documentation',
    links: [
      { label: 'CLI Reference', href: '/reference/cli' },
      { label: 'MCP Server', href: '/mcp-server', badge: 'AI' },
      { label: 'Guided Tour', href: '/tour' },
      { label: 'FAQ', href: '/reference/faq' },
    ],
  },
  {
    title: 'Community',
    links: [
      { label: 'GitHub', href: 'https://github.com/jongio/azd-app', external: true },
      { label: 'Discussions', href: 'https://github.com/jongio/azd-app/discussions', external: true },
      { label: 'Issues', href: 'https://github.com/jongio/azd-app/issues', external: true },
      { label: 'Contributing', href: '/contributing' },
    ],
  },
  {
    title: 'Legal',
    links: [
      { label: 'Privacy', href: '/privacy' },
      { label: 'Terms', href: '/terms' },
      { label: 'Code of Conduct', href: '/code-of-conduct' },
      { label: 'License', href: '/license' },
    ],
  },
];
```

---

## 4. Breadcrumb Paths

### Configuration

```typescript
const breadcrumbConfig: Record<string, BreadcrumbItem[]> = {
  '/': [],
  '/quick-start': [
    { label: 'Home', href: '/' },
    { label: 'Quick Start' },
  ],
  '/mcp-server': [
    { label: 'Home', href: '/' },
    { label: 'MCP Server' },
  ],
  '/docs/installation': [
    { label: 'Home', href: '/' },
    { label: 'Docs', href: '/docs' },
    { label: 'Installation' },
  ],
  '/docs/mcp/overview': [
    { label: 'Home', href: '/' },
    { label: 'Docs', href: '/docs' },
    { label: 'MCP Server', href: '/mcp-server' },
    { label: 'Overview' },
  ],
  '/docs/commands/run': [
    { label: 'Home', href: '/' },
    { label: 'Docs', href: '/docs' },
    { label: 'Commands', href: '/reference/cli' },
    { label: 'run' },
  ],
  // ... additional paths
};
```

---

## 5. Page Navigation (Prev/Next)

### Order Configuration

For documentation pages, define the reading order:

```typescript
const docPageOrder: string[] = [
  // Getting Started
  '/docs/quick-start',
  '/docs/installation',
  '/docs/configuration',
  '/docs/upgrading',
  
  // MCP Server
  '/docs/mcp/overview',
  '/docs/mcp/setup',
  '/docs/mcp/tools',
  '/docs/mcp/integration',
  '/docs/mcp/troubleshooting',
  
  // Commands
  '/docs/commands/run',
  '/docs/commands/logs',
  '/docs/commands/health',
  '/docs/commands/deps',
  '/docs/commands/info',
  '/docs/commands/version',
  '/docs/commands/notifications',
  '/docs/commands/reqs',
  '/docs/commands/mcp',
  
  // Reference
  '/reference/cli',
  '/reference/schema',
  '/reference/env',
  '/reference/exit-codes',
  '/reference/faq',
];

function getPageNavigation(currentPath: string): {
  prev?: PageLink;
  next?: PageLink;
} {
  const currentIndex = docPageOrder.indexOf(currentPath);
  if (currentIndex === -1) return {};
  
  return {
    prev: currentIndex > 0 
      ? { label: getPageLabel(docPageOrder[currentIndex - 1]), href: docPageOrder[currentIndex - 1] }
      : undefined,
    next: currentIndex < docPageOrder.length - 1
      ? { label: getPageLabel(docPageOrder[currentIndex + 1]), href: docPageOrder[currentIndex + 1] }
      : undefined,
  };
}
```

---

## 6. Active State Logic

### Matching Algorithm

```typescript
function isActiveLink(href: string, currentPath: string): boolean {
  // Exact match
  if (href === currentPath) return true;
  
  // Parent path match (for nested pages)
  if (href !== '/' && currentPath.startsWith(href)) {
    // Ensure it's a proper parent (not just prefix)
    const remaining = currentPath.slice(href.length);
    return remaining === '' || remaining.startsWith('/');
  }
  
  return false;
}

function getSectionForPath(currentPath: string): string | null {
  for (const section of sidebarNavigation) {
    for (const item of section.items) {
      if (isActiveLink(item.href, currentPath)) {
        return section.id;
      }
    }
  }
  return null;
}
```

---

## 7. Navigation Accessibility

### Keyboard Shortcuts (Optional Enhancement)

```typescript
const keyboardShortcuts: Record<string, string> = {
  'g h': '/',           // Go to Home
  'g q': '/quick-start', // Go to Quick Start
  'g m': '/mcp-server',  // Go to MCP Server
  'g t': '/tour',        // Go to Tour
  'g r': '/reference',   // Go to Reference
  '/': 'focus-search',   // Focus search input
};
```

### Link Announcements

```typescript
const linkAnnouncements: Record<string, string> = {
  '/mcp-server': 'MCP Server - AI-powered development assistance',
  '/quick-start': 'Quick Start - Get up and running in 5 minutes',
  '/tour': 'Guided Tour - Interactive walkthrough of features',
};
```

---

## 8. Mobile Navigation Order

On mobile, the navigation should prioritize:

1. **Home** - Always accessible
2. **MCP Server** - Key feature, highlighted
3. **Quick Start** - Entry point for new users
4. **Guided Tour** - Discovery path
5. **Reference** - For returning users

```typescript
const mobileNavOrder = [
  { ...mainNavigation[0] }, // Home
  { ...mainNavigation[2] }, // MCP Server (prominent)
  { ...mainNavigation[1] }, // Quick Start
  { ...mainNavigation[3] }, // Guided Tour
  { ...mainNavigation[4] }, // Reference
];
```

---

## 9. Search Integration

### Searchable Content

```typescript
interface SearchableItem {
  title: string;
  description: string;
  href: string;
  category: 'page' | 'command' | 'config' | 'concept';
  keywords: string[];
}

const searchIndex: SearchableItem[] = [
  {
    title: 'Quick Start',
    description: 'Get up and running with azd-app in 5 minutes',
    href: '/quick-start',
    category: 'page',
    keywords: ['start', 'begin', 'install', 'setup'],
  },
  {
    title: 'MCP Server',
    description: 'AI-powered development assistant for your local environment',
    href: '/mcp-server',
    category: 'page',
    keywords: ['ai', 'model context protocol', 'assistant', 'copilot'],
  },
  {
    title: 'run command',
    description: 'Start your local development environment',
    href: '/docs/commands/run',
    category: 'command',
    keywords: ['start', 'execute', 'launch'],
  },
  // ... additional items
];
```

---

## 10. URL Structure

### Path Conventions

| Path Pattern         | Description                      |
|----------------------|----------------------------------|
| `/`                  | Landing/home page                |
| `/quick-start`       | Quick start guide                |
| `/mcp-server`        | MCP Server landing               |
| `/tour`              | Guided tour index                |
| `/tour/[step]`       | Individual tour steps            |
| `/docs/[section]`    | Documentation section            |
| `/docs/commands/[cmd]`| Command documentation           |
| `/reference/[topic]` | Reference documentation          |
| `/changelog`         | Version changelog                |

### Redirects

```typescript
const redirects: Record<string, string> = {
  '/docs': '/docs/quick-start',
  '/getting-started': '/quick-start',
  '/ai': '/mcp-server',
  '/mcp': '/mcp-server',
  '/commands': '/reference/cli',
};
```
