"use client"

import Link from "next/link"
import { usePathname, useRouter, useSearchParams } from "next/navigation"
import { ColumnDef } from "@tanstack/react-table"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { DataTable } from "@/components/data-table-basic"
import { useSession } from "@/features/auth/hooks"
import { Tender } from "@/lib/api/generated/model"
import { useTenders, useTendersInfinite } from "@/features/tenders/hooks/use-tenders"
import { FavouriteButton } from "@/features/tenders/components/favourite-button"
import {
  daysUntilClosing,
  encodeTenderId,
  fmtDate,
  isClosingUrgent,
} from "@/features/tenders/lib/format"
import {
  OpenState,
  SortOption,
  TendersFilterState,
  TendersToolbar,
  TendersView,
  tendersSortToParams,
} from "@/features/tenders/components/tenders-toolbar"
import { TendersListView } from "@/features/tenders/components/tenders-list-view"

const PAGE_SIZE = 10
const DEFAULT_SORT: SortOption = "publishedDate:desc"

export const tenderColumns: ColumnDef<Tender>[] = [
  {
    id: "select",
    header: ({ table }) => (
      <Checkbox
        checked={table.getIsAllPageRowsSelected()}
        onCheckedChange={(v) => table.toggleAllPageRowsSelected(!!v)}
        aria-label="Select all"
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        checked={row.getIsSelected()}
        onCheckedChange={(v) => row.toggleSelected(!!v)}
        aria-label="Select row"
      />
    ),
    enableHiding: false,
  },
  {
    id: "favourite",
    header: "",
    cell: ({ row }) => (
      <FavouriteButton tenderId={row.original.id} isFavourite={row.original.isFavourite} />
    ),
  },
  {
    accessorKey: "source",
    header: "Source",
    cell: ({ row }) => <Badge variant="outline">{row.original.source.toUpperCase()}</Badge>,
  },
  {
    accessorKey: "title",
    header: "Title",
    cell: ({ row }) => (
      <Link
        href={`/tenders/${encodeTenderId(row.original.id)}`}
        className="max-w-md truncate font-medium underline-offset-4 hover:underline block"
      >
        {row.original.title || "—"}
      </Link>
    ),
  },
  {
    accessorKey: "agency",
    header: "Agency",
    cell: ({ row }) => (
      <p className="max-w-xs truncate text-muted-foreground">
        {row.original.agency || "—"}
      </p>
    ),
  },
  {
    accessorKey: "closingDate",
    header: "Closing",
    cell: ({ row }) => {
      const days = daysUntilClosing(row.original.closingDate)
      const urgent = isClosingUrgent(row.original.closingDate)
      return (
        <div className="flex flex-col gap-1">
          <span className="font-mono tabular-nums">{fmtDate(row.original.closingDate)}</span>
          {days != null && (
            <Badge variant={urgent ? "warning" : "secondary"} className="w-fit">
              {days < 0 ? "Closed" : `${days}d left`}
            </Badge>
          )}
        </div>
      )
    },
  },
]

export default function TendersPage() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const pathname = usePathname()
  const { data: user } = useSession()

  // Signed-in users default to relevance (ranked against their onboarding company
  // profile); signed-out visitors default to latest. An explicit ?sort= always wins.
  const defaultSort: SortOption = user ? "relevance:desc" : DEFAULT_SORT

  const page = Math.max(1, Number(searchParams.get("page")) || 1)
  const filters: TendersFilterState = {
    q: searchParams.get("q") ?? "",
    source: searchParams.get("source") ?? "",
    agency: searchParams.get("agency") ?? "",
    openState: (searchParams.get("openState") as OpenState) ?? "",
    sort: (searchParams.get("sort") as SortOption) || defaultSort,
    view: (searchParams.get("view") as TendersView) || "list",
  }

  const setParams = (next: Partial<TendersFilterState & { page: number }>) => {
    const params = new URLSearchParams(searchParams.toString())
    const merged = { ...filters, page, ...next }
    params.set("page", String(merged.page))
    params.set("q", merged.q)
    params.set("source", merged.source)
    params.set("agency", merged.agency)
    params.set("openState", merged.openState)
    params.set("sort", merged.sort)
    params.set("view", merged.view)
    router.push(`${pathname}?${params.toString()}`)
  }

  const goToPage = (p: number) => setParams({ page: p })
  // Any filter/search/sort change resets back to page 1.
  const onFilterChange = (next: Partial<TendersFilterState>) =>
    setParams({ ...next, page: 1 })

  const { sortBy, sortDir } = tendersSortToParams(filters.sort)
  const commonParams = {
    q: filters.q || undefined,
    source: filters.source || undefined,
    agency: filters.agency || undefined,
    openState: filters.openState || undefined,
    sortBy,
    sortDir,
  }
  const isTable = filters.view === "table"

  const { items: tableItems, total, isLoading: isTableLoading } = useTenders(
    { ...commonParams, page, pageSize: PAGE_SIZE },
    isTable,
  )
  const {
    items: listItems,
    isLoading: isListLoading,
    isLoadingMore,
    hasMore,
    loadMore,
  } = useTendersInfinite(commonParams, !isTable)
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <div
      className={`mx-auto w-full min-w-0 px-4 lg:px-6 transition-[max-width] duration-300 ease-out ${isTable ? "max-w-7xl" : "max-w-5xl"}`}
    >
      <TendersToolbar filters={filters} onChange={onFilterChange} />
      <div key={filters.view} className="animate-in fade-in slide-in-from-bottom-1 duration-300">
        {isTable ? (
          isTableLoading ? (
            <p className="text-sm text-muted-foreground">Loading tenders…</p>
          ) : (
            <DataTable columns={tenderColumns} data={tableItems} />
          )
        ) : isListLoading ? (
          <p className="text-sm text-muted-foreground">Loading tenders…</p>
        ) : (
          <TendersListView
            items={listItems}
            hasMore={hasMore}
            isLoadingMore={isLoadingMore}
            onLoadMore={loadMore}
          />
        )}
      </div>
      {isTable && !isTableLoading && (
        <div className="flex items-center justify-end gap-4 pt-4">
          <p className="text-sm text-muted-foreground">
            Page {page} of {pageCount} ({total} total)
          </p>
          <Button
            variant="outline"
            size="sm"
            onClick={() => goToPage(page - 1)}
            disabled={page <= 1}
          >
            Previous
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => goToPage(page + 1)}
            disabled={page >= pageCount}
          >
            Next
          </Button>
        </div>
      )}
    </div>
  )
}
