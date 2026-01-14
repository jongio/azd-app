/**
 * Skip links component for keyboard navigation
 * Allows users to skip to main content areas
 */

import * as React from 'react'

export interface SkipLink {
  id: string
  label: string
  target: string
}

export interface SkipLinksProps {
  links: SkipLink[]
}

/**
 * SkipLinks component
 * Renders hidden links that become visible on focus
 */
export function SkipLinks({ links }: SkipLinksProps) {
  const handleClick = (e: React.MouseEvent<HTMLAnchorElement>, targetId: string) => {
    e.preventDefault()
    const target = document.getElementById(targetId)
    if (target) {
      target.focus()
      target.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }
  }

  return (
    <div className="skip-links">
      {links.map((link) => (
        <a
          key={link.id}
          href={`#${link.target}`}
          className="sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4 focus:z-50 focus:px-4 focus:py-2 focus:bg-primary focus:text-primary-foreground focus:rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
          onClick={(e) => handleClick(e, link.target)}
        >
          {link.label}
        </a>
      ))}
    </div>
  )
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
