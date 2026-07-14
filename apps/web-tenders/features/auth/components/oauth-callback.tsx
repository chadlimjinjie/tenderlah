"use client"

import { useEffect, useRef } from "react"
import { useRouter } from "next/navigation"

import { authErrorMessage, useExchangeCode } from "@/features/auth/hooks"

/**
 * Lands here after a cross-site OAuth redirect (see auth-service's
 * oauthCallback, which can't set the session cookie itself for tenderlah.sg
 * — see useExchangeCode). Redeems the one-time code via a direct fetch, then
 * continues to the dashboard.
 */
export function OAuthCallback({ code }: { code?: string }) {
  const router = useRouter()
  const exchangeCode = useExchangeCode()
  const attempted = useRef(false)

  useEffect(() => {
    if (attempted.current) return
    attempted.current = true

    if (!code) {
      router.replace("/sign-in?error=" + encodeURIComponent("Invalid sign-in response"))
      return
    }
    exchangeCode.mutate(code, {
      onSuccess: () => router.replace("/"),
      onError: (err) =>
        router.replace("/sign-in?error=" + encodeURIComponent(authErrorMessage(err))),
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [code])

  return null
}
