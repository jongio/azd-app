/**
 * Command Search
 * Fuzzy search implementation for command palette using Fuse.js
 */

import Fuse from 'fuse.js'
import type { IFuseOptions, FuseResultMatch } from 'fuse.js'
import type { Command, CommandSearchResult } from './command-types'

// Fuse.js configuration for optimal command searching
const FUSE_OPTIONS: IFuseOptions<Command> = {
  // Include score and match indices in results
  includeScore: true,
  includeMatches: true,
  
  // Search threshold (0.0 = perfect match, 1.0 = match anything)
  threshold: 0.4,
  
  // Location and distance for positional matching
  location: 0,
  distance: 100,
  
  // Minimum match character length
  minMatchCharLength: 1,
  
  // Keys to search (with weights)
  keys: [
    { name: 'label', weight: 2 },        // Highest priority
    { name: 'description', weight: 1.5 },
    { name: 'keywords', weight: 1 },
  ],
}

/**
 * Command search engine
 */
export class CommandSearch {
  private fuse: Fuse<Command>
  private commands: Command[]
  
  constructor(commands: Command[]) {
    this.commands = commands
    this.fuse = new Fuse(commands, FUSE_OPTIONS)
  }
  
  /**
   * Update the command list
   */
  updateCommands(commands: Command[]): void {
    this.commands = commands
    this.fuse.setCollection(commands)
  }
  
  /**
   * Search commands by query
   * Returns all commands if query is empty (for showing recent commands)
   */
  search(query: string, maxResults = 50): CommandSearchResult[] {
    // Empty query - return all commands sorted by category
    if (!query.trim()) {
      return this.getAllCommands().slice(0, maxResults)
    }
    
    // Perform fuzzy search
    const results = this.fuse.search(query, { limit: maxResults })
    
    // Transform Fuse.js results to our format
    return results.map((result) => ({
      command: result.item,
      score: 1 - (result.score || 0), // Invert score (higher is better)
      matches: this.extractMatchIndices(result.matches),
    }))
  }
  
  /**
   * Get all commands (for empty query)
   */
  private getAllCommands(): CommandSearchResult[] {
    return this.commands.map((command) => ({
      command,
      score: 1,
      matches: undefined,
    }))
  }
  
  /**
   * Extract character indices from Fuse.js matches
   */
  private extractMatchIndices(
    matches?: readonly FuseResultMatch[]
  ): number[] | undefined {
    if (!matches || matches.length === 0) {
      return undefined
    }
    
    // Get indices from the first (highest-weighted) match
    const firstMatch = matches[0]
    if (!firstMatch.indices || firstMatch.indices.length === 0) {
      return undefined
    }
    
    // Convert index ranges to individual indices
    const indices: number[] = []
    for (const [start, end] of firstMatch.indices) {
      for (let i = start; i <= end; i++) {
        indices.push(i)
      }
    }
    
    return indices
  }
}

/**
 * Group search results by category
 * Limits results per category and sorts by score
 */
export function groupResultsByCategory(
  results: CommandSearchResult[],
  maxPerCategory = 5
): Map<string, CommandSearchResult[]> {
  const grouped = new Map<string, CommandSearchResult[]>()
  
  // Group results by category
  for (const result of results) {
    const category = result.command.category
    
    if (!grouped.has(category)) {
      grouped.set(category, [])
    }
    
    grouped.get(category)!.push(result)
  }
  
  // Limit and sort each category
  for (const [category, categoryResults] of grouped.entries()) {
    categoryResults.sort((a, b) => b.score - a.score)
    grouped.set(category, categoryResults.slice(0, maxPerCategory))
  }
  
  return grouped
}

/**
 * Filter results by recent command IDs
 * Boosts recently used commands to the top
 */
export function filterRecentCommands(
  results: CommandSearchResult[],
  recentIds: string[]
): CommandSearchResult[] {
  const recentSet = new Set(recentIds)
  
  return results.filter((result) => recentSet.has(result.command.id))
}

/**
 * Highlight matching characters in text
 * Returns parts array for rendering in React components
 */
export function getHighlightedParts(text: string, indices?: number[]): Array<{ text: string; highlight: boolean }> {
  if (!indices || indices.length === 0) {
    return [{ text, highlight: false }]
  }
  
  const parts: Array<{ text: string; highlight: boolean }> = []
  let lastIndex = 0
  
  for (let i = 0; i < indices.length; i++) {
    const index = indices[i]
    
    // Add non-matching text before this match
    if (index > lastIndex) {
      parts.push({ text: text.slice(lastIndex, index), highlight: false })
    }
    
    // Find consecutive matches to create a highlighted span
    let endIndex = index + 1
    while (i + 1 < indices.length && indices[i + 1] === endIndex) {
      endIndex++
      i++
    }
    
    // Add highlighted match
    parts.push({ text: text.slice(index, endIndex), highlight: true })
    
    lastIndex = endIndex
  }
  
  // Add remaining text
  if (lastIndex < text.length) {
    parts.push({ text: text.slice(lastIndex), highlight: false })
  }
  
  return parts
}
