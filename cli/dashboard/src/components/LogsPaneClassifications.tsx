import type { ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import { Info, AlertTriangle, XCircle, Check } from 'lucide-react'

export type ClassificationLevel = 'info' | 'warning' | 'error'

export interface LogsPaneClassificationOverlayProps {
  selectionPosition: { x: number; y: number } | null
  selectedText: string
  handleClassifySelection: (level: ClassificationLevel) => void
}

export function LogsPaneClassificationOverlay({
  selectionPosition,
  selectedText,
  handleClassifySelection,
}: Readonly<LogsPaneClassificationOverlayProps>): ReactNode {
  if (!selectionPosition || !selectedText) {
    return null
  }

  return (
    <div
      className="classification-popup fixed z-50 flex gap-1 bg-popover border rounded-md shadow-lg p-1"
      style={{
        left: `${selectionPosition.x}px`,
        top: `${selectionPosition.y}px`,
        transform: 'translate(-50%, -100%)'
      }}
    >
      <Button
        size="sm"
        variant="ghost"
        onClick={() => handleClassifySelection('info')}
        className="h-8 px-2 bg-blue-500 hover:bg-blue-600"
        title="Classify as Info"
      >
        <Info className="w-4 h-4 text-white" />
      </Button>
      <Button
        size="sm"
        variant="ghost"
        onClick={() => handleClassifySelection('warning')}
        className="h-8 px-2 bg-yellow-500 hover:bg-yellow-600"
        title="Classify as Warning"
      >
        <AlertTriangle className="w-4 h-4 text-white" />
      </Button>
      <Button
        size="sm"
        variant="ghost"
        onClick={() => handleClassifySelection('error')}
        className="h-8 px-2 bg-red-500 hover:bg-red-600"
        title="Classify as Error"
      >
        <XCircle className="w-4 h-4 text-white" />
      </Button>
    </div>
  )
}

export interface LogsPaneClassificationToastProps {
  show: boolean
}

export function LogsPaneClassificationToast({
  show,
}: Readonly<LogsPaneClassificationToastProps>): ReactNode {
  if (!show) {
    return null
  }

  return (
    <div
      className="fixed z-50 bg-green-500 text-white px-4 py-2 rounded-md shadow-lg flex items-center gap-2"
      style={{
        top: '20px',
        right: '20px'
      }}
    >
      <Check className="w-4 h-4" />
      <span>Classification saved</span>
    </div>
  )
}
