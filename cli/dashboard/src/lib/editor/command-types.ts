/**
 * Command Palette Types
 * Type definitions for command palette commands and results
 */

export type CommandCategory = 'navigation' | 'action' | 'field' | 'help'

export interface Command {
  /** Unique command identifier */
  id: string
  
  /** Display name */
  label: string
  
  /** Description/subtitle */
  description?: string
  
  /** Command category */
  category: CommandCategory
  
  /** Icon name (from lucide-react) */
  icon?: string
  
  /** Keywords for search matching */
  keywords?: string[]
  
  /** Keyboard shortcut hint */
  shortcut?: string
  
  /** Command action */
  action: CommandAction
}

export type CommandAction = 
  | { type: 'navigate'; path: string }
  | { type: 'execute'; handler: () => void }
  | { type: 'jump-to-field'; fieldPath: string }
  | { type: 'open-help'; topic: string }

export interface CommandSearchResult {
  /** The matched command */
  command: Command
  
  /** Match score (0-1, higher is better) */
  score: number
  
  /** Indices of matched characters */
  matches?: number[]
}

export interface CommandHistory {
  /** Recently executed commands (up to 10) */
  recent: string[]
  
  /** Last updated timestamp */
  lastUpdated: number
}

export interface CommandPaletteState {
  /** Whether palette is open */
  isOpen: boolean
  
  /** Current search query */
  query: string
  
  /** Filtered and ranked results */
  results: CommandSearchResult[]
  
  /** Currently selected result index */
  selectedIndex: number
  
  /** Command history */
  history: CommandHistory
}
