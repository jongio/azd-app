/**
 * Enhanced Form Controls with Editor Design System
 * Task 21: Visual Design and Styling
 * 
 * Provides styled form inputs, selects, textareas, and other controls
 */

import * as React from 'react'
import { cn } from '@/lib/utils'

/* ============================================
   Input Component
   ============================================ */

export interface EditorInputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  /** Show error state */
  error?: boolean
  /** Show success state */
  success?: boolean
  /** Icon to display at the start of the input */
  leftIcon?: React.ReactNode
  /** Icon to display at the end of the input */
  rightIcon?: React.ReactNode
  /** Helper text below input */
  helperText?: string
  /** Error message */
  errorMessage?: string
}

export const EditorInput = React.forwardRef<HTMLInputElement, EditorInputProps>(
  (
    {
      className,
      error,
      success,
      leftIcon,
      rightIcon,
      helperText,
      errorMessage,
      ...props
    },
    ref
  ) => {
    const hasError = error || !!errorMessage

    return (
      <div className="w-full">
        <div className="relative">
          {leftIcon && (
            <div className="absolute left-3 top-1/2 -translate-y-1/2 text-editor-fg-muted">
              {leftIcon}
            </div>
          )}
          <input
            ref={ref}
            className={cn(
              'editor-input',
              leftIcon && 'pl-10',
              rightIcon && 'pr-10',
              hasError && 'border-destructive focus:border-destructive',
              success && 'border-success focus:border-success',
              className
            )}
            aria-invalid={hasError}
            aria-describedby={
              errorMessage
                ? `${props.id}-error`
                : helperText
                ? `${props.id}-helper`
                : undefined
            }
            {...props}
          />
          {rightIcon && (
            <div className="absolute right-3 top-1/2 -translate-y-1/2 text-editor-fg-muted">
              {rightIcon}
            </div>
          )}
        </div>
        {errorMessage && (
          <p
            id={`${props.id}-error`}
            className="mt-1 text-xs text-destructive"
            role="alert"
          >
            {errorMessage}
          </p>
        )}
        {!errorMessage && helperText && (
          <p id={`${props.id}-helper`} className="mt-1 text-xs text-editor-fg-tertiary">
            {helperText}
          </p>
        )}
      </div>
    )
  }
)

EditorInput.displayName = 'EditorInput'

/* ============================================
   Textarea Component
   ============================================ */

export interface EditorTextareaProps
  extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  /** Show error state */
  error?: boolean
  /** Helper text below textarea */
  helperText?: string
  /** Error message */
  errorMessage?: string
}

export const EditorTextarea = React.forwardRef<HTMLTextAreaElement, EditorTextareaProps>(
  ({ className, error, helperText, errorMessage, ...props }, ref) => {
    const hasError = error || !!errorMessage

    return (
      <div className="w-full">
        <textarea
          ref={ref}
          className={cn(
            'editor-textarea',
            hasError && 'border-destructive focus:border-destructive',
            className
          )}
          aria-invalid={hasError}
          aria-describedby={
            errorMessage
              ? `${props.id}-error`
              : helperText
              ? `${props.id}-helper`
              : undefined
          }
          {...props}
        />
        {errorMessage && (
          <p
            id={`${props.id}-error`}
            className="mt-1 text-xs text-destructive"
            role="alert"
          >
            {errorMessage}
          </p>
        )}
        {!errorMessage && helperText && (
          <p id={`${props.id}-helper`} className="mt-1 text-xs text-editor-fg-tertiary">
            {helperText}
          </p>
        )}
      </div>
    )
  }
)

EditorTextarea.displayName = 'EditorTextarea'

/* ============================================
   Select Component
   ============================================ */

export interface EditorSelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  /** Show error state */
  error?: boolean
  /** Helper text below select */
  helperText?: string
  /** Error message */
  errorMessage?: string
}

export const EditorSelect = React.forwardRef<HTMLSelectElement, EditorSelectProps>(
  ({ className, error, helperText, errorMessage, children, ...props }, ref) => {
    const hasError = error || !!errorMessage

    return (
      <div className="w-full">
        <select
          ref={ref}
          className={cn(
            'editor-select',
            hasError && 'border-destructive focus:border-destructive',
            className
          )}
          aria-invalid={hasError}
          aria-describedby={
            errorMessage
              ? `${props.id}-error`
              : helperText
              ? `${props.id}-helper`
              : undefined
          }
          {...props}
        >
          {children}
        </select>
        {errorMessage && (
          <p
            id={`${props.id}-error`}
            className="mt-1 text-xs text-destructive"
            role="alert"
          >
            {errorMessage}
          </p>
        )}
        {!errorMessage && helperText && (
          <p id={`${props.id}-helper`} className="mt-1 text-xs text-editor-fg-tertiary">
            {helperText}
          </p>
        )}
      </div>
    )
  }
)

EditorSelect.displayName = 'EditorSelect'

/* ============================================
   Checkbox Component
   ============================================ */

export interface EditorCheckboxProps
  extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'type'> {
  /** Label for the checkbox */
  label?: string
  /** Helper text below checkbox */
  helperText?: string
}

export const EditorCheckbox = React.forwardRef<HTMLInputElement, EditorCheckboxProps>(
  ({ className, label, helperText, id, ...props }, ref) => {
    const checkboxId = id || `checkbox-${Math.random().toString(36).substr(2, 9)}`

    return (
      <div className="flex items-start gap-2">
        <input
          ref={ref}
          type="checkbox"
          id={checkboxId}
          className={cn('editor-checkbox', className)}
          {...props}
        />
        {(label || helperText) && (
          <div className="flex-1">
            {label && (
              <label
                htmlFor={checkboxId}
                className="text-sm font-medium text-editor-fg-primary cursor-pointer"
              >
                {label}
              </label>
            )}
            {helperText && (
              <p className="text-xs text-editor-fg-tertiary mt-0.5">{helperText}</p>
            )}
          </div>
        )}
      </div>
    )
  }
)

EditorCheckbox.displayName = 'EditorCheckbox'

/* ============================================
   Toggle/Switch Component
   ============================================ */

export interface EditorToggleProps {
  /** Whether the toggle is checked */
  checked?: boolean
  /** Callback when toggle state changes */
  onCheckedChange?: (checked: boolean) => void
  /** Whether the toggle is disabled */
  disabled?: boolean
  /** Label for the toggle */
  label?: string
  /** Helper text below toggle */
  helperText?: string
  /** Custom className */
  className?: string
  /** ID for accessibility */
  id?: string
}

export const EditorToggle = React.forwardRef<HTMLButtonElement, EditorToggleProps>(
  (
    { checked = false, onCheckedChange, disabled, label, helperText, className, id },
    ref
  ) => {
    const toggleId = id || `toggle-${Math.random().toString(36).substr(2, 9)}`

    const handleClick = () => {
      if (!disabled && onCheckedChange) {
        onCheckedChange(!checked)
      }
    }

    return (
      <div className={cn('flex items-start gap-3', className)}>
        <button
          ref={ref}
          type="button"
          role="switch"
          aria-checked={checked}
          aria-labelledby={label ? `${toggleId}-label` : undefined}
          onClick={handleClick}
          disabled={disabled}
          className={cn(
            'editor-toggle',
            disabled && 'opacity-50 cursor-not-allowed'
          )}
          data-state={checked ? 'checked' : 'unchecked'}
        >
          <span className="editor-toggle-thumb" />
        </button>
        {(label || helperText) && (
          <div className="flex-1">
            {label && (
              <label
                id={`${toggleId}-label`}
                htmlFor={toggleId}
                className="text-sm font-medium text-editor-fg-primary cursor-pointer"
                onClick={handleClick}
              >
                {label}
              </label>
            )}
            {helperText && (
              <p className="text-xs text-editor-fg-tertiary mt-0.5">{helperText}</p>
            )}
          </div>
        )}
      </div>
    )
  }
)

EditorToggle.displayName = 'EditorToggle'

/* ============================================
   Label Component
   ============================================ */

export interface EditorLabelProps extends React.LabelHTMLAttributes<HTMLLabelElement> {
  /** Whether the field is required */
  required?: boolean
}

export const EditorLabel = React.forwardRef<HTMLLabelElement, EditorLabelProps>(
  ({ className, required, children, ...props }, ref) => {
    return (
      <label
        ref={ref}
        className={cn(
          'block text-sm font-medium text-editor-fg-primary mb-1.5',
          className
        )}
        {...props}
      >
        {children}
        {required && <span className="text-destructive ml-1" aria-label="required">*</span>}
      </label>
    )
  }
)

EditorLabel.displayName = 'EditorLabel'

/* ============================================
   Form Field Component (combines label + input)
   ============================================ */

export interface EditorFormFieldProps {
  /** Field label */
  label?: string
  /** Whether the field is required */
  required?: boolean
  /** Helper text below input */
  helperText?: string
  /** Error message */
  errorMessage?: string
  /** Field ID */
  id?: string
  /** Custom className for the container */
  className?: string
  /** The form control (input, select, textarea, etc.) */
  children: React.ReactNode
}

export function EditorFormField({
  label,
  required,
  helperText,
  errorMessage,
  id,
  className,
  children,
}: EditorFormFieldProps) {
  const fieldId = id || `field-${Math.random().toString(36).substr(2, 9)}`

  return (
    <div className={cn('editor-space-y-sm', className)}>
      {label && (
        <EditorLabel htmlFor={fieldId} required={required}>
          {label}
        </EditorLabel>
      )}
      {React.Children.map(children, (child) => {
        if (React.isValidElement(child)) {
          return React.cloneElement(child, {
            id: fieldId,
            'aria-describedby': errorMessage
              ? `${fieldId}-error`
              : helperText
              ? `${fieldId}-helper`
              : undefined,
            'aria-invalid': !!errorMessage,
            ...(child.props || {}),
          } as React.HTMLAttributes<HTMLElement>)
        }
        return child
      })}
      {errorMessage && (
        <p id={`${fieldId}-error`} className="text-xs text-destructive" role="alert">
          {errorMessage}
        </p>
      )}
      {!errorMessage && helperText && (
        <p id={`${fieldId}-helper`} className="text-xs text-editor-fg-tertiary">
          {helperText}
        </p>
      )}
    </div>
  )
}
