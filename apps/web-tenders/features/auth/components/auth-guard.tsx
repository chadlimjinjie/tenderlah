"use client"

import { useEffect } from "react"
import { usePathname, useRouter } from "next/navigation"

import { useSession } from "@/features/auth/hooks"
import { useOnboardingStatus } from "@/features/onboarding/hooks"

/**
 * Client-side guard for the authenticated app shell. Auth is cookie-based and
 * validated through the gateway (`getMe`), so there is no server session to
 * check — we gate rendering on the `useSession` query, redirect signed-out
 * users to `/sign-in`, and redirect signed-in users who haven't completed the
 * onboarding profile (tracked separately by api-server-go) to `/onboarding`.
 */
export function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const pathname = usePathname()
  const { data: user, isPending: userPending } = useSession()
  const { data: onboarding, isPending: onboardingPending } = useOnboardingStatus()
  const needsOnboarding = !!user && !onboardingPending && onboarding === null && pathname !== "/onboarding"

  useEffect(() => {
    if (userPending) return
    if (!user) {
      router.replace("/sign-in")
    } else if (needsOnboarding) {
      router.replace("/onboarding")
    }
  }, [userPending, user, needsOnboarding, router])

  // While the session/onboarding status resolves, or when redirecting, render
  // nothing to avoid flashing protected content.
  if (userPending || !user || (onboardingPending && pathname !== "/onboarding") || needsOnboarding) {
    return null
  }

  return <>{children}</>
}
