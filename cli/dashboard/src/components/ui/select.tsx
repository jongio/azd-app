import * as React from "react"

export interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  children: React.ReactNode
}

export function Select({ children, className, ...props }: SelectProps) {
  return (
    <select
      className={`glass h-10 w-full rounded-md px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-primary hover:border-white/20 transition-all-smooth disabled:cursor-not-allowed disabled:opacity-50 [&>option]:bg-card [&>option]:text-foreground ${className || ''}`}
      {...props}
    >
      {children}
    </select>
  )
}
