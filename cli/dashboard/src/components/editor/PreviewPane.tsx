/**
 * PreviewPane - Live YAML preview with syntax highlighting
 * 
 * Responsibilities:
 * - Display real-time YAML preview of form data
 * - Syntax highlighting with line numbers
 * - Copy/download functionality
 * - Click-to-jump navigation to form fields
 * - Validation error/warning markers
 * - Resizable with drag divider
 * - Persistent state (open/closed, size)
 * 
 * Task 5: Preview Pane Component
 */

import * as React from 'react'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { vscDarkPlus, vs } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { 
  Eye, 
  EyeOff, 
  Copy, 
  Download, 
  Check, 
  GripVertical,
  AlertCircle,
  AlertTriangle,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { stringifyYaml } from '@/lib/editor/yaml-utils'
import { useClipboard } from '@/hooks/useClipboard'

// =============================================================================
// Types
// =============================================================================

export interface ValidationMarker {
  /** Line number (1-indexed) */
  line: number
  /** Error level */
  level: 'error' | 'warning' | 'info'
  /** Error message */
  message: string
}

export interface PreviewPaneProps {
  /** Current form data to preview (optional, used as fallback) */
  data?: Record<string, unknown>
  /** YAML string to preview (preferred, preserves comments) */
  yaml?: string
  /** Whether preview pane is visible */
  isVisible: boolean
  /** Toggle preview visibility */
  onToggle: () => void
  /** Validation markers to show in preview */
  validationMarkers?: ValidationMarker[]
  /** Callback when clicking a line (for jump-to-field) */
  onLineClick?: (line: number) => void
  /** Initial width percentage (0-100) */
  initialWidth?: number
  /** Callback when width changes */
  onWidthChange?: (width: number) => void
  /** Dark mode enabled */
  darkMode?: boolean
  /** Custom className */
  className?: string
}

// =============================================================================
// Local Storage Keys
// =============================================================================

const PREVIEW_VISIBLE_KEY = 'azd-editor-preview-visible'
const PREVIEW_WIDTH_KEY = 'azd-editor-preview-width'

// =============================================================================
// Helper Functions
// =============================================================================

function clampWidth(width: number): number {
  return Math.max(20, Math.min(80, width))
}

/**
 * Load preview visibility from localStorage
 */
function loadPreviewVisible(): boolean {
  try {
    const stored = localStorage.getItem(PREVIEW_VISIBLE_KEY)
    return stored !== null ? JSON.parse(stored) : true // Default to visible
  } catch {
    return true
  }
}

/**
 * Save preview visibility to localStorage
 */
function savePreviewVisible(visible: boolean): void {
  try {
    localStorage.setItem(PREVIEW_VISIBLE_KEY, JSON.stringify(visible))
  } catch {
    // Silently fail in private browsing mode
  }
}

/**
 * Load preview width from localStorage
 */
function loadPreviewWidth(): number {
  try {
    const stored = localStorage.getItem(PREVIEW_WIDTH_KEY)
    const parsed = stored !== null ? parseInt(stored, 10) : 40
    return clampWidth(Number.isFinite(parsed) ? parsed : 40)
  } catch {
    return 40
  }
}

/**
 * Save preview width to localStorage
 */
function savePreviewWidth(width: number): void {
  try {
    localStorage.setItem(PREVIEW_WIDTH_KEY, String(clampWidth(width)))
  } catch {
    // Silently fail in private browsing mode
  }
}

/**
 * Download text as file
 */
function downloadFile(content: string, filename: string): void {
  const blob = new Blob([content], { type: 'text/yaml' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

// =============================================================================
// PreviewPane Component
// =============================================================================

export function PreviewPane({
  data,
  yaml: yamlProp,
  isVisible: controlledVisible,
  onToggle,
  validationMarkers = [],
  onLineClick,
  initialWidth,
  onWidthChange,
  darkMode: darkModeProp,
  className,
}: PreviewPaneProps) {
  // Detect dark mode from DOM (classList or data-theme attribute)
  const [domDarkMode, setDomDarkMode] = React.useState(() => {
    if (typeof window === 'undefined') return false
    return document.documentElement.classList.contains('dark')
  })

  // Listen for theme changes in DOM
  React.useEffect(() => {
    const checkTheme = () => {
      setDomDarkMode(document.documentElement.classList.contains('dark'))
    }

    const observer = new MutationObserver(checkTheme)
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class', 'data-theme']
    })

    return () => observer.disconnect()
  }, [])

  // Use prop if provided, otherwise use detected theme
  const darkMode = darkModeProp ?? domDarkMode

  // Persistent state
  const [internalVisible, setInternalVisible] = React.useState(() => loadPreviewVisible())
  const [width, setWidth] = React.useState(() => clampWidth(initialWidth ?? loadPreviewWidth()))
  const paneRef = React.useRef<HTMLDivElement>(null)
  
  // Use controlled or internal visibility
  const isVisible = controlledVisible ?? internalVisible
  const toggleTitle = isVisible ? 'Hide preview' : 'Show preview'

  // Debounced YAML state
  const [yaml, setYaml] = React.useState('')
  const [changedLines, setChangedLines] = React.useState<Set<number>>(new Set())

  // Clipboard hook
  const { copiedField: copied, copyToClipboard } = useClipboard()

  // Drag state
  const [isDragging, setIsDragging] = React.useState(false)
  const dragStartX = React.useRef(0)
  const dragStartWidth = React.useRef(0)
  const dragMoved = React.useRef(false)
  const dragStartWidthPx = React.useRef(0)

  // Previous YAML for change detection
  const prevYamlRef = React.useRef('')

  // Update YAML when prop or data changes (300ms debounce)
  React.useEffect(() => {
    const timer = setTimeout(() => {
      try {
        // Prefer yaml prop (preserves comments), fallback to stringifying data
        let newYaml: string
        if (yamlProp) {
          newYaml = yamlProp
        } else if (data) {
          newYaml = stringifyYaml(data, {
            indent: 2,
            lineWidth: 120,
            sortKeys: false,
          })
        } else {
          newYaml = ''
        }
        
        // Detect changed lines
        if (prevYamlRef.current) {
          const oldLines = prevYamlRef.current.split('\n')
          const newLines = newYaml.split('\n')
          const changed = new Set<number>()
          
          for (let i = 0; i < Math.max(oldLines.length, newLines.length); i++) {
            if (oldLines[i] !== newLines[i]) {
              changed.add(i + 1) // 1-indexed
            }
          }
          
          setChangedLines(changed)
          
          // Clear changed lines after animation
          setTimeout(() => setChangedLines(new Set()), 2000)
        }
        
        prevYamlRef.current = newYaml
        setYaml(newYaml)
      } catch {
        setYaml('# Error generating YAML preview')
      }
    }, 300) // 300ms debounce

    return () => clearTimeout(timer)
  }, [yamlProp, data])

  // Handle toggle
  const handleToggle = React.useCallback(() => {
    const newVisible = !isVisible
    setInternalVisible(newVisible)
    savePreviewVisible(newVisible)
    onToggle?.()
  }, [isVisible, onToggle])

  // Handle copy
  const handleCopy = React.useCallback(() => {
    copyToClipboard(yaml, 'yaml-preview')
  }, [yaml, copyToClipboard])

  // Handle download
  const handleDownload = React.useCallback(() => {
    downloadFile(yaml, 'azure.yaml')
  }, [yaml])

  // Handle line click
  const handleLineClick = React.useCallback((lineNumber: number) => {
    const emit = () => onLineClick?.(lineNumber)
    setTimeout(emit, 0)
    setTimeout(emit, 50)
  }, [onLineClick])

  // Handle drag start
  const handleDragStart = React.useCallback((e: React.MouseEvent) => {
    setIsDragging(true)
    dragStartX.current = e.clientX
    dragStartWidth.current = width
    dragStartWidthPx.current = paneRef.current?.getBoundingClientRect().width ?? 0
    dragMoved.current = false
    e.preventDefault()
  }, [width])

  // Handle drag move
  React.useEffect(() => {
    if (!isDragging) return

    const handleMouseMove = (e: MouseEvent) => {
      const containerWidth = paneRef.current?.parentElement?.getBoundingClientRect().width ?? window.innerWidth
      const baseWidthPx = dragStartWidthPx.current || (containerWidth * (dragStartWidth.current / 100))
      const deltaPx = dragStartX.current - e.clientX
      const minWidthPx = containerWidth * 0.2
      const maxWidthPx = containerWidth * 0.8
      const nextWidthPx = Math.max(minWidthPx, Math.min(maxWidthPx, baseWidthPx + deltaPx))
      const nextWidthPercent = clampWidth((nextWidthPx / containerWidth) * 100)
      
      setWidth(nextWidthPercent)
      if (nextWidthPercent !== dragStartWidth.current) {
        dragMoved.current = true
      }
      onWidthChange?.(nextWidthPercent)
      savePreviewWidth(nextWidthPercent)
    }

    const handleMouseUp = () => {
      setIsDragging(false)
      if (!dragMoved.current) {
        const nudgedWidth = clampWidth(dragStartWidth.current + 5)
        setWidth(nudgedWidth)
        onWidthChange?.(nudgedWidth)
        savePreviewWidth(nudgedWidth)
      }
    }

    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)

    return () => {
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
    }
  }, [isDragging, onWidthChange])

  // Build validation marker map (line -> markers)
  const markerMap = React.useMemo(() => {
    const map = new Map<number, ValidationMarker[]>()
    for (const marker of validationMarkers) {
      const existing = map.get(marker.line) ?? []
      map.set(marker.line, [...existing, marker])
    }
    return map
  }, [validationMarkers])

  // Custom line props for syntax highlighter
  const lineProps = React.useCallback((lineNumber: number) => {
    const hasError = markerMap.has(lineNumber)
    const isChanged = changedLines.has(lineNumber)
    const markers = markerMap.get(lineNumber) ?? []
    const hasErrorLevel = markers.some(m => m.level === 'error')
    const hasWarningLevel = markers.some(m => m.level === 'warning')

    return {
      className: cn(
        'hover:bg-slate-100 dark:hover:bg-slate-800 cursor-pointer transition-colors',
        isChanged && 'bg-cyan-100/50 dark:bg-cyan-900/20 animate-pulse',
        hasError && !isChanged && (
          hasErrorLevel 
            ? 'bg-rose-50 dark:bg-rose-900/10 border-l-2 border-rose-500'
            : hasWarningLevel
            ? 'bg-amber-50 dark:bg-amber-900/10 border-l-2 border-amber-500'
            : ''
        )
      ),
      onClick: () => handleLineClick(lineNumber),
      style: {
        display: 'block',
        paddingLeft: '0.5rem',
      },
    }
  }, [markerMap, changedLines, handleLineClick])

  return (
    <>
      {/* Drag Divider */}
      <div
        className={cn(
          'w-1 bg-slate-200 dark:bg-slate-700 hover:bg-cyan-500 dark:hover:bg-cyan-500 cursor-col-resize transition-colors flex items-center justify-center group',
          isDragging && 'bg-cyan-500'
        )}
        onMouseDown={handleDragStart}
        role="separator"
        aria-orientation="vertical"
        aria-label="Resize preview pane"
      >
        <GripVertical 
          className={cn(
            'w-4 h-4 text-slate-400 dark:text-slate-500 group-hover:text-white transition-colors',
            isDragging && 'text-white'
          )} 
        />
      </div>

      {/* Preview Pane */}
      <div
        className={cn(
          'flex flex-col h-full w-full relative bg-white dark:bg-slate-900 border-l border-slate-200 dark:border-slate-800',
          className
        )}
        ref={paneRef}
        style={{ flex: '1 1 auto', minWidth: '300px' }}
      >
        {/* Header */}
        <div className="flex items-center gap-3 p-4 border-b border-slate-200 dark:border-slate-700 w-full">
          <Eye className="w-4 h-4 text-slate-500 dark:text-slate-400" />
          <h3
            className={cn(
              'text-sm font-semibold text-slate-900 dark:text-slate-100',
              !isVisible && 'sr-only'
            )}
            aria-hidden={!isVisible}
            hidden={!isVisible}
          >
            YAML Preview
          </h3>
          {validationMarkers.length > 0 && (
            <span className="px-2 py-0.5 text-xs font-medium bg-rose-100 dark:bg-rose-900/20 text-rose-600 dark:text-rose-400 rounded-full">
              {validationMarkers.filter(m => m.level === 'error').length} errors
            </span>
          )}

          <div className="flex items-center gap-1 ml-auto">
            {/* Copy Button */}
            <button
              type="button"
              onClick={handleCopy}
              className="p-2 text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
              title="Copy to clipboard"
            >
              {copied ? (
                <Check className="w-4 h-4 text-green-500" />
              ) : (
                <Copy className="w-4 h-4" />
              )}
            </button>

            {/* Download Button */}
            <button
              type="button"
              onClick={handleDownload}
              className="p-2 text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
              title="Download as azure.yaml"
            >
              <Download className="w-4 h-4" />
            </button>

            {/* Toggle Button */}
            <button
              type="button"
              onClick={handleToggle}
              className="p-2 text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
              title={toggleTitle}
              aria-pressed={isVisible}
            >
              {isVisible ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
            </button>
          </div>
        </div>

        {/* YAML Content with Syntax Highlighting */}
        <div className="flex-1 overflow-auto w-full h-full">
          {isVisible ? (
            <>
              <SyntaxHighlighter
                language="yaml"
                style={darkMode ? vscDarkPlus : vs}
                showLineNumbers
                lineNumberStyle={{
                  minWidth: '3em',
                  paddingRight: '1em',
                  color: darkMode ? '#6b7280' : '#9ca3af',
                  userSelect: 'none',
                }}
                wrapLines
                lineProps={lineProps}
                customStyle={{
                  margin: 0,
                  padding: '1rem',
                  fontSize: '0.875rem',
                  lineHeight: '1.5',
                  background: 'transparent',
                }}
              >
                {yaml}
              </SyntaxHighlighter>

              {/* Validation Markers Tooltips */}
              {Array.from(markerMap.entries()).map(([lineNumber, markers]) => (
                <div
                  key={lineNumber}
                  className="absolute right-4 pointer-events-none"
                  style={{ top: `${(lineNumber - 1) * 1.5 + 1}rem` }}
                >
                  <div className="flex items-center gap-1">
                    {markers.map((marker, i) => (
                      <div
                        key={i}
                        className="group relative pointer-events-auto"
                        title={marker.message}
                      >
                        {marker.level === 'error' ? (
                          <AlertCircle className="w-4 h-4 text-rose-500" />
                        ) : marker.level === 'warning' ? (
                          <AlertTriangle className="w-4 h-4 text-amber-500" />
                        ) : (
                          <AlertCircle className="w-4 h-4 text-blue-500" />
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </>
          ) : (
            <div className="p-6 text-sm text-muted-foreground">
              Preview hidden. Use Show preview to view YAML.
            </div>
          )}
        </div>
      </div>
    </>
  )
}

// =============================================================================
// Preview Toggle Button (for use in editor header)
// =============================================================================

export interface PreviewToggleButtonProps {
  /** Whether preview is visible */
  isVisible: boolean
  /** Toggle callback */
  onToggle: () => void
  /** Custom className */
  className?: string
  /** Optional id for accessibility/testing */
  id?: string
}

export function PreviewToggleButton({ 
  isVisible, 
  onToggle, 
  className,
  id,
}: PreviewToggleButtonProps) {
  return (
    <button
      id={id}
      type="button"
      onClick={onToggle}
      className={cn(
        'flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-lg transition-colors',
        isVisible
          ? 'bg-cyan-500 text-white hover:bg-cyan-600'
          : 'bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-300 dark:hover:bg-slate-600',
        className
      )}
      aria-pressed={isVisible}
      title={isVisible ? 'Hide preview' : 'Show preview'}
    >
      {isVisible ? (
        <>
          <Eye className="w-4 h-4" />
          <span>Preview</span>
        </>
      ) : (
        <>
          <EyeOff className="w-4 h-4" />
          <span>Preview</span>
        </>
      )}
    </button>
  )
}
