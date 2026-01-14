/**
 * Command History
 * Manages recent command history with localStorage persistence
 */

import type { CommandHistory } from './command-types'

const STORAGE_KEY = 'azd-command-palette-history'
const MAX_RECENT = 10

/**
 * Load command history from localStorage
 */
export function loadHistory(): CommandHistory {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    
    if (!stored) {
      return { recent: [], lastUpdated: Date.now() }
    }
    
    const parsed = JSON.parse(stored) as CommandHistory
    
    // Validate structure
    if (!Array.isArray(parsed.recent) || typeof parsed.lastUpdated !== 'number') {
      return { recent: [], lastUpdated: Date.now() }
    }
    
    return parsed
  } catch {
    return { recent: [], lastUpdated: Date.now() }
  }
}

/**
 * Save command history to localStorage
 */
export function saveHistory(history: CommandHistory): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(history))
  } catch {
    // Ignore localStorage errors (quota exceeded, etc.)
  }
}

/**
 * Add command to history
 * Moves command to front if already present
 */
export function addToHistory(commandId: string, history: CommandHistory): CommandHistory {
  // Remove command if already in history
  const filtered = history.recent.filter((id) => id !== commandId)
  
  // Add to front
  const recent = [commandId, ...filtered].slice(0, MAX_RECENT)
  
  const updated = {
    recent,
    lastUpdated: Date.now(),
  }
  
  // Persist to localStorage
  saveHistory(updated)
  
  return updated
}

/**
 * Clear command history
 */
export function clearHistory(): CommandHistory {
  const empty = { recent: [], lastUpdated: Date.now() }
  saveHistory(empty)
  return empty
}

/**
 * Remove specific command from history
 */
export function removeFromHistory(
  commandId: string,
  history: CommandHistory
): CommandHistory {
  const recent = history.recent.filter((id) => id !== commandId)
  
  const updated = {
    recent,
    lastUpdated: Date.now(),
  }
  
  saveHistory(updated)
  
  return updated
}
