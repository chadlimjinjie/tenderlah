import Link from "next/link"
import { useEffect, useRef } from "react"

import { Badge } from "@/components/ui/badge"
import { Card } from "@/components/ui/card"
import { Tender } from "@/lib/api/generated/model"
import { FavouriteButton } from "@/features/tenders/components/favourite-button"
import {
  daysUntilClosing,
  encodeTenderId,
  fmtDate,
  isClosingUrgent,
} from "@/features/tenders/lib/format"

interface TendersListViewProps {
  items: Tender[]
  hasMore: boolean
  isLoadingMore: boolean
  onLoadMore: () => void
}

export function TendersListView({
  items,
  hasMore,
  isLoadingMore,
  onLoadMore,
}: TendersListViewProps) {
  const sentinelRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const sentinel = sentinelRef.current
    if (!sentinel || !hasMore) return
    const observer = new IntersectionObserver((entries) => {
      if (entries[0].isIntersecting) onLoadMore()
    })
    observer.observe(sentinel)
    return () => observer.disconnect()
  }, [hasMore, onLoadMore])

  if (items.length === 0) {
    return (
      <p className="py-12 text-center text-sm text-muted-foreground">
        No tenders match these filters yet. Loosen a filter or save this search to get alerted when one appears.
      </p>
    )
  }

  return (
    <>
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-3">
        {items.map((tender, i) => {
          const days = daysUntilClosing(tender.closingDate)
          const urgent = isClosingUrgent(tender.closingDate)
          const daysLabel = days == null ? null : days < 0 ? "Closed" : `${days}d left`

          return (
            <Card
              key={tender.id}
              className={`min-w-0 flex-col gap-3 border-l-[3px] p-4 duration-400 animate-in fade-in slide-in-from-bottom-1 fill-mode-both motion-reduce:animate-none sm:flex-row sm:items-start sm:gap-4 ${urgent ? "border-l-warning" : "border-l-transparent"
                }`}
              style={{ animationDelay: `${(i % 10) * 50}ms` }}
            >
              <div className="min-w-0 flex-1">
                <div className="flex min-w-0 flex-wrap items-center gap-2">
                  <Badge variant="outline" className="h-auto max-w-full whitespace-normal break-words text-center">
                    {tender.agency || tender.source.toUpperCase()}
                  </Badge>
                  <span className="min-w-0 truncate text-xs text-muted-foreground">{tender.category}</span>
                </div>
                <p className="mt-2 line-clamp-2 break-words text-base font-medium">
                  <Link
                    href={`/tenders/${encodeTenderId(tender.id)}`}
                    className="underline-offset-4 hover:underline"
                  >
                    {tender.title || "—"}
                  </Link>
                </p>
                <p className="mt-1.5 break-all font-mono text-xs tabular-nums text-muted-foreground">
                  {tender.referenceNo || tender.id}
                </p>
                <p className="mt-1.5 break-all font-mono text-xs tabular-nums text-muted-foreground">
                  Closes {fmtDate(tender.closingDate)}
                </p>
              </div>
              <div className="flex flex-row flex-wrap items-center justify-between gap-2 sm:flex-none sm:flex-col sm:items-end">
                <div className="flex items-center gap-1.5">
                  {daysLabel && (
                    <Badge variant={urgent ? "warning" : "secondary"}>{daysLabel}</Badge>
                  )}
                  <FavouriteButton tenderId={tender.id} isFavourite={tender.isFavourite} />
                </div>
              </div>
            </Card>
          )
        })}
      </div>
      {hasMore && (
        <div ref={sentinelRef} className="py-6 text-center text-sm text-muted-foreground">
          {isLoadingMore ? "Loading more…" : ""}
        </div>
      )}
    </>
  )
}
