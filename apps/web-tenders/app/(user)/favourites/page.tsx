import { Suspense } from "react"

import FavouritesPage from "@/features/favourites/components/favourites-page"

export default function Page() {
  return (
    <div className="flex flex-1 flex-col">
      <div className="@container/main flex flex-1 flex-col gap-2">
        <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
          <Suspense>
            <FavouritesPage />
          </Suspense>
        </div>
      </div>
    </div>
  )
}
