````markdown
# Terminal Component Specification

## Overview

The Terminal component displays command-line output with a realistic terminal appearance. It supports showing commands with their output, optional typing animations for demos, and adapts to light/dark themes.

---

## 1. Component Hierarchy

```
Terminal (organism)
├── TerminalHeader (molecule)
│   ├── TerminalDots (atom) [macOS style]
│   ├── TerminalTitle (atom)
│   └── CopyButton (atom)
├── TerminalContent (molecule)
│   ├── TerminalPrompt (atom)
│   ├── TerminalCommand (atom)
│   │   └── TypedText (atom) [for animation]
│   ├── TerminalOutput (atom)
│   │   └── OutputLine (atom) × n
│   └── TerminalCursor (atom) [for animation]
└── TerminalFooter (molecule) [optional, for controls]
    └── ReplayButton (atom)
```

---

## 2. Props Interface

### Terminal

```typescript
interface TerminalProps {
  /** Lines of terminal content */
  lines: TerminalLine[];
  /** Terminal title (shown in header) */
  title?: string;
  /** Show macOS-style window dots */
  showDots?: boolean;
  /** Show copy button */
  showCopy?: boolean;
  /** Enable typing animation */
  animate?: boolean;
  /** Animation typing speed (ms per character) */
  typingSpeed?: number;
  /** Delay before starting animation (ms) */
  startDelay?: number;
  /** Show replay button after animation */
  showReplay?: boolean;
  /** Terminal prompt string */
  prompt?: string;
  /** Custom class name */
  className?: string;
  /** Maximum height before scrolling */
  maxHeight?: string;
  /** Auto-start animation when in viewport */
  autoPlay?: boolean;
}

interface TerminalLine {
  /** Type of line */
  type: 'command' | 'output' | 'error' | 'success' | 'info';
  /** Content of the line */
  content: string;
  /** Delay before this line appears (ms) */
  delay?: number;
  /** For commands: whether to animate typing */
  animated?: boolean;
}
```

### TerminalPrompt

```typescript
interface TerminalPromptProps {
  /** Prompt string (default: "$") */
  prompt?: string;
  /** Current working directory */
  cwd?: string;
  /** Username (optional) */
  user?: string;
  /** Show timestamp */
  showTime?: boolean;
  /** Custom class name */
  className?: string;
}
```

### TypedText

```typescript
interface TypedTextProps {
  /** Text to type */
  text: string;
  /** Characters per second */
  speed?: number;
  /** Delay before starting (ms) */
  delay?: number;
  /** Show cursor while typing */
  showCursor?: boolean;
  /** Callback when typing completes */
  onComplete?: () => void;
  /** Whether animation is active */
  isActive?: boolean;
}
```

---

## 3. States

### Terminal States

| State     | Trigger                | Visual Changes                        |
|-----------|------------------------|---------------------------------------|
| Default   | Initial (no animation) | All content visible                   |
| Loading   | Animation starting     | Only prompt visible                   |
| Typing    | Animation running      | Command typing character by character |
| Output    | Command "executed"     | Output lines appearing                |
| Complete  | Animation finished     | All content visible, replay available |
| Hover     | Mouse enter            | Copy button visible                   |

### Cursor States

| State    | Trigger              | Visual Changes                        |
|----------|---------------------|---------------------------------------|
| Hidden   | No animation         | Not rendered                          |
| Blinking | Typing paused        | Block cursor blinking                 |
| Solid    | Typing active        | Solid block cursor                    |

### Output Line States

| Type    | Color Token              | Example                             |
|---------|-------------------------|-------------------------------------|
| command | --terminal-command      | `azd app run`                       |
| output  | --terminal-output       | `Starting services...`              |
| success | --terminal-success      | `✓ api started on port 5000`        |
| error   | --terminal-error        | `✗ Error: port 3000 in use`         |
| info    | --terminal-info         | `ℹ Checking requirements...`        |

---

## 4. Visual Specifications

### Layout

```
┌──────────────────────────────────────────────────────────────┐
│ ● ● ●                  Terminal                          📋  │  ← Header
├──────────────────────────────────────────────────────────────┤
│ $ azd app run█                                               │  ← Command with cursor
│ Starting services...                                         │
│ ✓ api started on http://localhost:5000                       │  ← Success output
│ ✓ web started on http://localhost:3000                       │
│ ✗ Error: database connection failed                          │  ← Error output
│                                                              │
│ $ █                                                          │  ← Blinking cursor
└──────────────────────────────────────────────────────────────┘
```

### Dimensions

| Property               | Value                    |
|------------------------|--------------------------|
| Container border-radius| 8px (--radius-lg)        |
| Header height          | 36px                     |
| Header padding         | 8px 16px                 |
| Window dot size        | 12px                     |
| Window dot gap         | 8px                      |
| Content padding        | 16px                     |
| Font size              | 14px (--font-size-sm)    |
| Line height            | 1.6                      |
| Cursor width           | 0.5ch (block cursor)     |
| Max height (default)   | 300px                    |

### Window Dots

```
┌─────────┐
│ ● ● ●   │
└─────────┘
  ↑ ↑ ↑
  │ │ └─ Green  (#4ade80)
  │ └─── Yellow (#fbbf24)
  └───── Red    (#f87171)
```

### Colors

#### Light Theme

| Element                    | Property    | Value               |
|----------------------------|-------------|---------------------|
| Container background       | background  | #1e293b             |
| Header background          | background  | #334155             |
| Header border              | border      | #475569             |
| Title color                | color       | #94a3b8             |
| Prompt color               | color       | #10b981             |
| Command color              | color       | #f8fafc             |
| Output color               | color       | #cbd5e1             |
| Success color              | color       | #4ade80             |
| Error color                | color       | #f87171             |
| Info color                 | color       | #60a5fa             |
| Cursor color               | background  | #f8fafc             |
| Selection background       | background  | rgba(59, 130, 246, 0.3) |

#### Dark Theme

| Element                    | Property    | Value               |
|----------------------------|-------------|---------------------|
| Container background       | background  | #0c0a09             |
| Header background          | background  | #1c1917             |
| Header border              | border      | #292524             |
| Title color                | color       | #78716c             |
| Prompt color               | color       | #22c55e             |
| Command color              | color       | #fafaf9             |
| Output color               | color       | #a8a29e             |
| Success color              | color       | #22c55e             |
| Error color                | color       | #ef4444             |
| Info color                 | color       | #3b82f6             |
| Cursor color               | background  | #fafaf9             |

---

## 5. Prompt Styles

### Basic Prompt

```
$ 
```

### With Directory

```
~/projects/my-app $ 
```

### With User (optional)

```
user@host:~/projects $ 
```

### PowerShell Style

```
PS C:\code\my-app> 
```

### Windows CMD Style

```
C:\code\my-app>
```

### Custom Prompt Configuration

```typescript
const promptConfigs = {
  default: { prompt: '$', color: '--terminal-prompt' },
  powershell: { prompt: 'PS>', color: '--terminal-ps-prompt' },
  cmd: { prompt: '>', color: '--terminal-cmd-prompt' },
  zsh: { prompt: '❯', color: '--terminal-prompt' },
};
```

---

## 6. Typing Animation

### Animation Behavior

```typescript
interface TypingAnimationConfig {
  /** Base typing speed (ms per character) */
  speed: number;
  /** Random variation (+/- ms) for natural feel */
  jitter: number;
  /** Delay after completing command before output */
  commandDelay: number;
  /** Delay between output lines */
  outputDelay: number;
  /** Cursor blink interval (ms) */
  cursorBlink: number;
}

const defaultConfig: TypingAnimationConfig = {
  speed: 50,
  jitter: 20,
  commandDelay: 300,
  outputDelay: 100,
  cursorBlink: 530,
};
```

### Animation Sequence

1. Terminal appears with prompt and blinking cursor
2. Command types character by character
3. Brief pause after command completes
4. Output lines appear one at a time
5. Final prompt appears with blinking cursor
6. Replay button appears (if enabled)

### Typing Speed Variations

```typescript
// Add natural variation to typing
function getTypingDelay(baseSpeed: number, jitter: number): number {
  return baseSpeed + (Math.random() * 2 - 1) * jitter;
}

// Pause longer for spaces and punctuation
function getCharacterDelay(char: string, baseSpeed: number): number {
  if (char === ' ') return baseSpeed * 1.5;
  if (['.', ',', '!', '?'].includes(char)) return baseSpeed * 2;
  return baseSpeed;
}
```

---

## 7. Interactions

### Copy Behavior

| Target         | What Gets Copied                              |
|----------------|-----------------------------------------------|
| Copy button    | All commands only (no output)                 |
| Text selection | Selected text                                 |
| Keyboard copy  | Selected text                                 |

### Hover Behavior

| Element        | Hover Effect                                  |
|----------------|-----------------------------------------------|
| Container      | Copy button appears                           |
| Copy button    | Background highlight                          |
| Replay button  | Background highlight                          |

### Replay Button

| Trigger        | Action                                        |
|----------------|-----------------------------------------------|
| Click          | Restart typing animation from beginning       |
| Keyboard       | Enter/Space activates                         |

### Pause/Resume (optional)

| Trigger        | Action                                        |
|----------------|-----------------------------------------------|
| Click anywhere | Pause/resume animation                        |
| Keyboard Space | Pause/resume animation                        |

---

## 8. Accessibility

### Semantic HTML Structure

```html
<figure 
  class="terminal" 
  role="figure" 
  aria-label="Terminal showing azd app run command"
>
  <div class="terminal-header" aria-hidden="true">
    <div class="terminal-dots">
      <span class="dot dot-red"></span>
      <span class="dot dot-yellow"></span>
      <span class="dot dot-green"></span>
    </div>
    <span class="terminal-title">Terminal</span>
    <button
      type="button"
      class="copy-button"
      aria-label="Copy commands to clipboard"
    >
      <svg aria-hidden="true"><!-- copy icon --></svg>
    </button>
  </div>
  
  <div 
    class="terminal-content" 
    role="log" 
    aria-live="polite" 
    aria-label="Terminal output"
    tabindex="0"
  >
    <div class="terminal-line terminal-line--command">
      <span class="terminal-prompt" aria-hidden="true">$</span>
      <span class="terminal-command">azd app run</span>
    </div>
    <div class="terminal-line terminal-line--output">
      <span class="terminal-output">Starting services...</span>
    </div>
    <div class="terminal-line terminal-line--success">
      <span class="terminal-output">✓ api started on port 5000</span>
    </div>
  </div>
  
  <div class="terminal-footer" aria-hidden="true">
    <button
      type="button"
      class="replay-button"
      aria-label="Replay terminal animation"
    >
      <svg aria-hidden="true"><!-- replay icon --></svg>
      Replay
    </button>
  </div>
</figure>
```

### ARIA Attributes

| Element          | Attribute            | Purpose                          |
|------------------|---------------------|----------------------------------|
| Container        | role="figure"       | Semantic grouping                |
| Container        | aria-label          | Describe terminal purpose        |
| Content          | role="log"          | Announce new content             |
| Content          | aria-live="polite"  | Announce output as it appears    |
| Content          | tabindex="0"        | Make focusable                   |
| Copy button      | aria-label          | Describe action                  |
| Replay button    | aria-label          | Describe action                  |

### Screen Reader Announcements

```typescript
// For animated content
"azd app run command entered"
"Starting services output"
"Success: api started on port 5000"
"Error: database connection failed"

// For copy
"Commands copied to clipboard"

// For replay
"Animation restarted"
```

### Keyboard Navigation

| Key            | Action                                        |
|----------------|-----------------------------------------------|
| Tab            | Focus terminal, then copy button              |
| Enter/Space    | Activate focused button                       |
| Arrow Up/Down  | Scroll terminal content                       |
| Escape         | Pause animation (if running)                  |

### Reduced Motion

```css
@media (prefers-reduced-motion: reduce) {
  .terminal-cursor {
    animation: none;
    opacity: 1;
  }
  
  /* Show all content immediately, no typing */
  .terminal--animated .terminal-line {
    opacity: 1;
    transition: none;
  }
  
  /* Disable replay for reduced motion */
  .replay-button {
    display: none;
  }
}
```

---

## 9. Responsive Design

### Breakpoint Behavior

| Breakpoint      | Changes                                       |
|-----------------|-----------------------------------------------|
| Mobile (<640px) | Full width, smaller font (13px), no dots     |
| Tablet (640-1023px) | Standard layout                          |
| Desktop (≥1024px) | Standard layout                            |

### Mobile Considerations

```css
@media (max-width: 640px) {
  .terminal {
    border-radius: 0;
    margin-left: calc(-1 * var(--spacing-4));
    margin-right: calc(-1 * var(--spacing-4));
    font-size: 13px;
  }
  
  .terminal-header {
    padding: 6px 12px;
  }
  
  .terminal-dots {
    display: none; /* Hide macOS dots on mobile */
  }
  
  .terminal-title {
    text-align: left;
  }
  
  .terminal-content {
    padding: 12px;
    overflow-x: auto;
  }
  
  .copy-button,
  .replay-button {
    min-width: 44px;
    min-height: 44px;
  }
}
```

---

## 10. Animation Specifications

### Cursor Blink

```css
.terminal-cursor {
  display: inline-block;
  width: 0.5ch;
  height: 1.2em;
  background-color: var(--terminal-cursor);
  animation: cursor-blink 1.06s step-end infinite;
}

@keyframes cursor-blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

/* Solid cursor while typing */
.terminal--typing .terminal-cursor {
  animation: none;
  opacity: 1;
}
```

### Line Appearance

```css
.terminal-line {
  opacity: 0;
  transform: translateY(4px);
  transition: opacity 0.2s ease-out, transform 0.2s ease-out;
}

.terminal-line--visible {
  opacity: 1;
  transform: translateY(0);
}
```

### Output Stagger

```typescript
// Stagger output lines
const outputLines = document.querySelectorAll('.terminal-line--output');
outputLines.forEach((line, index) => {
  setTimeout(() => {
    line.classList.add('terminal-line--visible');
  }, index * 100); // 100ms stagger
});
```

---

## 11. Example Content

### Basic Command + Output

```typescript
const basicExample: TerminalLine[] = [
  { type: 'command', content: 'azd app run' },
  { type: 'output', content: 'Starting services...' },
  { type: 'success', content: '✓ api started on http://localhost:5000' },
  { type: 'success', content: '✓ web started on http://localhost:3000' },
];
```

### Mixed Success/Error

```typescript
const mixedExample: TerminalLine[] = [
  { type: 'command', content: 'azd app deps' },
  { type: 'info', content: 'ℹ Checking dependencies...' },
  { type: 'success', content: '✓ Python 3.11 installed' },
  { type: 'success', content: '✓ Node.js 18.17 installed' },
  { type: 'error', content: '✗ Docker not found' },
  { type: 'output', content: '' },
  { type: 'info', content: 'Run: brew install docker' },
];
```

### Multi-Command Demo

```typescript
const demoExample: TerminalLine[] = [
  { type: 'command', content: 'azd init -t azd-app-demo', animated: true },
  { type: 'output', content: 'Initializing project...', delay: 500 },
  { type: 'success', content: '✓ Project initialized', delay: 100 },
  { type: 'output', content: '', delay: 200 },
  { type: 'command', content: 'azd app run', animated: true },
  { type: 'output', content: 'Starting services...', delay: 500 },
  { type: 'success', content: '✓ All services started', delay: 100 },
  { type: 'info', content: 'Dashboard: http://localhost:5050', delay: 100 },
];
```

---

## 12. Implementation Notes

### Copy Commands Only

```typescript
function extractCommands(lines: TerminalLine[]): string {
  return lines
    .filter(line => line.type === 'command')
    .map(line => line.content)
    .join('\n');
}
```

### Intersection Observer for Auto-Play

```typescript
function setupAutoPlay(element: HTMLElement, callback: () => void): void {
  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          callback();
          observer.unobserve(entry.target);
        }
      });
    },
    { threshold: 0.5 }
  );
  
  observer.observe(element);
}
```

### Performance Considerations

- Use `will-change: opacity, transform` sparingly
- Batch DOM updates for line appearances
- Cancel animations when element leaves viewport
- Clean up timers on unmount

### Testing Checklist

- [ ] Commands copy correctly
- [ ] Output lines appear in sequence
- [ ] Typing animation smooth
- [ ] Cursor blinks correctly
- [ ] Replay restarts animation
- [ ] Screen reader announces content
- [ ] Keyboard navigation works
- [ ] Focus states visible
- [ ] Works in both themes
- [ ] Reduced motion respected
- [ ] Touch targets ≥ 44x44px
- [ ] Horizontal scroll on mobile

---

## 13. CSS Custom Properties

```css
/* Terminal Tokens */
--terminal-bg: #1e293b;
--terminal-bg-dark: #0c0a09;
--terminal-header-bg: #334155;
--terminal-header-bg-dark: #1c1917;
--terminal-header-border: #475569;
--terminal-title-color: #94a3b8;
--terminal-prompt: #10b981;
--terminal-command: #f8fafc;
--terminal-output: #cbd5e1;
--terminal-success: #4ade80;
--terminal-error: #f87171;
--terminal-info: #60a5fa;
--terminal-cursor: #f8fafc;
--terminal-dot-red: #f87171;
--terminal-dot-yellow: #fbbf24;
--terminal-dot-green: #4ade80;
--terminal-font-size: 0.875rem;
--terminal-line-height: 1.6;
--terminal-border-radius: 0.5rem;
--terminal-padding: 1rem;
```

````