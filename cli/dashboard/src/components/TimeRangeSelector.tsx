/**
 * TimeRangeSelector - Time range picker for historical log queries
 * Provides preset buttons (15m, 30m, 6h, 24h) and custom datetime inputs.
 */
import * as React from 'react'
import { Calendar, Clock } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TimeRangePreset, TimeRange } from '@/hooks/useHistoricalLogs'

// =============================================================================
// Types
// =============================================================================

export interface TimeRangeSelectorProps {
  /** Current time range selection */
  value: TimeRange
  /** Callback when time range changes */
  onChange: (timeRange: TimeRange) => void
  /** Whether the selector is disabled */
  disabled?: boolean
  /** Additional class names */
  className?: string
}

// =============================================================================
// Constants
// =============================================================================

const TIME_PRESETS: { value: TimeRangePreset; label: string }[] = [
  { value: '15m', label: '15m' },
  { value: '30m', label: '30m' },
  { value: '6h', label: '6h' },
  { value: '24h', label: '24h' },
  { value: 'custom', label: 'Custom' },
]

// =============================================================================
// Helper Functions
// =============================================================================

/**
 * Formats a Date to datetime-local input value (YYYY-MM-DDTHH:mm)
 */
function formatDateTimeLocal(date: Date): string {
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

/**
 * Gets default start date (1 hour ago) for custom range
 */
function getDefaultStart(): Date {
  const date = new Date()
  date.setHours(date.getHours() - 1)
  return date
}

/**
 * Gets default end date (now) for custom range
 */
function getDefaultEnd(): Date {
  return new Date()
}

/**
 * Gets the maximum allowed date (7 days ago)
 */
function getMinAllowedDate(): string {
  const date = new Date()
  date.setDate(date.getDate() - 7)
  return formatDateTimeLocal(date)
}

/**
 * Gets the maximum allowed date (now)
 */
function getMaxAllowedDate(): string {
  return formatDateTimeLocal(new Date())
}

// =============================================================================
// TimeRangeSelector Component
// =============================================================================

export function TimeRangeSelector({
  value,
  onChange,
  disabled = false,
  className,
}: Readonly<TimeRangeSelectorProps>) {
  // Local state for custom range inputs
  const [customStart, setCustomStart] = React.useState<string>(() => 
    formatDateTimeLocal(value.start ?? getDefaultStart())
  )
  const [customEnd, setCustomEnd] = React.useState<string>(() => 
    formatDateTimeLocal(value.end ?? getDefaultEnd())
  )

  // Update local state when value changes externally
  React.useEffect(() => {
    if (value.preset === 'custom' && value.start && value.end) {
      setCustomStart(formatDateTimeLocal(value.start))
      setCustomEnd(formatDateTimeLocal(value.end))
    }
  }, [value])

  const handlePresetClick = (preset: TimeRangePreset) => {
    if (disabled) return

    if (preset === 'custom') {
      // Initialize custom range with defaults
      const start = getDefaultStart()
      const end = getDefaultEnd()
      setCustomStart(formatDateTimeLocal(start))
      setCustomEnd(formatDateTimeLocal(end))
      onChange({ preset: 'custom', start, end })
    } else {
      onChange({ preset })
    }
  }

  const handleCustomStartChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newStart = e.target.value
    setCustomStart(newStart)
  }

  const handleCustomEndChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newEnd = e.target.value
    setCustomEnd(newEnd)
  }

  const handleApplyCustomRange = () => {
    if (!customStart || !customEnd) return

    const start = new Date(customStart)
    const end = new Date(customEnd)

    // Validate range
    if (start >= end) {
      // Could show error, for now just swap
      onChange({ preset: 'custom', start: end, end: start })
      return
    }

    // Check max 7 days
    const diffDays = (end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24)
    if (diffDays > 7) {
      // Clamp to 7 days from end
      const clampedStart = new Date(end.getTime() - 7 * 24 * 60 * 60 * 1000)
      onChange({ preset: 'custom', start: clampedStart, end })
      return
    }

    onChange({ preset: 'custom', start, end })
  }

  return (
    <div className={cn('space-y-3', className)}>
      {/* Label */}
      <label className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
        Time Range
      </label>

      {/* Preset Buttons */}
      <div 
        className="flex items-center gap-1 p-1 rounded-lg bg-slate-100 dark:bg-slate-800"
        role="radiogroup"
        aria-label="Time range selection"
      >
        {TIME_PRESETS.map(({ value: preset, label }) => (
          <button
            key={preset}
            type="button"
            role="radio"
            aria-checked={value.preset === preset}
            onClick={() => handlePresetClick(preset)}
            disabled={disabled}
            className={cn(
              'flex-1 px-3 py-1.5 rounded-md text-sm font-medium',
              'transition-all duration-200 ease-out',
              value.preset === preset ? [
                'bg-white dark:bg-slate-700',
                'text-slate-900 dark:text-slate-100',
                'shadow-sm',
              ] : [
                'text-slate-500 dark:text-slate-400',
                'hover:text-slate-700 dark:hover:text-slate-300',
              ],
              'focus-visible:outline-none focus-visible:ring-2',
              'focus-visible:ring-cyan-500 focus-visible:ring-offset-1',
              'disabled:opacity-50 disabled:cursor-not-allowed',
            )}
          >
            {label}
          </button>
        ))}
      </div>

      {/* Custom Range Picker */}
      {value.preset === 'custom' && (
        <div className="p-3 rounded-lg bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 space-y-3">
          {/* Start Date/Time */}
          <div className="space-y-1.5">
            <label htmlFor="custom-start-datetime" className="flex items-center gap-1.5 text-xs font-medium text-slate-500 dark:text-slate-400">
              <Calendar className="w-3.5 h-3.5" />
              Start
            </label>
            <input
              id="custom-start-datetime"
              type="datetime-local"
              value={customStart}
              onChange={handleCustomStartChange}
              min={getMinAllowedDate()}
              max={customEnd || getMaxAllowedDate()}
              disabled={disabled}
              className={cn(
                'w-full px-3 py-2 rounded-md text-sm',
                'bg-white dark:bg-slate-800',
                'border border-slate-200 dark:border-slate-600',
                'text-slate-900 dark:text-slate-100',
                'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-transparent',
                'disabled:opacity-50 disabled:cursor-not-allowed',
              )}
            />
          </div>

          {/* End Date/Time */}
          <div className="space-y-1.5">
            <label htmlFor="custom-end-datetime" className="flex items-center gap-1.5 text-xs font-medium text-slate-500 dark:text-slate-400">
              <Clock className="w-3.5 h-3.5" />
              End
            </label>
            <input
              id="custom-end-datetime"
              type="datetime-local"
              value={customEnd}
              onChange={handleCustomEndChange}
              min={customStart || getMinAllowedDate()}
              max={getMaxAllowedDate()}
              disabled={disabled}
              className={cn(
                'w-full px-3 py-2 rounded-md text-sm',
                'bg-white dark:bg-slate-800',
                'border border-slate-200 dark:border-slate-600',
                'text-slate-900 dark:text-slate-100',
                'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-transparent',
                'disabled:opacity-50 disabled:cursor-not-allowed',
              )}
            />
          </div>

          {/* Apply Button */}
          <button
            type="button"
            onClick={handleApplyCustomRange}
            disabled={disabled || !customStart || !customEnd}
            className={cn(
              'w-full px-4 py-2 rounded-md text-sm font-medium',
              'bg-cyan-600 hover:bg-cyan-700 text-white',
              'transition-colors duration-200',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
              'disabled:opacity-50 disabled:cursor-not-allowed',
            )}
          >
            Apply Range
          </button>

          {/* Hint */}
          <p className="text-xs text-slate-400 dark:text-slate-500 text-center">
            Maximum range: 7 days
          </p>
        </div>
      )}
    </div>
  )
}

export default TimeRangeSelector
