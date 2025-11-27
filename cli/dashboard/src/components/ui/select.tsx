import * as React from "react"

export interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  children: React.ReactNode
}

export function Select({ children, className, ...props }: SelectProps) {
  return (
    <select
      className={`h-10 w-full rounded-md px-3 py-2 text-sm bg-input-background text-input-foreground border border-input-border focus:outline-none focus:ring-2 focus:ring-primary focus:border-primary hover:border-border-secondary transition-colors disabled:cursor-not-allowed disabled:opacity-50 [&>option]:bg-input-background [&>option]:text-input-foreground ${className || ''}`}
      {...props}
    >
      {children}
    </select>
  )
}
