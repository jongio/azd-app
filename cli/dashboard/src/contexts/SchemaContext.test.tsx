/**
 * Tests for SchemaContext
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SchemaProvider, useSchema } from './SchemaContext'
import * as schemaLoader from '@/lib/schema/schema-loader'

// Test component that uses the schema context
function TestComponent() {
  const { schema, isLoading, error, source } = useSchema()

  if (isLoading) {
    return <div>Loading...</div>
  }

  if (error) {
    return <div>Error: {error}</div>
  }

  return (
    <div>
      <div data-testid="schema-name">{schema?.name}</div>
      <div data-testid="schema-source">{source}</div>
      <div data-testid="properties-count">
        {schema?.properties ? Object.keys(schema.properties).length : 0}
      </div>
    </div>
  )
}

describe('SchemaContext', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('should load and parse schema successfully', async () => {
    const mockSchema = {
      $schema: 'http://json-schema.org/draft-07/schema#',
      title: 'Test Schema',
      type: 'object',
      required: ['name'],
      properties: {
        name: {
          type: 'string',
          description: 'Name field',
        },
        count: {
          type: 'number',
        },
      },
    }

    vi.spyOn(schemaLoader, 'loadSchema').mockResolvedValueOnce({
      success: true,
      schema: mockSchema,
      source: 'remote',
    })

    render(
      <SchemaProvider>
        <TestComponent />
      </SchemaProvider>
    )

    // Should show loading initially
    expect(screen.getByText('Loading...')).toBeInTheDocument()

    // Wait for schema to load
    await waitFor(() => {
      expect(screen.getByTestId('schema-name')).toHaveTextContent('Test Schema')
    })

    expect(screen.getByTestId('schema-source')).toHaveTextContent('remote')
    expect(screen.getByTestId('properties-count')).toHaveTextContent('2')
  })

  it('should handle schema load errors', async () => {
    vi.spyOn(schemaLoader, 'loadSchema').mockResolvedValueOnce({
      success: false,
      schema: null,
      source: 'remote',
      error: 'Network error',
    })

    render(
      <SchemaProvider>
        <TestComponent />
      </SchemaProvider>
    )

    await waitFor(() => {
      expect(screen.getByText(/Error:/)).toBeInTheDocument()
    })
  })

  it('should use bundled schema as fallback', async () => {
    const mockBundledSchema = {
      $schema: 'http://json-schema.org/draft-07/schema#',
      title: 'Bundled Schema',
      type: 'object',
      properties: {
        name: { type: 'string' },
      },
    }

    vi.spyOn(schemaLoader, 'loadSchema').mockResolvedValueOnce({
      success: true,
      schema: mockBundledSchema,
      source: 'bundled',
      error: 'Remote fetch failed',
    })

    render(
      <SchemaProvider>
        <TestComponent />
      </SchemaProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId('schema-name')).toHaveTextContent('Bundled Schema')
    })

    expect(screen.getByTestId('schema-source')).toHaveTextContent('bundled')
  })

  it('should provide refreshSchema function', async () => {
    const mockSchema = {
      $schema: 'http://json-schema.org/draft-07/schema#',
      title: 'Test Schema',
      type: 'object',
      properties: {
        name: { type: 'string' },
      },
    }

    const loadSchemaSpy = vi.spyOn(schemaLoader, 'loadSchema').mockResolvedValue({
      success: true,
      schema: mockSchema,
      source: 'remote',
    })

    function TestRefreshComponent() {
      const { refreshSchema } = useSchema()

      return <button onClick={() => void refreshSchema()}>Refresh</button>
    }

    const user = userEvent.setup()
    
    render(
      <SchemaProvider>
        <TestRefreshComponent />
      </SchemaProvider>
    )

    await waitFor(() => {
      expect(loadSchemaSpy).toHaveBeenCalledTimes(1)
    })

    // Click refresh button using userEvent
    const refreshButton = screen.getByText('Refresh')
    await user.click(refreshButton)

    await waitFor(() => {
      expect(loadSchemaSpy).toHaveBeenCalledTimes(2)
    })
  })

  it('should throw error when useSchema is used outside provider', () => {
    // Suppress console.error for this test
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})

    expect(() => {
      render(<TestComponent />)
    }).toThrow('useSchema must be used within SchemaProvider')

    consoleError.mockRestore()
  })

  it('should cache parsed schema in memory', async () => {
    const mockSchema = {
      $schema: 'http://json-schema.org/draft-07/schema#',
      title: 'Test Schema',
      type: 'object',
      properties: {
        name: { type: 'string' },
      },
    }

    vi.spyOn(schemaLoader, 'loadSchema').mockResolvedValueOnce({
      success: true,
      schema: mockSchema,
      source: 'remote',
    })

    function TestCachingComponent() {
      const { schema } = useSchema()
      return <div data-testid="schema-obj">{schema ? 'cached' : 'null'}</div>
    }

    const { rerender } = render(
      <SchemaProvider>
        <TestCachingComponent />
      </SchemaProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId('schema-obj')).toHaveTextContent('cached')
    })

    // Re-render should use cached schema
    rerender(
      <SchemaProvider>
        <TestCachingComponent />
      </SchemaProvider>
    )

    // Schema should still be available (cached)
    expect(screen.getByTestId('schema-obj')).toHaveTextContent('cached')
  })
})
