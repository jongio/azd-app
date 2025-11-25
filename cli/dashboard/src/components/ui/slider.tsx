import * as React from "react"
import { cn } from "@/lib/utils"

export interface SliderProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string
  showValue?: boolean
}

const Slider = React.forwardRef<HTMLInputElement, SliderProps>(
  ({ className, label, showValue = true, ...props }, ref) => {
    return (
      <div className="flex flex-col gap-2">
        {label && (
          <div className="flex justify-between items-center">
            <label className="text-sm font-medium">{label}</label>
            {showValue && <span className="text-sm text-muted-foreground">{props.value}</span>}
          </div>
        )}
        <input
          type="range"
          className={cn(
            "w-full h-2 bg-secondary rounded-lg appearance-none cursor-pointer",
            "[&::-webkit-slider-thumb]:appearance-none",
            "[&::-webkit-slider-thumb]:w-4",
            "[&::-webkit-slider-thumb]:h-4",
            "[&::-webkit-slider-thumb]:rounded-full",
            "[&::-webkit-slider-thumb]:bg-primary",
            "[&::-webkit-slider-thumb]:cursor-pointer",
            "[&::-moz-range-thumb]:w-4",
            "[&::-moz-range-thumb]:h-4",
            "[&::-moz-range-thumb]:rounded-full",
            "[&::-moz-range-thumb]:bg-primary",
            "[&::-moz-range-thumb]:border-0",
            "[&::-moz-range-thumb]:cursor-pointer",
            className
          )}
          ref={ref}
          {...props}
        />
      </div>
    )
  }
)
Slider.displayName = "Slider"

export { Slider }
