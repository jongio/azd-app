import { Activity, Terminal, FileText, GitBranch, BarChart3 } from 'lucide-react'

interface SidebarProps {
  activeView: string
  onViewChange: (view: string) => void
  hasActiveErrors?: boolean
}

export function Sidebar({ activeView, onViewChange, hasActiveErrors = false }: SidebarProps) {
  const navItems = [
    { id: 'resources', label: 'Resources', icon: Activity },
    { id: 'console', label: 'Console', icon: Terminal },
    { id: 'structured', label: 'Structured', icon: FileText },
    { id: 'traces', label: 'Traces', icon: GitBranch },
    { id: 'metrics', label: 'Metrics', icon: BarChart3 },
  ]

  return (
    <aside className="w-20 bg-background border-r border-border flex flex-col items-center py-4">
      {navItems.map((item) => {
        const Icon = item.icon
        const isActive = activeView === item.id
        const showErrorIndicator = item.id === 'console' && hasActiveErrors
        
        return (
          <button
            key={item.id}
            onClick={() => onViewChange(item.id)}
            className={`
              w-16 py-3 mb-1 rounded-md flex flex-col items-center gap-1.5
              transition-all duration-200 cursor-pointer relative
              ${isActive 
                ? 'bg-accent text-accent-foreground' 
                : 'text-foreground-tertiary hover:text-foreground hover:bg-secondary'
              }
              ${showErrorIndicator && !isActive ? 'ring-2 ring-red-500/50' : ''}
            `}
          >
            <Icon className="w-5 h-5" />
            <span className="text-[10px] font-medium leading-tight text-center">{item.label}</span>
            {showErrorIndicator && (
              <span 
                className="absolute top-1 right-1 w-2 h-2 bg-red-500 rounded-full animate-pulse"
                title="Active errors detected"
              />
            )}
          </button>
        )
      })}
    </aside>
  )
}
