/**
 * Type-safe localStorage utilities with error handling.
 * Centralizes localStorage access patterns to eliminate duplication.
 */

/**
 * Safely parse JSON from localStorage with validation.
 * Returns defaultValue if the key doesn't exist, parsing fails, or validation fails.
 * 
 * @param key - localStorage key
 * @param defaultValue - Value to return if parsing fails
 * @param validator - Optional validation function to verify the parsed data
 */
export function getStorageItem<T>(
  key: string,
  defaultValue: T,
  validator?: (value: unknown) => value is T
): T {
  if (typeof window === 'undefined') {
    return defaultValue
  }

  try {
    const stored = localStorage.getItem(key)
    if (stored === null) {
      return defaultValue
    }

    const parsed: unknown = JSON.parse(stored)
    
    if (validator) {
      if (validator(parsed)) {
        return parsed
      }
      console.warn(`Invalid data format in localStorage key "${key}", using defaults`)
      return defaultValue
    }

    return parsed as T
  } catch (e) {
    console.warn(`Failed to parse localStorage key "${key}":`, e)
    return defaultValue
  }
}

/**
 * Safely set a value in localStorage with JSON serialization.
 * 
 * @param key - localStorage key
 * @param value - Value to store
 */
export function setStorageItem<T>(key: string, value: T): void {
  if (typeof window === 'undefined') {
    return
  }

  try {
    localStorage.setItem(key, JSON.stringify(value))
  } catch (e) {
    console.error(`Failed to save to localStorage key "${key}":`, e)
  }
}

/**
 * Remove an item from localStorage.
 * 
 * @param key - localStorage key
 */
export function removeStorageItem(key: string): void {
  if (typeof window === 'undefined') {
    return
  }

  try {
    localStorage.removeItem(key)
  } catch (e) {
    console.error(`Failed to remove localStorage key "${key}":`, e)
  }
}

// ============================================================================
// Type Validators for common data types
// ============================================================================

/**
 * Validates that a value is a boolean.
 */
export function isBoolean(value: unknown): value is boolean {
  return typeof value === 'boolean'
}

/**
 * Validates that a value is a Record<string, boolean>.
 */
export function isRecordOfBooleans(value: unknown): value is Record<string, boolean> {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    return false
  }
  return Object.values(value).every(v => typeof v === 'boolean')
}

/**
 * Creates a validator for string arrays where each string must be in a valid set.
 */
export function createStringArrayValidator<T extends string>(validValues: readonly T[]) {
  return (value: unknown): value is T[] => {
    if (!Array.isArray(value)) {
      return false
    }
    return value.every(v => typeof v === 'string' && validValues.includes(v as T))
  }
}
