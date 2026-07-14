"use client"

import { StarIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { useSession } from "@/features/auth/hooks"
import { useToggleTenderFavourite } from "@/features/tenders/hooks/use-tenders"

interface FavouriteButtonProps {
  tenderId: string
  isFavourite?: boolean
}

export function FavouriteButton({ tenderId, isFavourite = false }: FavouriteButtonProps) {
  const { data: user } = useSession()
  const { toggle, isPending } = useToggleTenderFavourite()

  if (!user) return null

  return (
    <Button
      variant="ghost"
      size="icon"
      aria-label={isFavourite ? "Remove from favourites" : "Add to favourites"}
      disabled={isPending}
      onClick={(e) => {
        e.preventDefault()
        e.stopPropagation()
        toggle(tenderId, isFavourite)
      }}
    >
      <StarIcon className={isFavourite ? "fill-current text-yellow-500" : ""} />
    </Button>
  )
}
