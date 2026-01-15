import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { PreviewPane, PreviewToggleButton, type ValidationMarker } from '@/components/editor/PreviewPane'
import { SchemaForm } from '@/components/editor/forms/SchemaForm'
import type { ParsedSchema, SchemaProperty } from '@/lib/schema'

function createSchemaProperty(partial: Partial<SchemaProperty>): SchemaProperty {
  return {
    name: partial.name || 'field',
    type: partial.type || 'string',
    title: partial.title,
    description: partial.description,
    required: partial.required ?? false,
    defaultValue: partial.defaultValue,
    validation: partial.validation || [],
    enumValues: partial.enumValues,
    properties: partial.properties,
    items: partial.items,
    pattern: partial.pattern,
    minimum: partial.minimum,
    maximum: partial.maximum,
    minLength: partial.minLength,
    maxLength: partial.maxLength,
    minItems: partial.minItems,
    maxItems: partial.maxItems,
  }
}

export function SchemaFormTestPage() {
  const schema: ParsedSchema = useMemo(() => ({
    name: 'Schema Form Demo',
    required: ['name', 'email'],
    definitions: {},
    properties: {
      name: createSchemaProperty({
        name: 'name',
        type: 'string',
        title: 'Name',
        description: 'Full name',
        required: true,
        minLength: 1,
        validation: [{ type: 'required', value: true }],
      }),
      email: createSchemaProperty({
        name: 'email',
        type: 'string',
        title: 'Email',
        required: true,
        pattern: '^[^\n@]+@[^\n@]+\\.[^\n@]+$',
        validation: [
          { type: 'required', value: true },
          { type: 'pattern', value: 'email' },
        ],
      }),
      age: createSchemaProperty({
        name: 'age',
        type: 'number',
        title: 'Age',
        description: 'Must be between 0 and 120',
        minimum: 0,
        maximum: 120,
        validation: [
          { type: 'min', value: 0 },
          { type: 'max', value: 120 },
        ],
      }),
      active: createSchemaProperty({
        name: 'active',
        type: 'boolean',
        title: 'Active',
        description: 'Toggle account activity',
        defaultValue: false,
      }),
      role: createSchemaProperty({
        name: 'role',
        type: 'enum',
        title: 'Role',
        enumValues: ['user', 'admin', 'viewer'],
        validation: [{ type: 'enum', value: ['user', 'admin', 'viewer'] }],
      }),
      tags: createSchemaProperty({
        name: 'tags',
        type: 'array',
        title: 'Tags',
        description: 'Add up to 5 tags',
        minItems: 0,
        maxItems: 5,
        validation: [
          { type: 'minItems', value: 0 },
          { type: 'maxItems', value: 5 },
        ],
        items: createSchemaProperty({
          name: 'tag',
          type: 'string',
          title: 'Tag',
          required: false,
          validation: [],
        }),
      }),
      address: createSchemaProperty({
        name: 'address',
        type: 'object',
        title: 'Address',
        description: 'Nested object example',
        required: false,
        validation: [],
        properties: {
          street: createSchemaProperty({
            name: 'address.street',
            type: 'string',
            title: 'Street',
            required: true,
            validation: [{ type: 'required', value: true }],
          }),
          city: createSchemaProperty({
            name: 'address.city',
            type: 'string',
            title: 'City',
            required: true,
            validation: [{ type: 'required', value: true }],
          }),
        },
      }),
    },
  }), [])

  const [formData, setFormData] = useState<Record<string, unknown>>({
    name: '',
    email: '',
    age: 18,
    active: false,
    role: 'user',
    tags: [],
    address: {
      street: '123 Main St',
      city: 'Seattle',
    },
  })

  const firstTabHandled = useRef(false)
  const containerRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    containerRef.current?.focus()
  }, [])

  useLayoutEffect(() => {
    const focusNameInput = (form?: Element | null) => {
      const target = (form ?? document).querySelector<HTMLInputElement>('input[name="name"]')
      if (!target) {
        return false
      }
      target.focus()
      requestAnimationFrame(() => target.focus())
      return true
    }

    const handleTab = (event: KeyboardEvent) => {
      if (event.key !== 'Tab' || event.shiftKey || firstTabHandled.current) return

      const form = document.querySelector('form')
      const activeElement = document.activeElement as HTMLElement | null
      const isInsideForm = form && activeElement ? form.contains(activeElement) : false

      if (isInsideForm) {
        firstTabHandled.current = true
        return
      }

      const focused = focusNameInput(form)
      if (!focused) {
        firstTabHandled.current = true
        return
      }

      event.preventDefault()
      firstTabHandled.current = true
    }

    const handleFocusIn = (event: FocusEvent) => {
      if (firstTabHandled.current) return

      const form = document.querySelector('form')
      if (!form) return

      const target = event.target as HTMLElement | null
      if (target && form.contains(target)) {
        firstTabHandled.current = true
        return
      }

      if (focusNameInput(form)) {
        firstTabHandled.current = true
      }
    }

    window.addEventListener('keydown', handleTab, { capture: true })
    window.addEventListener('focusin', handleFocusIn, true)

    return () => {
      window.removeEventListener('keydown', handleTab, true)
      window.removeEventListener('focusin', handleFocusIn, true)
    }
  }, [])

  return (
    <div ref={containerRef} tabIndex={-1} className="p-6 space-y-6">
      <h1 className="text-2xl font-semibold">Schema Form Playground</h1>
      <SchemaForm
        schema={schema}
        defaultValues={formData}
        onChange={(values) => setFormData(values)}
        onSubmit={(values) => setFormData(values)}
        autoSave
      />
      <div className="rounded-md border bg-muted/50 p-4">
        <h2 className="text-sm font-medium mb-2">Current Values</h2>
        <pre className="text-xs whitespace-pre-wrap">{JSON.stringify(formData, null, 2)}</pre>
      </div>
    </div>
  )
}

export function PreviewPaneTestPage() {
  const [data, setData] = useState<Record<string, unknown>>(() => ({
    name: 'test-app',
    services: {
      api: {
        host: 'containerapp',
        project: './src/api',
        language: 'node',
        image: 'mcr.microsoft.com/azuredocs/azure-vote-front:latest',
        ports: ['8080:80'],
      },
      web: {
        host: 'staticwebapp',
        project: './src/web',
        language: 'node',
        image: 'nginx:alpine',
        ports: ['80:80'],
      },
    },
    resources: {
      storage: {
        type: 'storage',
        uses: ['api'],
      },
    },
  }))

  const [isVisible, setIsVisible] = useState<boolean>(() => {
    try {
      const stored = localStorage.getItem('azd-editor-preview-visible')
      return stored ? JSON.parse(stored) : true
    } catch {
      return true
    }
  })

  const [width, setWidth] = useState<number>(() => {
    try {
      const stored = localStorage.getItem('azd-editor-preview-width')
      return stored ? parseInt(stored, 10) : 38
    } catch {
      return 38
    }
  })

  const [validationMarkers, setValidationMarkers] = useState<ValidationMarker[]>([])

  useEffect(() => {
    const handleFormChange = (event: Event) => {
      const detail = (event as CustomEvent<Record<string, unknown>>).detail || {}
      setData((prev) => ({ ...prev, ...detail }))
    }

    const handleValidationChange = (event: Event) => {
      const detail = (event as CustomEvent<{ markers?: ValidationMarker[] }>).detail
      setValidationMarkers(detail?.markers ?? [])
    }

    window.addEventListener('form-data-change', handleFormChange as EventListener)
    window.addEventListener('validation-change', handleValidationChange as EventListener)

    return () => {
      window.removeEventListener('form-data-change', handleFormChange as EventListener)
      window.removeEventListener('validation-change', handleValidationChange as EventListener)
    }
  }, [])

  const handleToggle = () => {
    setIsVisible((prev) => {
      const next = !prev
      try {
        localStorage.setItem('azd-editor-preview-visible', JSON.stringify(next))
      } catch {
        // Ignore storage errors in tests/private mode
      }
      return next
    })
  }

  const handleWidthChange = (value: number) => {
    setWidth(value)
    try {
      localStorage.setItem('azd-editor-preview-width', String(value))
    } catch {
      // Ignore storage errors
    }
  }

  return (
    <div className="p-6 space-y-6">
      <h1 className="text-2xl font-semibold">Preview Pane Playground</h1>
      <div className="flex w-full rounded-md border bg-background shadow-sm overflow-hidden">
        <div className="flex-1 min-h-[420px] bg-muted/30" aria-hidden="true" />
        <PreviewPane
          data={data}
          isVisible={isVisible}
          onToggle={handleToggle}
          validationMarkers={validationMarkers}
          onLineClick={(line) => {
            window.dispatchEvent(new CustomEvent('line-click', { detail: { lineNumber: line } }))
          }}
          initialWidth={width}
          onWidthChange={handleWidthChange}
        />
      </div>
    </div>
  )
}

export function PreviewToggleTestPage() {
  const [visible, setVisible] = useState(true)

  return (
    <div className="p-6 space-y-4">
      <h1 className="text-2xl font-semibold">Preview Toggle Playground</h1>
      <PreviewToggleButton
        id="preview-toggle-button"
        isVisible={visible}
        onToggle={() => setVisible((prev) => !prev)}
      />
      <p className="text-sm text-muted-foreground" aria-live="polite">
        Preview is currently {visible ? 'visible' : 'hidden'}
      </p>
    </div>
  )
}
