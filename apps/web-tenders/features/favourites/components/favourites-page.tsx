"use client"

import { useSearchParams, useRouter, usePathname } from "next/navigation"

import { Button } from "@/components/ui/button"
import { DataTable } from "@/components/data-table-basic"
import { useTenders } from "@/features/tenders/hooks/use-tenders"
import { tenderColumns } from "@/features/tenders/components/tenders-page"

const PAGE_SIZE = 10

export default function FavouritesPage() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const pathname = usePathname()

  const page = Math.max(1, Number(searchParams.get("page")) || 1)
  const goToPage = (p: number) => router.push(`${pathname}?page=${p}`)

  const { items, total, isLoading } = useTenders({
    favouritesOnly: true,
    page,
    pageSize: PAGE_SIZE,
  })
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <div className="px-4 lg:px-6">
      {isLoading ? (
        <p className="text-sm text-muted-foreground">Loading favourites…</p>
      ) : (
        <>
          <DataTable columns={tenderColumns} data={items} />
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
        </>
      )}
    </div>
  )
}
