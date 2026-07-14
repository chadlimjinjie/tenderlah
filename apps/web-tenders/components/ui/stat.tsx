import * as React from "react"
import { ArrowDownIcon, ArrowUpIcon, MinusIcon } from "lucide-react"

import { cn } from "@/lib/utils"

const TREND_ICON = {
  up: ArrowUpIcon,
  down: ArrowDownIcon,
  flat: MinusIcon,
} as const

const TREND_CLASS = {
  up: "text-success",
  down: "text-destructive",
  flat: "text-muted-foreground",
} as const

function Stat({
  label,
  value,
  delta,
  trend = "up",
  sub,
  icon,
  className,
  ...props
}: React.ComponentProps<"div"> & {
  label: React.ReactNode
  value: React.ReactNode
  delta?: React.ReactNode
  trend?: "up" | "down" | "flat"
  sub?: React.ReactNode
  icon?: React.ReactNode
}) {
  const TrendIcon = TREND_ICON[trend]

  return (
    <div data-slot="stat" className={cn("flex flex-col gap-1.5", className)} {...props}>
      <div className="flex items-center gap-2">
        {icon && (
          <span className="flex size-7.5 items-center justify-center rounded-md bg-brand-subtle text-brand-active [&_svg]:size-4">
            {icon}
          </span>
        )}
        <span className="text-sm font-semibold text-muted-foreground">{label}</span>
      </div>
      <div className="text-2xl font-extrabold tracking-tight tabular-nums text-foreground">
        {value}
      </div>
      {(delta || sub) && (
        <div className="flex items-center gap-1.5 text-sm">
          {delta && (
            <span className={cn("inline-flex items-center gap-0.5 font-semibold", TREND_CLASS[trend])}>
              <TrendIcon className="size-3.5" />
              {delta}
            </span>
          )}
          {sub && <span className="text-muted-foreground">{sub}</span>}
        </div>
      )}
    </div>
  )
}

export { Stat }
