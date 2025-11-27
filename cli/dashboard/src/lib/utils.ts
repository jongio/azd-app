import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

// Re-export from centralized log-utils for backwards compatibility
export { convertAnsiToHtml } from '@/lib/log-utils'
