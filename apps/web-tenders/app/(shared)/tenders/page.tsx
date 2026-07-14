import type { Metadata } from "next"
import { Suspense } from "react"

import TendersPage from "@/features/tenders/components/tenders-page"

export const metadata: Metadata = {
  title: "Tenders",
  description: "Browse Singapore tender opportunities.",
}

export default function Page() {
  return (
    <div className="flex min-w-0 flex-1 flex-col">
      <div className="@container/main flex min-w-0 flex-1 flex-col gap-2">
        <div className="flex min-w-0 flex-col gap-4 py-4 md:gap-6 md:py-6">
          <Suspense>
            <TendersPage />
          </Suspense>
        </div>
      </div>
    </div>
  )
}
