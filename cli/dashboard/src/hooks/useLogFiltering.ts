import { useMemo, useCallback } from 'react'
import type { LogEntry } from '@/components/LogsPane'
import { LOG_LEVELS } from '@/lib/log-utils'
import { useLogClassifications } from '@/hooks/useLogClassifications'
import {
  isErrorLine as baseIsErrorLine,
  isWarningLine as baseIsWarningLine,
} from '@/lib/log-utils'

export type PaneLogLevel = 'info' | 'warning' | 'error'

export function useLogFiltering(
  logs: LogEntry[],
  globalSearchTerm: string,
  levelFilter: Set<PaneLogLevel>
) {
  const { getClassificationForText } = useLogClassifications()

  const isErrorLine = useCallback((message: string) => {
    const classificationLevel = getClassificationForText(message)
    if (classificationLevel === 'error') return true
    if (classificationLevel === 'info' || classificationLevel === 'warning') return false
    return baseIsErrorLine(message)
  }, [getClassificationForText])

  const isWarningLine = useCallback((message: string) => {
    const classificationLevel = getClassificationForText(message)
    if (classificationLevel === 'warning') return true
    if (classificationLevel === 'info' || classificationLevel === 'error') return false
    return baseIsWarningLine(message)
  }, [getClassificationForText])

  const getPaneLogLevel = useCallback((log: LogEntry): PaneLogLevel => {
    const overrideLevel = getClassificationForText(log.message)
    const isError =
      overrideLevel === 'error' ||
      (!overrideLevel && (isErrorLine(log.message) || log.level === LOG_LEVELS.ERROR))

    if (isError) {
      return 'error'
    }

    const isWarning =
      overrideLevel === 'warning' ||
      (!overrideLevel && (isWarningLine(log.message) || log.level === LOG_LEVELS.WARNING))

    if (isWarning) {
      return 'warning'
    }

    return 'info'
  }, [getClassificationForText, isErrorLine, isWarningLine])

  const paneStatus = useMemo(() => {
    const hasError = logs.some(log => 
      isErrorLine(log.message) || log.level === LOG_LEVELS.ERROR
    )
    const hasWarning = logs.some(log => isWarningLine(log.message) || log.level === LOG_LEVELS.WARNING)

    if (hasError) return 'error'
    if (hasWarning) return 'warning'
    return 'info'
  }, [logs, isErrorLine, isWarningLine])

  const filteredLogs = useMemo(() => {
    return logs.filter(log => {
      if (!log?.message) return false
      
      // Text search filter
      if (!log.message.toLowerCase().includes(globalSearchTerm.toLowerCase())) return false
      
      // Level filter
      const logLevel = getPaneLogLevel(log)
      return levelFilter.has(logLevel)
    })
  }, [logs, globalSearchTerm, levelFilter, getPaneLogLevel])

  return {
    filteredLogs,
    getPaneLogLevel,
    paneStatus,
    isErrorLine,
    isWarningLine,
  }
}
