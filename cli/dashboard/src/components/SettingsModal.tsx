import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Slider } from '@/components/ui/slider'
import { X } from 'lucide-react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { PatternsList, PatternEditor } from './PatternsManager'
import { useLogPatterns, type LogPattern } from '@/hooks/useLogPatterns'
import { usePreferences } from '@/hooks/usePreferences'

interface SettingsModalProps {
  isOpen: boolean
  onClose: () => void
}

export function SettingsModal({ isOpen, onClose }: SettingsModalProps) {
  const { patterns, addPattern, updatePattern, deletePattern } = useLogPatterns()
  const { preferences, updateUI } = usePreferences()
  const [editingPattern, setEditingPattern] = useState<LogPattern | null>(null)
  const [creatingSource, setCreatingSource] = useState<'user' | 'app' | null>(null)
  const [activeTab, setActiveTab] = useState<string>('general')

  if (!isOpen) return null

  const handleSavePattern = async (pattern: Omit<LogPattern, 'id' | 'createdAt'>) => {
    if (editingPattern) {
      await updatePattern(editingPattern.id, pattern)
      setEditingPattern(null)
    } else if (creatingSource) {
      await addPattern({ ...pattern, source: creatingSource })
      setCreatingSource(null)
    }
  }

  const handleToggle = async (id: string, enabled: boolean) => {
    await updatePattern(id, { enabled })
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={onClose}>
      <div className="bg-background rounded-lg shadow-xl max-w-3xl w-full max-h-[80vh] overflow-hidden border border-border" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between p-4 border-b border-border bg-background">
          <h2 className="text-lg font-semibold text-foreground">Logs Dashboard Settings</h2>
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="w-4 h-4" />
          </Button>
        </div>

        <div className="p-4 overflow-y-auto max-h-[calc(80vh-80px)] bg-background">
          {editingPattern || creatingSource ? (
            <PatternEditor
              pattern={editingPattern || undefined}
              source={creatingSource || editingPattern!.source}
              onSave={handleSavePattern}
              onCancel={() => {
                setEditingPattern(null)
                setCreatingSource(null)
              }}
            />
          ) : (
            <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
              <TabsList className="grid w-full grid-cols-3">
                <TabsTrigger value="general">General</TabsTrigger>
                <TabsTrigger value="user">User Patterns</TabsTrigger>
                <TabsTrigger value="app">Project Patterns</TabsTrigger>
              </TabsList>
              
              <TabsContent value="general" className="mt-4 space-y-4">
                <div>
                  <h3 className="font-medium mb-4">Grid Layout</h3>
                  <Slider
                    label="Grid Columns"
                    min={1}
                    max={6}
                    value={preferences.ui.gridColumns}
                    onChange={(e) => updateUI({ gridColumns: parseInt(e.target.value) })}
                    showValue
                  />
                </div>
              </TabsContent>
              
              <TabsContent value="user" className="mt-4">
                <PatternsList
                  patterns={patterns}
                  source="user"
                  onAdd={() => setCreatingSource('user')}
                  onEdit={setEditingPattern}
                  onDelete={(id) => void deletePattern(id)}
                  onToggle={(id, enabled) => void handleToggle(id, enabled)}
                />
              </TabsContent>
              
              <TabsContent value="app" className="mt-4">
                <PatternsList
                  patterns={patterns}
                  source="app"
                  onAdd={() => setCreatingSource('app')}
                  onEdit={setEditingPattern}
                  onDelete={(id) => void deletePattern(id)}
                  onToggle={(id, enabled) => void handleToggle(id, enabled)}
                />
              </TabsContent>
            </Tabs>
          )}
        </div>
      </div>
    </div>
  )
}
