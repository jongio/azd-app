# Schema Infrastructure

Schema loading, parsing, and caching infrastructure for the Azure YAML Editor.

## Overview

This module provides the foundation for schema-driven form generation in the Azure YAML Editor. It handles loading the azure.yaml JSON Schema from remote sources with fallback to bundled versions, parsing schema definitions into an internal model, and caching the parsed schema in memory.

## Architecture

### Components

1. **schema-loader.ts** - Loads JSON Schema from remote URL with local fallback
2. **schema-parser.ts** - Parses JSON Schema into internal TypeScript model
3. **SchemaContext.tsx** - React context provider for schema state management
4. **bundled-schema.json** - Local copy of azure.yaml JSON Schema for offline use

### Data Flow

```
Remote Schema URL
       ↓
  loadSchema()
       ↓ (on error)
Bundled Schema ──→ parseSchema() ──→ ParsedSchema ──→ SchemaContext
       ↓
   In-Memory Cache
       ↓
  React Components
```

## Usage

### Basic Usage

```typescript
import { SchemaProvider, useSchema } from '@/contexts/SchemaContext'

// Wrap your app with SchemaProvider
function App() {
  return (
    <SchemaProvider>
      <YourComponents />
    </SchemaProvider>
  )
}

// Use schema in components
function YourComponent() {
  const { schema, isLoading, error, source } = useSchema()

  if (isLoading) return <Loading />
  if (error) return <Error message={error} />

  return (
    <div>
      <h1>{schema.name}</h1>
      {/* Use schema.properties for form generation */}
    </div>
  )
}
```

### Advanced Usage

```typescript
import { parseSchema, getPropertyByPath } from '@/lib/schema'

// Parse a custom schema
const parsed = parseSchema(customSchema)

// Get nested property
const hostProperty = getPropertyByPath(parsed, 'services.api.host')
console.log(hostProperty.type) // 'enum'
console.log(hostProperty.enumValues) // ['local', 'containerapp', ...]
```

## API Reference

### schema-loader.ts

#### `loadSchema(): Promise<SchemaLoadResult>`

Loads JSON Schema from remote URL with automatic fallback to bundled schema.

**Returns:**
```typescript
{
  success: boolean
  schema: Record<string, unknown> | null
  source: 'remote' | 'bundled'
  error?: string
}
```

**Behavior:**
- Attempts to fetch from `https://raw.githubusercontent.com/jongio/azd-app/main/schemas/v1.1/azure.yaml.json`
- 5-second timeout on network requests
- Automatic fallback to bundled schema on any error
- Errors logged to console but don't throw

#### `getBundledSchema(): Record<string, unknown>`

Returns the bundled schema synchronously.

**Use case:** When you need immediate schema access without async loading.

---

### schema-parser.ts

#### `parseSchema(schema: Record<string, unknown>): ParsedSchema`

Parses JSON Schema into internal TypeScript model.

**Returns:**
```typescript
{
  name: string
  properties: Record<string, SchemaProperty>
  required: string[]
  definitions: Record<string, SchemaProperty>
}
```

**Parsed property structure:**
```typescript
{
  name: string
  type: 'string' | 'number' | 'boolean' | 'object' | 'array' | 'enum'
  title?: string
  description?: string
  required: boolean
  defaultValue?: unknown
  validation: ValidationRule[]
  enumValues?: string[]
  properties?: Record<string, SchemaProperty>  // for objects
  items?: SchemaProperty                        // for arrays
  pattern?: string
  minimum?: number
  maximum?: number
  minLength?: number
  maxLength?: number
  minItems?: number
  maxItems?: number
}
```

**Validation rules:**
```typescript
{
  type: 'required' | 'pattern' | 'min' | 'max' | 'minLength' | 'maxLength' | 'minItems' | 'maxItems' | 'enum'
  value: unknown
  message?: string
}
```

#### `getPropertyByPath(schema: ParsedSchema, path: string): SchemaProperty | null`

Retrieves a property by dot-notation path.

**Examples:**
```typescript
getPropertyByPath(schema, 'name')                    // top-level property
getPropertyByPath(schema, 'services.api.host')       // nested property
getPropertyByPath(schema, 'service')                 // from definitions
```

---

### SchemaContext.tsx

#### `<SchemaProvider>`

React context provider that loads and caches schema on mount.

**Props:**
```typescript
{
  children: ReactNode
}
```

**Behavior:**
- Loads schema automatically on mount
- Caches parsed schema in memory
- Provides loading/error states
- Exposes refresh function for manual reload

#### `useSchema()`

Hook to access schema context.

**Returns:**
```typescript
{
  schema: ParsedSchema | null
  rawSchema: Record<string, unknown> | null
  isLoading: boolean
  error: string | null
  source: 'remote' | 'bundled' | null
  refreshSchema: () => Promise<void>
}
```

**Throws:** Error if used outside `<SchemaProvider>`

## Implementation Details

### Schema Loading Strategy

1. **Primary:** Fetch from GitHub (remote URL)
   - Fast CDN delivery
   - Always up-to-date
   - 5-second timeout

2. **Fallback:** Bundled schema (local JSON)
   - Guaranteed availability
   - Zero network dependency
   - Updated with each build

### Error Handling

All errors are caught and logged, but never thrown. The system always provides a valid schema through the bundled fallback.

**Error scenarios handled:**
- Network timeout (5s)
- HTTP errors (404, 500, etc.)
- Invalid JSON response
- Parse errors
- Missing schema file

### Performance

- **Initial load:** <500ms (remote) or <100ms (bundled)
- **Parse time:** <200ms for full azure.yaml schema
- **Memory usage:** ~500KB for parsed schema
- **Caching:** In-memory, survives re-renders

### Type Safety

All types are fully typed with TypeScript:
- JSON Schema → `Record<string, unknown>`
- Parsed Schema → `ParsedSchema` (strongly typed)
- Properties → `SchemaProperty` (discriminated union by type)

## Testing

### Test Coverage

- **Overall:** 91.79%
- **schema-loader.ts:** 100%
- **schema-parser.ts:** 88.42%
- **SchemaContext.tsx:** 100%

### Running Tests

```bash
# Unit tests
pnpm test -- src/lib/schema/ src/contexts/SchemaContext --run

# Coverage report
pnpm test:coverage -- src/lib/schema/ src/contexts/SchemaContext
```

### Test Scenarios

**schema-loader.test.ts:**
- ✓ Load from remote URL successfully
- ✓ Fallback on network error
- ✓ Fallback on HTTP error
- ✓ Fallback on invalid JSON
- ✓ Get bundled schema synchronously
- ✓ Validate bundled schema structure

**schema-parser.test.ts:**
- ✓ Parse basic properties
- ✓ Parse enums
- ✓ Parse nested objects
- ✓ Parse arrays
- ✓ Extract validation rules
- ✓ Handle default values
- ✓ Handle boolean properties
- ✓ Handle union types
- ✓ Parse definitions
- ✓ Handle missing properties
- ✓ Get property by path
- ✓ Get nested property
- ✓ Get property from definitions
- ✓ Handle non-existent paths
- ✓ Handle invalid nested paths

**SchemaContext.test.tsx:**
- ✓ Load and parse schema successfully
- ✓ Handle schema load errors
- ✓ Use bundled schema as fallback
- ✓ Provide refreshSchema function
- ✓ Throw error when used outside provider
- ✓ Cache parsed schema in memory

## Future Enhancements

1. **Schema Versioning**
   - Support multiple schema versions
   - Version selection UI
   - Migration helpers

2. **Schema Validation**
   - Validate azure.yaml against schema using Ajv
   - Real-time validation feedback
   - Detailed error messages

3. **Schema Extensions**
   - Custom properties
   - Plugin system for additional fields
   - Schema composition

4. **Performance Optimization**
   - Lazy parsing (parse on demand)
   - Incremental updates
   - Worker thread parsing

## License

MIT
