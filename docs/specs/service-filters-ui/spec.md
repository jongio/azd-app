# Service Filters UI Redesign

## Overview
Redesign the Services filter section in ConsoleView to match the visual style of Log Levels, State, and Health Status filters. Replace checkbox-based selection with modern pill buttons (icon + text) for visual consistency.

## Current State
- Services use traditional checkboxes with text labels
- Log Levels, State, and Health Status use icon buttons
- Visual inconsistency creates design friction

## Goals
1. **Visual Consistency**: Services section uses pill-style buttons matching other filter sections
2. **Usability**: Clear service identification with icon + text (not just icons)
3. **Accessibility**: Maintain WCAG AA compliance
4. **Responsive**: Handle 1-20+ services gracefully with automatic wrapping

## Design Requirements

### Visual Style
- Pill-style buttons with icon + text (px-2.5 py-1.5, rounded-md)
- Icon (w-3.5 h-3.5 shrink-0) + service name text (text-xs font-medium truncate)
- Text handling: single line, truncate with ellipsis if exceeds 150px (max-w-[150px])
- Buttons grow to fit content, flex-wrap handles multi-row layout automatically
- Selected state: 
  - Colored background (e.g., `bg-emerald-100 dark:bg-emerald-500/20`)
  - Colored text (e.g., `text-emerald-700 dark:text-emerald-300`)
  - Colored ring (e.g., `ring-1 ring-emerald-300 dark:ring-emerald-500/50`)
  - Icon and text both use selected color
- Unselected state:
  - Transparent background (`bg-transparent`)
  - Gray text (`text-slate-600 dark:text-slate-400`)
  - Gray icon (same color as text)
  - Hover: subtle background (`hover:bg-slate-200/60 dark:hover:bg-slate-700/60`)
- Each service gets unique color scheme cycling through 8-color palette

### Service Icons
Use contextual icons based on service name/type:
- `web`, `frontend`, `ui`, `app` → Globe
- `api`, `backend`, `server` → Server
- `worker`, `queue`, `background` → Cpu
- `functions`, `function`, `func` → Zap
- `containerapp`, `container` → Box
- `database`, `db`, `postgres`, `redis`, `mongo`, `mysql` → Database
- Default → Package

### Icon Library
Import from lucide-react: `Globe, Server, Database, Box, Cpu, Zap, Package`

### Color Palette
Cycle through these tailwind color schemes (use `index % 8`):
1. Emerald: `bg-emerald-100 dark:bg-emerald-500/20` / `text-emerald-700 dark:text-emerald-300` / `ring-emerald-300 dark:ring-emerald-500/50`
2. Purple: `bg-purple-100 dark:bg-purple-500/20` / `text-purple-700 dark:text-purple-300` / `ring-purple-300 dark:ring-purple-500/50`
3. Blue: `bg-blue-100 dark:bg-blue-500/20` / `text-blue-700 dark:text-blue-300` / `ring-blue-300 dark:ring-blue-500/50`
4. Rose: `bg-rose-100 dark:bg-rose-500/20` / `text-rose-700 dark:text-rose-300` / `ring-rose-300 dark:ring-rose-500/50`
5. Cyan: `bg-cyan-100 dark:bg-cyan-500/20` / `text-cyan-700 dark:text-cyan-300` / `ring-cyan-300 dark:ring-cyan-500/50`
6. Violet: `bg-violet-100 dark:bg-violet-500/20` / `text-violet-700 dark:text-violet-300` / `ring-violet-300 dark:ring-violet-500/50`
7. Amber: `bg-amber-100 dark:bg-amber-500/20` / `text-amber-700 dark:text-amber-300` / `ring-amber-300 dark:ring-amber-500/50`
8. Teal: `bg-teal-100 dark:bg-teal-500/20` / `text-teal-700 dark:text-teal-300` / `ring-teal-300 dark:ring-teal-500/50`

### Layout & Wrapping

**Few services** (single row):
```
Services
[🌐 web] [⚡ api] [⚙️ worker]
```

**Many services** (automatic multi-row wrapping):
```
Services
[🌐 appservice-web] [⚡ containerapp-api] [⚙️ functions-worker] [💾 postgres]
[📦 redis] [☁️ azurite] [🔧 worker] [🔐 auth] [📊 analytics] [🎨 frontend]
[🗄️ database] [⚡ queue-processor] [🌍 global-api] ...
```

- Container uses `flex flex-wrap gap-2`
- Buttons automatically wrap to new rows when horizontal space fills
- Each button: max-width 150px with text truncate + ellipsis
- `title` attribute shows full service name on hover (for truncated text)
- Scales from 1 to 100+ services responsively

### Selection State Examples

**Selected** (emerald):
```
[🌐 web]  ← emerald bg, emerald text/icon, emerald ring - fully colored
```

**Unselected**:
```
[🌐 web]  ← transparent bg, gray text/icon, subtle hover effect
```

**Multiple selections** (different colors):
```
[🌐 web]   [⚡ api]    [⚙️ worker]
↑ emerald  ↑ purple   ↑ blue
```

## Technical Approach

### Component Structure
```tsx
// In FiltersBar component - Replace checkbox section with pill buttons
<div className="flex flex-col gap-2">
  <span className="text-xs font-medium text-slate-500">Services</span>
  <div className="flex flex-wrap gap-2">
    {services.sort((a, b) => a.name.localeCompare(b.name)).map((service, idx) => {
      const { icon: IconComponent, colorScheme } = getServiceIconAndColor(service.name, idx)
      const isSelected = selectedServices.has(service.name)
      
      return (
        <button
          key={service.name}
          type="button"
          onClick={() => onToggleService(service.name)}
          className={cn(
            'flex items-center gap-1.5 px-2.5 py-1.5 rounded-md transition-all max-w-[150px]',
            isSelected
              ? colorScheme.selected
              : 'bg-transparent text-slate-600 dark:text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
          )}
          aria-label={`Toggle ${service.name}`}
          title={service.name} // Full name on hover
        >
          <IconComponent className="w-3.5 h-3.5 shrink-0" />
          <span className="text-xs font-medium truncate">{service.name}</span>
        </button>
      )
    })}
  </div>
</div>
```

### Helper Function
```tsx
import { Globe, Server, Database, Box, Cpu, Zap, Package } from 'lucide-react'

interface ServiceIconColor {
  icon: typeof Globe // LucideIcon type
  colorScheme: {
    selected: string // Combined className for selected state
  }
}

function getServiceIconAndColor(serviceName: string, index: number): ServiceIconColor {
  const lowerName = serviceName.toLowerCase()
  
  // Determine icon based on service name patterns
  let icon = Package // default
  if (lowerName.includes('web') || lowerName.includes('frontend') || lowerName.includes('ui') || lowerName.includes('app')) {
    icon = Globe
  } else if (lowerName.includes('api') || lowerName.includes('backend') || lowerName.includes('server')) {
    icon = Server
  } else if (lowerName.includes('worker') || lowerName.includes('queue') || lowerName.includes('background')) {
    icon = Cpu
  } else if (lowerName.includes('function') || lowerName.includes('func')) {
    icon = Zap
  } else if (lowerName.includes('container')) {
    icon = Box
  } else if (lowerName.includes('db') || lowerName.includes('database') || lowerName.includes('postgres') || lowerName.includes('redis') || lowerName.includes('mongo') || lowerName.includes('mysql')) {
    icon = Database
  }
  
  // Color scheme cycling (8 colors)
  const colorSchemes = [
    { selected: 'bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-300 ring-1 ring-emerald-300 dark:ring-emerald-500/50' },
    { selected: 'bg-purple-100 dark:bg-purple-500/20 text-purple-700 dark:text-purple-300 ring-1 ring-purple-300 dark:ring-purple-500/50' },
    { selected: 'bg-blue-100 dark:bg-blue-500/20 text-blue-700 dark:text-blue-300 ring-1 ring-blue-300 dark:ring-blue-500/50' },
    { selected: 'bg-rose-100 dark:bg-rose-500/20 text-rose-700 dark:text-rose-300 ring-1 ring-rose-300 dark:ring-rose-500/50' },
    { selected: 'bg-cyan-100 dark:bg-cyan-500/20 text-cyan-700 dark:text-cyan-300 ring-1 ring-cyan-300 dark:ring-cyan-500/50' },
    { selected: 'bg-violet-100 dark:bg-violet-500/20 text-violet-700 dark:text-violet-300 ring-1 ring-violet-300 dark:ring-violet-500/50' },
    { selected: 'bg-amber-100 dark:bg-amber-500/20 text-amber-700 dark:text-amber-300 ring-1 ring-amber-300 dark:ring-amber-500/50' },
    { selected: 'bg-teal-100 dark:bg-teal-500/20 text-teal-700 dark:text-teal-300 ring-1 ring-teal-300 dark:ring-teal-500/50' },
  ]
  
  return {
    icon,
    colorScheme: colorSchemes[index % colorSchemes.length]
  }
}
```

## Files to Modify
- `cli/dashboard/src/components/ConsoleView.tsx` (FiltersBar component)

## Acceptance Criteria
✅ Services section uses pill buttons with icon + text instead of checkboxes  
✅ Each service has contextual icon based on name pattern  
✅ Service name text is visible in button  
✅ Services cycle through 8-color palette deterministically (by index)  
✅ Selected state: colored background + text + ring  
✅ Unselected state: transparent with gray text  
✅ Text truncates with ellipsis for long names (>150px)  
✅ `title` attribute shows full name on hover  
✅ Automatic wrapping with flex-wrap for 1-100+ services  
✅ Accessible (aria-label, keyboard navigation)  
✅ Dark mode support  
✅ Maintains all existing filter toggle behavior

## Out of Scope
- Changing Log Levels, State, or Health Status sections
- Custom service icons from azure.yaml configuration
- Service grouping/categories
- Service ordering (keeps existing alphabetical sort)
