/**
 * Retry Utilities - Exponential backoff retry logic
 */

export interface RetryOptions {
  /** Maximum number of retries (default: 3) */
  maxRetries?: number
  /** Initial delay in ms (default: 1000) */
  initialDelay?: number
  /** Backoff multiplier (default: 2) */
  backoffMultiplier?: number
  /** Maximum delay in ms (default: 10000) */
  maxDelay?: number
  /** Whether to retry on this error */
  shouldRetry?: (error: unknown) => boolean
  /** Callback before retry */
  onRetry?: (attempt: number, error: unknown) => void
}

/**
 * Default retry check - retry on network errors, not 4xx errors
 */
function defaultShouldRetry(error: unknown): boolean {
  // Don't retry on client errors (4xx)
  if (error && typeof error === 'object' && 'status' in error) {
    const status = (error as { status: number }).status
    return status >= 500 || status === 408 || status === 429 // Server errors, timeout, rate limit
  }

  // Retry on network errors
  if (error instanceof TypeError && error.message.includes('fetch')) {
    return true
  }

  // Retry on timeout
  if (error instanceof Error && error.message.includes('timeout')) {
    return true
  }

  return false
}

/**
 * Retry an async function with exponential backoff
 */
export async function retryWithBackoff<T>(
  fn: () => Promise<T>,
  options: RetryOptions = {}
): Promise<T> {
  const {
    maxRetries = 3,
    initialDelay = 1000,
    backoffMultiplier = 2,
    maxDelay = 10000,
    shouldRetry = defaultShouldRetry,
    onRetry,
  } = options

  let lastError: unknown
  let delay = initialDelay

  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      return await fn()
    } catch (error) {
      lastError = error

      // Don't retry if it's the last attempt
      if (attempt === maxRetries) {
        break
      }

      // Check if we should retry
      if (!shouldRetry(error)) {
        throw error
      }

      // Call retry callback
      if (onRetry) {
        onRetry(attempt + 1, error)
      }

      // Wait before retrying with exponential backoff
      await new Promise((resolve) => setTimeout(resolve, delay))
      delay = Math.min(delay * backoffMultiplier, maxDelay)
    }
  }

  // All retries failed
  throw lastError
}

/**
 * Retry status for UI display
 */
export interface RetryStatus {
  attempt: number
  maxAttempts: number
  isRetrying: boolean
  error?: unknown
}

/**
 * Hook for retry state management
 */
export function createRetryState() {
  let status: RetryStatus = {
    attempt: 0,
    maxAttempts: 0,
    isRetrying: false,
  }

  return {
    getStatus: () => status,
    setRetrying: (attempt: number, maxAttempts: number, error?: unknown) => {
      status = { attempt, maxAttempts, isRetrying: true, error }
    },
    setComplete: () => {
      status = { ...status, isRetrying: false }
    },
    reset: () => {
      status = { attempt: 0, maxAttempts: 0, isRetrying: false }
    },
  }
}

/**
 * Format retry status for display
 */
export function formatRetryStatus(status: RetryStatus): string {
  if (!status.isRetrying) {
    return ''
  }
  return `Retrying... ${status.attempt}/${status.maxAttempts}`
}
