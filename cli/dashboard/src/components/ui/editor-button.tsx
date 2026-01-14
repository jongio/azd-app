/**
 * Enhanced Button Component with Editor Design System
 * Task 21: Visual Design and Styling
 * 
 * Implements comprehensive button variants with micro-interactions
 */

import * as React from 'react'
import { cn } from '@/lib/utils'

export interface EditorButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  /** Button variant */
  variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger'
  /** Button size */
  size?: 'sm' | 'md' | 'lg' | 'icon'
  /** Show loading spinner */
  loading?: boolean
  /** Icon element to display before label */
  leftIcon?: React.ReactNode
  /** Icon element to display after label */
  rightIcon?: React.ReactNode
}

/**
 * Enhanced Editor Button Component
 * 
 * @example
 * ```tsx
 * <EditorButton variant="primary" onClick={handleSave}>
 *   Save Changes
 * </EditorButton>
 * 
 * <EditorButton variant="outline" loading leftIcon={<SaveIcon />}>
 *   Saving...
 * </EditorButton>
 * ```
 */
export const EditorButton = React.forwardRef<HTMLButtonElement, EditorButtonProps>(
  (
    {
      className,
      variant = 'primary',
      size = 'md',
      loading = false,
      leftIcon,
      rightIcon,
      children,
      disabled,
      ...props
    },
    ref
  ) => {
    const variantStyles = {
      primary: 'editor-btn-primary',
      secondary: 'editor-btn-secondary',
      outline: 'editor-btn-outline',
      ghost: 'editor-btn-ghost',
      danger: 'editor-btn-danger',
    }

    const sizeStyles = {
      sm: 'h-8 px-3 text-xs',
      md: 'h-10 px-4 text-sm',
      lg: 'h-12 px-6 text-base',
      icon: 'h-10 w-10 p-0',
    }

    return (
      <button
        ref={ref}
        className={cn(
          'inline-flex items-center justify-center gap-2',
          'rounded-md font-medium',
          'ring-offset-background transition-all duration-200',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
          'disabled:pointer-events-none disabled:opacity-50',
          variantStyles[variant],
          sizeStyles[size],
          className
        )}
        disabled={disabled || loading}
        {...props}
      >
        {loading && (
          <svg
            className="animate-spin h-4 w-4"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              className="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="4"
            />
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            />
          </svg>
        )}
        {!loading && leftIcon && <span className="flex-shrink-0">{leftIcon}</span>}
        {size !== 'icon' && children}
        {!loading && rightIcon && <span className="flex-shrink-0">{rightIcon}</span>}
      </button>
    )
  }
)

EditorButton.displayName = 'EditorButton'

/**
 * Button Group Component
 * Groups multiple buttons together with proper spacing
 */
export interface EditorButtonGroupProps {
  children: React.ReactNode
  className?: string
  orientation?: 'horizontal' | 'vertical'
}

export function EditorButtonGroup({
  children,
  className,
  orientation = 'horizontal',
}: EditorButtonGroupProps) {
  return (
    <div
      className={cn(
        'inline-flex',
        orientation === 'horizontal' ? 'gap-2' : 'flex-col gap-2',
        className
      )}
      role="group"
    >
      {children}
    </div>
  )
}
