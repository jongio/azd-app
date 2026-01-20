/**
 * Default skip links configuration
 * Separated from component for Fast Refresh compatibility
 */

export interface SkipLink {
  id: string
  label: string
  target: string
}

/**
 * Default skip links for editor
 */
export const DEFAULT_SKIP_LINKS: SkipLink[] = [
  {
    id: 'skip-to-main',
    label: 'Skip to main content',
    target: 'main-content',
  },
  {
    id: 'skip-to-nav',
    label: 'Skip to navigation',
    target: 'navigation-sidebar',
  },
  {
    id: 'skip-to-editor',
    label: 'Skip to editor',
    target: 'editor-pane',
  },
  {
    id: 'skip-to-preview',
    label: 'Skip to preview',
    target: 'preview-pane',
  },
]
