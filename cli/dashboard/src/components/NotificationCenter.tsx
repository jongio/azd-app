import { useState, useMemo } from 'react'
import { X, ChevronDown, Search, Check, Trash2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { NotificationBadge } from './NotificationBadge'

export interface NotificationHistoryItem {
  id: string
  serviceName: string
  message: string
  severity: 'critical' | 'warning' | 'info'
  timestamp: Date
  read: boolean
  acknowledged: boolean
}

export interface NotificationCenterProps {
  notifications: NotificationHistoryItem[]
  onMarkAsRead: (id: string) => void
  onMarkAllAsRead: () => void
  onClearAll: () => void
  onNotificationClick: (id: string) => void
  onClose?: () => void
  isOpen: boolean
  className?: string
}

export function NotificationCenter({
  notifications,
  onMarkAsRead,
  onMarkAllAsRead,
  onClearAll,
  onNotificationClick,
  onClose,
  isOpen,
  className
}: NotificationCenterProps) {
  const [searchQuery, setSearchQuery] = useState('')
  const [severityFilter, setSeverityFilter] = useState<'all' | 'critical' | 'warning' | 'info'>('all')
  const [groupBy, setGroupBy] = useState<'service' | 'severity' | 'time'>('service')
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set())

  const unreadCount = notifications.filter(n => !n.read).length

  // Filter notifications
  const filteredNotifications = useMemo(() => {
    return notifications.filter(n => {
      if (searchQuery && !n.message.toLowerCase().includes(searchQuery.toLowerCase()) &&
          !n.serviceName.toLowerCase().includes(searchQuery.toLowerCase())) {
        return false
      }
      if (severityFilter !== 'all' && n.severity !== severityFilter) {
        return false
      }
      return true
    }).sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())
  }, [notifications, searchQuery, severityFilter])

  // Group notifications
  const groupedNotifications = useMemo(() => {
    const groups: Record<string, NotificationHistoryItem[]> = {}
    
    filteredNotifications.forEach(notification => {
      let key: string
      if (groupBy === 'service') {
        key = notification.serviceName
      } else if (groupBy === 'severity') {
        key = notification.severity
      } else {
        const date = notification.timestamp
        const now = new Date()
        const diff = now.getTime() - date.getTime()
        if (diff < 3600000) key = 'Last hour'
        else if (diff < 86400000) key = 'Today'
        else if (diff < 604800000) key = 'This week'
        else key = 'Older'
      }
      
      if (!groups[key]) groups[key] = []
      groups[key].push(notification)
    })
    
    return groups
  }, [filteredNotifications, groupBy])

  const toggleGroup = (groupName: string) => {
    setExpandedGroups(prev => {
      const next = new Set(prev)
      if (next.has(groupName)) {
        next.delete(groupName)
      } else {
        next.add(groupName)
      }
      return next
    })
  }

  const getRelativeTime = (date: Date): string => {
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    
    if (diff < 60000) return 'Just now'
    if (diff < 3600000) return `${Math.floor(diff / 60000)} minutes ago`
    if (diff < 86400000) return `${Math.floor(diff / 3600000)} hours ago`
    return `${Math.floor(diff / 86400000)} days ago`
  }

  if (!isOpen) return null

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/20 dark:bg-black/40 z-[900] backdrop-blur-sm"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Panel */}
      <aside
        role="complementary"
        aria-label="Notification Center"
        className={cn(
          'fixed top-0 right-0 h-full w-full sm:w-[400px] bg-background border-l shadow-2xl z-[1000]',
          'transform transition-transform duration-300 ease-out',
          isOpen ? 'translate-x-0' : 'translate-x-full',
          className
        )}
      >
        {/* Header */}
        <div className="sticky top-0 bg-background-secondary border-b p-4 flex items-center justify-between z-10">
          <div className="flex items-center gap-2">
            <h2 className="text-lg font-semibold">Notifications</h2>
            {unreadCount > 0 && <NotificationBadge count={unreadCount} size="sm" />}
          </div>
          <button
            onClick={onClose}
            aria-label="Close notification center"
            className="p-1 rounded hover:bg-secondary transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Toolbar */}
        <div className="sticky top-[57px] bg-background border-b p-3 space-y-2 z-10">
          {/* Search */}
          <div className="relative" role="search">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <input
              type="text"
              placeholder="Search notifications..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-3 py-2 text-sm rounded border bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
            />
          </div>

          {/* Filters and Actions */}
          <div className="flex items-center gap-2 flex-wrap">
            <select
              value={severityFilter}
              onChange={(e) => setSeverityFilter(e.target.value as 'all' | 'info' | 'warning' | 'critical')}
              className="text-xs px-2 py-1 rounded border bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
              aria-label="Filter by severity"
            >
              <option value="all">All</option>
              <option value="critical">Critical</option>
              <option value="warning">Warning</option>
              <option value="info">Info</option>
            </select>

            <select
              value={groupBy}
              onChange={(e) => setGroupBy(e.target.value as 'time' | 'service' | 'severity')}
              className="text-xs px-2 py-1 rounded border bg-background focus:outline-none focus:ring-2 focus:ring-primary/50"
              aria-label="Group by"
            >
              <option value="service">By Service</option>
              <option value="severity">By Severity</option>
              <option value="time">By Time</option>
            </select>

            <button
              onClick={onMarkAllAsRead}
              disabled={unreadCount === 0}
              className="text-xs px-2 py-1 rounded border hover:bg-secondary transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1"
              aria-label="Mark all as read"
            >
              <Check className="w-3 h-3" />
              Mark All Read
            </button>

            <button
              onClick={onClearAll}
              className="text-xs px-2 py-1 rounded border hover:bg-destructive/10 hover:text-destructive transition-colors flex items-center gap-1"
              aria-label="Clear all notifications"
            >
              <Trash2 className="w-3 h-3" />
              Clear All
            </button>
          </div>
        </div>

        {/* Notification List */}
        <div className="overflow-y-auto h-[calc(100%-129px)]" role="list">
          {Object.keys(groupedNotifications).length === 0 ? (
            <div className="p-8 text-center text-muted-foreground">
              No notifications found
            </div>
          ) : (
            Object.entries(groupedNotifications).map(([groupName, items]) => {
              const isExpanded = expandedGroups.has(groupName) || expandedGroups.size === 0
              
              return (
                <div key={groupName} role="group" aria-labelledby={`group-${groupName}`}>
                  {/* Group Header */}
                  <button
                    onClick={() => toggleGroup(groupName)}
                    className="w-full px-4 py-2 flex items-center gap-2 hover:bg-secondary/50 transition-colors sticky top-0 bg-background border-b"
                    aria-expanded={isExpanded}
                  >
                    <ChevronDown
                      className={cn(
                        'w-4 h-4 transition-transform',
                        isExpanded ? 'rotate-0' : '-rotate-90'
                      )}
                    />
                    <span id={`group-${groupName}`} className="font-semibold text-sm flex-1 text-left">
                      {groupName}
                    </span>
                    <NotificationBadge count={items.length} size="sm" variant="default" />
                  </button>

                  {/* Group Items */}
                  {isExpanded && (
                    <div>
                      {items.map((notification) => (
                        <div
                          key={notification.id}
                          role="listitem"
                          className={cn(
                            'px-4 py-3 border-b hover:bg-secondary/30 cursor-pointer transition-colors relative',
                            !notification.read && 'bg-primary/5 border-l-2 border-l-primary'
                          )}
                          onClick={() => {
                            onNotificationClick(notification.id)
                            if (!notification.read) onMarkAsRead(notification.id)
                          }}
                        >
                          <div className="flex gap-3">
                            {!notification.read && (
                              <div className="w-2 h-2 rounded-full bg-primary mt-1.5 flex-shrink-0" aria-label="Unread" />
                            )}
                            <div className="flex-1 min-w-0">
                              <div className={cn('text-sm font-medium mb-1', !notification.read && 'font-semibold')}>
                                {notification.serviceName}
                              </div>
                              <div className="text-sm text-foreground-secondary mb-1">
                                {notification.message}
                              </div>
                              <div className="text-xs text-muted-foreground">
                                {getRelativeTime(notification.timestamp)}
                              </div>
                            </div>
                            <button
                              onClick={(e) => {
                                e.stopPropagation()
                                onMarkAsRead(notification.id)
                              }}
                              className="flex-shrink-0 p-1 rounded hover:bg-secondary/50 transition-colors"
                              aria-label="Mark as read"
                            >
                              <Check className="w-4 h-4" />
                            </button>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )
            })
          )}
        </div>
      </aside>
    </>
  )
}
