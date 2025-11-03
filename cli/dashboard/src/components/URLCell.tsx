import { ExternalLink } from 'lucide-react'
import { Badge } from '@/components/ui/badge'

interface URLCellProps {
  localUrl?: string
  azureUrl?: string
}

export function URLCell({ localUrl, azureUrl }: URLCellProps) {
  // No URLs available
  if (!localUrl && !azureUrl) {
    return <span className="text-muted-foreground">-</span>
  }

  // Truncate URL for display
  const truncateUrl = (url: string, maxLength: number = 40) => {
    if (url.length <= maxLength) return url
    const protocol = url.match(/^https?:\/\//)
    if (protocol) {
      const rest = url.slice(protocol[0].length)
      if (rest.length > maxLength - protocol[0].length) {
        return protocol[0] + rest.slice(0, maxLength - protocol[0].length - 3) + '...'
      }
    }
    return url.slice(0, maxLength - 3) + '...'
  }

  // Primary URL (prefer local)
  const primaryUrl = localUrl || azureUrl
  const hasMultipleUrls = localUrl && azureUrl

  return (
    <div className="flex items-center gap-2">
      <a
        href={primaryUrl}
        target="_blank"
        rel="noopener noreferrer"
        className="text-primary hover:underline flex items-center gap-1 transition-colors"
        title={primaryUrl}
      >
        <span className="truncate max-w-[300px]">{truncateUrl(primaryUrl!)}</span>
        <ExternalLink className="w-3 h-3 shrink-0" />
      </a>
      
      {hasMultipleUrls && (
        <div className="relative group">
          <Badge 
            variant="secondary" 
            className="bg-blue-500/20 text-blue-300 cursor-help hover:bg-blue-500/30 transition-colors"
            title="Azure URL available"
          >
            +1
          </Badge>
          
          {/* Tooltip on hover */}
          <div className="absolute left-0 top-full mt-2 hidden group-hover:block z-50 w-max max-w-md">
            <div className="glass border border-white/10 rounded-lg p-3 shadow-xl">
              <div className="text-xs text-muted-foreground mb-2">Azure URL:</div>
              <a
                href={azureUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="text-blue-300 hover:text-blue-200 text-sm flex items-center gap-2 transition-colors"
              >
                <span className="truncate max-w-[350px]">{azureUrl}</span>
                <ExternalLink className="w-3 h-3 shrink-0" />
              </a>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
