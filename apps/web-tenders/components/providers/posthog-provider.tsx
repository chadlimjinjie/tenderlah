"use client"

import { useEffect, useState, Suspense } from "react"
import { usePathname, useSearchParams } from "next/navigation"
import posthog from "posthog-js"
import { PostHogProvider as PHProvider } from "posthog-js/react"
import {
  CookieConsentBanner,
  getStoredConsent,
  type Consent
} from "@/components/cookie-consent-banner"

function PostHogPageView() {
  const pathname = usePathname()
  const searchParams = useSearchParams()

  useEffect(() => {
    if (pathname) {
      let url = window.origin + pathname
      const search = searchParams.toString()
      if (search) url += "?" + search
      posthog.capture("$pageview", { $current_url: url })
    }
  }, [pathname, searchParams])

  return null
}

export function PostHogProvider({ children }: { children: React.ReactNode }) {
  const [consent, setConsent] = useState<Consent | null>(null)
  const [ready, setReady] = useState(false)

  useEffect(() => {
    setConsent(getStoredConsent())
    setReady(true)
  }, [])

  useEffect(() => {
    if (process.env.NODE_ENV !== "production") return // ponytail: no analytics in dev
    if (consent !== "granted" || posthog.__loaded) return // gate on consent (GDPR/ePrivacy H1)
    posthog.init(process.env.NEXT_PUBLIC_POSTHOG_KEY!, {
      api_host: process.env.NEXT_PUBLIC_POSTHOG_HOST,
      person_profiles: "identified_only",
      capture_pageview: true
    })
  }, [consent])

  return (
    <PHProvider client={posthog}>
      {consent === "granted" && (
        <Suspense fallback={null}>
          <PostHogPageView />
        </Suspense>
      )}
      {children}
      {ready && consent === null && process.env.NODE_ENV === "production" && (
        <CookieConsentBanner onChoice={setConsent} />
      )}
    </PHProvider>
  )
}
