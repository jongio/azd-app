````markdown
# Language Indicator Component Specification

## Overview

The Language Indicator is an atom component that displays the programming language of a code block. It provides visual identification and accessibility information about the code's language.

---

## 1. Component Hierarchy

```
LanguageIndicator (atom)
├── LanguageIcon (atom) [optional]
└── LanguageLabel (atom)
```

---

## 2. Props Interface

```typescript
interface LanguageIndicatorProps {
  /** The programming language */
  language: SupportedLanguage;
  /** Size variant */
  size?: 'sm' | 'md';
  /** Show language icon */
  showIcon?: boolean;
  /** Visual variant */
  variant?: 'badge' | 'text' | 'pill';
  /** Custom class name */
  className?: string;
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

---

## 3. Language Mappings

### Display Names and Icons

```typescript
const languageConfig: Record<SupportedLanguage, LanguageConfig> = {
  bash: {
    displayName: 'Bash',
    shortName: 'bash',
    icon: '❯',
    color: '#4EAA25',
  },
  shell: {
    displayName: 'Shell',
    shortName: 'sh',
    icon: '❯',
    color: '#89E051',
  },
  yaml: {
    displayName: 'YAML',
    shortName: 'yaml',
    icon: null,
    color: '#CB171E',
  },
  json: {
    displayName: 'JSON',
    shortName: 'json',
    icon: '{ }',
    color: '#292929',
  },
  typescript: {
    displayName: 'TypeScript',
    shortName: 'ts',
    icon: 'TS',
    color: '#3178C6',
  },
  javascript: {
    displayName: 'JavaScript',
    shortName: 'js',
    icon: 'JS',
    color: '#F7DF1E',
  },
  python: {
    displayName: 'Python',
    shortName: 'py',
    icon: '🐍',
    color: '#3776AB',
  },
  go: {
    displayName: 'Go',
    shortName: 'go',
    icon: null,
    color: '#00ADD8',
  },
  csharp: {
    displayName: 'C#',
    shortName: 'c#',
    icon: null,
    color: '#512BD4',
  },
  dockerfile: {
    displayName: 'Dockerfile',
    shortName: 'docker',
    icon: '🐳',
    color: '#2496ED',
  },
  plaintext: {
    displayName: 'Plain Text',
    shortName: 'text',
    icon: null,
    color: '#6B7280',
  },
};

interface LanguageConfig {
  displayName: string;
  shortName: string;
  icon: string | null;
  color: string;
}
```

---

## 4. States

### Visual States

| State    | Trigger          | Visual Changes                        |
|----------|------------------|---------------------------------------|
| Default  | Initial render   | Standard appearance                   |
| Hover    | Mouse enter      | Subtle brightness increase (optional) |

Note: Language Indicator is typically non-interactive, so states are minimal.

---

## 5. Visual Specifications

### Variants

```
Badge (default):
┌────────┐
│  yaml  │
└────────┘

Text:
yaml

Pill:
╭────────╮
│  yaml  │
╰────────╯

With Icon:
┌─────────────┐
│  🐍 Python  │
└─────────────┘
```

### Dimensions

| Property         | sm        | md (default) |
|------------------|-----------|--------------|
| Height           | 20px      | 24px         |
| Padding          | 4px 6px   | 4px 8px      |
| Font size        | 11px      | 12px         |
| Border radius    | 4px       | 4px          |
| Icon size        | 12px      | 14px         |
| Icon gap         | 4px       | 6px          |

### Colors

#### Light Theme (Dark Code Block Background)

| Element        | Property    | Value               |
|----------------|-------------|---------------------|
| Background     | background  | rgba(255, 255, 255, 0.1) |
| Text           | color       | #94a3b8             |
| Icon           | color       | #94a3b8             |

#### Dark Theme (Darker Code Block Background)

| Element        | Property    | Value               |
|----------------|-------------|---------------------|
| Background     | background  | rgba(255, 255, 255, 0.05) |
| Text           | color       | #64748b             |
| Icon           | color       | #64748b             |

### Language-Specific Colors (Optional)

For colored variants, each language can have its own accent color:

```css
.language-indicator--typescript {
  background: rgba(49, 120, 198, 0.2);
  color: #60a5fa;
}

.language-indicator--python {
  background: rgba(55, 118, 171, 0.2);
  color: #60a5fa;
}

.language-indicator--bash {
  background: rgba(78, 170, 37, 0.2);
  color: #86efac;
}
```

---

## 6. Accessibility

### HTML Structure

```html
<span 
  class="language-indicator" 
  aria-label="Language: YAML"
>
  <span class="language-indicator-icon" aria-hidden="true">
    <!-- Icon if applicable -->
  </span>
  <span class="language-indicator-label">
    yaml
  </span>
</span>
```

### ARIA Attributes

| Attribute    | Value              | Purpose                          |
|--------------|-------------------|----------------------------------|
| aria-label   | "Language: X"     | Full accessible name             |
| aria-hidden  | "true" (on icon)  | Hide decorative icon from SR     |

### Screen Reader Announcement

The `aria-label` provides context: "Language: YAML" rather than just "yaml".

---

## 7. Responsive Design

### Breakpoint Behavior

| Breakpoint      | Changes                                       |
|-----------------|-----------------------------------------------|
| Mobile (<640px) | Use `sm` size, short names                    |
| Tablet+         | Standard layout                               |

### Mobile Adjustments

```css
@media (max-width: 640px) {
  .language-indicator {
    /* Use smaller size on mobile */
    font-size: 11px;
    padding: 4px 6px;
  }
}
```

---

## 8. Implementation Notes

### Component Implementation

```typescript
export function LanguageIndicator({
  language,
  size = 'md',
  showIcon = false,
  variant = 'badge',
  className,
}: LanguageIndicatorProps) {
  const config = languageConfig[language] || languageConfig.plaintext;
  
  return (
    <span
      className={cn(
        'language-indicator',
        `language-indicator--${size}`,
        `language-indicator--${variant}`,
        `language-indicator--${language}`,
        className
      )}
      aria-label={`Language: ${config.displayName}`}
    >
      {showIcon && config.icon && (
        <span className="language-indicator-icon" aria-hidden="true">
          {config.icon}
        </span>
      )}
      <span className="language-indicator-label">
        {size === 'sm' ? config.shortName : language}
      </span>
    </span>
  );
}
```

### Usage in CodeBlock

```html
<div class="code-block-header">
  <LanguageIndicator language="typescript" size="sm" />
  <span class="filename">utils.ts</span>
  <CopyButton text={code} />
</div>
```

---

## 9. CSS Custom Properties

```css
/* Language Indicator Tokens */
--language-indicator-bg: rgba(255, 255, 255, 0.1);
--language-indicator-color: #94a3b8;
--language-indicator-font-size-sm: 11px;
--language-indicator-font-size-md: 12px;
--language-indicator-padding-sm: 4px 6px;
--language-indicator-padding-md: 4px 8px;
--language-indicator-border-radius: 4px;
--language-indicator-font-weight: 500;
--language-indicator-font-family: var(--font-family-mono);
```

---

## 10. Testing Checklist

- [ ] All languages display correctly
- [ ] Screen reader announces full language name
- [ ] Icons render correctly when enabled
- [ ] Color contrast meets WCAG AA
- [ ] Sizing works at all breakpoints
- [ ] Works in both themes

````