````markdown
# Code Block Component Specification

## Overview

The Code Block component displays syntax-highlighted code with optional features like copy functionality, language indicators, filename headers, and line highlighting. It supports multiple programming languages and adapts to light/dark themes.

---

## 1. Component Hierarchy

```
CodeBlock (organism)
├── CodeBlockHeader (molecule) [optional]
│   ├── LanguageIndicator (atom)
│   ├── FilenameLabel (atom) [optional]
│   └── CopyButton (atom)
├── CodeBlockContent (molecule)
│   ├── LineNumbers (atom) [optional]
│   └── CodeLines (atom)
│       └── SyntaxToken (atom) × n
└── CodeBlockFooter (molecule) [optional, for expanded view]
```

---

## 2. Props Interface

### CodeBlock

```typescript
interface CodeBlockProps {
  /** The code content to display */
  code: string;
  /** Programming language for syntax highlighting */
  language: SupportedLanguage;
  /** Optional filename to display in header */
  filename?: string;
  /** Show line numbers */
  showLineNumbers?: boolean;
  /** Lines to highlight (1-indexed) */
  highlightLines?: number[];
  /** Starting line number (for code snippets) */
  startLineNumber?: number;
  /** Show copy button */
  showCopy?: boolean;
  /** Show language indicator */
  showLanguage?: boolean;
  /** Maximum height before scrolling (CSS value) */
  maxHeight?: string;
  /** Custom class name */
  className?: string;
  /** Caption text below code block */
  caption?: string;
  /** Enable word wrap */
  wordWrap?: boolean;
}

type SupportedLanguage = 
  | 'bash' 
  | 'shell' 
  | 'yaml' 
  | 'json' 
  | 'typescript' 
  | 'javascript' 
  | 'python' 
  | 'go' 
  | 'csharp' 
  | 'dockerfile'
  | 'plaintext';
```

### LanguageIndicator

```typescript
interface LanguageIndicatorProps {
  /** The language to display */
  language: SupportedLanguage;
  /** Size variant */
  size?: 'sm' | 'md';
  /** Custom class name */
  className?: string;
}
```

### CopyButton

```typescript
interface CopyButtonProps {
  /** Text to copy to clipboard */
  text: string;
  /** Size variant */
  size?: 'sm' | 'md';
  /** Position variant */
  position?: 'header' | 'floating';
  /** Custom class name */
  className?: string;
  /** Callback on successful copy */
  onCopy?: () => void;
}
```

---

## 3. States

### CodeBlock States

| State     | Trigger            | Visual Changes                        |
|-----------|-------------------|---------------------------------------|
| Default   | Initial render     | Code displayed with syntax highlighting |
| Hover     | Mouse enter        | Copy button becomes more visible      |
| Focused   | Keyboard focus     | Focus ring around container           |
| Scrolling | Content overflow   | Scroll indicators visible             |
| Collapsed | maxHeight exceeded | "Show more" button visible            |

### CopyButton States

| State   | Trigger               | Visual Changes                        |
|---------|----------------------|---------------------------------------|
| Default | Initial              | Copy icon visible                     |
| Hover   | Mouse enter          | Background highlight                  |
| Focus   | Keyboard focus       | Focus ring visible                    |
| Active  | Mouse down           | Scale down slightly                   |
| Copied  | After click (2s)     | Check icon, "Copied!" text            |
| Error   | Copy failed          | Error icon, "Failed" text             |

### LineNumber States

| State       | Trigger            | Visual Changes                      |
|-------------|-------------------|-------------------------------------|
| Default     | Normal line        | Muted color                         |
| Highlighted | In highlightLines  | Brighter, background highlight      |

---

## 4. Visual Specifications

### Layout

```
With Header:
┌─────────────────────────────────────────────────────┐
│  yaml                              azure.yaml  📋  │  ← Header
├─────────────────────────────────────────────────────┤
│  1 │ name: my-app                                   │
│  2 │ services:                                      │
│  3 │   api:                                         │  ← Code Content
│  4 │     project: ./src/api                         │
│  5 │     language: python                           │
└─────────────────────────────────────────────────────┘

Without Header:
┌─────────────────────────────────────────────────────┐
│  $ azd app run                                   📋 │
│  Starting services...                               │
│  ✓ api started on port 5000                         │
└─────────────────────────────────────────────────────┘
```

### Dimensions

| Property               | Value                    |
|------------------------|--------------------------|
| Container border-radius| 8px (--radius-lg)        |
| Header height          | 40px                     |
| Header padding         | 8px 16px                 |
| Code padding           | 16px                     |
| Line number width      | 48px                     |
| Line number padding    | 0 16px                   |
| Font size              | 14px (--font-size-sm)    |
| Line height            | 1.5                      |
| Tab size               | 2 spaces                 |
| Max height (default)   | 400px (scrollable)       |

### Colors

#### Light Theme

| Element                    | Property    | Value               |
|----------------------------|-------------|---------------------|
| Container background       | background  | #1e293b             |
| Header background          | background  | #0f172a             |
| Header border              | border      | #334155             |
| Line number color          | color       | #64748b             |
| Line number bg             | background  | #0f172a             |
| Highlighted line bg        | background  | rgba(59, 130, 246, 0.15) |
| Highlighted line border    | border-left | 3px solid #3b82f6   |
| Selection background       | background  | rgba(59, 130, 246, 0.3)  |
| Scrollbar track            | background  | #334155             |
| Scrollbar thumb            | background  | #475569             |

#### Dark Theme

| Element                    | Property    | Value               |
|----------------------------|-------------|---------------------|
| Container background       | background  | #0f172a             |
| Header background          | background  | #020617             |
| Header border              | border      | #1e293b             |
| Line number color          | color       | #475569             |
| Line number bg             | background  | #020617             |
| Highlighted line bg        | background  | rgba(96, 165, 250, 0.1) |
| Highlighted line border    | border-left | 3px solid #60a5fa   |

---

## 5. Syntax Highlighting Color Tokens

### Light Theme (Dark Background)

```css
/* Code block uses dark background even in light theme for contrast */
--syntax-text: #e2e8f0;           /* Default text */
--syntax-comment: #64748b;         /* Comments, dimmed */
--syntax-keyword: #f472b6;         /* Keywords (if, for, const, etc.) */
--syntax-string: #a5f3fc;          /* String literals */
--syntax-number: #fcd34d;          /* Numeric literals */
--syntax-function: #a78bfa;        /* Function names */
--syntax-variable: #93c5fd;        /* Variables */
--syntax-operator: #f1f5f9;        /* Operators (+, -, =, etc.) */
--syntax-punctuation: #94a3b8;     /* Brackets, semicolons */
--syntax-class: #67e8f9;           /* Class names */
--syntax-property: #86efac;        /* Object properties */
--syntax-tag: #f472b6;             /* HTML/XML tags */
--syntax-attribute: #fcd34d;       /* HTML attributes */
--syntax-selector: #c4b5fd;        /* CSS selectors */
--syntax-builtin: #67e8f9;         /* Built-in functions */
--syntax-regex: #fca5a5;           /* Regular expressions */
--syntax-deleted: #fca5a5;         /* Git deleted lines */
--syntax-inserted: #86efac;        /* Git inserted lines */
```

### Dark Theme (Darker Background)

```css
/* Slightly adjusted for darker background */
--syntax-text: #e2e8f0;
--syntax-comment: #475569;
--syntax-keyword: #f472b6;
--syntax-string: #67e8f9;
--syntax-number: #fbbf24;
--syntax-function: #a78bfa;
--syntax-variable: #7dd3fc;
--syntax-operator: #f1f5f9;
--syntax-punctuation: #64748b;
--syntax-class: #5eead4;
--syntax-property: #4ade80;
--syntax-tag: #f472b6;
--syntax-attribute: #fbbf24;
--syntax-selector: #a78bfa;
--syntax-builtin: #5eead4;
--syntax-regex: #f87171;
--syntax-deleted: #f87171;
--syntax-inserted: #4ade80;
```

### Language-Specific Token Mapping

#### Bash/Shell

| Token Type    | CSS Variable        | Example                    |
|---------------|--------------------|-----------------------------|
| Command       | --syntax-function  | `azd`, `npm`, `git`        |
| Flag          | --syntax-keyword   | `--help`, `-v`             |
| String        | --syntax-string    | `"hello"`                  |
| Variable      | --syntax-variable  | `$HOME`, `${PATH}`         |
| Comment       | --syntax-comment   | `# this is a comment`      |
| Operator      | --syntax-operator  | `|`, `>`, `&&`             |
| Path          | --syntax-property  | `./src/api`                |

#### YAML

| Token Type    | CSS Variable        | Example                    |
|---------------|--------------------|-----------------------------|
| Key           | --syntax-property  | `name:`, `services:`       |
| String value  | --syntax-string    | `"my-app"`                 |
| Boolean       | --syntax-keyword   | `true`, `false`            |
| Number        | --syntax-number    | `8080`, `3.14`             |
| Null          | --syntax-keyword   | `null`, `~`                |
| Anchor        | --syntax-selector  | `&anchor`, `*alias`        |
| Comment       | --syntax-comment   | `# comment`                |

#### JSON

| Token Type    | CSS Variable        | Example                    |
|---------------|--------------------|-----------------------------|
| Key           | --syntax-property  | `"name"`                   |
| String value  | --syntax-string    | `"value"`                  |
| Number        | --syntax-number    | `123`, `-5.6`              |
| Boolean       | --syntax-keyword   | `true`, `false`            |
| Null          | --syntax-keyword   | `null`                     |
| Punctuation   | --syntax-punctuation | `{`, `}`, `[`, `]`, `,`  |

#### TypeScript/JavaScript

| Token Type    | CSS Variable        | Example                    |
|---------------|--------------------|-----------------------------|
| Keyword       | --syntax-keyword   | `const`, `async`, `import` |
| Function      | --syntax-function  | `fetchData`, `console.log` |
| String        | --syntax-string    | `'hello'`, `` `template` ``|
| Number        | --syntax-number    | `42`, `0xFF`               |
| Variable      | --syntax-variable  | `myVar`, `props`           |
| Class         | --syntax-class     | `Promise`, `Array`         |
| Property      | --syntax-property  | `.length`, `.map`          |
| Type          | --syntax-class     | `string`, `number`         |
| Comment       | --syntax-comment   | `// comment`               |
| Decorator     | --syntax-selector  | `@Component`               |

#### Python

| Token Type    | CSS Variable        | Example                    |
|---------------|--------------------|-----------------------------|
| Keyword       | --syntax-keyword   | `def`, `class`, `import`   |
| Function      | --syntax-function  | `print`, `len`, `range`    |
| String        | --syntax-string    | `'hello'`, `"""docstring"""`|
| Number        | --syntax-number    | `42`, `3.14`               |
| Variable      | --syntax-variable  | `my_var`, `self`           |
| Class         | --syntax-class     | `Exception`, `List`        |
| Builtin       | --syntax-builtin   | `True`, `None`, `print`    |
| Decorator     | --syntax-selector  | `@property`, `@staticmethod`|
| Comment       | --syntax-comment   | `# comment`                |

#### Go

| Token Type    | CSS Variable        | Example                    |
|---------------|--------------------|-----------------------------|
| Keyword       | --syntax-keyword   | `func`, `package`, `import`|
| Function      | --syntax-function  | `fmt.Println`, `make`      |
| String        | --syntax-string    | `"hello"`, `` `raw` ``     |
| Number        | --syntax-number    | `42`, `0x1F`               |
| Type          | --syntax-class     | `string`, `int`, `error`   |
| Variable      | --syntax-variable  | `err`, `ctx`               |
| Builtin       | --syntax-builtin   | `nil`, `true`, `false`     |
| Comment       | --syntax-comment   | `// comment`               |

#### C#

| Token Type    | CSS Variable        | Example                    |
|---------------|--------------------|-----------------------------|
| Keyword       | --syntax-keyword   | `class`, `async`, `using`  |
| Function      | --syntax-function  | `Console.WriteLine`        |
| String        | --syntax-string    | `"hello"`, `$"interpolated"`|
| Number        | --syntax-number    | `42`, `3.14m`              |
| Type          | --syntax-class     | `string`, `Task<T>`        |
| Variable      | --syntax-variable  | `myVar`, `args`            |
| Attribute     | --syntax-selector  | `[HttpGet]`, `[Required]`  |
| Comment       | --syntax-comment   | `// comment`               |
| Preprocessor  | --syntax-regex     | `#if`, `#region`           |

---

## 6. Language Display Names

```typescript
const languageDisplayNames: Record<SupportedLanguage, string> = {
  bash: 'Bash',
  shell: 'Shell',
  yaml: 'YAML',
  json: 'JSON',
  typescript: 'TypeScript',
  javascript: 'JavaScript',
  python: 'Python',
  go: 'Go',
  csharp: 'C#',
  dockerfile: 'Dockerfile',
  plaintext: 'Plain Text',
};

const languageIcons: Record<SupportedLanguage, string> = {
  bash: '❯',
  shell: '❯',
  yaml: '📄',
  json: '{ }',
  typescript: 'TS',
  javascript: 'JS',
  python: '🐍',
  go: 'Go',
  csharp: 'C#',
  dockerfile: '🐳',
  plaintext: '📝',
};
```

---

## 7. Interactions

### Copy Button

| Trigger              | Action                                       |
|---------------------|----------------------------------------------|
| Click               | Copy code to clipboard, show "Copied!" for 2s |
| Keyboard Enter/Space| Same as click                                |
| Copy success        | Change icon to checkmark, green color        |
| Copy failure        | Change icon to X, show error briefly         |
| Timer (2s)          | Reset to default copy icon                   |

### Hover Behavior

| Element       | Hover Effect                                  |
|---------------|-----------------------------------------------|
| Container     | Copy button opacity increases                 |
| Copy button   | Background highlight                          |
| Line numbers  | Slightly brighter                             |

### Selection

| Action        | Behavior                                      |
|---------------|-----------------------------------------------|
| Click + drag  | Select code text                              |
| Triple-click  | Select entire line                            |
| Cmd/Ctrl + A  | Select all code (within block)                |

### Scroll

| Trigger              | Behavior                                    |
|---------------------|----------------------------------------------|
| Mouse wheel         | Vertical scroll                              |
| Shift + wheel       | Horizontal scroll                            |
| Overflow            | Scrollbar appears (styled)                   |
| Near edges          | Gradient fade indicator                      |

---

## 8. Accessibility

### Semantic HTML Structure

```html
<figure class="code-block" role="figure" aria-label="Code example in YAML">
  <div class="code-block-header">
    <span class="language-indicator" aria-label="Language: YAML">
      yaml
    </span>
    <span class="filename" aria-label="Filename: azure.yaml">
      azure.yaml
    </span>
    <button
      type="button"
      class="copy-button"
      aria-label="Copy code to clipboard"
      aria-live="polite"
    >
      <svg class="copy-icon" aria-hidden="true"><!-- copy icon --></svg>
      <span class="sr-only copy-status">Copy</span>
    </button>
  </div>
  
  <div class="code-block-content" tabindex="0" role="region" aria-label="Code content">
    <pre><code class="language-yaml">
      <!-- Syntax highlighted code -->
    </code></pre>
  </div>
  
  <figcaption class="code-block-caption">
    Example azure.yaml configuration
  </figcaption>
</figure>
```

### ARIA Attributes

| Element        | Attribute                    | Purpose                         |
|----------------|-----------------------------|---------------------------------|
| Container      | role="figure"               | Semantic grouping               |
| Container      | aria-label                  | Describe code block purpose     |
| Language       | aria-label="Language: X"    | Announce language               |
| Filename       | aria-label="Filename: X"    | Announce filename               |
| Copy button    | aria-label                  | Describe action                 |
| Copy button    | aria-live="polite"          | Announce copy status            |
| Code content   | tabindex="0"                | Make focusable for keyboard     |
| Code content   | role="region"               | Landmark for navigation         |

### Screen Reader Announcements

```typescript
// Copy feedback
"Code copied to clipboard"
"Failed to copy code"

// Code block identification
"Code example in YAML, 15 lines"
"azure.yaml file content"
```

### Keyboard Navigation

| Key            | Action                                        |
|----------------|-----------------------------------------------|
| Tab            | Focus code block, then copy button            |
| Enter/Space    | Activate copy button                          |
| Arrow keys     | Scroll within code block (when focused)       |
| Cmd/Ctrl + C   | Copy selected text                            |
| Cmd/Ctrl + A   | Select all code                               |

---

## 9. Responsive Design

### Breakpoint Behavior

| Breakpoint      | Changes                                       |
|-----------------|-----------------------------------------------|
| Mobile (<640px) | Full width, smaller font (13px)               |
| Tablet (640-1023px) | Slight padding adjustment              |
| Desktop (≥1024px) | Standard layout                            |

### Mobile Considerations

```css
/* Mobile adjustments */
@media (max-width: 640px) {
  .code-block {
    border-radius: 0;
    margin-left: calc(-1 * var(--spacing-4));
    margin-right: calc(-1 * var(--spacing-4));
    font-size: 13px;
  }
  
  .code-block-header {
    padding: 6px 12px;
  }
  
  .code-block-content {
    padding: 12px;
  }
  
  .line-numbers {
    display: none; /* Hide line numbers on mobile */
  }
  
  .copy-button {
    /* Ensure touch target */
    min-width: 44px;
    min-height: 44px;
  }
}
```

### Horizontal Overflow

- On small screens, code blocks scroll horizontally
- Fade gradient indicates scrollable content
- Touch-friendly horizontal scrolling

---

## 10. Animation Specifications

### Copy Button Feedback

```css
.copy-button {
  transition: background-color 0.15s ease-out,
              transform 0.15s ease-out;
}

.copy-button:active {
  transform: scale(0.95);
}

.copy-button--copied {
  animation: copy-success 0.3s ease-out;
}

@keyframes copy-success {
  0% { transform: scale(1); }
  50% { transform: scale(1.1); }
  100% { transform: scale(1); }
}

.copy-icon {
  transition: opacity 0.15s ease-out;
}

.check-icon {
  animation: check-appear 0.2s ease-out;
}

@keyframes check-appear {
  0% {
    opacity: 0;
    transform: scale(0.5);
  }
  100% {
    opacity: 1;
    transform: scale(1);
  }
}
```

### Line Highlight

```css
.code-line--highlighted {
  animation: line-highlight-in 0.3s ease-out;
}

@keyframes line-highlight-in {
  0% {
    background-color: transparent;
  }
  100% {
    background-color: var(--syntax-highlight-bg);
  }
}
```

### Reduced Motion

```css
@media (prefers-reduced-motion: reduce) {
  .copy-button,
  .copy-icon,
  .check-icon,
  .code-line--highlighted {
    animation: none;
    transition: none;
  }
}
```

---

## 11. Copy Button Implementation

```typescript
interface CopyButtonState {
  status: 'idle' | 'copied' | 'error';
}

const COPY_FEEDBACK_DURATION = 2000; // 2 seconds

async function handleCopy(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
    setStatus('copied');
    
    // Announce to screen readers
    announceToScreenReader('Code copied to clipboard');
    
    // Reset after duration
    setTimeout(() => setStatus('idle'), COPY_FEEDBACK_DURATION);
    
  } catch (error) {
    setStatus('error');
    announceToScreenReader('Failed to copy code');
    setTimeout(() => setStatus('idle'), COPY_FEEDBACK_DURATION);
  }
}

function announceToScreenReader(message: string): void {
  const announcer = document.getElementById('copy-announcer');
  if (announcer) {
    announcer.textContent = message;
  }
}
```

### Copy Button Visual States

```
Idle:
┌────────┐
│  📋   │
└────────┘

Copied:
┌────────────────┐
│  ✓  Copied!    │
└────────────────┘

Error:
┌────────────────┐
│  ✗  Failed     │
└────────────────┘
```

---

## 12. Implementation Notes

### Syntax Highlighting Library

Recommend using **Shiki** for syntax highlighting:
- Built-in VS Code themes
- Supports all required languages
- Static HTML output (no runtime JS)
- Works with Astro

### Performance Considerations

- Syntax highlight at build time (SSG)
- Lazy load large code blocks
- Use `content-visibility: auto` for long pages
- Virtualize very long code blocks

### Error Handling

| Scenario               | Behavior                                    |
|------------------------|---------------------------------------------|
| Unknown language       | Fall back to plaintext                      |
| Empty code             | Show placeholder message                    |
| Very long lines        | Horizontal scroll, no wrap (default)        |
| Clipboard API blocked  | Show fallback instructions                  |

### Testing Checklist

- [ ] All languages render correctly
- [ ] Syntax colors have sufficient contrast
- [ ] Copy button works on all browsers
- [ ] Copy feedback is announced to screen readers
- [ ] Line numbers align with content
- [ ] Line highlighting works
- [ ] Horizontal scroll works on mobile
- [ ] Keyboard navigation works
- [ ] Focus states visible
- [ ] Works in light and dark themes
- [ ] Reduced motion preference respected
- [ ] Touch targets ≥ 44x44px

---

## 13. CSS Custom Properties

```css
/* Code Block Tokens */
--code-block-bg: #1e293b;
--code-block-bg-dark: #0f172a;
--code-header-bg: #0f172a;
--code-header-bg-dark: #020617;
--code-header-border: #334155;
--code-line-number-color: #64748b;
--code-line-highlight-bg: rgba(59, 130, 246, 0.15);
--code-line-highlight-border: #3b82f6;
--code-selection-bg: rgba(59, 130, 246, 0.3);
--code-scrollbar-track: #334155;
--code-scrollbar-thumb: #475569;
--code-font-size: 0.875rem;
--code-line-height: 1.5;
--code-border-radius: 0.5rem;
--code-padding: 1rem;
```

````