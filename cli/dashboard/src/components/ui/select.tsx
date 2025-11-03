import * as React from "react"

export interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  children: React.ReactNode
}

export function Select({ children, className, ...props }: SelectProps) {
  return (
    <select
      className={`h-10 w-full rounded-md px-3 py-2 text-sm bg-[#0d0d0d] text-foreground border border-[#2a2a2a] focus:outline-none focus:ring-2 focus:ring-primary focus:border-primary hover:border-[#3a3a3a] transition-colors disabled:cursor-not-allowed disabled:opacity-50 [&>option]:bg-[#0d0d0d] [&>option]:text-foreground ${className || ''}`}
      {...props}
    >
      {children}
    </select>
  )
}
