import { Activity, Terminal, FileText, GitBranch, BarChart3, Settings2, Zap, Network } from 'lucide-react'

interface SidebarProps {
  activeView: string
  onViewChange: (view: string) => void
}

export function Sidebar({ activeView, onViewChange }: SidebarProps) {
  const navItems = [
    { id: 'resources', label: 'Resources', icon: Activity },
    { id: 'console', label: 'Console', icon: Terminal },
    { id: 'metrics', label: 'Metrics', icon: BarChart3 },
    { id: 'environment', label: 'Environment', icon: Settings2 },
    { id: 'actions', label: 'Actions', icon: Zap },
    { id: 'dependencies', label: 'Dependencies', icon: Network },
    { id: 'structured', label: 'Structured', icon: FileText },
    { id: 'traces', label: 'Traces', icon: GitBranch },
  ]

  return (
    <aside className="w-20 bg-[#0d0d0d] border-r border-[#2a2a2a] flex flex-col items-center py-4">
      {navItems.map((item) => {
        const Icon = item.icon
        const isActive = activeView === item.id
        
        return (
          <button
            key={item.id}
            onClick={() => onViewChange(item.id)}
            className={`
              w-16 py-3 mb-1 rounded-md flex flex-col items-center gap-1.5
              transition-all duration-200
              ${isActive 
                ? 'bg-purple-500/15 text-purple-400' 
                : 'text-gray-500 hover:text-gray-300 hover:bg-white/5'
              }
            `}
          >
            <Icon className="w-5 h-5" />
            <span className="text-[10px] font-medium leading-tight text-center">{item.label}</span>
          </button>
        )
      })}
    </aside>
  )
}
