"use client"

import Link from "next/link"
import { Button } from "@/components/ui/button"

const KEY = "ph_consent"
export type Consent = "granted" | "denied"

export function getStoredConsent(): Consent | null {
  if (typeof window === "undefined") return null
  const v = localStorage.getItem(KEY)
  return v === "granted" || v === "denied" ? v : null
}

export function CookieConsentBanner({ onChoice }: { onChoice: (c: Consent) => void }) {
  const choose = (c: Consent) => {
    localStorage.setItem(KEY, c)
    onChoice(c)
  }
  return (
    <div className="fixed inset-x-0 bottom-0 z-50 flex flex-col gap-3 border-t bg-background p-4 sm:flex-row sm:items-center sm:justify-between">
      <p className="text-sm text-muted-foreground">
        We use analytics cookies to understand how the site is used. They load only if you accept.{" "}
        <Link href="/privacy" className="underline">
          Learn more
        </Link>
        .
      </p>
      <div className="flex shrink-0 gap-2">
        <Button variant="outline" size="sm" onClick={() => choose("denied")}>
          Decline
        </Button>
        <Button size="sm" onClick={() => choose("granted")}>
          Accept
        </Button>
      </div>
    </div>
  )
}
