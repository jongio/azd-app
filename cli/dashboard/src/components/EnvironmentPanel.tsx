import { useState } from 'react'
import { Eye, EyeOff, Copy, Search, Filter } from 'lucide-react'
import type { Service } from '@/types'

interface EnvironmentPanelProps {
  services: Service[]
}

export function EnvironmentPanel({ services }: EnvironmentPanelProps) {
  const [showValues, setShowValues] = useState(false)
  const [searchTerm, setSearchTerm] = useState('')
  const [selectedService, setSelectedService] = useState<string>('all')
  const [copiedKey, setCopiedKey] = useState<string | null>(null)

  // Collect all environment variables from all services
  const getAllEnvVars = () => {
    const envVarsMap = new Map<string, { value: string; services: string[] }>()
    
    services.forEach(service => {
      const vars = service.environmentVariables || {}
      Object.entries(vars).forEach(([key, value]) => {
        if (envVarsMap.has(key)) {
          const existing = envVarsMap.get(key)!
          if (!existing.services.includes(service.name)) {
            existing.services.push(service.name)
          }
        } else {
          envVarsMap.set(key, { value, services: [service.name] })
        }
      })
    })

    return Array.from(envVarsMap.entries()).map(([key, data]) => ({
      key,
      value: data.value,
      services: data.services
    }))
  }

  const envVars = getAllEnvVars()

  // Filter environment variables
  const filteredEnvVars = envVars.filter(env => {
    const matchesSearch = env.key.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         env.value.toLowerCase().includes(searchTerm.toLowerCase())
    const matchesService = selectedService === 'all' || env.services.includes(selectedService)
    return matchesSearch && matchesService
  })

  const copyToClipboard = (key: string, value: string) => {
    navigator.clipboard.writeText(value)
    setCopiedKey(key)
    setTimeout(() => setCopiedKey(null), 2000)
  }

  const maskValue = (value: string) => {
    if (showValues) return value
    // Mask sensitive-looking values (containing "key", "secret", "password", "token")
    const key = value.toLowerCase()
    if (key.includes('key') || key.includes('secret') || key.includes('password') || key.includes('token')) {
      return '•'.repeat(Math.min(value.length, 20))
    }
    return value.length > 40 ? value.substring(0, 40) + '...' : value
  }

  return (
    <div className="space-y-4">
      {/* Header Controls */}
      <div className="flex items-center gap-3">
        <div className="relative flex-1">
          <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-500" />
          <input
            type="text"
            placeholder="Search environment variables..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="pl-9 pr-4 py-2 bg-[#0d0d0d] border border-[#2a2a2a] rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 w-full text-foreground"
          />
        </div>
        
        <div className="relative">
          <Filter className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-500" />
          <select
            value={selectedService}
            onChange={(e) => setSelectedService(e.target.value)}
            className="pl-9 pr-10 py-2 bg-[#0d0d0d] border border-[#2a2a2a] rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 text-foreground appearance-none cursor-pointer"
          >
            <option value="all">All Services</option>
            {services.map(service => (
              <option key={service.name} value={service.name}>{service.name}</option>
            ))}
          </select>
        </div>

        <button
          onClick={() => setShowValues(!showValues)}
          className="px-4 py-2 bg-[#0d0d0d] border border-[#2a2a2a] rounded-md text-sm hover:bg-white/5 transition-colors flex items-center gap-2"
        >
          {showValues ? (
            <>
              <EyeOff className="w-4 h-4" />
              Hide Values
            </>
          ) : (
            <>
              <Eye className="w-4 h-4" />
              Show Values
            </>
          )}
        </button>
      </div>

      {/* Environment Variables List */}
      <div className="glass rounded-xl border border-white/10 overflow-hidden">
        {filteredEnvVars.length === 0 ? (
          <div className="p-8 text-center text-muted-foreground">
            {searchTerm || selectedService !== 'all' 
              ? 'No environment variables match your filters'
              : 'No environment variables available'}
          </div>
        ) : (
          <div className="max-h-[600px] overflow-y-auto">
            <table className="w-full">
              <thead className="sticky top-0 bg-[#0d0d0d] border-b border-white/10 z-10">
                <tr>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                    Variable
                  </th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                    Value
                  </th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                    Services
                  </th>
                  <th className="text-right px-4 py-3 text-xs font-semibold text-muted-foreground uppercase tracking-wider w-20">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {filteredEnvVars.map((env) => (
                  <tr 
                    key={env.key}
                    className="hover:bg-white/5 transition-colors group"
                  >
                    <td className="px-4 py-3">
                      <code className="text-sm font-mono text-foreground">{env.key}</code>
                    </td>
                    <td className="px-4 py-3">
                      <code className="text-sm font-mono text-muted-foreground">
                        {maskValue(env.value)}
                      </code>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1">
                        {env.services.map(service => (
                          <span
                            key={service}
                            className="px-2 py-0.5 bg-primary/10 text-primary text-xs rounded-md"
                          >
                            {service}
                          </span>
                        ))}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-right">
                      <button
                        onClick={() => copyToClipboard(env.key, env.value)}
                        className="opacity-0 group-hover:opacity-100 p-1.5 hover:bg-white/10 rounded transition-all"
                        title="Copy value"
                      >
                        {copiedKey === env.key ? (
                          <span className="text-xs text-success">✓</span>
                        ) : (
                          <Copy className="w-4 h-4 text-muted-foreground" />
                        )}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Summary */}
      <div className="flex items-center justify-between text-sm text-muted-foreground">
        <span>
          Showing {filteredEnvVars.length} of {envVars.length} variables
        </span>
        <span>
          {services.length} service{services.length !== 1 ? 's' : ''}
        </span>
      </div>
    </div>
  )
}
