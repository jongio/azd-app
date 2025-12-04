````markdown
# Quick Start Page Component Specification

## Overview

The Quick Start page provides a streamlined, gamified onboarding experience that guides users through installing azd-app and completing a "Fix the Bug" challenge using AI assistance. The page is designed to be completable in 5 minutes, with clear progress tracking and copy-paste commands.

---

## 1. Page Structure

```
QuickStartPage
├── Header (organism) [from header.md]
├── QuickStartHero (organism)
│   ├── PageTitle (atom)
│   ├── TimeEstimate (molecule)
│   │   ├── ClockIcon (atom)
│   │   └── EstimateText (atom)
│   └── ChallengeTeaser (molecule)
├── ProgressIndicator (molecule)
│   └── ProgressStep (atom) × 4
├── StepsContainer (organism)
│   ├── StepCard (molecule) × 4
│   │   ├── StepHeader (molecule)
│   │   │   ├── StepNumber (atom)
│   │   │   ├── StepTitle (atom)
│   │   │   └── StepStatus (atom) [optional]
│   │   ├── StepContent (molecule)
│   │   │   ├── StepDescription (atom)
│   │   │   ├── PlatformTabs (molecule) [Step 1 & 2]
│   │   │   │   └── PlatformTab (atom) × 3
│   │   │   ├── CodeBlock (molecule) [from code-block.md]
│   │   │   └── StepTips (molecule) [optional]
│   │   └── ChallengeCallout (molecule) [Step 4 only]
│   │       ├── ChallengeIcon (atom)
│   │       ├── ChallengeTitle (atom)
│   │       ├── ChallengeDescription (atom)
│   │       └── CopilotPrompt (molecule)
├── NextStepsSection (organism)
│   ├── SectionHeader (molecule)
│   └── NextStepCards (molecule)
│       └── NextStepCard (molecule) × 3
└── Footer (organism) [from footer.md]
```

---

## 2. Component Specifications

### 2.1 Quick Start Hero

#### Purpose
Introduce the quick start experience with clear time commitment and challenge preview.

#### Props Interface

```typescript
interface QuickStartHeroProps {
  /** Page title */
  title: string;
  /** Subtitle/description */
  subtitle: string;
  /** Estimated completion time */
  timeEstimate: string;
  /** Challenge teaser text */
  challengeTeaser?: string;
  /** Custom class name */
  className?: string;
}
```

#### Content

```typescript
const heroContent = {
  title: "Quick Start",
  subtitle: "Get up and running with azd-app in 5 minutes. Install, run a demo, and fix your first bug with AI!",
  timeEstimate: "~5 minutes",
  challengeTeaser: "🎯 Challenge: Can you fix the bug using AI?",
};
```

#### Layout

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│                                                                                  │
│                              Quick Start                                         │
│                                                                                  │
│      Get up and running with azd-app in 5 minutes.                              │
│      Install, run a demo, and fix your first bug with AI!                       │
│                                                                                  │
│                         ⏱ ~5 minutes                                            │
│                                                                                  │
│                 🎯 Challenge: Can you fix the bug using AI?                     │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

#### Dimensions

| Property            | Desktop         | Tablet          | Mobile          |
|---------------------|-----------------|-----------------|-----------------|
| Section padding     | 48px 0          | 40px 0          | 32px 0          |
| Title font          | 48px/1.1        | 40px/1.15       | 32px/1.2        |
| Subtitle font       | 20px/1.6        | 18px/1.6        | 16px/1.6        |
| Time estimate font  | 16px            | 16px            | 14px            |
| Challenge font      | 18px            | 16px            | 16px            |

#### Colors

| Element                    | Light Theme         | Dark Theme          |
|----------------------------|---------------------|---------------------|
| Title color                | --color-text-primary| --color-text-primary|
| Subtitle color             | --color-text-secondary| --color-text-secondary|
| Time estimate icon         | --color-azure-500   | --color-azure-400   |
| Time estimate text         | --color-text-tertiary| --color-text-tertiary|
| Challenge background       | --color-mcp-badge-bg| --color-mcp-badge-bg|
| Challenge text             | --color-mcp-badge-text| --color-mcp-badge-text|
| Challenge border           | --color-mcp-badge-border| --color-mcp-badge-border|

---

### 2.2 Time Estimate Display

#### Purpose
Show users how long the quick start will take, setting expectations.

#### Props Interface

```typescript
interface TimeEstimateProps {
  /** Time estimate string (e.g., "~5 minutes") */
  estimate: string;
  /** Icon to display */
  icon?: React.ReactNode;
  /** Size variant */
  size?: 'sm' | 'md' | 'lg';
  /** Custom class name */
  className?: string;
}
```

#### Layout

```
┌───────────────────┐
│  ⏱  ~5 minutes   │
└───────────────────┘
```

#### Dimensions

| Size | Icon Size | Font Size | Padding    |
|------|-----------|-----------|------------|
| sm   | 14px      | 12px      | 4px 8px    |
| md   | 16px      | 14px      | 6px 12px   |
| lg   | 20px      | 16px      | 8px 16px   |

#### Colors

| Element       | Light Theme         | Dark Theme          |
|---------------|---------------------|---------------------|
| Background    | --color-bg-tertiary | --color-bg-tertiary |
| Icon          | --color-azure-500   | --color-azure-400   |
| Text          | --color-text-secondary| --color-text-secondary|
| Border        | --color-border-default| --color-border-default|

---

### 2.3 Progress Indicator

#### Purpose
Show visual progress through the 4 steps, helping users track their position.

#### Props Interface

```typescript
interface ProgressIndicatorProps {
  /** Total number of steps */
  totalSteps: number;
  /** Current active step (1-indexed) */
  currentStep: number;
  /** Completed steps */
  completedSteps: number[];
  /** Step labels */
  labels: string[];
  /** Orientation */
  orientation?: 'horizontal' | 'vertical';
  /** Custom class name */
  className?: string;
}
```

#### Content

```typescript
const progressContent = {
  totalSteps: 4,
  labels: [
    "Install Azure CLI",
    "Enable Extensions",
    "Clone & Run",
    "Fix with AI"
  ],
};
```

#### Layout

```
Horizontal (Desktop/Tablet):
┌──────────────────────────────────────────────────────────────────────────────────┐
│                                                                                  │
│     ①──────────②──────────③──────────④                                          │
│   Install    Enable     Clone &    Fix with                                     │
│   Azure CLI  Extensions Run        AI                                           │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘

Vertical (Mobile):
┌─────────────────────────────┐
│  ① Install Azure CLI        │
│  │                          │
│  ② Enable Extensions        │
│  │                          │
│  ③ Clone & Run              │
│  │                          │
│  ④ Fix with AI              │
└─────────────────────────────┘
```

#### Step States

| State     | Visual                                              |
|-----------|-----------------------------------------------------|
| Inactive  | Outlined circle, muted colors                       |
| Active    | Filled circle, primary color, pulsing indicator     |
| Completed | Filled circle with checkmark, success color         |

#### Dimensions

| Property            | Desktop         | Mobile          |
|---------------------|-----------------|-----------------|
| Step circle size    | 32px            | 28px            |
| Step number font    | 14px            | 12px            |
| Connector width     | 100px           | 2px (vertical)  |
| Connector height    | 2px             | 24px            |
| Label font          | 14px            | 12px            |
| Gap between circles | 16px            | 8px             |

#### Colors

| Element                    | Light Theme         | Dark Theme          |
|----------------------------|---------------------|---------------------|
| Inactive circle bg         | transparent         | transparent         |
| Inactive circle border     | --color-border-strong| --color-border-strong|
| Inactive number color      | --color-text-muted  | --color-text-muted  |
| Active circle bg           | --color-azure-500   | --color-azure-500   |
| Active number color        | white               | white               |
| Active glow                | rgba(59,130,246,0.3)| rgba(96,165,250,0.3)|
| Completed circle bg        | --color-success     | --color-success     |
| Completed checkmark        | white               | white               |
| Connector inactive         | --color-border-default| --color-border-default|
| Connector completed        | --color-success     | --color-success     |
| Label active color         | --color-text-primary| --color-text-primary|
| Label inactive color       | --color-text-tertiary| --color-text-tertiary|

#### Animation

```css
/* Active step pulse */
.progress-step--active .step-circle {
  animation: step-pulse 2s ease-in-out infinite;
}

@keyframes step-pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.4); }
  50% { box-shadow: 0 0 0 8px rgba(59, 130, 246, 0); }
}

/* Completion checkmark */
.progress-step--completed .step-check {
  animation: check-appear 0.3s ease-out;
}

@keyframes check-appear {
  0% { transform: scale(0); opacity: 0; }
  100% { transform: scale(1); opacity: 1; }
}
```

---

### 2.4 Step Card

#### Purpose
Display each step with clear instructions, code commands, and copy functionality.

#### Props Interface

```typescript
interface StepCardProps {
  /** Step number (1-4) */
  number: number;
  /** Step title */
  title: string;
  /** Step description */
  description: string;
  /** Content blocks (code, text, tips) */
  content: StepContentBlock[];
  /** Whether step is active/current */
  isActive?: boolean;
  /** Whether step is completed */
  isCompleted?: boolean;
  /** Platform-specific content */
  platformContent?: PlatformContent;
  /** Challenge callout (for step 4) */
  challenge?: ChallengeConfig;
  /** Custom class name */
  className?: string;
}

interface StepContentBlock {
  /** Block type */
  type: 'text' | 'code' | 'terminal' | 'tip' | 'warning';
  /** Block content */
  content: string;
  /** For code blocks: language */
  language?: string;
  /** For code blocks: title */
  title?: string;
}

interface PlatformContent {
  windows: StepContentBlock[];
  macos: StepContentBlock[];
  linux: StepContentBlock[];
}

interface ChallengeConfig {
  title: string;
  description: string;
  prompt: string;
  icon?: string;
}
```

#### Layout

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│  ┌─────┐                                                                         │
│  │  1  │  Install Azure Developer CLI                                            │
│  └─────┘                                                                         │
│                                                                                  │
│  First, you'll need to install the Azure Developer CLI (azd).                   │
│  Choose your platform below:                                                     │
│                                                                                  │
│  ┌─────────────┬─────────────┬─────────────┐                                    │
│  │   Windows   │    macOS    │    Linux    │                                    │
│  └─────────────┴─────────────┴─────────────┘                                    │
│                                                                                  │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │ powershell                                                             📋  │ │
│  ├────────────────────────────────────────────────────────────────────────────┤ │
│  │ winget install microsoft.azd                                               │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                  │
│  💡 Tip: After installation, restart your terminal for the changes to apply.   │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

#### Dimensions

| Property            | Desktop         | Tablet          | Mobile          |
|---------------------|-----------------|-----------------|-----------------|
| Card padding        | 32px            | 24px            | 20px            |
| Card margin-bottom  | 24px            | 20px            | 16px            |
| Step number size    | 40px            | 36px            | 32px            |
| Step number font    | 20px bold       | 18px bold       | 16px bold       |
| Title font          | 24px semibold   | 22px semibold   | 20px semibold   |
| Description font    | 16px            | 15px            | 15px            |
| Content gap         | 20px            | 16px            | 14px            |

#### Step Number Styles

```
Active:
┌─────────┐
│         │
│    1    │  ← Filled, Azure color
│         │
└─────────┘

Completed:
┌─────────┐
│         │
│    ✓    │  ← Filled, Success color
│         │
└─────────┘

Upcoming:
┌─────────┐
│         │
│    3    │  ← Outlined, muted
│         │
└─────────┘
```

#### Colors

| Element                    | Light Theme         | Dark Theme          |
|----------------------------|---------------------|---------------------|
| Card background            | white               | --color-bg-secondary|
| Card border                | --color-border-default| --color-border-default|
| Card shadow                | --shadow-sm         | none                |
| Active card border         | --color-azure-300   | --color-azure-700   |
| Active card shadow         | --shadow-md         | --shadow-sm         |
| Step number active bg      | --color-azure-500   | --color-azure-500   |
| Step number active text    | white               | white               |
| Step number completed bg   | --color-success     | --color-success     |
| Title color                | --color-text-primary| --color-text-primary|
| Description color          | --color-text-secondary| --color-text-secondary|
| Tip background             | --color-azure-50    | --color-azure-900/30|
| Tip border                 | --color-azure-200   | --color-azure-700   |
| Tip icon color             | --color-azure-600   | --color-azure-400   |

---

### 2.5 Platform Tabs

#### Purpose
Allow users to switch between platform-specific installation commands.

#### Props Interface

```typescript
interface PlatformTabsProps {
  /** Currently selected platform */
  selectedPlatform: 'windows' | 'macos' | 'linux';
  /** Callback when platform changes */
  onPlatformChange: (platform: 'windows' | 'macos' | 'linux') => void;
  /** Auto-detect user's platform */
  autoDetect?: boolean;
  /** Custom class name */
  className?: string;
}
```

#### Layout

```
┌─────────────────────────────────────────────────────┐
│  ┌─────────────┬─────────────┬─────────────┐       │
│  │   Windows   │    macOS    │    Linux    │       │
│  │  [active]   │             │             │       │
│  └─────────────┴─────────────┴─────────────┘       │
└─────────────────────────────────────────────────────┘
```

#### Dimensions

| Property            | Value           |
|---------------------|-----------------|
| Tab height          | 44px            |
| Tab min-width       | 100px           |
| Tab padding         | 12px 20px       |
| Tab gap             | 0               |
| Tab font            | 14px medium     |
| Border radius       | 8px (container) |
| Icon size           | 18px            |

#### Tab States

| State    | Visual                                              |
|----------|-----------------------------------------------------|
| Default  | No background, muted text                           |
| Hover    | Light background, primary text                      |
| Active   | Filled background, bold text, border indicator      |
| Focus    | Focus ring visible                                  |

#### Colors

| Element                    | Light Theme         | Dark Theme          |
|----------------------------|---------------------|---------------------|
| Container background       | --color-bg-tertiary | --color-bg-tertiary |
| Tab text default           | --color-text-secondary| --color-text-secondary|
| Tab text hover             | --color-text-primary| --color-text-primary|
| Tab text active            | --color-azure-700   | --color-azure-300   |
| Tab bg active              | white               | --color-bg-secondary|
| Tab shadow active          | --shadow-sm         | none                |
| Tab border active          | --color-azure-500   | --color-azure-500   |

#### Platform Icons

```typescript
const platformIcons = {
  windows: WindowsIcon,  // Windows logo
  macos: AppleIcon,      // Apple logo  
  linux: LinuxIcon,      // Tux/Linux logo
};
```

---

### 2.6 Challenge Callout (Step 4)

#### Purpose
Highlight the "Fix the Bug" challenge in Step 4, making it visually distinct and engaging.

#### Props Interface

```typescript
interface ChallengeCalloutProps {
  /** Challenge title */
  title: string;
  /** Challenge description */
  description: string;
  /** Copilot prompt to copy */
  prompt: string;
  /** Challenge icon/emoji */
  icon?: string;
  /** Custom class name */
  className?: string;
}
```

#### Content

```typescript
const challengeContent = {
  icon: "🎯",
  title: "Your Challenge",
  description: "The demo template has an intentional bug! The API service is failing with a connection timeout. Use GitHub Copilot to diagnose and fix the issue.",
  prompt: "Why is my API service failing? Can you help me fix it?",
};
```

#### Layout

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│  ╔═══════════════════════════════════════════════════════════════════════════╗  │
│  ║                                                                           ║  │
│  ║  🎯 Your Challenge                                                        ║  │
│  ║                                                                           ║  │
│  ║  The demo template has an intentional bug! The API service is failing    ║  │
│  ║  with a connection timeout. Use GitHub Copilot to diagnose and fix       ║  │
│  ║  the issue.                                                               ║  │
│  ║                                                                           ║  │
│  ║  ┌─────────────────────────────────────────────────────────────────────┐ ║  │
│  ║  │  💬 Ask Copilot:                                                    │ ║  │
│  ║  │                                                                     │ ║  │
│  ║  │  "Why is my API service failing? Can you help me fix it?"      📋  │ ║  │
│  ║  └─────────────────────────────────────────────────────────────────────┘ ║  │
│  ║                                                                           ║  │
│  ╚═══════════════════════════════════════════════════════════════════════════╝  │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

#### Dimensions

| Property            | Desktop         | Tablet          | Mobile          |
|---------------------|-----------------|-----------------|-----------------|
| Callout padding     | 24px            | 20px            | 16px            |
| Callout margin      | 24px 0          | 20px 0          | 16px 0          |
| Icon size           | 32px            | 28px            | 24px            |
| Title font          | 22px bold       | 20px bold       | 18px bold       |
| Description font    | 16px            | 15px            | 15px            |
| Prompt box padding  | 16px            | 14px            | 12px            |
| Border width        | 3px             | 2px             | 2px             |
| Border radius       | 12px            | 10px            | 8px             |

#### Colors

| Element                    | Light Theme         | Dark Theme          |
|----------------------------|---------------------|---------------------|
| Callout background         | linear-gradient(135deg, #fef3c7, #fef9c3) | linear-gradient(135deg, #422006, #451a03) |
| Callout border             | --color-mcp-badge-border | --color-mcp-badge-border |
| Icon color                 | --color-mcp-badge-text | --color-mcp-badge-text |
| Title color                | --color-text-primary | --color-text-primary |
| Description color          | --color-text-secondary | --color-text-secondary |
| Prompt box background      | white               | --color-bg-primary  |
| Prompt box border          | --color-border-default | --color-border-default |
| Prompt text color          | --color-text-primary | --color-text-primary |
| Prompt label color         | --color-azure-600   | --color-azure-400   |

#### Copilot Prompt Block

```
┌─────────────────────────────────────────────────────────────┐
│  💬 Ask Copilot:                                            │
│                                                             │
│  "Why is my API service failing? Can you help me fix it?"   │
│                                                         📋  │
└─────────────────────────────────────────────────────────────┘
```

---

### 2.7 Next Steps Section

#### Purpose
Guide users to continue learning after completing the quick start.

#### Props Interface

```typescript
interface NextStepsSectionProps {
  /** Section title */
  title: string;
  /** Section description */
  description?: string;
  /** Next step cards */
  steps: NextStepCard[];
  /** Custom class name */
  className?: string;
}

interface NextStepCard {
  /** Card icon */
  icon: string | React.ReactNode;
  /** Card title */
  title: string;
  /** Card description */
  description: string;
  /** Link destination */
  href: string;
  /** Primary action? (styled differently) */
  isPrimary?: boolean;
}
```

#### Content

```typescript
const nextStepsContent = {
  title: "What's Next?",
  description: "You've completed the quick start! Here's what to explore next:",
  steps: [
    {
      icon: "🎯",
      title: "Guided Tour",
      description: "Take a comprehensive tour of all azd-app features",
      href: "/tour",
      isPrimary: true,
    },
    {
      icon: "🤖",
      title: "AI Features",
      description: "Learn about MCP server and AI-powered debugging",
      href: "/mcp",
    },
    {
      icon: "📚",
      title: "Documentation",
      description: "Explore the full CLI reference and commands",
      href: "/cli",
    },
  ],
};
```

#### Layout

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│                                                                                  │
│                              What's Next?                                        │
│         You've completed the quick start! Here's what to explore next:          │
│                                                                                  │
│  ┌────────────────────────┐ ┌────────────────────────┐ ┌────────────────────────┐│
│  │  🎯                    │ │  🤖                    │ │  📚                    ││
│  │  Guided Tour           │ │  AI Features           │ │  Documentation         ││
│  │                        │ │                        │ │                        ││
│  │  Take a comprehensive  │ │  Learn about MCP       │ │  Explore the full CLI  ││
│  │  tour of all azd-app   │ │  server and AI-powered │ │  reference and         ││
│  │  features              │ │  debugging             │ │  commands              ││
│  │                        │ │                        │ │                        ││
│  │  [primary button]      │ │  Learn more →          │ │  Learn more →          ││
│  └────────────────────────┘ └────────────────────────┘ └────────────────────────┘│
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

#### Dimensions

| Property            | Desktop         | Tablet          | Mobile          |
|---------------------|-----------------|-----------------|-----------------|
| Section padding     | 64px 0          | 48px 0          | 40px 0          |
| Grid columns        | 3               | 2               | 1               |
| Card padding        | 24px            | 20px            | 20px            |
| Card gap            | 24px            | 20px            | 16px            |
| Icon size           | 40px            | 36px            | 32px            |
| Title font          | 20px semibold   | 18px semibold   | 18px semibold   |
| Description font    | 15px            | 14px            | 14px            |

---

## 3. Step-by-Step Content

### Step 1: Install Azure Developer CLI

```typescript
const step1Content = {
  number: 1,
  title: "Install Azure Developer CLI",
  description: "First, you'll need the Azure Developer CLI (azd) installed on your machine. Choose your platform below:",
  platformContent: {
    windows: [
      {
        type: 'code',
        language: 'powershell',
        title: 'PowerShell',
        content: 'winget install microsoft.azd',
      },
      {
        type: 'tip',
        content: 'Restart your terminal after installation.',
      },
    ],
    macos: [
      {
        type: 'code',
        language: 'bash',
        title: 'Terminal',
        content: 'brew tap azure/azd && brew install azd',
      },
    ],
    linux: [
      {
        type: 'code',
        language: 'bash',
        title: 'Terminal',
        content: 'curl -fsSL https://aka.ms/install-azd.sh | bash',
      },
    ],
  },
};
```

### Step 2: Enable Extensions & Install azd-app

```typescript
const step2Content = {
  number: 2,
  title: "Enable Extensions & Install azd-app",
  description: "Enable the azd extensions feature and install the azd-app extension:",
  platformContent: {
    windows: [
      {
        type: 'code',
        language: 'powershell',
        title: 'PowerShell',
        content: `# Enable extensions
azd config set alpha.extensions.enabled on

# Add azd-app extension source
azd extension source add app https://raw.githubusercontent.com/jongio/azd-app/main/registry.json

# Install the extension
azd extension install jongio.azd.app`,
      },
    ],
    macos: [
      {
        type: 'code',
        language: 'bash',
        title: 'Terminal',
        content: `# Enable extensions
azd config set alpha.extensions.enabled on

# Add azd-app extension source
azd extension source add app https://raw.githubusercontent.com/jongio/azd-app/main/registry.json

# Install the extension
azd extension install jongio.azd.app`,
      },
    ],
    linux: [
      {
        type: 'code',
        language: 'bash',
        title: 'Terminal',
        content: `# Enable extensions
azd config set alpha.extensions.enabled on

# Add azd-app extension source
azd extension source add app https://raw.githubusercontent.com/jongio/azd-app/main/registry.json

# Install the extension
azd extension install jongio.azd.app`,
      },
    ],
  },
};
```

### Step 3: Clone Demo Template & Run

```typescript
const step3Content = {
  number: 3,
  title: "Clone Demo Template & Run",
  description: "Initialize the demo template and start all services. This template includes an intentional bug for you to fix!",
  content: [
    {
      type: 'code',
      language: 'bash',
      title: 'Terminal',
      content: `# Initialize the demo project
azd init -t jongio/azd-app-demo

# Navigate to the project
cd azd-app-demo

# Start all services
azd app run`,
    },
    {
      type: 'terminal',
      content: `$ azd app run
Starting services...
✓ web started on http://localhost:3000
✗ api: Connection timeout to database

💡 There's a bug! Continue to Step 4 to fix it with AI.`,
    },
    {
      type: 'tip',
      content: 'The dashboard will open at http://localhost:5050',
    },
  ],
};
```

### Step 4: Fix the Bug with AI

```typescript
const step4Content = {
  number: 4,
  title: "Notice the API Error, Then Ask Copilot to Help Fix It!",
  description: "You'll see an error in the logs. Now it's time to use AI to diagnose and fix the issue!",
  content: [
    {
      type: 'text',
      content: "Open the Copilot Chat panel in VS Code (Ctrl+Shift+I / Cmd+Shift+I) and ask about the error:",
    },
  ],
  challenge: {
    icon: "🎯",
    title: "Your Challenge",
    description: "The demo template has an intentional bug! The API service is failing with a connection timeout. Use GitHub Copilot to diagnose and fix the issue.",
    prompt: "Why is my API service failing? Can you help me fix it?",
  },
};
```

---

## 4. One-Click Start Command

The primary command for the quick start should be prominently displayed:

```typescript
const oneClickCommand = "azd init -t jongio/azd-app-demo";
```

#### Prominent Display

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│                                                                                  │
│  One-Click Start:                                                                │
│                                                                                  │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │                                                                            │ │
│  │  azd init -t jongio/azd-app-demo                                       📋 │ │
│  │                                                                            │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

#### Styling

```css
.one-click-command {
  font-size: 1.25rem;
  font-weight: 600;
  font-family: var(--font-family-mono);
  background: linear-gradient(135deg, var(--color-azure-50), var(--color-azure-100));
  border: 2px solid var(--color-azure-200);
  border-radius: 12px;
  padding: 20px 24px;
}

[data-theme="dark"] .one-click-command {
  background: linear-gradient(135deg, var(--color-azure-900), var(--color-azure-800));
  border-color: var(--color-azure-700);
}
```

---

## 5. Accessibility

### Semantic Structure

```html
<main id="main-content">
  <section aria-labelledby="quickstart-heading" class="quickstart-hero">
    <h1 id="quickstart-heading">Quick Start</h1>
    <!-- Hero content -->
  </section>
  
  <nav aria-label="Quick start progress" class="progress-indicator">
    <ol role="list">
      <li aria-current="step">Step 1: Install Azure CLI</li>
      <li>Step 2: Enable Extensions</li>
      <li>Step 3: Clone & Run</li>
      <li>Step 4: Fix with AI</li>
    </ol>
  </nav>
  
  <section aria-labelledby="steps-heading" class="steps-container">
    <h2 id="steps-heading" class="sr-only">Quick Start Steps</h2>
    
    <article aria-labelledby="step-1-title" class="step-card" id="step-1">
      <h3 id="step-1-title">
        <span class="step-number" aria-label="Step 1">1</span>
        Install Azure Developer CLI
      </h3>
      <!-- Step content -->
    </article>
    
    <!-- More steps... -->
  </section>
  
  <section aria-labelledby="next-steps-heading" class="next-steps">
    <h2 id="next-steps-heading">What's Next?</h2>
    <!-- Next steps cards -->
  </section>
</main>
```

### ARIA Attributes by Component

| Component        | Element              | ARIA Attribute                    |
|------------------|---------------------|-----------------------------------|
| Progress         | Container           | aria-label="Quick start progress" |
| Progress         | Step list           | role="list"                       |
| Progress         | Current step        | aria-current="step"               |
| Progress         | Completed step      | aria-label="Completed: Step 1..." |
| Step Card        | Card                | role="article"                    |
| Step Card        | Step number         | aria-label="Step N"               |
| Platform Tabs    | Tab container       | role="tablist"                    |
| Platform Tabs    | Tab                 | role="tab", aria-selected         |
| Platform Tabs    | Tab panel           | role="tabpanel", aria-labelledby  |
| Challenge        | Callout             | role="region", aria-label         |
| Challenge        | Prompt              | aria-label="Copilot prompt to copy"|
| Time Estimate    | Container           | role="status"                     |

### Keyboard Navigation

| Element           | Key            | Action                           |
|-------------------|----------------|----------------------------------|
| Progress steps    | Tab            | Move through interactive steps   |
| Platform tabs     | Arrow Left/Right | Switch between platforms       |
| Platform tabs     | Enter/Space    | Select platform                  |
| Copy buttons      | Enter/Space    | Copy content                     |
| Step cards        | Tab            | Navigate through step content    |
| Next step cards   | Tab            | Navigate between cards           |
| Next step cards   | Enter          | Navigate to page                 |

### Screen Reader Announcements

```typescript
// Progress updates
"Step 1 of 4: Install Azure CLI, current step"
"Step 2 completed"

// Platform selection
"Windows selected, tab 1 of 3"
"Tab panel: Windows installation commands"

// Copy feedback
"Command copied to clipboard"
"Copilot prompt copied to clipboard"

// Challenge callout
"Challenge region: Your Challenge. The demo template has an intentional bug..."
```

### Focus Management

- Skip link jumps to `#step-1`
- Platform tabs maintain focus when switching
- Copy button focus stays after copy action
- Anchor links scroll smoothly and set focus

### Reduced Motion

```css
@media (prefers-reduced-motion: reduce) {
  .progress-step--active .step-circle {
    animation: none;
  }
  
  .step-card {
    transition: none;
  }
  
  .challenge-callout {
    animation: none;
  }
}
```

---

## 6. Responsive Design

### Breakpoint Behavior

| Component         | Desktop (≥1024px) | Tablet (768-1023px) | Mobile (<768px) |
|-------------------|-------------------|---------------------|-----------------|
| Hero              | Centered, wide    | Centered            | Full width      |
| Progress          | Horizontal        | Horizontal          | Vertical        |
| Step Cards        | Max-width 800px   | Full width          | Full width      |
| Platform Tabs     | Inline tabs       | Inline tabs         | Full width tabs |
| Next Steps        | 3-column          | 2-column            | 1-column        |
| Time Estimate     | Inline            | Inline              | Below subtitle  |

### Mobile-First CSS

```css
/* Base (Mobile) */
.progress-indicator {
  flex-direction: column;
  align-items: flex-start;
  gap: var(--spacing-4);
}

.progress-connector {
  width: 2px;
  height: 24px;
  margin-left: 15px;
}

.step-card {
  padding: var(--spacing-5);
  margin: 0 calc(-1 * var(--spacing-4));
  border-radius: 0;
}

.platform-tabs {
  flex-direction: column;
}

.platform-tab {
  width: 100%;
}

/* Tablet */
@media (min-width: 768px) {
  .progress-indicator {
    flex-direction: row;
    justify-content: center;
    gap: var(--spacing-6);
  }
  
  .progress-connector {
    width: 80px;
    height: 2px;
    margin: 0;
  }
  
  .step-card {
    padding: var(--spacing-6);
    margin: 0;
    border-radius: var(--radius-lg);
  }
  
  .platform-tabs {
    flex-direction: row;
  }
  
  .platform-tab {
    width: auto;
  }
}

/* Desktop */
@media (min-width: 1024px) {
  .step-card {
    padding: var(--spacing-8);
    max-width: 800px;
    margin: 0 auto;
  }
}
```

---

## 7. Animation Specifications

### Progress Step Activation

```css
.progress-step {
  transition: all 0.3s var(--ease-default);
}

.progress-step--active .step-circle {
  animation: step-pulse 2s ease-in-out infinite;
  transform: scale(1.1);
}

@keyframes step-pulse {
  0%, 100% {
    box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.4);
  }
  50% {
    box-shadow: 0 0 0 12px rgba(59, 130, 246, 0);
  }
}
```

### Step Card Entrance

```css
.step-card {
  animation: step-card-in 0.5s var(--ease-out) both;
}

.step-card:nth-child(1) { animation-delay: 0.1s; }
.step-card:nth-child(2) { animation-delay: 0.2s; }
.step-card:nth-child(3) { animation-delay: 0.3s; }
.step-card:nth-child(4) { animation-delay: 0.4s; }

@keyframes step-card-in {
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

### Challenge Callout Attention

```css
.challenge-callout {
  animation: challenge-attention 0.5s var(--ease-bounce) 0.5s both;
}

@keyframes challenge-attention {
  0% {
    opacity: 0;
    transform: scale(0.95);
  }
  50% {
    transform: scale(1.02);
  }
  100% {
    opacity: 1;
    transform: scale(1);
  }
}
```

---

## 8. CSS Custom Properties

```css
/* Quick Start Page Tokens */
--quickstart-hero-padding: var(--spacing-12);
--quickstart-section-gap: var(--spacing-16);
--quickstart-content-max-width: 800px;

/* Progress Indicator Tokens */
--progress-step-size: 32px;
--progress-step-font: var(--font-size-sm);
--progress-connector-width: 80px;
--progress-connector-height: 2px;
--progress-active-color: var(--color-azure-500);
--progress-completed-color: var(--color-success);
--progress-inactive-color: var(--color-border-strong);

/* Step Card Tokens */
--step-card-padding: var(--spacing-8);
--step-card-radius: var(--radius-lg);
--step-card-gap: var(--spacing-5);
--step-number-size: 40px;
--step-title-font: var(--font-size-2xl);

/* Platform Tabs Tokens */
--platform-tab-height: 44px;
--platform-tab-padding: var(--spacing-3) var(--spacing-5);
--platform-tab-gap: 0;

/* Challenge Callout Tokens */
--challenge-padding: var(--spacing-6);
--challenge-radius: var(--radius-xl);
--challenge-border-width: 3px;
--challenge-icon-size: 32px;

/* Time Estimate Tokens */
--time-estimate-padding: var(--spacing-2) var(--spacing-3);
--time-estimate-radius: var(--radius-full);
```

---

## 9. Testing Checklist

### Functional Tests

- [ ] All platform tabs switch content correctly
- [ ] Copy buttons work for all code blocks
- [ ] Copilot prompt copies correctly
- [ ] Progress indicator reflects current step
- [ ] Next step cards navigate correctly
- [ ] Auto-detect platform works
- [ ] Anchor links work for step navigation

### Accessibility Tests

- [ ] All headings in correct hierarchy
- [ ] Skip link works
- [ ] Platform tabs keyboard accessible
- [ ] Screen reader announces progress
- [ ] Copy feedback announced
- [ ] Challenge callout announced correctly
- [ ] Focus visible on all elements
- [ ] Color contrast ≥ 4.5:1
- [ ] Touch targets ≥ 44x44px
- [ ] Reduced motion respected

### Responsive Tests

- [ ] Progress indicator switches orientation
- [ ] Step cards full-width on mobile
- [ ] Platform tabs stack on mobile
- [ ] Next steps grid adjusts
- [ ] Text readable at all sizes
- [ ] No horizontal scroll

### Cross-Browser Tests

- [ ] Chrome (latest)
- [ ] Firefox (latest)
- [ ] Safari (latest)
- [ ] Edge (latest)
- [ ] iOS Safari
- [ ] Chrome for Android

---

## 10. Implementation Notes

### Component Files

```
web/src/components/
├── quickstart/
│   ├── QuickStartHero.astro
│   ├── TimeEstimate.astro
│   ├── ProgressIndicator.astro
│   ├── ProgressStep.astro
│   ├── StepCard.astro
│   ├── PlatformTabs.astro
│   ├── ChallengeCallout.astro
│   ├── CopilotPrompt.astro
│   └── NextStepsSection.astro
```

### Page File

```astro
---
// web/src/pages/quickstart.astro
import Layout from '../layouts/Layout.astro';
import QuickStartHero from '../components/quickstart/QuickStartHero.astro';
import ProgressIndicator from '../components/quickstart/ProgressIndicator.astro';
import StepCard from '../components/quickstart/StepCard.astro';
import NextStepsSection from '../components/quickstart/NextStepsSection.astro';
import { steps, nextSteps } from '../data/quickstart';
---

<Layout title="Quick Start | azd-app">
  <QuickStartHero />
  <ProgressIndicator steps={steps} />
  
  <div class="steps-container">
    {steps.map((step) => (
      <StepCard {...step} />
    ))}
  </div>
  
  <NextStepsSection steps={nextSteps} />
</Layout>
```

### Data Files

```typescript
// web/src/data/quickstart.ts
export const steps = [ ... ];
export const nextSteps = [ ... ];
export const platformCommands = { ... };
```

---

## 11. Related Components

- [Header](./header.md) - Top navigation
- [Footer](./footer.md) - Bottom navigation and links
- [Code Block](./code-block.md) - Code display with copy
- [Copy Button](./copy-button.md) - Copy to clipboard functionality
- [Terminal](./terminal.md) - Terminal demo display
- [Landing Page](./landing-page.md) - Home page with similar sections

````
