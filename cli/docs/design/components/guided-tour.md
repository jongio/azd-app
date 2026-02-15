````markdown
# Guided Tour Component Specification

## Overview

The Guided Tour provides a progressive, 8-step tutorial experience for learning azd-app features. It includes persistent progress tracking with localStorage, step-by-step navigation, estimated completion times, and interactive "Try it yourself" prompts. The tour is designed to be completable across multiple sessions with full progress restoration.

---

## 1. Component Hierarchy

```
GuidedTourLayout (template)
├── Header (organism) [from header.md]
├── TourProgressSidebar (organism)
│   ├── TourHeader (molecule)
│   │   ├── TourTitle (atom)
│   │   └── TourProgress (atom)
│   ├── TourStepList (molecule)
│   │   └── TourStepItem (atom) × 8
│   └── TourActions (molecule)
│       ├── ResetButton (atom)
│       └── TotalTimeEstimate (atom)
├── TourMainContent (organism)
│   ├── TourStepPage (template)
│   │   ├── TourStepHeader (molecule)
│   │   │   ├── StepBreadcrumb (atom)
│   │   │   ├── StepNumber (atom)
│   │   │   ├── StepTitle (atom)
│   │   │   ├── TimeEstimate (atom)
│   │   │   └── CompletionCheckbox (molecule)
│   │   ├── TourStepContent (organism)
│   │   │   ├── StepIntro (molecule)
│   │   │   ├── StepInstructions (molecule)
│   │   │   │   ├── InstructionBlock (molecule)
│   │   │   │   ├── CodeBlock (molecule) [from code-block.md]
│   │   │   │   └── Screenshot (molecule) [from screenshot.md]
│   │   │   ├── TryItYourself (molecule)
│   │   │   │   ├── PromptIcon (atom)
│   │   │   │   ├── PromptText (atom)
│   │   │   │   └── PromptAction (atom)
│   │   │   └── LearnMoreSection (molecule)
│   │   │       ├── ExpandButton (atom)
│   │   │       └── ExpandableContent (molecule)
│   │   └── TourStepNavigation (molecule)
│   │       ├── PreviousButton (atom)
│   │       ├── StepIndicator (atom)
│   │       └── NextButton (atom)
│   └── TourCompletionPage (template) [after step 8]
│       ├── CompletionHero (molecule)
│       ├── AchievementBadges (molecule)
│       └── NextStepsCTA (molecule)
└── Footer (organism) [from footer.md]
```

---

## 2. Tour Structure

### Tour Steps Content

```typescript
interface TourStep {
  /** Step number (1-8) */
  number: number;
  /** URL slug */
  slug: string;
  /** Step title */
  title: string;
  /** Short description */
  description: string;
  /** Estimated completion time in minutes */
  estimatedTime: number;
  /** Step icon/emoji */
  icon: string;
  /** Prerequisites (previous step numbers) */
  prerequisites: number[];
  /** Related documentation links */
  relatedDocs: string[];
}

const tourSteps: TourStep[] = [
  {
    number: 1,
    slug: "install",
    title: "Install azd + extension",
    description: "Set up the Azure Developer CLI and install the azd-app extension",
    estimatedTime: 5,
    icon: "📦",
    prerequisites: [],
    relatedDocs: ["/docs/installation"],
  },
  {
    number: 2,
    slug: "requirements",
    title: "Check requirements",
    description: "Verify your system meets all requirements for running azd-app",
    estimatedTime: 3,
    icon: "✅",
    prerequisites: [1],
    relatedDocs: ["/docs/requirements"],
  },
  {
    number: 3,
    slug: "dependencies",
    title: "Install dependencies",
    description: "Install required dependencies and configure your development environment",
    estimatedTime: 5,
    icon: "🔧",
    prerequisites: [1, 2],
    relatedDocs: ["/docs/dependencies"],
  },
  {
    number: 4,
    slug: "first-app",
    title: "Run your first app",
    description: "Initialize and run a demo application using azd-app",
    estimatedTime: 7,
    icon: "🚀",
    prerequisites: [1, 2, 3],
    relatedDocs: ["/docs/quick-start", "/docs/commands/run"],
  },
  {
    number: 5,
    slug: "dashboard",
    title: "Explore the dashboard",
    description: "Navigate the azd-app dashboard and understand its features",
    estimatedTime: 5,
    icon: "📊",
    prerequisites: [4],
    relatedDocs: ["/docs/dashboard"],
  },
  {
    number: 6,
    slug: "logs",
    title: "View and filter logs",
    description: "Learn to view, filter, and search logs for debugging",
    estimatedTime: 5,
    icon: "📋",
    prerequisites: [4, 5],
    relatedDocs: ["/docs/commands/logs"],
  },
  {
    number: 7,
    slug: "health",
    title: "Monitor service health",
    description: "Monitor service health and troubleshoot issues",
    estimatedTime: 5,
    icon: "💓",
    prerequisites: [4, 5],
    relatedDocs: ["/docs/commands/health"],
  },
  {
    number: 8,
    slug: "mcp",
    title: "MCP server integration",
    description: "Connect GitHub Copilot to azd-app using the MCP server",
    estimatedTime: 10,
    icon: "🤖",
    prerequisites: [4],
    relatedDocs: ["/docs/mcp", "/docs/ai-features"],
  },
];

// Total estimated time: 45 minutes
```

---

## 3. Component Specifications

### 3.1 Tour Progress Sidebar

#### Purpose
Display persistent progress through the 8-step tour with completion status for each step.

#### Props Interface

```typescript
interface TourProgressSidebarProps {
  /** All tour steps */
  steps: TourStep[];
  /** Current step number */
  currentStep: number;
  /** Completed step numbers */
  completedSteps: number[];
  /** Callback when step is clicked */
  onStepClick: (stepNumber: number) => void;
  /** Callback to reset progress */
  onReset: () => void;
  /** Whether sidebar is open (mobile) */
  isOpen?: boolean;
  /** Callback when sidebar open state changes */
  onOpenChange?: (isOpen: boolean) => void;
  /** Custom class name */
  className?: string;
}
```

#### Layout

```
Desktop (Always Visible):
┌────────────────────────────────────────────────────────────────────────────────┐
│  ┌──────────────────────┬───────────────────────────────────────────────────┐  │
│  │  🎯 Guided Tour      │                                                   │  │
│  │  ─────────────────   │                                                   │  │
│  │  Progress: 3 of 8    │                                                   │  │
│  │  ▓▓▓▓▓▓░░░░░ 38%    │                                                   │  │
│  │                      │                                                   │  │
│  │  ✓ 1. Install azd    │               Main Content                       │  │
│  │  ✓ 2. Requirements   │                                                   │  │
│  │  ● 3. Dependencies   │                                                   │  │
│  │  ○ 4. First app      │                                                   │  │
│  │  ○ 5. Dashboard      │                                                   │  │
│  │  ○ 6. Logs           │                                                   │  │
│  │  ○ 7. Health         │                                                   │  │
│  │  ○ 8. MCP server     │                                                   │  │
│  │                      │                                                   │  │
│  │  ─────────────────   │                                                   │  │
│  │  Total: ~45 min      │                                                   │  │
│  │  [Reset Progress]    │                                                   │  │
│  └──────────────────────┴───────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────────────────┘

Mobile (Overlay):
┌─────────────────────────────────────┐
│  [☰ Show Progress]                  │
│                                     │
│  Step 3 of 8 content...             │
│                                     │
└─────────────────────────────────────┘
```

#### Dimensions

| Property              | Desktop    | Tablet         | Mobile      |
|-----------------------|------------|----------------|-------------|
| Width                 | 280px      | 240px          | 300px       |
| Min height            | 100vh - 64px | 100vh - 64px | auto        |
| Padding               | 24px       | 20px           | 20px        |
| Step item height      | 44px       | 44px           | 52px        |
| Step gap              | 4px        | 4px            | 8px         |
| Progress bar height   | 8px        | 8px            | 8px         |

#### Colors

| Element                    | Light Theme         | Dark Theme          |
|----------------------------|---------------------|---------------------|
| Sidebar bg                 | #f9fafb             | #1e293b             |
| Border                     | #e5e7eb             | #334155             |
| Title color                | --color-text-primary| --color-text-primary|
| Progress text              | --color-text-secondary| --color-text-secondary|
| Progress bar bg            | --color-bg-tertiary | --color-bg-tertiary |
| Progress bar fill          | --color-azure-500   | --color-azure-500   |
| Step completed icon        | --color-success     | --color-success     |
| Step current bg            | --color-azure-50    | --color-azure-900/30|
| Step current border        | --color-azure-500   | --color-azure-500   |
| Step upcoming text         | --color-text-tertiary| --color-text-tertiary|
| Reset button               | --color-error       | --color-error       |

#### Step Item States

```
Completed:
┌──────────────────────────────────────┐
│  ✓  1. Install azd + extension       │  ← Green checkmark
└──────────────────────────────────────┘

Current:
┌══════════════════════════════════════┐
│  ●  3. Install dependencies          │  ← Blue dot, highlighted bg
└══════════════════════════════════════┘

Upcoming:
┌──────────────────────────────────────┐
│  ○  4. Run your first app            │  ← Gray circle, muted
└──────────────────────────────────────┘

Locked (prerequisites not met):
┌──────────────────────────────────────┐
│  🔒 8. MCP server integration        │  ← Lock icon, disabled
└──────────────────────────────────────┘
```

---

### 3.2 Tour Step Page Layout

#### Purpose
Display a single step with all instructions, screenshots, and interactive elements.

#### Props Interface

```typescript
interface TourStepPageProps {
  /** Step data */
  step: TourStep;
  /** Whether step is completed */
  isCompleted: boolean;
  /** Callback when completion changes */
  onCompletionChange: (completed: boolean) => void;
  /** Previous step (null if first) */
  previousStep: TourStep | null;
  /** Next step (null if last) */
  nextStep: TourStep | null;
  /** Step content (MDX/Astro component) */
  children: React.ReactNode;
  /** Custom class name */
  className?: string;
}
```

#### Layout

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│  ◀ Back to Tour                                        Step 3 of 8              │
│                                                                                  │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │  🔧  Step 3                                                                │ │
│  │                                                                            │ │
│  │  Install Dependencies                                              ~5 min │ │
│  │                                                                            │ │
│  │  ┌─────────────────────────────────────────────────────────────────────┐  │ │
│  │  │  ☐  Mark as complete                                                │  │ │
│  │  └─────────────────────────────────────────────────────────────────────┘  │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                  │
│  In this step, you'll install the required dependencies...                      │
│                                                                                  │
│  ## Prerequisites                                                                │
│                                                                                  │
│  Before you begin, make sure you have:                                          │
│  • Node.js 18+ installed                                                        │
│  • npm or pnpm package manager                                                   │
│                                                                                  │
│  ## Instructions                                                                 │
│                                                                                  │
│  1. Open your terminal                                                           │
│                                                                                  │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │ bash                                                                   📋  │ │
│  ├────────────────────────────────────────────────────────────────────────────┤ │
│  │ azd app deps install                                                       │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                  │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │  📸 Screenshot                                                             │ │
│  │  ┌──────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                                                                      │ │ │
│  │  │              Terminal showing deps install output                    │ │ │
│  │  │                                                                      │ │ │
│  │  └──────────────────────────────────────────────────────────────────────┘ │ │
│  │  Dependencies installation output                                          │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                  │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │  🎯 Try It Yourself                                                        │ │
│  │                                                                            │ │
│  │  Run the deps install command and verify all dependencies are installed.  │ │
│  │  You should see a success message with the list of installed packages.    │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                  │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │  📚 Learn More                                                      [▼]   │ │
│  ├────────────────────────────────────────────────────────────────────────────┤ │
│  │  ### Understanding Dependencies                                            │ │
│  │                                                                            │ │
│  │  azd-app manages dependencies through...                                   │ │
│  │  (Expandable content with detailed explanation)                            │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                  │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │                                                                            │ │
│  │  ◀ Previous: Check requirements        Next: Run your first app ▶         │ │
│  │                                                                            │ │
│  │                          ○ ○ ● ○ ○ ○ ○ ○                                   │ │
│  │                                                                            │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

#### Dimensions

| Property              | Desktop         | Tablet          | Mobile          |
|-----------------------|-----------------|-----------------|-----------------|
| Content max-width     | 800px           | 100%            | 100%            |
| Content padding       | 48px            | 32px            | 20px            |
| Section gap           | 32px            | 24px            | 20px            |
| Step header padding   | 24px            | 20px            | 16px            |
| Navigation padding    | 24px            | 20px            | 16px            |
| Navigation gap        | 16px            | 12px            | 8px             |

---

### 3.3 Completion Checkbox

#### Purpose
Allow users to mark a step as complete with visual feedback and localStorage persistence.

#### Props Interface

```typescript
interface CompletionCheckboxProps {
  /** Step number */
  stepNumber: number;
  /** Whether step is completed */
  isCompleted: boolean;
  /** Callback when completion changes */
  onChange: (completed: boolean) => void;
  /** Label text */
  label?: string;
  /** Custom class name */
  className?: string;
}
```

#### Layout

```
Unchecked:
┌─────────────────────────────────────────────────────────┐
│  ☐  Mark as complete                                    │
└─────────────────────────────────────────────────────────┘

Checked:
┌─────────────────────────────────────────────────────────┐
│  ☑  Completed! Great job.                               │
└─────────────────────────────────────────────────────────┘

Hover (unchecked):
┌═════════════════════════════════════════════════════════┐
│  ☐  Mark as complete                                    │
└═════════════════════════════════════════════════════════┘
```

#### States

| State       | Visual Changes                                              |
|-------------|-------------------------------------------------------------|
| Unchecked   | Empty checkbox, neutral border, default text                |
| Hover       | Border highlight, slight bg color                           |
| Focus       | Focus ring visible                                          |
| Checked     | Filled checkbox with checkmark, success color, updated text |
| Animating   | Checkmark scales in with bounce                             |

#### Dimensions

| Property              | Value           |
|-----------------------|-----------------|
| Container padding     | 16px 20px       |
| Checkbox size         | 24px × 24px     |
| Checkbox border       | 2px             |
| Checkbox border-radius| 6px             |
| Label font size       | 16px            |
| Gap                   | 12px            |
| Border radius         | 8px             |

#### Colors

| Element                    | Light Theme (unchecked) | Light Theme (checked)   |
|----------------------------|-------------------------|-------------------------|
| Container bg               | --color-bg-tertiary     | rgba(16,185,129,0.1)    |
| Container border           | --color-border-default  | --color-success         |
| Checkbox border            | --color-border-strong   | --color-success         |
| Checkbox bg                | transparent             | --color-success         |
| Checkmark color            | -                       | white                   |
| Label color                | --color-text-secondary  | --color-text-primary    |

#### Animation

```css
.completion-checkbox input:checked + .checkbox-icon {
  animation: checkbox-check 0.3s var(--ease-bounce);
}

@keyframes checkbox-check {
  0% { transform: scale(0.8); }
  50% { transform: scale(1.1); }
  100% { transform: scale(1); }
}

.completion-checkbox input:checked ~ .checkbox-label {
  animation: label-update 0.2s ease-out;
}

@keyframes label-update {
  0% { opacity: 0.5; }
  100% { opacity: 1; }
}
```

---

### 3.4 Tour Navigation Controls

#### Purpose
Navigate between tour steps with previous/next buttons and step indicator dots.

#### Props Interface

```typescript
interface TourNavigationProps {
  /** Current step number */
  currentStep: number;
  /** Total steps */
  totalSteps: number;
  /** Previous step (null if first) */
  previousStep: TourStep | null;
  /** Next step (null if last) */
  nextStep: TourStep | null;
  /** Callback for previous */
  onPrevious: () => void;
  /** Callback for next */
  onNext: () => void;
  /** Completed steps for dot indicators */
  completedSteps: number[];
  /** Custom class name */
  className?: string;
}
```

#### Layout

```
With both buttons:
┌──────────────────────────────────────────────────────────────────────────────────┐
│                                                                                  │
│  ◀ Previous: Check requirements              Next: Run your first app ▶         │
│                                                                                  │
│                          ● ● ● ○ ○ ○ ○ ○                                        │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘

First step (no previous):
┌──────────────────────────────────────────────────────────────────────────────────┐
│                                                                                  │
│                                     Next: Check requirements ▶                   │
│                                                                                  │
│                          ● ○ ○ ○ ○ ○ ○ ○                                        │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘

Last step:
┌──────────────────────────────────────────────────────────────────────────────────┐
│                                                                                  │
│  ◀ Previous: Monitor health              Complete Tour! 🎉                      │
│                                                                                  │
│                          ● ● ● ● ● ● ● ●                                        │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘

Mobile (stacked):
┌─────────────────────────────────────┐
│                                     │
│  ● ● ● ○ ○ ○ ○ ○                   │
│                                     │
│  ┌────────────────────────────────┐│
│  │ ◀ Previous: Check requirements ││
│  └────────────────────────────────┘│
│                                     │
│  ┌────────────────────────────────┐│
│  │ Next: Run your first app ▶     ││
│  └────────────────────────────────┘│
│                                     │
└─────────────────────────────────────┘
```

#### Dimensions

| Property              | Desktop         | Tablet          | Mobile          |
|-----------------------|-----------------|-----------------|-----------------|
| Container padding     | 24px            | 20px            | 16px            |
| Button padding        | 12px 24px       | 12px 20px       | 12px 16px       |
| Button font size      | 16px            | 15px            | 15px            |
| Dot size              | 10px            | 10px            | 8px             |
| Dot gap               | 8px             | 8px             | 6px             |
| Nav gap               | 16px            | 16px            | 12px            |

#### Colors

| Element                    | Light Theme         | Dark Theme          |
|----------------------------|---------------------|---------------------|
| Previous button bg         | transparent         | transparent         |
| Previous button text       | --color-text-secondary| --color-text-secondary|
| Previous button border     | --color-border-default| --color-border-default|
| Next button bg             | --color-azure-500   | --color-azure-500   |
| Next button text           | white               | white               |
| Dot inactive               | --color-border-strong| --color-border-strong|
| Dot completed              | --color-success     | --color-success     |
| Dot current                | --color-azure-500   | --color-azure-500   |

---

### 3.5 Try It Yourself Prompt

#### Purpose
Encourage users to practice with hands-on actions and provide clear expectations.

#### Props Interface

```typescript
interface TryItYourselfProps {
  /** Prompt text */
  prompt: string;
  /** Expected outcome */
  expectedOutcome?: string;
  /** Optional action button */
  action?: {
    label: string;
    href?: string;
    onClick?: () => void;
  };
  /** Custom class name */
  className?: string;
}
```

#### Layout

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│  ╔═══════════════════════════════════════════════════════════════════════════╗  │
│  ║                                                                           ║  │
│  ║  🎯 Try It Yourself                                                       ║  │
│  ║                                                                           ║  │
│  ║  Run the deps install command and verify all dependencies are installed. ║  │
│  ║                                                                           ║  │
│  ║  Expected: You should see a success message listing all packages.        ║  │
│  ║                                                                           ║  │
│  ║  ┌─────────────────────────────────────────────────────────────────────┐ ║  │
│  ║  │  Open Terminal →                                                    │ ║  │
│  ║  └─────────────────────────────────────────────────────────────────────┘ ║  │
│  ║                                                                           ║  │
│  ╚═══════════════════════════════════════════════════════════════════════════╝  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

#### Dimensions

| Property              | Value           |
|-----------------------|-----------------|
| Container padding     | 24px            |
| Container margin      | 24px 0          |
| Border width          | 2px             |
| Border radius         | 12px            |
| Title font size       | 18px            |
| Content font size     | 16px            |
| Action button height  | 44px            |

#### Colors

| Element                    | Light Theme         | Dark Theme          |
|----------------------------|---------------------|---------------------|
| Background                 | linear-gradient(135deg, #fef3c7, #fef9c3) | linear-gradient(135deg, #422006, #451a03) |
| Border                     | --color-mcp-badge-border | --color-mcp-badge-border |
| Title color                | --color-text-primary | --color-text-primary |
| Content color              | --color-text-secondary | --color-text-secondary |
| Action button bg           | white               | --color-bg-primary  |
| Action button text         | --color-text-primary | --color-text-primary |
| Action button border       | --color-border-default | --color-border-default |

---

### 3.6 Learn More Section

#### Purpose
Provide expandable supplementary content without overwhelming the main flow.

#### Props Interface

```typescript
interface LearnMoreSectionProps {
  /** Section title */
  title?: string;
  /** Whether section is expanded */
  isExpanded?: boolean;
  /** Expandable content */
  children: React.ReactNode;
  /** Custom class name */
  className?: string;
}
```

#### Layout

```
Collapsed:
┌──────────────────────────────────────────────────────────────────────────────────┐
│  📚 Learn More                                                            [▶]   │
└──────────────────────────────────────────────────────────────────────────────────┘

Expanded:
┌──────────────────────────────────────────────────────────────────────────────────┐
│  📚 Learn More                                                            [▼]   │
├──────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  ### Understanding Dependencies                                                  │
│                                                                                  │
│  azd-app uses a sophisticated dependency management system that...              │
│                                                                                  │
│  Key concepts:                                                                   │
│  • Local dependencies vs. cloud dependencies                                     │
│  • Version resolution strategy                                                   │
│  • Caching and performance                                                       │
│                                                                                  │
│  For more details, see the [Dependencies Documentation](/docs/dependencies).    │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

#### States

| State       | Chevron Direction | Content Visibility |
|-------------|-------------------|-------------------|
| Collapsed   | ▶ (pointing right)| Hidden            |
| Expanded    | ▼ (pointing down) | Visible           |
| Animating   | Rotating          | Transitioning     |

#### Dimensions

| Property              | Value           |
|-----------------------|-----------------|
| Header padding        | 16px 20px       |
| Content padding       | 0 20px 20px     |
| Header font size      | 16px            |
| Chevron size          | 20px            |
| Border radius         | 8px             |

#### Animation

```css
.learn-more-content {
  display: grid;
  grid-template-rows: 0fr;
  transition: grid-template-rows 0.3s ease-out;
}

.learn-more--expanded .learn-more-content {
  grid-template-rows: 1fr;
}

.learn-more-content__inner {
  overflow: hidden;
}

.learn-more-chevron {
  transition: transform 0.2s ease-out;
}

.learn-more--expanded .learn-more-chevron {
  transform: rotate(90deg);
}
```

---

### 3.7 Tour Completion Page

#### Purpose
Celebrate user completion and guide them to next actions.

#### Props Interface

```typescript
interface TourCompletionPageProps {
  /** Total time spent (calculated) */
  totalTimeSpent?: number;
  /** Completion date */
  completedAt: Date;
  /** Next step suggestions */
  nextSteps: NextStepCard[];
  /** Callback to restart tour */
  onRestartTour: () => void;
  /** Custom class name */
  className?: string;
}
```

#### Layout

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│                                                                                  │
│                              🎉                                                  │
│                                                                                  │
│                    Congratulations!                                              │
│                 You've completed the tour!                                       │
│                                                                                  │
│              All 8 steps completed • Total time: ~45 min                        │
│                                                                                  │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐                    │
│  │  📦 Install     │ │  ✅ Reqs        │ │  🔧 Deps        │                    │
│  │  Completed      │ │  Completed      │ │  Completed      │                    │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘                    │
│                                                                                  │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐                    │
│  │  🚀 First App   │ │  📊 Dashboard   │ │  📋 Logs        │                    │
│  │  Completed      │ │  Completed      │ │  Completed      │                    │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘                    │
│                                                                                  │
│  ┌─────────────────┐ ┌─────────────────┐                                        │
│  │  💓 Health      │ │  🤖 MCP         │                                        │
│  │  Completed      │ │  Completed      │                                        │
│  └─────────────────┘ └─────────────────┘                                        │
│                                                                                  │
│                            What's Next?                                          │
│                                                                                  │
│  ┌────────────────────────┐ ┌────────────────────────┐ ┌────────────────────────┐│
│  │  📚 Documentation      │ │  🤖 AI Features        │ │  🔁 Restart Tour       ││
│  │  Explore full docs     │ │  Deep dive into MCP    │ │  Review the steps      ││
│  └────────────────────────┘ └────────────────────────┘ └────────────────────────┘│
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Progress Persistence

### localStorage Schema

```typescript
interface TourProgress {
  /** Version for migration */
  version: 1;
  /** Completed step numbers */
  completedSteps: number[];
  /** Last active step */
  lastActiveStep: number;
  /** Timestamp of last update */
  lastUpdated: string; // ISO date string
  /** Expanded "Learn More" sections */
  expandedSections: string[];
}

const STORAGE_KEY = 'azd-app:tour:progress';

// Initial state
const defaultProgress: TourProgress = {
  version: 1,
  completedSteps: [],
  lastActiveStep: 1,
  lastUpdated: new Date().toISOString(),
  expandedSections: [],
};
```

### Persistence Functions

```typescript
// Load progress from localStorage
function loadTourProgress(): TourProgress {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (!stored) return defaultProgress;
    
    const parsed = JSON.parse(stored);
    // Validate and migrate if needed
    if (parsed.version !== 1) {
      return migrateProgress(parsed);
    }
    return parsed;
  } catch {
    return defaultProgress;
  }
}

// Save progress to localStorage
function saveTourProgress(progress: TourProgress): void {
  try {
    const updated = {
      ...progress,
      lastUpdated: new Date().toISOString(),
    };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(updated));
  } catch (error) {
    console.error('Failed to save tour progress:', error);
  }
}

// Mark step as complete
function markStepComplete(stepNumber: number): void {
  const progress = loadTourProgress();
  if (!progress.completedSteps.includes(stepNumber)) {
    progress.completedSteps.push(stepNumber);
    progress.completedSteps.sort((a, b) => a - b);
  }
  saveTourProgress(progress);
}

// Reset progress
function resetTourProgress(): void {
  localStorage.removeItem(STORAGE_KEY);
}
```

### State Restoration

```typescript
// On page load
useEffect(() => {
  const progress = loadTourProgress();
  setCompletedSteps(progress.completedSteps);
  
  // If returning user, offer to resume
  if (progress.completedSteps.length > 0 && currentStep === 1) {
    showResumePrompt(progress.lastActiveStep);
  }
}, []);

// On step change
useEffect(() => {
  const progress = loadTourProgress();
  progress.lastActiveStep = currentStep;
  saveTourProgress(progress);
}, [currentStep]);
```

---

## 5. Step Page Templates

### Step 1: Install azd + extension

```typescript
const step1: TourStepContent = {
  title: "Install azd + extension",
  intro: "Let's start by installing the Azure Developer CLI and the azd-app extension.",
  sections: [
    {
      type: "prerequisites",
      items: [
        "Windows 10/11, macOS 10.15+, or Ubuntu 20.04+",
        "Administrator/sudo access for installation",
      ],
    },
    {
      type: "platform-instructions",
      platforms: {
        windows: [
          { type: "code", language: "powershell", content: "winget install microsoft.azd" },
        ],
        macos: [
          { type: "code", language: "bash", content: "brew tap azure/azd && brew install azd" },
        ],
        linux: [
          { type: "code", language: "bash", content: "curl -fsSL https://aka.ms/install-azd.sh | bash" },
        ],
      },
    },
    {
      type: "code",
      title: "Install the extension",
      language: "bash",
      content: `# Add extension source
azd extension source add -n jongio -t url -l https://jongio.github.io/azd-extensions/registry.json

# Install
azd extension install jongio.azd.app`,
    },
    {
      type: "screenshot",
      lightSrc: "tour/step1-install-light.png",
      darkSrc: "tour/step1-install-dark.png",
      alt: "Terminal showing successful azd installation",
      caption: "Expected output after installation",
    },
    {
      type: "try-it-yourself",
      prompt: "Run `azd version` to verify the installation was successful.",
      expectedOutcome: "You should see the azd version number displayed.",
    },
    {
      type: "learn-more",
      content: "### Alternative Installation Methods\n\n...",
    },
  ],
};
```

### Step 4: Run your first app

```typescript
const step4: TourStepContent = {
  title: "Run your first app",
  intro: "Now let's run a demo application to see azd-app in action.",
  sections: [
    {
      type: "code",
      title: "Initialize and run",
      language: "bash",
      content: `# Initialize the demo project
azd init -t jongio/azd-app-demo

# Navigate to the project
cd azd-app-demo

# Start all services
azd app run`,
    },
    {
      type: "screenshot",
      lightSrc: "tour/step4-dashboard-light.png",
      darkSrc: "tour/step4-dashboard-dark.png",
      alt: "azd-app dashboard showing running services",
      caption: "The dashboard opens at http://localhost:5050",
    },
    {
      type: "try-it-yourself",
      prompt: "Wait for all services to start, then open the dashboard URL in your browser.",
      expectedOutcome: "You should see the dashboard with green health indicators for all services.",
    },
  ],
};
```

### Step 8: MCP server integration

```typescript
const step8: TourStepContent = {
  title: "MCP server integration",
  intro: "Connect GitHub Copilot to your running services using the Model Context Protocol server.",
  badge: { text: "AI", variant: "ai" },
  sections: [
    {
      type: "code",
      title: "Start the MCP server",
      language: "bash",
      content: "azd app mcp",
    },
    {
      type: "code",
      title: "Configure VS Code settings",
      language: "json",
      filename: "settings.json",
      content: `{
  "github.copilot.chat.experimental.mcp": true,
  "github.copilot.chat.mcp.servers": {
    "azd-app": {
      "command": "azd",
      "args": ["app", "mcp"]
    }
  }
}`,
    },
    {
      type: "screenshot",
      lightSrc: "tour/step8-mcp-light.png",
      darkSrc: "tour/step8-mcp-dark.png",
      alt: "GitHub Copilot Chat using azd-app MCP tools",
      caption: "Copilot can now access your service logs and health status",
    },
    {
      type: "try-it-yourself",
      prompt: "Open Copilot Chat and ask: 'What services are running and what's their health status?'",
      expectedOutcome: "Copilot should respond with information about your running services.",
    },
  ],
};
```

---

## 6. Accessibility

### Semantic Structure

```html
<main id="main-content" class="tour-layout">
  <aside class="tour-sidebar" aria-label="Tour progress">
    <nav aria-label="Tour steps">
      <h2 id="tour-title">Guided Tour</h2>
      
      <div role="status" aria-live="polite" class="tour-progress">
        <span>Progress: 3 of 8 steps completed</span>
        <progress value="3" max="8" aria-label="Tour progress: 3 of 8 steps"></progress>
      </div>
      
      <ol role="list" aria-label="Tour steps">
        <li>
          <a 
            href="/tour/install"
            aria-current="false"
            aria-label="Step 1: Install azd, completed"
          >
            <span aria-hidden="true">✓</span>
            <span>1. Install azd + extension</span>
          </a>
        </li>
        <li>
          <a 
            href="/tour/dependencies"
            aria-current="page"
            aria-label="Step 3: Install dependencies, current step"
          >
            <span aria-hidden="true">●</span>
            <span>3. Install dependencies</span>
          </a>
        </li>
        <!-- ... more steps ... -->
      </ol>
    </nav>
  </aside>
  
  <article class="tour-step" aria-labelledby="step-title">
    <header class="tour-step-header">
      <nav aria-label="Breadcrumb">
        <ol>
          <li><a href="/tour">Guided Tour</a></li>
          <li aria-current="page">Step 3</li>
        </ol>
      </nav>
      
      <div class="step-meta">
        <span aria-hidden="true">🔧</span>
        <span class="step-number" aria-label="Step 3">Step 3</span>
        <h1 id="step-title">Install Dependencies</h1>
        <span class="time-estimate" role="status">
          <span aria-hidden="true">⏱</span>
          <span>~5 minutes</span>
        </span>
      </div>
      
      <div class="completion-checkbox">
        <label>
          <input 
            type="checkbox" 
            aria-describedby="completion-hint"
            checked={isCompleted}
          />
          <span>{isCompleted ? "Completed!" : "Mark as complete"}</span>
        </label>
        <span id="completion-hint" class="sr-only">
          Marks this step as complete and saves your progress
        </span>
      </div>
    </header>
    
    <div class="tour-step-content">
      <!-- Content sections -->
      
      <section 
        class="try-it-yourself" 
        role="region" 
        aria-label="Try it yourself challenge"
      >
        <h3>🎯 Try It Yourself</h3>
        <p>...</p>
      </section>
      
      <section class="learn-more">
        <button
          aria-expanded="false"
          aria-controls="learn-more-content"
          class="learn-more-toggle"
        >
          <span>📚 Learn More</span>
          <span aria-hidden="true">▶</span>
        </button>
        <div id="learn-more-content" hidden>
          <!-- Expandable content -->
        </div>
      </section>
    </div>
    
    <nav class="tour-navigation" aria-label="Step navigation">
      <a href="/tour/requirements" class="nav-previous">
        <span aria-hidden="true">◀</span>
        <span>Previous: Check requirements</span>
      </a>
      
      <div class="step-indicators" role="group" aria-label="Step progress">
        <span 
          v-for="step in 8"
          :aria-label="`Step ${step}, ${getStepStatus(step)}`"
          :class="getStepClass(step)"
        ></span>
      </div>
      
      <a href="/tour/first-app" class="nav-next">
        <span>Next: Run your first app</span>
        <span aria-hidden="true">▶</span>
      </a>
    </nav>
  </article>
</main>
```

### ARIA Attributes by Component

| Component           | Element          | ARIA Attribute                            |
|---------------------|------------------|-------------------------------------------|
| Sidebar             | Container        | aria-label="Tour progress"                |
| Progress            | Container        | role="status", aria-live="polite"         |
| Progress            | Bar              | aria-label="Tour progress: X of 8 steps"  |
| Step List           | Container        | role="list", aria-label="Tour steps"      |
| Step Item           | Link             | aria-current="page" (current step)        |
| Step Item           | Link             | aria-label with completion status         |
| Step Header         | Breadcrumb       | aria-label="Breadcrumb"                   |
| Completion Checkbox | Input            | aria-describedby="completion-hint"        |
| Try It Yourself     | Section          | role="region", aria-label                 |
| Learn More          | Button           | aria-expanded, aria-controls              |
| Navigation          | Container        | aria-label="Step navigation"              |
| Step Indicators     | Container        | role="group", aria-label="Step progress"  |
| Step Indicator      | Span             | aria-label with step number and status    |

### Screen Reader Announcements

```typescript
// Progress updates
"Tour progress: 3 of 8 steps completed"
"Step 3 marked as complete. Progress: 4 of 8 steps."

// Step navigation
"Step 3: Install Dependencies, current step"
"Navigated to Step 4: Run your first app"

// Learn more toggle
"Learn More, collapsed. Press Enter to expand."
"Learn More, expanded"

// Completion
"Congratulations! Tour completed. All 8 steps finished."

// Reset confirmation
"Progress reset. Starting from Step 1."
```

### Keyboard Navigation

| Key             | Context           | Action                                    |
|-----------------|-------------------|-------------------------------------------|
| Tab             | Sidebar           | Move through step links                   |
| Enter           | Step link         | Navigate to that step                     |
| Tab             | Step page         | Move through interactive elements         |
| Space/Enter     | Checkbox          | Toggle completion                         |
| Enter           | Learn More button | Toggle expanded state                     |
| Arrow Down      | Learn More        | (When expanded) scroll content            |
| Tab             | Navigation        | Move between Previous/Next buttons        |
| Enter           | Nav button        | Navigate to step                          |
| Escape          | Mobile sidebar    | Close sidebar overlay                     |

### Focus Management

- On page load: Focus on main content heading
- On step change: Focus moves to step title
- On completion toggle: Focus stays on checkbox
- On Learn More toggle: Focus stays on button
- Mobile sidebar: Focus trapped when open
- On tour completion: Focus on completion message

---

## 7. Responsive Design

### Breakpoint Behavior

| Component           | Desktop (≥1024px) | Tablet (768-1023px) | Mobile (<768px)   |
|---------------------|-------------------|---------------------|-------------------|
| Sidebar             | Fixed left, 280px | Collapsible, 240px  | Overlay, 300px    |
| Step content        | Max 800px, padded | Full width          | Full width        |
| Navigation          | Horizontal        | Horizontal          | Stacked           |
| Screenshots         | Inline            | Full width          | Full width, bleed |
| Learn More          | Inline            | Inline              | Full width        |
| Progress bar        | Visible           | Visible             | Compact           |

### Mobile Sidebar

```css
/* Mobile sidebar trigger */
.tour-sidebar-trigger {
  display: none;
  position: fixed;
  bottom: 20px;
  right: 20px;
  z-index: var(--z-fixed);
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background: var(--color-azure-500);
  color: white;
  box-shadow: var(--shadow-lg);
}

@media (max-width: 767px) {
  .tour-sidebar {
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    width: 300px;
    z-index: var(--z-modal);
    transform: translateX(-100%);
    transition: transform 0.3s var(--ease-out);
  }
  
  .tour-sidebar--open {
    transform: translateX(0);
  }
  
  .tour-sidebar-trigger {
    display: flex;
    align-items: center;
    justify-content: center;
  }
  
  .tour-navigation {
    flex-direction: column;
    gap: 12px;
  }
  
  .nav-previous,
  .nav-next {
    width: 100%;
    justify-content: center;
  }
}
```

---

## 8. Animation Specifications

### Progress Bar Fill

```css
.tour-progress-bar {
  width: 100%;
  height: 8px;
  background: var(--color-bg-tertiary);
  border-radius: 4px;
  overflow: hidden;
}

.tour-progress-fill {
  height: 100%;
  background: var(--color-azure-500);
  border-radius: 4px;
  transition: width 0.5s var(--ease-out);
}
```

### Step Completion Animation

```css
/* Checkmark appearance */
.completion-checkbox input:checked + .checkbox-visual {
  animation: check-fill 0.3s var(--ease-bounce) forwards;
}

@keyframes check-fill {
  0% {
    background: transparent;
    transform: scale(0.9);
  }
  50% {
    transform: scale(1.1);
  }
  100% {
    background: var(--color-success);
    transform: scale(1);
  }
}

/* Checkmark icon */
.checkbox-checkmark {
  animation: checkmark-draw 0.3s ease-out 0.15s forwards;
  stroke-dasharray: 20;
  stroke-dashoffset: 20;
}

@keyframes checkmark-draw {
  to {
    stroke-dashoffset: 0;
  }
}
```

### Sidebar Step Update

```css
.tour-step-item {
  transition: background-color 0.2s ease-out,
              color 0.2s ease-out,
              border-color 0.2s ease-out;
}

.tour-step-item--completing {
  animation: step-complete 0.5s ease-out;
}

@keyframes step-complete {
  0% { background: transparent; }
  50% { background: rgba(16, 185, 129, 0.2); }
  100% { background: transparent; }
}
```

### Tour Completion Celebration

```css
.tour-completion-hero {
  animation: completion-entrance 0.6s var(--ease-out);
}

@keyframes completion-entrance {
  0% {
    opacity: 0;
    transform: translateY(20px) scale(0.95);
  }
  100% {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.completion-emoji {
  animation: emoji-bounce 0.6s var(--ease-bounce) 0.3s;
}

@keyframes emoji-bounce {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-20px); }
}

.achievement-badge {
  opacity: 0;
  animation: badge-reveal 0.4s ease-out forwards;
}

.achievement-badge:nth-child(1) { animation-delay: 0.5s; }
.achievement-badge:nth-child(2) { animation-delay: 0.6s; }
.achievement-badge:nth-child(3) { animation-delay: 0.7s; }
/* ... */

@keyframes badge-reveal {
  0% {
    opacity: 0;
    transform: scale(0.8);
  }
  100% {
    opacity: 1;
    transform: scale(1);
  }
}
```

### Reduced Motion

```css
@media (prefers-reduced-motion: reduce) {
  .tour-progress-fill,
  .tour-step-item,
  .completion-checkbox input,
  .learn-more-content,
  .tour-completion-hero,
  .achievement-badge {
    transition-duration: 0.01ms !important;
    animation-duration: 0.01ms !important;
  }
}
```

---

## 9. CSS Custom Properties

```css
/* Tour Layout Tokens */
--tour-sidebar-width: 280px;
--tour-sidebar-width-tablet: 240px;
--tour-sidebar-width-mobile: 300px;
--tour-content-max-width: 800px;
--tour-content-padding: var(--spacing-12);

/* Tour Progress Tokens */
--tour-progress-bar-height: 8px;
--tour-progress-bar-radius: 4px;
--tour-progress-fill-color: var(--color-azure-500);

/* Tour Step Item Tokens */
--tour-step-item-height: 44px;
--tour-step-item-padding: var(--spacing-3) var(--spacing-4);
--tour-step-item-gap: var(--spacing-3);
--tour-step-icon-size: 20px;

/* Tour Step Page Tokens */
--tour-step-header-padding: var(--spacing-6);
--tour-step-section-gap: var(--spacing-8);
--tour-step-border-radius: var(--radius-lg);

/* Completion Checkbox Tokens */
--checkbox-size: 24px;
--checkbox-border-radius: 6px;
--checkbox-border-width: 2px;
--checkbox-gap: var(--spacing-3);

/* Try It Yourself Tokens */
--try-it-padding: var(--spacing-6);
--try-it-border-width: 2px;
--try-it-border-radius: var(--radius-xl);

/* Learn More Tokens */
--learn-more-header-padding: var(--spacing-4) var(--spacing-5);
--learn-more-content-padding: 0 var(--spacing-5) var(--spacing-5);
--learn-more-border-radius: var(--radius-lg);

/* Navigation Tokens */
--nav-button-padding: var(--spacing-3) var(--spacing-6);
--nav-button-font-size: var(--font-size-base);
--nav-dot-size: 10px;
--nav-dot-gap: var(--spacing-2);
```

---

## 10. Testing Checklist

### Functional Tests

- [ ] Progress saves to localStorage on step completion
- [ ] Progress restores on page reload
- [ ] Progress restores in new browser tab
- [ ] Reset progress clears all data
- [ ] Previous/Next navigation works correctly
- [ ] Step indicator dots update correctly
- [ ] Learn More sections expand/collapse
- [ ] Completion checkbox toggles state
- [ ] Screenshots display correct theme variant
- [ ] Code blocks have working copy buttons
- [ ] Mobile sidebar opens/closes correctly
- [ ] Resume prompt appears for returning users

### Accessibility Tests

- [ ] All headings in correct hierarchy
- [ ] Skip link works
- [ ] Sidebar is properly labeled landmark
- [ ] Progress announces via live region
- [ ] Step completion announced to screen readers
- [ ] Keyboard navigation through all interactive elements
- [ ] Learn More toggle announces expanded state
- [ ] Focus visible on all interactive elements
- [ ] Mobile sidebar traps focus when open
- [ ] Reduced motion preference respected
- [ ] Color contrast ≥ 4.5:1 for all text
- [ ] Touch targets ≥ 44x44px on mobile

### Responsive Tests

- [ ] Sidebar visible on desktop
- [ ] Sidebar collapsible on tablet
- [ ] Sidebar overlay on mobile
- [ ] Navigation stacks on mobile
- [ ] Content readable at all sizes
- [ ] Screenshots scale properly
- [ ] No horizontal scroll
- [ ] Mobile trigger button visible

### Cross-Browser Tests

- [ ] Chrome (latest)
- [ ] Firefox (latest)
- [ ] Safari (latest)
- [ ] Edge (latest)
- [ ] iOS Safari
- [ ] Chrome for Android
- [ ] localStorage works in all browsers

### Performance Tests

- [ ] localStorage operations don't block UI
- [ ] Animations run at 60fps
- [ ] Images lazy load correctly
- [ ] Page transitions are smooth

---

## 11. Implementation Files

### Component Structure

```
web/src/
├── components/
│   └── tour/
│       ├── TourLayout.astro
│       ├── TourProgressSidebar.astro
│       ├── TourStepItem.astro
│       ├── TourStepPage.astro
│       ├── TourStepHeader.astro
│       ├── TourNavigation.astro
│       ├── CompletionCheckbox.astro
│       ├── TryItYourself.astro
│       ├── LearnMoreSection.astro
│       ├── TourCompletionPage.astro
│       └── TourProgress.ts (client script)
├── pages/
│   └── tour/
│       ├── index.astro (tour overview)
│       ├── install.astro (step 1)
│       ├── requirements.astro (step 2)
│       ├── dependencies.astro (step 3)
│       ├── first-app.astro (step 4)
│       ├── dashboard.astro (step 5)
│       ├── logs.astro (step 6)
│       ├── health.astro (step 7)
│       ├── mcp.astro (step 8)
│       └── complete.astro (completion page)
├── data/
│   └── tour-steps.ts
└── styles/
    └── tour.css
```

### Data File

```typescript
// web/src/data/tour-steps.ts
export const tourSteps = [...];
export const getTotalEstimatedTime = () => tourSteps.reduce((sum, step) => sum + step.estimatedTime, 0);
export const getStepBySlug = (slug: string) => tourSteps.find(s => s.slug === slug);
export const getStepByNumber = (num: number) => tourSteps.find(s => s.number === num);
```

---

## 12. Related Components

- [Header](./header.md) - Top navigation with tour link
- [Footer](./footer.md) - Bottom navigation and links
- [Sidebar](./sidebar.md) - Similar navigation pattern
- [Code Block](./code-block.md) - Code display with copy
- [Screenshot](./screenshot.md) - Themed screenshot display
- [Quick Start Page](./quick-start-page.md) - Simpler 4-step version
- [Lightbox](./lightbox.md) - Full-size image viewing

````
