/**
 * File Upload Tab Component
 * Allows uploading .yaml/.yml files from disk
 */

import * as React from 'react'
import { cn } from '@/lib/utils'
import { Upload, FileText, Check } from 'lucide-react'

export interface FileUploadTabProps {
  onFileLoad: (content: string) => void
}

/**
 * File Upload Tab Component
 */
export function FileUploadTab({ onFileLoad }: FileUploadTabProps) {
  const [isDragging, setIsDragging] = React.useState(false)
  const [fileName, setFileName] = React.useState<string | null>(null)
  const [fileLoaded, setFileLoaded] = React.useState(false)
  const fileInputRef = React.useRef<HTMLInputElement>(null)

  const handleFileSelect = (file: File) => {
    if (!file.name.match(/\.(yaml|yml)$/i)) {
      alert('Please select a .yaml or .yml file')
      return
    }

    const reader = new FileReader()
    reader.onload = (e) => {
      const content = e.target?.result as string
      onFileLoad(content)
      setFileName(file.name)
      setFileLoaded(true)
    }
    reader.onerror = () => {
      alert('Failed to read file')
    }
    reader.readAsText(file)
  }

  const handleFileInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      handleFileSelect(file)
    }
  }

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(true)
  }

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(false)
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(false)

    const file = e.dataTransfer.files[0]
    if (file) {
      handleFileSelect(file)
    }
  }

  const handleBrowseClick = () => {
    fileInputRef.current?.click()
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-slate-600 dark:text-slate-400">
        Upload an azure.yaml file from your computer
      </p>

      {/* Drop Zone */}
      <div
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        className={cn(
          'relative border-2 border-dashed rounded-lg p-12 text-center transition-all duration-150',
          isDragging
            ? 'border-cyan-500 bg-cyan-50 dark:bg-cyan-900/20'
            : 'border-slate-300 dark:border-slate-700 hover:border-slate-400 dark:hover:border-slate-600',
          fileLoaded && 'bg-green-50 dark:bg-green-900/20 border-green-500'
        )}
      >
        <input
          ref={fileInputRef}
          type="file"
          accept=".yaml,.yml"
          onChange={handleFileInputChange}
          className="hidden"
        />

        <div className="flex flex-col items-center gap-4">
          {fileLoaded ? (
            <>
              <div className="w-16 h-16 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center">
                <Check className="w-8 h-8 text-green-600 dark:text-green-400" />
              </div>
              <div>
                <h3 className="text-base font-semibold text-green-900 dark:text-green-100 mb-1">
                  File Loaded
                </h3>
                <p className="text-sm text-green-700 dark:text-green-300 flex items-center gap-2 justify-center">
                  <FileText className="w-4 h-4" />
                  {fileName}
                </p>
              </div>
            </>
          ) : (
            <>
              <div className="w-16 h-16 rounded-full bg-slate-100 dark:bg-slate-800 flex items-center justify-center">
                <Upload className="w-8 h-8 text-slate-400" />
              </div>
              <div>
                <h3 className="text-base font-semibold text-slate-900 dark:text-slate-100 mb-1">
                  {isDragging ? 'Drop file here' : 'Drag and drop your file'}
                </h3>
                <p className="text-sm text-slate-600 dark:text-slate-400">
                  or{' '}
                  <button
                    type="button"
                    onClick={handleBrowseClick}
                    className="text-cyan-600 dark:text-cyan-400 hover:underline font-medium"
                  >
                    browse to select
                  </button>
                </p>
              </div>
            </>
          )}

          <p className="text-xs text-slate-500 dark:text-slate-500">
            Accepted formats: .yaml, .yml
          </p>
        </div>
      </div>

      {fileLoaded && (
        <button
          type="button"
          onClick={handleBrowseClick}
          className={cn(
            'w-full px-4 py-2 rounded-lg text-sm font-semibold',
            'text-slate-700 dark:text-slate-300',
            'border border-slate-200 dark:border-slate-700',
            'hover:bg-slate-100 dark:hover:bg-slate-800',
            'transition-colors duration-150'
          )}
        >
          Choose Different File
        </button>
      )}
    </div>
  )
}
