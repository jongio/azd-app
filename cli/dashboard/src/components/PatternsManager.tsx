import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { X } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { LogPattern } from '@/hooks/useLogPatterns'

interface PatternEditorProps {
  pattern?: LogPattern
  onSave: (pattern: Omit<LogPattern, 'id' | 'createdAt'>) => Promise<void>
  onCancel: () => void
  source: 'user' | 'app'
}

export function PatternEditor({ pattern, onSave, onCancel, source }: PatternEditorProps) {
  const [name, setName] = useState(pattern?.name || '')
  const [regex, setRegex] = useState(pattern?.regex || '')
  const [description, setDescription] = useState(pattern?.description || '')
  const [testString, setTestString] = useState('')
  const [testResult, setTestResult] = useState<boolean | null>(null)
  const [isSaving, setIsSaving] = useState(false)

  const handleTest = () => {
    try {
      const re = new RegExp(regex, 'i')
      setTestResult(re.test(testString))
    } catch {
      setTestResult(false)
    }
  }

  const handleSave = async () => {
    if (!name || !regex) return

    setIsSaving(true)
    try {
      await onSave({
        name,
        regex,
        description,
        enabled: true,
        source
      })
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <label className="text-sm font-medium">Pattern Name</label>
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g., Zero Errors"
        />
      </div>

      <div>
        <label className="text-sm font-medium">Regular Expression</label>
        <Input
          value={regex}
          onChange={(e) => setRegex(e.target.value)}
          placeholder="e.g., ^.*\\b0\\s+errors?\\b.*$"
          className="font-mono"
        />
      </div>

      <div>
        <label className="text-sm font-medium">Description</label>
        <Input
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="What does this pattern match?"
        />
      </div>

      <div className="border-t pt-4">
        <label className="text-sm font-medium">Test Pattern</label>
        <div className="flex gap-2 mt-2">
          <Input
            value={testString}
            onChange={(e) => setTestString(e.target.value)}
            placeholder="Enter a log line to test"
            className="flex-1"
          />
          <Button onClick={handleTest} variant="outline">
            Test
          </Button>
        </div>
        {testResult !== null && (
          <div className={cn(
            "mt-2 p-2 rounded text-sm",
            testResult ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" : "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
          )}>
            {testResult ? '✓ Pattern matches' : '✗ Pattern does not match'}
          </div>
        )}
      </div>

      <div className="flex justify-end gap-2 pt-4 border-t">
        <Button variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button onClick={() => void handleSave()} disabled={!name || !regex || isSaving}>
          {isSaving ? 'Saving...' : 'Save Pattern'}
        </Button>
      </div>
    </div>
  )
}

interface PatternsListProps {
  patterns: LogPattern[]
  source: 'user' | 'app'
  onAdd: () => void
  onEdit: (pattern: LogPattern) => void
  onDelete: (id: string) => void
  onToggle: (id: string, enabled: boolean) => void
}

export function PatternsList({ patterns, source, onAdd, onEdit, onDelete, onToggle }: PatternsListProps) {
  const sourcePatterns = patterns.filter(p => p.source === source)

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h3 className="font-medium">{source === 'user' ? 'User Patterns' : 'Project Patterns'}</h3>
        <Button size="sm" onClick={onAdd}>
          Add Pattern
        </Button>
      </div>

      {sourcePatterns.length === 0 ? (
        <div className="text-center text-muted-foreground py-8 bg-card rounded-lg">
          No {source} patterns defined
        </div>
      ) : (
        <div className="space-y-2">
          {sourcePatterns.map(pattern => (
            <div key={pattern.id} className="border border-border rounded-lg p-3 space-y-2 bg-card">
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={pattern.enabled}
                      onChange={(e) => onToggle(pattern.id, e.target.checked)}
                      className="w-4 h-4"
                    />
                    <h4 className="font-medium">{pattern.name}</h4>
                  </div>
                  <p className="text-sm text-muted-foreground mt-1">{pattern.description}</p>
                  <code className="text-xs bg-secondary px-2 py-1 rounded mt-2 inline-block">
                    {pattern.regex}
                  </code>
                </div>
                <div className="flex gap-1">
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => onEdit(pattern)}
                  >
                    Edit
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => onDelete(pattern.id)}
                  >
                    <X className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
