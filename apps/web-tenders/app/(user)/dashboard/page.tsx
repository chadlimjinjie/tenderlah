"use client"

import Link from "next/link"
import { BookmarkIcon, FileTextIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Stat } from "@/components/ui/stat"
import { useTenders } from "@/features/tenders/hooks/use-tenders"
import { isClosingUrgent } from "@/features/tenders/lib/format"

export default function Page() {
  const { items: saved, total: savedCount } = useTenders({
    favouritesOnly: true,
    sortBy: "closingDate",
    sortDir: "asc",
    page: 1,
    pageSize: 50,
  })
  const closingSoon = saved.filter((t) => isClosingUrgent(t.closingDate))

  return (
    <div className="flex flex-1 flex-col">
      <div className="@container/main flex flex-1 flex-col gap-6">
        <div className="flex flex-col gap-6 px-4 py-4 md:py-6 lg:px-6">

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Stat
              icon={<BookmarkIcon />}
              label="Saved tenders"
              value={savedCount}
              sub={
                <Link href="/favourites" className="underline-offset-4 hover:underline">
                  View all
                </Link>
              }
            />
            <Stat icon={<FileTextIcon />} label="Closing within 7 days" value={closingSoon.length} />
          </div>

        </div>
      </div>
    </div>
  )
}
