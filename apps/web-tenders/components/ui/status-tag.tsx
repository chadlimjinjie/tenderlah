import * as React from "react"
import { cva } from "class-variance-authority"

import { cn } from "@/lib/utils"

const statusTagVariants = cva(
  "inline-flex h-6 shrink-0 items-center gap-1.5 rounded-full border px-2.5 text-xs font-semibold whitespace-nowrap",
  {
    variants: {
      status: {
        open: "border-success-border bg-success-subtle text-success",
        closing: "border-warning-border bg-warning-subtle text-warning",
        new: "border-info-border bg-info-subtle text-info",
        awarded: "border-brand-border bg-brand-subtle text-brand-active",
        closed: "border-border bg-muted text-muted-foreground",
        cancelled: "border-border bg-muted text-muted-foreground line-through",
      },
    },
    defaultVariants: {
      status: "open",
    },
  }
)

const STATUS_LABELS = {
  open: "Open",
  closing: "Closing soon",
  new: "New",
  awarded: "Awarded",
  closed: "Closed",
  cancelled: "Cancelled",
} as const

type KnownStatus = keyof typeof STATUS_LABELS

function StatusTag({
  status,
  className,
  children,
  ...props
}: React.ComponentProps<"span"> & { status?: string | null }) {
  const key = (status?.toLowerCase() as KnownStatus) ?? "open"
  const variant = STATUS_LABELS[key] ? key : undefined

  return (
    <span
      data-slot="status-tag"
      className={cn(statusTagVariants({ status: variant ?? "closed" }), className)}
      {...props}
    >
      <span className="size-1.5 shrink-0 rounded-full bg-current" />
      {children ?? STATUS_LABELS[key] ?? status ?? "—"}
    </span>
  )
}

export { StatusTag, statusTagVariants }
