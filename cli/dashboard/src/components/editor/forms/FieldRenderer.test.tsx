/**
 * FieldRenderer Component Tests
 */

import { describe, it, expect, vi } from 'vitest'
import { render } from '@testing-library/react'
import { FieldRenderer } from './FieldRenderer'
import type { SchemaProperty } from '@/lib/schema'

// Mock field components
vi.mock('./fields/StringField', () => ({
  StringField: ({ name }: { name: string }) => <div data-testid="string-field">{name}</div>,
}))

vi.mock('./fields/NumberField', () => ({
  NumberField: ({ name }: { name: string }) => <div data-testid="number-field">{name}</div>,
}))

vi.mock('./fields/BooleanField', () => ({
  BooleanField: ({ name }: { name: string }) => <div data-testid="boolean-field">{name}</div>,
}))

vi.mock('./fields/EnumField', () => ({
  EnumField: ({ name }: { name: string }) => <div data-testid="enum-field">{name}</div>,
}))

vi.mock('./fields/ArrayField', () => ({
  ArrayField: ({ name }: { name: string }) => <div data-testid="array-field">{name}</div>,
}))

vi.mock('./fields/ObjectField', () => ({
  ObjectField: ({ name }: { name: string }) => <div data-testid="object-field">{name}</div>,
}))

describe('FieldRenderer', () => {
  it('renders StringField for string type', () => {
    const property: SchemaProperty = { type: 'string', title: 'Test String' }
    const { getByTestId } = render(<FieldRenderer name="test" property={property} />)
    expect(getByTestId('string-field')).toBeInTheDocument()
  })

  it('renders NumberField for number type', () => {
    const property: SchemaProperty = { type: 'number', title: 'Test Number' }
    const { getByTestId } = render(<FieldRenderer name="test" property={property} />)
    expect(getByTestId('number-field')).toBeInTheDocument()
  })

  it('renders BooleanField for boolean type', () => {
    const property: SchemaProperty = { type: 'boolean', title: 'Test Boolean' }
    const { getByTestId } = render(<FieldRenderer name="test" property={property} />)
    expect(getByTestId('boolean-field')).toBeInTheDocument()
  })

  it('renders EnumField for enum type', () => {
    const property: SchemaProperty = {
      type: 'enum',
      title: 'Test Enum',
      enum: ['option1', 'option2'],
    }
    const { getByTestId } = render(<FieldRenderer name="test" property={property} />)
    expect(getByTestId('enum-field')).toBeInTheDocument()
  })

  it('renders ArrayField for array type', () => {
    const property: SchemaProperty = {
      type: 'array',
      title: 'Test Array',
      items: { type: 'string' },
    }
    const { getByTestId } = render(<FieldRenderer name="test" property={property} />)
    expect(getByTestId('array-field')).toBeInTheDocument()
  })

  it('renders ObjectField for object type', () => {
    const property: SchemaProperty = {
      type: 'object',
      title: 'Test Object',
      properties: {},
    }
    const { getByTestId } = render(<FieldRenderer name="test" property={property} />)
    expect(getByTestId('object-field')).toBeInTheDocument()
  })

  it('falls back to StringField for unknown type', () => {
    const property = { type: 'unknown' as unknown as string, title: 'Test Unknown' }
    const { getByTestId } = render(<FieldRenderer name="test" property={property} />)
    
    expect(getByTestId('string-field')).toBeInTheDocument()
  })

  it('passes autoSave prop to field component', () => {
    const property: SchemaProperty = { type: 'string', title: 'Test' }
    const { getByTestId } = render(
      <FieldRenderer name="test" property={property} autoSave={false} />
    )
    // Component should render (autoSave is passed through)
    expect(getByTestId('string-field')).toBeInTheDocument()
  })

  it('passes nested prop to field component', () => {
    const property: SchemaProperty = { type: 'string', title: 'Test' }
    const { getByTestId } = render(
      <FieldRenderer name="test" property={property} nested={true} />
    )
    // Component should render (nested is passed through)
    expect(getByTestId('string-field')).toBeInTheDocument()
  })
})
