/**
 * ArrayField and ObjectField Tests
 */

import { describe, it, expect } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FormProvider, useForm } from 'react-hook-form'
import { ArrayField } from './ArrayField'
import { ObjectField } from './ObjectField'
import { StringField } from './StringField'
import type { SchemaProperty } from '@/lib/schema'
import type { FieldRendererProps } from '../FieldRenderer'

// Mock FieldRenderer for testing nested fields
function MockFieldRenderer({ name, property }: FieldRendererProps) {
  // For simplicity, just render string fields in tests
  return <StringField name={name} property={property} />
}

// Test wrapper component with React Hook Form context
function TestWrapper({ children, defaultValues = {} }: { children: React.ReactNode; defaultValues?: Record<string, unknown> }) {
  const methods = useForm({ defaultValues })
  return <FormProvider {...methods}>{children}</FormProvider>
}

describe('ArrayField', () => {
  const property: SchemaProperty = {
    name: 'tags',
    type: 'array',
    title: 'Tags',
    description: 'List of tags',
    required: false,
    minItems: 1,
    maxItems: 5,
    items: {
      name: 'tag',
      type: 'string',
      title: 'Tag',
      required: false,
      validation: [],
    } as SchemaProperty,
    validation: [
      { type: 'minItems', value: 1 },
      { type: 'maxItems', value: 5 },
    ],
  }

  it('renders empty array field with add button', () => {
    render(
      <TestWrapper defaultValues={{ tags: [] }}>
        <ArrayField name="tags" property={property} FieldRenderer={MockFieldRenderer} />
      </TestWrapper>
    )

    expect(screen.getByText('Tags')).toBeInTheDocument()
    expect(screen.getByText(/no items/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /add item/i })).toBeInTheDocument()
  })

  it('adds new item when add button is clicked', async () => {
    const user = userEvent.setup()

    render(
      <TestWrapper defaultValues={{ tags: [] }}>
        <ArrayField name="tags" property={property} FieldRenderer={MockFieldRenderer} />
      </TestWrapper>
    )

    const addButton = screen.getByRole('button', { name: /add item/i })
    await user.click(addButton)

    await waitFor(() => {
      expect(screen.queryByText(/no items/i)).not.toBeInTheDocument()
      expect(screen.getByLabelText(/tag #1/i)).toBeInTheDocument()
    })
  })

  it('removes item when remove button is clicked', async () => {
    const user = userEvent.setup()

    // Use 3 items so after removal we still have 2 (above minItems of 1)
    render(
      <TestWrapper defaultValues={{ tags: ['tag1', 'tag2', 'tag3'] }}>
        <ArrayField name="tags" property={property} FieldRenderer={MockFieldRenderer} />
      </TestWrapper>
    )

    // Wait for initial render - should have 3 remove buttons
    await waitFor(() => {
      const removeButtons = screen.queryAllByRole('button', { name: /remove item/i })
      expect(removeButtons).toHaveLength(3)
    })

    const removeButtons = screen.getAllByRole('button', { name: /remove item/i })
    await user.click(removeButtons[0])

    // After removing 1 item, should have 2 remove buttons (2 items > minItems of 1)
    await waitFor(() => {
      const updatedButtons = screen.queryAllByRole('button', { name: /remove item/i })
      expect(updatedButtons.length).toBe(2)
    })
  })

  it('disables add button when max items reached', () => {
    render(
      <TestWrapper defaultValues={{ tags: ['tag1', 'tag2', 'tag3', 'tag4', 'tag5'] }}>
        <ArrayField name="tags" property={property} FieldRenderer={MockFieldRenderer} />
      </TestWrapper>
    )

    expect(screen.queryByRole('button', { name: /add item/i })).not.toBeInTheDocument()
    expect(screen.getByText(/maximum 5 items reached/i)).toBeInTheDocument()
  })

  it('shows minimum items message when below minimum', () => {
    render(
      <TestWrapper defaultValues={{ tags: [] }}>
        <ArrayField name="tags" property={property} FieldRenderer={MockFieldRenderer} />
      </TestWrapper>
    )

    expect(screen.getByText(/minimum 1 item required/i)).toBeInTheDocument()
  })

  it('supports drag and drop reordering', async () => {
    const { container } = render(
      <TestWrapper defaultValues={{ tags: ['tag1', 'tag2', 'tag3'] }}>
        <ArrayField name="tags" property={property} FieldRenderer={MockFieldRenderer} />
      </TestWrapper>
    )

    // Find draggable containers
    const draggableItems = container.querySelectorAll('[draggable="true"]')
    expect(draggableItems.length).toBe(3)
  })
})

describe('ObjectField', () => {
  const property: SchemaProperty = {
    name: 'address',
    type: 'object',
    title: 'Address',
    description: 'Your address',
    required: false,
    properties: {
      street: {
        name: 'street',
        type: 'string',
        title: 'Street',
        required: true,
        validation: [{ type: 'required', value: true }],
      } as SchemaProperty,
      city: {
        name: 'city',
        type: 'string',
        title: 'City',
        required: true,
        validation: [{ type: 'required', value: true }],
      } as SchemaProperty,
      zipCode: {
        name: 'zipCode',
        type: 'string',
        title: 'Zip Code',
        required: false,
        validation: [],
      } as SchemaProperty,
    },
    validation: [],
  }

  it('renders object field with nested properties', () => {
    render(
      <TestWrapper defaultValues={{ address: {} }}>
        <ObjectField name="address" property={property} FieldRenderer={MockFieldRenderer} />
      </TestWrapper>
    )

    expect(screen.getAllByText('Address')[0]).toBeInTheDocument()
    expect(screen.getByLabelText(/street/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/city/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/zip code/i)).toBeInTheDocument()
  })

  it('expands and collapses on header click', async () => {
    const user = userEvent.setup()

    render(
      <TestWrapper defaultValues={{ address: {} }}>
        <ObjectField name="address" property={property} FieldRenderer={MockFieldRenderer} />
      </TestWrapper>
    )

    // Initially expanded
    expect(screen.getByLabelText(/street/i)).toBeInTheDocument()

    // Click to collapse - find the first button that's not a Help button
    const buttons = screen.getAllByRole('button')
    const toggleButton = buttons.find(btn => {
      const ariaLabel = btn.getAttribute('aria-label')
      return ariaLabel !== 'Help'
    })
    expect(toggleButton).toBeDefined()
    await user.click(toggleButton!)

    await waitFor(() => {
      expect(screen.queryByLabelText(/street/i)).not.toBeInTheDocument()
    })

    // Click to expand again
    await user.click(toggleButton!)

    await waitFor(() => {
      expect(screen.getByLabelText(/street/i)).toBeInTheDocument()
    })
  })

  it('shows empty state when no properties defined', () => {
    const emptyProperty: SchemaProperty = {
      name: 'empty',
      type: 'object',
      title: 'Empty Object',
      required: false,
      properties: {},
      validation: [],
    }

    render(
      <TestWrapper>
        <ObjectField name="empty" property={emptyProperty} FieldRenderer={MockFieldRenderer} />
      </TestWrapper>
    )

    expect(screen.getByText(/no properties defined/i)).toBeInTheDocument()
  })

  it('applies nested styling', () => {
    const { container } = render(
      <TestWrapper defaultValues={{ address: {} }}>
        <ObjectField name="address" property={property} nested FieldRenderer={MockFieldRenderer} />
      </TestWrapper>
    )

    const outerDiv = container.querySelector('.ml-4')
    expect(outerDiv).toBeInTheDocument()
  })
})
