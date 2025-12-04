````markdown
# Landing Page Component Specification

## Overview

The Landing Page is the primary marketing entry point for azd-app. It showcases AI-powered debugging with GitHub Copilot, highlights key features, and provides clear paths to getting started. The page is designed to convert visitors into users through compelling visuals and minimal friction CTAs.

---

## 1. Page Structure

```
LandingPage
├── Header (organism) [from header.md]
├── HeroSection (organism)
│   ├── HeroContent (molecule)
│   │   ├── HeroHeadline (atom)
│   │   ├── HeroSubheadline (atom)
│   │   ├── HeroCTAGroup (molecule)
│   │   │   ├── PrimaryButton (atom)
│   │   │   └── SecondaryButton (atom)
│   │   └── SocialProofBadge (atom)
│   └── HeroIllustration (molecule)
│       └── AIChatDemo (organism) [interactive]
├── FeaturesSection (organism)
│   ├── SectionHeader (molecule)
│   └── FeatureCardsGrid (molecule)
│       └── FeatureCard (molecule) × 6
│           ├── FeatureIcon (atom)
│           ├── FeatureTitle (atom)
│           └── FeatureDescription (atom)
├── MCPSection (organism)
│   ├── SectionHeader (molecule)
│   ├── MCPBenefits (molecule)
│   │   └── BenefitItem (atom) × 3
│   └── MCPDemoTerminal (molecule)
├── DemoTemplateSection (organism)
│   ├── SectionHeader (molecule)
│   ├── TerminalPreview (molecule) [from terminal.md]
│   │   └── CopyButton (atom)
│   └── DemoSteps (molecule)
│       └── DemoStep (atom) × 4
├── InstallSection (organism)
│   ├── SectionHeader (molecule)
│   ├── PlatformTabs (molecule)
│   │   └── PlatformTab (atom) × 3
│   ├── InstallCodeBlock (molecule) [from code-block.md]
│   └── InstallCTA (molecule)
├── SocialProofSection (organism)
│   ├── TestimonialsGrid (molecule)
│   │   └── TestimonialCard (molecule) × 3
│   └── StatsRow (molecule)
│       └── StatItem (atom) × 4
└── Footer (organism) [from footer.md]
```

---

## 2. Section Specifications

### 2.1 Hero Section

#### Purpose
Immediately communicate the value proposition: AI-powered debugging for Azure Developer CLI with GitHub Copilot.

#### Props Interface

```typescript
interface HeroSectionProps {
  /** Primary headline text */
  headline: string;
  /** Supporting subheadline */
  subheadline: string;
  /** Primary CTA button config */
  primaryCTA: CTAConfig;
  /** Secondary CTA button config */
  secondaryCTA: CTAConfig;
  /** Social proof text (e.g., "1000+ developers") */
  socialProof?: string;
  /** Show animated AI demo */
  showDemo?: boolean;
  /** Custom class name */
  className?: string;
}

interface CTAConfig {
  label: string;
  href: string;
  icon?: React.ReactNode;
  variant?: 'primary' | 'secondary' | 'ghost';
}
```

#### Content

```typescript
const heroContent = {
  headline: "Debug Azure Apps with AI",
  subheadline: "azd-app brings MCP-powered AI debugging to Azure Developer CLI. Let GitHub Copilot analyze logs, diagnose issues, and suggest fixes.",
  primaryCTA: {
    label: "Get Started",
    href: "/quickstart",
    icon: <ArrowRightIcon />,
  },
  secondaryCTA: {
    label: "View Demo",
    href: "#demo",
    icon: <PlayIcon />,
  },
  socialProof: "Used by 1,000+ Azure developers",
};
```

#### Layout

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│                                                                                  │
│  ┌────────────────────────────────┐   ┌────────────────────────────────────────┐ │
│  │                                │   │  ╔══════════════════════════════════╗  │ │
│  │  Debug Azure Apps             │   │  ║  🤖 Copilot                       ║  │ │
│  │  with AI                      │   │  ╠══════════════════════════════════╣  │ │
│  │                                │   │  ║  Why is my API failing?          ║  │ │
│  │  azd-app brings MCP-powered   │   │  ║                                  ║  │ │
│  │  AI debugging to Azure...     │   │  ║  I found the issue in your logs: ║  │ │
│  │                                │   │  ║  Connection timeout to database  ║  │ │
│  │  ┌────────────────┐ ┌───────┐ │   │  ║                                  ║  │ │
│  │  │ Get Started  → │ │ Demo  │ │   │  ║  📍 src/api/db.py:42            ║  │ │
│  │  └────────────────┘ └───────┘ │   │  ╚══════════════════════════════════╝  │ │
│  │                                │   │                                        │ │
│  │  ✓ Used by 1,000+ developers  │   └────────────────────────────────────────┘ │
│  └────────────────────────────────┘                                             │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘

Mobile:
┌─────────────────────────────┐
│                             │
│   Debug Azure Apps          │
│   with AI                   │
│                             │
│   azd-app brings MCP-       │
│   powered AI debugging...   │
│                             │
│   ┌───────────────────────┐ │
│   │    Get Started    →   │ │
│   └───────────────────────┘ │
│                             │
│   ┌───────────────────────┐ │
│   │      View Demo        │ │
│   └───────────────────────┘ │
│                             │
│   ╔═══════════════════════╗ │
│   ║  🤖 Copilot Chat Demo ║ │
│   ╚═══════════════════════╝ │
│                             │
└─────────────────────────────┘
```

#### Dimensions

| Property            | Desktop         | Tablet          | Mobile          |
|---------------------|-----------------|-----------------|-----------------|
| Section padding     | 96px 0          | 80px 0          | 64px 0          |
| Content max-width   | 1280px          | 100%            | 100%            |
| Content gap         | 64px            | 48px            | 32px            |
| Headline font       | 60px/1.1        | 48px/1.15       | 36px/1.2        |
| Subheadline font    | 20px/1.6        | 18px/1.6        | 16px/1.6        |
| CTA button height   | 48px            | 48px            | 56px (touch)    |
| Demo width          | 560px           | 480px           | 100%            |

#### Typography

| Element       | Font Size | Weight    | Line Height | Color Token          |
|---------------|-----------|-----------|-------------|----------------------|
| Headline      | 60px      | Bold (700)| 1.1         | --color-text-primary |
| Subheadline   | 20px      | Normal    | 1.6         | --color-text-secondary|
| CTA Primary   | 16px      | Semibold  | 1.5         | --color-text-inverse |
| CTA Secondary | 16px      | Medium    | 1.5         | --color-text-primary |
| Social Proof  | 14px      | Medium    | 1.5         | --color-text-tertiary|

#### Colors

| Element                    | Light Theme         | Dark Theme          |
|----------------------------|---------------------|---------------------|
| Section background         | linear-gradient     | linear-gradient     |
|                            | (to-b, azure-50,    | (to-b, slate-900,   |
|                            | white)              | slate-800)          |
| Primary CTA bg             | --color-azure-600   | --color-azure-500   |
| Primary CTA hover          | --color-azure-700   | --color-azure-400   |
| Secondary CTA bg           | transparent         | transparent         |
| Secondary CTA border       | --color-border-strong| --color-border-strong|
| Social proof icon          | --color-success     | --color-success     |

---

### 2.2 AI Chat Demo (Hero Illustration)

#### Purpose
Show a realistic GitHub Copilot conversation demonstrating AI debugging capabilities.

#### Props Interface

```typescript
interface AIChatDemoProps {
  /** Conversation messages to display */
  messages: ChatMessage[];
  /** Animate typing effect */
  animate?: boolean;
  /** Typing speed (ms per character) */
  typingSpeed?: number;
  /** Auto-start when in viewport */
  autoPlay?: boolean;
  /** Show replay button */
  showReplay?: boolean;
  /** Custom class name */
  className?: string;
}

interface ChatMessage {
  /** Who is speaking */
  role: 'user' | 'assistant';
  /** Message content */
  content: string;
  /** Optional code snippet */
  codeSnippet?: {
    language: string;
    code: string;
    filename?: string;
  };
  /** Optional tool call indicator */
  toolCall?: {
    name: string;
    status: 'calling' | 'complete';
  };
  /** Delay before this message (ms) */
  delay?: number;
}
```

#### Demo Conversation

```typescript
const demoConversation: ChatMessage[] = [
  {
    role: 'user',
    content: "Why is my API service failing?",
  },
  {
    role: 'assistant',
    content: "Let me check the logs for your API service...",
    toolCall: { name: 'get_service_logs', status: 'calling' },
    delay: 500,
  },
  {
    role: 'assistant',
    content: "I found the issue. Your API is failing due to a database connection timeout.",
    delay: 800,
  },
  {
    role: 'assistant',
    content: "The error is in your connection configuration:",
    codeSnippet: {
      language: 'python',
      filename: 'src/api/db.py',
      code: `# Line 42 - timeout is too short
connection = Database(
    host="localhost",
    timeout=1  # Should be at least 5
)`,
    },
    delay: 300,
  },
  {
    role: 'assistant',
    content: "Increase the timeout to 5 seconds and the connection will succeed.",
    delay: 200,
  },
];
```

#### Layout

```
┌────────────────────────────────────────────────────────┐
│  🤖 GitHub Copilot                               ✕    │  ← Header
├────────────────────────────────────────────────────────┤
│                                                        │
│                 Why is my API failing?              ⬤ │  ← User message (right)
│                                                        │
│  ⬤ Let me check the logs...                          │  ← Assistant (left)
│                                                        │
│  ⬤ ┌──────────────────────────────────────────────┐  │  ← Tool call indicator
│    │ 🔧 Calling get_service_logs...               │  │
│    └──────────────────────────────────────────────┘  │
│                                                        │
│  ⬤ I found the issue. Your API is failing due to... │
│                                                        │
│  ⬤ ┌──────────────────────────────────────────────┐  │  ← Code snippet
│    │ python                      src/api/db.py 📋 │  │
│    ├──────────────────────────────────────────────┤  │
│    │ # Line 42 - timeout is too short             │  │
│    │ connection = Database(                       │  │
│    │     timeout=1  # Should be 5                 │  │
│    │ )                                            │  │
│    └──────────────────────────────────────────────┘  │
│                                                        │
│  ⬤ Increase the timeout to 5 seconds...             │
│                                                        │
└────────────────────────────────────────────────────────┘
│                    ↻ Replay                            │  ← Replay button
└────────────────────────────────────────────────────────┘
```

#### Colors

| Element                    | Light Theme         | Dark Theme          |
|----------------------------|---------------------|---------------------|
| Container background       | white               | --color-bg-secondary|
| Container border           | --color-border-default| --color-border-default|
| Container shadow           | --shadow-xl         | --shadow-lg         |
| Header background          | --color-azure-50    | --color-azure-900/20|
| Copilot icon               | --color-azure-600   | --color-azure-400   |
| User message bg            | --color-azure-600   | --color-azure-500   |
| User message text          | white               | white               |
| Assistant message bg       | --color-bg-tertiary | --color-bg-tertiary |
| Tool call bg               | --color-mcp-badge-bg| --color-mcp-badge-bg|
| Tool call border           | --color-mcp-badge-border| --color-mcp-badge-border|

---

### 2.3 Features Section

#### Purpose
Highlight the 6 key capabilities of azd-app in a scannable grid format.

#### Props Interface

```typescript
interface FeaturesSectionProps {
  /** Section title */
  title: string;
  /** Section description */
  description?: string;
  /** Feature cards to display */
  features: FeatureCardData[];
  /** Number of columns */
  columns?: 2 | 3;
  /** Custom class name */
  className?: string;
}

interface FeatureCardData {
  /** Feature icon component or emoji */
  icon: React.ReactNode | string;
  /** Feature title */
  title: string;
  /** Feature description */
  description: string;
  /** Optional badge (e.g., "AI", "New") */
  badge?: string;
  /** Link to learn more */
  href?: string;
}
```

#### Feature Content

```typescript
const features: FeatureCardData[] = [
  {
    icon: <CopilotIcon />,
    title: "AI-Powered Debugging",
    description: "Ask GitHub Copilot to analyze logs, find errors, and suggest fixes using MCP integration.",
    badge: "AI",
    href: "/mcp/ai-debugging",
  },
  {
    icon: "📊",
    title: "Real-time Dashboard",
    description: "Monitor all your services in one place with live status updates and resource usage.",
    href: "/tour/5-dashboard",
  },
  {
    icon: "📝",
    title: "Unified Logs",
    description: "Stream and filter logs from all services. Search, highlight, and export with ease.",
    href: "/tour/6-logs",
  },
  {
    icon: "❤️",
    title: "Health Monitoring",
    description: "Automatic health checks with visual indicators. Know when services need attention.",
    href: "/tour/7-health",
  },
  {
    icon: "🔌",
    title: "MCP Server",
    description: "Expose services to AI assistants via Model Context Protocol. Works with Copilot, Cursor, Claude.",
    badge: "AI",
    href: "/mcp",
  },
  {
    icon: "⚡",
    title: "One-Command Start",
    description: "Run `azd app run` and all services start with dependencies resolved automatically.",
    href: "/quickstart",
  },
];
```

#### Layout

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│                                                                                  │
│                          Everything You Need                                     │
│                    to Debug Azure Apps Locally                                   │
│                                                                                  │
│  ┌───────────────────────┐ ┌───────────────────────┐ ┌───────────────────────┐  │
│  │ 🤖        [AI]        │ │ 📊                    │ │ 📝                    │  │
│  │ AI-Powered Debugging  │ │ Real-time Dashboard   │ │ Unified Logs          │  │
│  │                       │ │                       │ │                       │  │
│  │ Ask GitHub Copilot    │ │ Monitor all services  │ │ Stream and filter     │  │
│  │ to analyze logs...    │ │ in one place...       │ │ logs from all...      │  │
│  │                       │ │                       │ │                       │  │
│  │ Learn more →          │ │ Learn more →          │ │ Learn more →          │  │
│  └───────────────────────┘ └───────────────────────┘ └───────────────────────┘  │
│                                                                                  │
│  ┌───────────────────────┐ ┌───────────────────────┐ ┌───────────────────────┐  │
│  │ ❤️                    │ │ 🔌        [AI]        │ │ ⚡                    │  │
│  │ Health Monitoring     │ │ MCP Server            │ │ One-Command Start     │  │
│  │                       │ │                       │ │                       │  │
│  │ Automatic health      │ │ Expose services to    │ │ Run azd app run       │  │
│  │ checks with visual... │ │ AI assistants...      │ │ and all services...   │  │
│  │                       │ │                       │ │                       │  │
│  │ Learn more →          │ │ Learn more →          │ │ Learn more →          │  │
│  └───────────────────────┘ └───────────────────────┘ └───────────────────────┘  │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘

Mobile (stacked):
┌─────────────────────────────┐
│ 🤖           [AI]           │
│ AI-Powered Debugging        │
│                             │
│ Ask GitHub Copilot to       │
│ analyze logs, find errors   │
│                             │
│ Learn more →                │
└─────────────────────────────┘
```

#### Feature Card Dimensions

| Property            | Desktop         | Tablet          | Mobile          |
|---------------------|-----------------|-----------------|-----------------|
| Grid columns        | 3               | 2               | 1               |
| Card padding        | 32px            | 24px            | 24px            |
| Card gap            | 24px            | 20px            | 16px            |
| Icon size           | 48px            | 40px            | 40px            |
| Title font          | 20px            | 18px            | 18px            |
| Description font    | 16px            | 15px            | 15px            |
| Badge font          | 12px            | 12px            | 12px            |

#### Feature Card States

| State    | Trigger         | Visual Changes                                |
|----------|-----------------|-----------------------------------------------|
| Default  | Initial         | Standard appearance                           |
| Hover    | Mouse enter     | Lift (translateY -4px), shadow increase       |
| Focus    | Keyboard focus  | Focus ring, same as hover                     |
| Active   | Mouse down      | Scale(0.98), shadow decrease                  |

#### Colors

| Element                    | Light Theme         | Dark Theme          |
|----------------------------|---------------------|---------------------|
| Card background            | white               | --color-bg-secondary|
| Card border                | --color-border-default| --color-border-default|
| Card shadow (default)      | --shadow-sm         | --shadow-sm         |
| Card shadow (hover)        | --shadow-lg         | --shadow-md         |
| Icon container bg          | --color-azure-50    | --color-azure-900/30|
| Title color                | --color-text-primary| --color-text-primary|
| Description color          | --color-text-secondary| --color-text-secondary|
| Link color                 | --color-azure-600   | --color-azure-400   |
| AI badge bg                | --color-mcp-badge-bg| --color-mcp-badge-bg|
| AI badge text              | --color-mcp-badge-text| --color-mcp-badge-text|

---

### 2.4 Demo Template Section

#### Purpose
Showcase the `azd init -t jongio/azd-app-demo` experience with a compelling terminal preview.

#### Props Interface

```typescript
interface DemoTemplateSectionProps {
  /** Section title */
  title: string;
  /** Section description */
  description: string;
  /** Terminal demo lines */
  terminalLines: TerminalLine[];
  /** Demo steps */
  steps: DemoStep[];
  /** Custom class name */
  className?: string;
}

interface DemoStep {
  /** Step number */
  number: number;
  /** Step title */
  title: string;
  /** Step description */
  description: string;
  /** Optional icon */
  icon?: React.ReactNode;
}
```

#### Content

```typescript
const demoContent = {
  title: "Try It Yourself",
  description: "Get started in under 5 minutes with our demo template. It includes an intentional bug for you to fix using AI!",
  terminalLines: [
    { type: 'command', content: 'azd init -t jongio/azd-app-demo' },
    { type: 'output', content: 'Initializing project from template...' },
    { type: 'success', content: '✓ Project initialized' },
    { type: 'output', content: '' },
    { type: 'command', content: 'azd app run' },
    { type: 'output', content: 'Starting services...' },
    { type: 'success', content: '✓ web started on http://localhost:3000' },
    { type: 'error', content: '✗ api: Connection timeout to database' },
    { type: 'output', content: '' },
    { type: 'info', content: '💡 Ask Copilot: "Why is my API failing?"' },
  ],
  steps: [
    { number: 1, title: "Initialize", description: "Clone the demo template" },
    { number: 2, title: "Run", description: "Start all services" },
    { number: 3, title: "Discover", description: "Find the bug in logs" },
    { number: 4, title: "Fix with AI", description: "Ask Copilot for help" },
  ],
};
```

#### Layout

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│                                                                                  │
│                              Try It Yourself                                     │
│     Get started in under 5 minutes with our demo template.                       │
│     It includes an intentional bug for you to fix using AI!                      │
│                                                                                  │
│  ┌────────────────────────────────────────────────────────────────────────────┐  │
│  │ ● ● ●                      Terminal                                   📋  │  │
│  ├────────────────────────────────────────────────────────────────────────────┤  │
│  │ $ azd init -t jongio/azd-app-demo                                         │  │
│  │ Initializing project from template...                                      │  │
│  │ ✓ Project initialized                                                      │  │
│  │                                                                             │  │
│  │ $ azd app run                                                              │  │
│  │ Starting services...                                                        │  │
│  │ ✓ web started on http://localhost:3000                                     │  │
│  │ ✗ api: Connection timeout to database                                      │  │
│  │                                                                             │  │
│  │ 💡 Ask Copilot: "Why is my API failing?"                                   │  │
│  └────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                  │
│      ┌─────────┐     ┌─────────┐     ┌─────────┐     ┌─────────────┐            │
│      │    1    │ ──▶ │    2    │ ──▶ │    3    │ ──▶ │      4      │            │
│      │Initialize│     │   Run   │     │Discover │     │ Fix with AI │            │
│      └─────────┘     └─────────┘     └─────────┘     └─────────────┘            │
│                                                                                  │
│                        ┌─────────────────────────────┐                           │
│                        │      Start Quick Start  →   │                           │
│                        └─────────────────────────────┘                           │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

#### Command Prominence

The `azd init -t jongio/azd-app-demo` command should be visually prominent:

```css
.demo-command {
  font-size: 1.125rem; /* Larger than normal terminal text */
  font-weight: 600;
  color: var(--color-azure-400);
  background: rgba(59, 130, 246, 0.1);
  padding: 2px 8px;
  border-radius: 4px;
}
```

---

### 2.5 Install Section

#### Purpose
Provide platform-specific installation commands with easy copy functionality.

#### Props Interface

```typescript
interface InstallSectionProps {
  /** Section title */
  title: string;
  /** Section description */
  description?: string;
  /** Platform install configs */
  platforms: PlatformInstall[];
  /** Default selected platform */
  defaultPlatform?: 'windows' | 'macos' | 'linux';
  /** Custom class name */
  className?: string;
}

interface PlatformInstall {
  /** Platform identifier */
  id: 'windows' | 'macos' | 'linux';
  /** Display name */
  name: string;
  /** Platform icon */
  icon: React.ReactNode;
  /** Install commands */
  commands: string[];
}
```

#### Content

```typescript
const installContent: PlatformInstall[] = [
  {
    id: 'windows',
    name: 'Windows',
    icon: <WindowsIcon />,
    commands: [
      '# Install Azure Developer CLI',
      'winget install microsoft.azd',
      '',
      '# Enable extensions and install azd-app',
      'azd config set alpha.extensions.enabled on',
      'azd extension source add app https://raw.githubusercontent.com/jongio/azd-app/main/registry.json',
      'azd extension install jongio.azd.app',
    ],
  },
  {
    id: 'macos',
    name: 'macOS',
    icon: <AppleIcon />,
    commands: [
      '# Install Azure Developer CLI',
      'brew tap azure/azd && brew install azd',
      '',
      '# Enable extensions and install azd-app',
      'azd config set alpha.extensions.enabled on',
      'azd extension source add app https://raw.githubusercontent.com/jongio/azd-app/main/registry.json',
      'azd extension install jongio.azd.app',
    ],
  },
  {
    id: 'linux',
    name: 'Linux',
    icon: <LinuxIcon />,
    commands: [
      '# Install Azure Developer CLI',
      'curl -fsSL https://aka.ms/install-azd.sh | bash',
      '',
      '# Enable extensions and install azd-app',
      'azd config set alpha.extensions.enabled on',
      'azd extension source add app https://raw.githubusercontent.com/jongio/azd-app/main/registry.json',
      'azd extension install jongio.azd.app',
    ],
  },
];
```

#### Layout

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│                                                                                  │
│                            Quick Install                                         │
│               Get up and running in less than 2 minutes                          │
│                                                                                  │
│            ┌─────────────┬─────────────┬─────────────┐                           │
│            │   Windows   │    macOS    │    Linux    │  ← Platform tabs          │
│            └─────────────┴─────────────┴─────────────┘                           │
│                                                                                  │
│  ┌────────────────────────────────────────────────────────────────────────────┐  │
│  │ bash                                                                   📋  │  │
│  ├────────────────────────────────────────────────────────────────────────────┤  │
│  │ # Install Azure Developer CLI                                              │  │
│  │ winget install microsoft.azd                                               │  │
│  │                                                                             │  │
│  │ # Enable extensions and install azd-app                                    │  │
│  │ azd config set alpha.extensions.enabled on                                 │  │
│  │ azd extension source add app https://raw.githubusercontent.com/...         │  │
│  │ azd extension install jongio.azd.app                                       │  │
│  └────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                  │
│                        ┌─────────────────────────────┐                           │
│                        │   Continue to Quick Start → │                           │
│                        └─────────────────────────────┘                           │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

#### Platform Tab States

| State    | Trigger         | Visual Changes                                |
|----------|-----------------|-----------------------------------------------|
| Default  | Not selected    | Muted text, no background                     |
| Hover    | Mouse enter     | Subtle background highlight                   |
| Active   | Selected        | Bold text, underline, filled background       |
| Focus    | Keyboard focus  | Focus ring                                    |

---

### 2.6 Social Proof Section

#### Purpose
Build trust through testimonials and usage statistics.

#### Props Interface

```typescript
interface SocialProofSectionProps {
  /** Testimonials to display */
  testimonials: Testimonial[];
  /** Statistics to display */
  stats: StatItem[];
  /** Custom class name */
  className?: string;
}

interface Testimonial {
  /** Quote text */
  quote: string;
  /** Author name */
  author: string;
  /** Author role/title */
  role: string;
  /** Author avatar URL */
  avatar?: string;
  /** Company/organization */
  company?: string;
}

interface StatItem {
  /** Statistic value */
  value: string;
  /** Statistic label */
  label: string;
  /** Optional icon */
  icon?: React.ReactNode;
}
```

#### Content

```typescript
const socialProofContent = {
  testimonials: [
    {
      quote: "azd-app with Copilot integration is a game-changer. I fixed a production bug in 5 minutes that would have taken hours to diagnose.",
      author: "Sarah Chen",
      role: "Senior Developer",
      company: "Contoso",
    },
    {
      quote: "The MCP server integration means I can just ask Copilot 'what's wrong' and get actual answers. Incredible.",
      author: "Marcus Johnson",
      role: "Platform Engineer",
      company: "Fabrikam",
    },
    {
      quote: "Finally, a local development experience that matches what we have in production. The dashboard is beautiful.",
      author: "Priya Sharma",
      role: "Tech Lead",
      company: "Northwind",
    },
  ],
  stats: [
    { value: "1,000+", label: "Active Users", icon: <UsersIcon /> },
    { value: "10", label: "MCP Tools", icon: <ToolIcon /> },
    { value: "<5min", label: "Setup Time", icon: <ClockIcon /> },
    { value: "100%", label: "Open Source", icon: <CodeIcon /> },
  ],
};
```

#### Layout

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│                                                                                  │
│                         Loved by Developers                                      │
│                                                                                  │
│  ┌─────────────────────────┐ ┌─────────────────────────┐ ┌─────────────────────┐ │
│  │ "azd-app with Copilot   │ │ "The MCP server         │ │ "Finally, a local   │ │
│  │  integration is a       │ │  integration means I    │ │  development        │ │
│  │  game-changer..."       │ │  can just ask..."       │ │  experience..."     │ │
│  │                         │ │                         │ │                     │ │
│  │  👤 Sarah Chen          │ │  👤 Marcus Johnson      │ │  👤 Priya Sharma    │ │
│  │  Senior Developer       │ │  Platform Engineer      │ │  Tech Lead          │ │
│  │  Contoso                │ │  Fabrikam               │ │  Northwind          │ │
│  └─────────────────────────┘ └─────────────────────────┘ └─────────────────────┘ │
│                                                                                  │
│  ───────────────────────────────────────────────────────────────────────────────│
│                                                                                  │
│     👥 1,000+        🔧 10            ⏱ <5min         📖 100%                   │
│     Active Users      MCP Tools        Setup Time      Open Source              │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Accessibility

### Semantic Structure

```html
<main id="main-content">
  <section aria-labelledby="hero-heading" class="hero-section">
    <h1 id="hero-heading">Debug Azure Apps with AI</h1>
    <!-- Hero content -->
  </section>
  
  <section aria-labelledby="features-heading" class="features-section">
    <h2 id="features-heading">Everything You Need</h2>
    <!-- Features grid -->
  </section>
  
  <section aria-labelledby="demo-heading" class="demo-section">
    <h2 id="demo-heading">Try It Yourself</h2>
    <!-- Demo terminal -->
  </section>
  
  <section aria-labelledby="install-heading" class="install-section">
    <h2 id="install-heading">Quick Install</h2>
    <!-- Install tabs and code -->
  </section>
  
  <section aria-labelledby="testimonials-heading" class="social-proof-section">
    <h2 id="testimonials-heading">Loved by Developers</h2>
    <!-- Testimonials and stats -->
  </section>
</main>
```

### ARIA Attributes by Section

| Section      | Element              | ARIA Attribute                    |
|--------------|---------------------|-----------------------------------|
| Hero         | AI Demo             | role="figure", aria-label         |
| Features     | Card grid           | role="list"                       |
| Features     | Feature card        | role="listitem"                   |
| Demo         | Terminal            | role="figure", aria-label         |
| Install      | Tab group           | role="tablist"                    |
| Install      | Tab                 | role="tab", aria-selected         |
| Install      | Tab panel           | role="tabpanel", aria-labelledby  |
| Social Proof | Stats               | role="list"                       |

### Keyboard Navigation

| Element           | Key            | Action                           |
|-------------------|----------------|----------------------------------|
| Platform tabs     | Arrow Left/Right | Move between tabs              |
| Platform tabs     | Enter/Space    | Select tab                       |
| Feature cards     | Tab            | Move between cards               |
| Feature cards     | Enter          | Navigate to feature page         |
| Copy buttons      | Enter/Space    | Copy content                     |
| CTA buttons       | Enter/Space    | Navigate or trigger action       |

### Screen Reader Announcements

```typescript
// Platform tab change
"Windows tab selected"

// Copy feedback
"Commands copied to clipboard"

// AI demo progress (if animated)
"Assistant message: I found the issue in your logs"

// Feature card focus
"AI-Powered Debugging feature. Ask GitHub Copilot to analyze logs..."
```

### Skip Link

```html
<a href="#main-content" class="skip-link">
  Skip to main content
</a>
```

### Reduced Motion

```css
@media (prefers-reduced-motion: reduce) {
  .hero-illustration,
  .feature-card,
  .terminal-animation {
    animation: none;
    transition: opacity 0.1s ease-out;
  }
  
  /* Show all demo content immediately */
  .ai-chat-demo .message {
    opacity: 1;
    transform: none;
  }
}
```

---

## 4. Responsive Design

### Breakpoint Behavior

| Section         | Desktop (≥1024px) | Tablet (768-1023px) | Mobile (<768px) |
|-----------------|-------------------|---------------------|-----------------|
| Hero            | 2-column          | 2-column stacked    | 1-column        |
| Features        | 3-column grid     | 2-column grid       | 1-column stack  |
| Demo Terminal   | Wide view         | Full width          | Full width      |
| Install Tabs    | Inline tabs       | Inline tabs         | Stacked buttons |
| Testimonials    | 3-column          | 2-column            | 1-column        |
| Stats           | 4-column inline   | 2x2 grid            | 2x2 grid        |

### Mobile-First CSS Approach

```css
/* Base (Mobile) */
.hero-section {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-8);
  padding: var(--spacing-16) var(--spacing-4);
}

.hero-content {
  text-align: center;
}

.hero-headline {
  font-size: var(--font-size-4xl);
}

/* Tablet */
@media (min-width: 768px) {
  .hero-section {
    gap: var(--spacing-12);
    padding: var(--spacing-20) var(--spacing-6);
  }
  
  .hero-headline {
    font-size: var(--font-size-5xl);
  }
}

/* Desktop */
@media (min-width: 1024px) {
  .hero-section {
    flex-direction: row;
    align-items: center;
    gap: var(--spacing-16);
    padding: var(--spacing-24) var(--spacing-8);
  }
  
  .hero-content {
    text-align: left;
    flex: 1;
  }
  
  .hero-illustration {
    flex: 1;
    max-width: 560px;
  }
  
  .hero-headline {
    font-size: 60px;
  }
}
```

---

## 5. Animation Specifications

### Hero Entrance

```css
.hero-content {
  animation: hero-content-in 0.6s ease-out;
}

.hero-illustration {
  animation: hero-illustration-in 0.8s ease-out 0.2s both;
}

@keyframes hero-content-in {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes hero-illustration-in {
  from {
    opacity: 0;
    transform: translateY(30px) scale(0.98);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}
```

### Feature Card Stagger

```css
.feature-card {
  animation: feature-card-in 0.5s ease-out both;
}

.feature-card:nth-child(1) { animation-delay: 0.1s; }
.feature-card:nth-child(2) { animation-delay: 0.2s; }
.feature-card:nth-child(3) { animation-delay: 0.3s; }
.feature-card:nth-child(4) { animation-delay: 0.4s; }
.feature-card:nth-child(5) { animation-delay: 0.5s; }
.feature-card:nth-child(6) { animation-delay: 0.6s; }

@keyframes feature-card-in {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
```

### AI Demo Typing

```css
.ai-message {
  animation: message-appear 0.3s ease-out;
}

@keyframes message-appear {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.typing-indicator {
  animation: typing-dots 1.4s infinite;
}

@keyframes typing-dots {
  0%, 20% { opacity: 0.3; }
  50% { opacity: 1; }
  80%, 100% { opacity: 0.3; }
}
```

### Scroll-Triggered Animations

Use Intersection Observer to trigger section animations when scrolling:

```typescript
function setupScrollAnimations(): void {
  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          entry.target.classList.add('animate-in');
          observer.unobserve(entry.target);
        }
      });
    },
    { threshold: 0.2 }
  );
  
  document.querySelectorAll('.animate-on-scroll').forEach((el) => {
    observer.observe(el);
  });
}
```

---

## 6. Performance Considerations

### Critical Rendering Path

1. **Above-the-fold content** (Hero) loads first
2. Defer below-fold images and animations
3. Use `loading="lazy"` for testimonial avatars
4. Preload hero illustration assets

### Asset Optimization

```html
<!-- Preload critical fonts -->
<link rel="preload" href="/fonts/inter-var.woff2" as="font" type="font/woff2" crossorigin>

<!-- Preload hero assets -->
<link rel="preload" href="/images/copilot-icon.svg" as="image">

<!-- Defer non-critical scripts -->
<script src="/scripts/animations.js" defer></script>
```

### Bundle Size

- Use tree-shaking for icon libraries
- Split AI demo component (code-split)
- Lazy-load terminal component for demo section

### Lighthouse Targets

| Metric            | Target    |
|-------------------|-----------|
| FCP               | < 1.5s    |
| LCP               | < 2.5s    |
| CLS               | < 0.1     |
| TBT               | < 200ms   |
| Accessibility     | 100       |
| Best Practices    | 100       |

---

## 7. CSS Custom Properties

```css
/* Landing Page Tokens */
--landing-hero-padding-y: var(--spacing-24);
--landing-section-padding-y: var(--spacing-20);
--landing-section-gap: var(--spacing-24);
--landing-content-max-width: 1280px;
--landing-content-padding-x: var(--spacing-6);

/* Hero Tokens */
--hero-headline-size: 3.75rem; /* 60px */
--hero-subheadline-size: 1.25rem; /* 20px */
--hero-cta-height: 3rem; /* 48px */
--hero-cta-padding: var(--spacing-6);

/* Feature Card Tokens */
--feature-card-padding: var(--spacing-8);
--feature-card-gap: var(--spacing-6);
--feature-card-radius: var(--radius-xl);
--feature-icon-size: 3rem; /* 48px */

/* AI Demo Tokens */
--ai-demo-width: 560px;
--ai-demo-radius: var(--radius-xl);
--ai-demo-shadow: var(--shadow-xl);

/* Install Section Tokens */
--install-tab-height: 3rem;
--install-tab-gap: var(--spacing-2);

/* Social Proof Tokens */
--testimonial-card-padding: var(--spacing-6);
--stat-value-size: var(--font-size-4xl);
```

---

## 8. Testing Checklist

### Functional Tests

- [ ] All CTA buttons navigate correctly
- [ ] Platform tabs switch content
- [ ] Copy buttons work on all browsers
- [ ] AI demo animation plays and replays
- [ ] Terminal demo animation works
- [ ] External links open correctly

### Accessibility Tests

- [ ] All headings in correct hierarchy (h1, h2, h3)
- [ ] Skip link works
- [ ] All interactive elements keyboard accessible
- [ ] Focus visible on all elements
- [ ] Screen reader announces all content correctly
- [ ] Color contrast ≥ 4.5:1 for text
- [ ] Touch targets ≥ 44x44px on mobile
- [ ] Reduced motion preference respected

### Responsive Tests

- [ ] Layout correct at all breakpoints
- [ ] No horizontal scroll on mobile
- [ ] Touch targets adequate on mobile
- [ ] Text readable at all sizes
- [ ] Images scale correctly

### Performance Tests

- [ ] LCP under 2.5s
- [ ] FCP under 1.5s
- [ ] No layout shift (CLS < 0.1)
- [ ] Images optimized
- [ ] Fonts loaded efficiently

### Cross-Browser Tests

- [ ] Chrome (latest)
- [ ] Firefox (latest)
- [ ] Safari (latest)
- [ ] Edge (latest)
- [ ] iOS Safari
- [ ] Chrome for Android

---

## 9. Implementation Notes

### Component Files

```
web/src/components/
├── landing/
│   ├── HeroSection.astro
│   ├── AIChatDemo.astro
│   ├── FeaturesSection.astro
│   ├── FeatureCard.astro
│   ├── DemoTemplateSection.astro
│   ├── InstallSection.astro
│   ├── PlatformTabs.astro
│   ├── SocialProofSection.astro
│   ├── TestimonialCard.astro
│   └── StatItem.astro
```

### Page File

```astro
---
// web/src/pages/index.astro
import Layout from '../layouts/Layout.astro';
import HeroSection from '../components/landing/HeroSection.astro';
import FeaturesSection from '../components/landing/FeaturesSection.astro';
import DemoTemplateSection from '../components/landing/DemoTemplateSection.astro';
import InstallSection from '../components/landing/InstallSection.astro';
import SocialProofSection from '../components/landing/SocialProofSection.astro';
---

<Layout title="azd-app - AI-Powered Azure Development">
  <HeroSection />
  <FeaturesSection />
  <DemoTemplateSection />
  <InstallSection />
  <SocialProofSection />
</Layout>
```

### Data Files

```typescript
// web/src/data/landing.ts
export const heroContent = { ... };
export const features = [ ... ];
export const demoContent = { ... };
export const installContent = [ ... ];
export const socialProofContent = { ... };
```

---

## 10. Related Components

- [Header](./header.md) - Top navigation
- [Footer](./footer.md) - Bottom navigation and links
- [Terminal](./terminal.md) - Terminal demo component
- [Code Block](./code-block.md) - Install code display
- [Copy Button](./copy-button.md) - Copy to clipboard functionality
- [Theme Toggle](./theme-toggle.md) - Light/dark mode switching

````
