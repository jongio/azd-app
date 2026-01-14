/**
 * Visual Design System Example
 * Task 21: Visual Design and Styling
 * 
 * Demonstrates usage of the enhanced editor design system components
 */

import * as React from 'react'
import { ThemeProvider, ThemeToggle } from '@/lib/editor/theme-provider'
import { EditorButton, EditorButtonGroup } from '@/components/ui/editor-button'
import {
  EditorInput,
  EditorTextarea,
  EditorSelect,
  EditorCheckbox,
  EditorToggle,
  EditorFormField,
} from '@/components/ui/editor-form-controls'

/**
 * Example component showing all design system elements
 */
export function DesignSystemExample() {
  const [inputValue, setInputValue] = React.useState('')
  const [selectValue, setSelectValue] = React.useState('option1')
  const [textareaValue, setTextareaValue] = React.useState('')
  const [checkboxChecked, setCheckboxChecked] = React.useState(false)
  const [toggleChecked, setToggleChecked] = React.useState(false)
  const [isLoading, setIsLoading] = React.useState(false)

  const handleSubmit = () => {
    setIsLoading(true)
    setTimeout(() => setIsLoading(false), 2000)
  }

  return (
    <ThemeProvider defaultTheme="system">
      <div className="min-h-screen p-8 bg-editor-bg-canvas text-editor-fg-primary">
        {/* Header with Theme Toggle */}
        <div className="flex items-center justify-between mb-8">
          <h1 className="editor-text-2xl font-bold">Azure YAML Editor - Design System</h1>
          <ThemeToggle className="editor-btn-icon" showLabel />
        </div>

        <div className="max-w-4xl mx-auto editor-space-y-xl">
          {/* Buttons Section */}
          <section className="editor-card">
            <h2 className="editor-text-xl font-semibold mb-4">Buttons</h2>
            
            <div className="editor-space-y-md">
              <div>
                <h3 className="editor-text-sm font-medium mb-2 text-editor-fg-secondary">
                  Variants
                </h3>
                <EditorButtonGroup>
                  <EditorButton variant="primary">Primary</EditorButton>
                  <EditorButton variant="secondary">Secondary</EditorButton>
                  <EditorButton variant="outline">Outline</EditorButton>
                  <EditorButton variant="ghost">Ghost</EditorButton>
                  <EditorButton variant="danger">Danger</EditorButton>
                </EditorButtonGroup>
              </div>

              <div>
                <h3 className="editor-text-sm font-medium mb-2 text-editor-fg-secondary">
                  Sizes
                </h3>
                <EditorButtonGroup>
                  <EditorButton size="sm">Small</EditorButton>
                  <EditorButton size="md">Medium</EditorButton>
                  <EditorButton size="lg">Large</EditorButton>
                </EditorButtonGroup>
              </div>

              <div>
                <h3 className="editor-text-sm font-medium mb-2 text-editor-fg-secondary">
                  States
                </h3>
                <EditorButtonGroup>
                  <EditorButton loading>Loading...</EditorButton>
                  <EditorButton disabled>Disabled</EditorButton>
                </EditorButtonGroup>
              </div>

              <div>
                <h3 className="editor-text-sm font-medium mb-2 text-editor-fg-secondary">
                  With Icons
                </h3>
                <EditorButtonGroup>
                  <EditorButton
                    variant="primary"
                    leftIcon={
                      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                    }
                  >
                    Save
                  </EditorButton>
                  <EditorButton
                    variant="outline"
                    rightIcon={
                      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                      </svg>
                    }
                  >
                    Next
                  </EditorButton>
                </EditorButtonGroup>
              </div>
            </div>
          </section>

          {/* Form Controls Section */}
          <section className="editor-card">
            <h2 className="editor-text-xl font-semibold mb-4">Form Controls</h2>

            <div className="editor-space-y-lg">
              {/* Input */}
              <EditorFormField
                label="Project Name"
                required
                helperText="Enter a unique name for your project"
              >
                <EditorInput
                  value={inputValue}
                  onChange={(e) => setInputValue(e.target.value)}
                  placeholder="my-awesome-project"
                />
              </EditorFormField>

              {/* Select */}
              <EditorFormField label="Host Type" required>
                <EditorSelect
                  value={selectValue}
                  onChange={(e) => setSelectValue(e.target.value)}
                >
                  <option value="option1">Container App</option>
                  <option value="option2">App Service</option>
                  <option value="option3">Function App</option>
                  <option value="option4">Static Web App</option>
                </EditorSelect>
              </EditorFormField>

              {/* Textarea */}
              <EditorFormField
                label="Description"
                helperText="Describe your service configuration"
              >
                <EditorTextarea
                  value={textareaValue}
                  onChange={(e) => setTextareaValue(e.target.value)}
                  placeholder="# Service configuration&#10;name: my-service&#10;host: containerapp"
                  rows={6}
                />
              </EditorFormField>

              {/* Checkbox */}
              <EditorCheckbox
                checked={checkboxChecked}
                onChange={(e) => setCheckboxChecked(e.target.checked)}
                label="Enable health check"
                helperText="Monitor service health automatically"
              />

              {/* Toggle */}
              <EditorToggle
                checked={toggleChecked}
                onCheckedChange={setToggleChecked}
                label="Production mode"
                helperText="Enable production optimizations"
              />

              {/* Input with Error */}
              <EditorFormField
                label="Email"
                required
                errorMessage="Please enter a valid email address"
              >
                <EditorInput
                  type="email"
                  placeholder="user@example.com"
                  error
                />
              </EditorFormField>
            </div>
          </section>

          {/* Badges Section */}
          <section className="editor-card">
            <h2 className="editor-text-xl font-semibold mb-4">Badges</h2>
            <div className="flex flex-wrap gap-2">
              <span className="editor-badge editor-badge-default">Default</span>
              <span className="editor-badge editor-badge-success">Success</span>
              <span className="editor-badge editor-badge-warning">Warning</span>
              <span className="editor-badge editor-badge-error">Error</span>
              <span className="editor-badge editor-badge-info">Info</span>
            </div>
          </section>

          {/* Cards Section */}
          <section className="editor-space-y-md">
            <h2 className="editor-text-xl font-semibold mb-4">Cards</h2>
            
            <div className="editor-card">
              <h3 className="editor-text-lg font-medium mb-2">Basic Card</h3>
              <p className="text-editor-fg-secondary">
                A simple card with hover shadow effect
              </p>
            </div>

            <div className="editor-card-elevated">
              <h3 className="editor-text-lg font-medium mb-2">Elevated Card</h3>
              <p className="text-editor-fg-secondary">
                An elevated card with lift effect on hover
              </p>
            </div>
          </section>

          {/* Typography Section */}
          <section className="editor-card">
            <h2 className="editor-text-xl font-semibold mb-4">Typography</h2>
            <div className="editor-space-y-sm">
              <p className="editor-text-2xl">Heading 2XL (24px)</p>
              <p className="editor-text-xl">Heading XL (20px)</p>
              <p className="editor-text-lg">Heading LG (18px)</p>
              <p className="editor-text-base">Body Base (16px)</p>
              <p className="editor-text-sm">Body Small (14px)</p>
              <p className="editor-text-xs">Caption XS (12px)</p>
              <p className="editor-font-mono">Monospace Font (Code)</p>
            </div>
          </section>

          {/* Animations Section */}
          <section className="editor-card">
            <h2 className="editor-text-xl font-semibold mb-4">Animations</h2>
            <div className="flex flex-wrap gap-4">
              <div className="editor-card-elevated editor-animate-fade-in p-4">
                <p>Fade In</p>
              </div>
              <div className="editor-card-elevated editor-animate-slide-up p-4">
                <p>Slide Up</p>
              </div>
              <div className="editor-card-elevated editor-animate-scale-in p-4">
                <p>Scale In</p>
              </div>
              <div className="editor-card-elevated p-4">
                <div className="editor-animate-spin h-6 w-6 border-2 border-primary border-t-transparent rounded-full" />
              </div>
            </div>
          </section>

          {/* Action Bar */}
          <div className="flex justify-end gap-3 pt-4 border-t border-editor-border-default">
            <EditorButton variant="outline">Cancel</EditorButton>
            <EditorButton
              variant="primary"
              loading={isLoading}
              onClick={handleSubmit}
            >
              {isLoading ? 'Saving...' : 'Save Changes'}
            </EditorButton>
          </div>
        </div>
      </div>
    </ThemeProvider>
  )
}

export default DesignSystemExample
