/**
 * Screen reader announcer utility
 * Creates live regions for screen reader announcements
 */

export type AnnouncementPriority = 'polite' | 'assertive'

export interface AnnouncementOptions {
  priority?: AnnouncementPriority
  delay?: number
}

/**
 * Singleton announcer instance
 */
class ScreenReaderAnnouncer {
  private politeRegion: HTMLElement | null = null
  private assertiveRegion: HTMLElement | null = null

  constructor() {
    this.createRegions()
  }

  private createRegions() {
    // Remove existing regions if present to avoid duplicates between test environments
    if (this.politeRegion && document.body.contains(this.politeRegion)) {
      document.body.removeChild(this.politeRegion)
    }
    if (this.assertiveRegion && document.body.contains(this.assertiveRegion)) {
      document.body.removeChild(this.assertiveRegion)
    }

    // Create polite region
    this.politeRegion = document.createElement('div')
    this.politeRegion.setAttribute('role', 'status')
    this.politeRegion.setAttribute('aria-live', 'polite')
    this.politeRegion.setAttribute('aria-atomic', 'true')
    this.politeRegion.className = 'sr-only'
    document.body.appendChild(this.politeRegion)

    // Create assertive region
    this.assertiveRegion = document.createElement('div')
    this.assertiveRegion.setAttribute('role', 'alert')
    this.assertiveRegion.setAttribute('aria-live', 'assertive')
    this.assertiveRegion.setAttribute('aria-atomic', 'true')
    this.assertiveRegion.className = 'sr-only'
    document.body.appendChild(this.assertiveRegion)
  }

  private ensureRegions() {
    if (!this.politeRegion || !document.body.contains(this.politeRegion) || !this.assertiveRegion || !document.body.contains(this.assertiveRegion)) {
      this.createRegions()
    }
  }

  /**
   * Announce a message to screen readers
   */
  announce(message: string, options: AnnouncementOptions = {}) {
    this.ensureRegions()
    const { priority = 'polite', delay = 100 } = options
    const region = priority === 'assertive' ? this.assertiveRegion : this.politeRegion

    if (!region) return

    // Clear previous message
    region.textContent = ''

    // Set new message after delay (allows screen reader to detect change)
    setTimeout(() => {
      region.textContent = message
    }, delay)

    // Clear after announcement
    setTimeout(() => {
      region.textContent = ''
    }, delay + 3000)
  }

  /**
   * Announce success message
   */
  announceSuccess(message: string) {
    this.announce(`Success: ${message}`, { priority: 'polite' })
  }

  /**
   * Announce error message
   */
  announceError(message: string) {
    this.announce(`Error: ${message}`, { priority: 'assertive' })
  }

  /**
   * Announce warning message
   */
  announceWarning(message: string) {
    this.announce(`Warning: ${message}`, { priority: 'polite' })
  }

  /**
   * Announce navigation change
   */
  announceNavigation(message: string) {
    this.announce(`Navigated to ${message}`, { priority: 'polite' })
  }

  /**
   * Announce modal state
   */
  announceModal(isOpen: boolean, modalTitle: string) {
    if (isOpen) {
      this.announce(`${modalTitle} dialog opened`, { priority: 'polite' })
    } else {
      this.announce(`${modalTitle} dialog closed`, { priority: 'polite' })
    }
  }

  /**
   * Cleanup regions
   */
  cleanup() {
    if (this.politeRegion) {
      document.body.removeChild(this.politeRegion)
      this.politeRegion = null
    }
    if (this.assertiveRegion) {
      document.body.removeChild(this.assertiveRegion)
      this.assertiveRegion = null
    }
  }
}

// Singleton instance
let announcer: ScreenReaderAnnouncer | null = null

/**
 * Get announcer instance
 */
export function getAnnouncer(): ScreenReaderAnnouncer {
  if (!announcer) {
    announcer = new ScreenReaderAnnouncer()
  } else {
    announcer.ensureRegions()
  }
  return announcer
}

/**
 * Announce a message to screen readers
 */
export function announce(message: string, options?: AnnouncementOptions) {
  getAnnouncer().announce(message, options)
}

/**
 * React hook for announcements
 */
export function useAnnouncer() {
  return React.useMemo(() => getAnnouncer(), [])
}

// For React import
import * as React from 'react'
