"use client"

import { useEffect, useState } from "react"
import { useQuery } from "@tanstack/react-query"

import { authorizePath, ISSUER } from "@/lib/auth/oidc"

interface LoginOptions {
  sign_up_enabled: boolean
  password_min_length: number
  socials: string[]
}

/** What the auth-service tenant allows: sign-up, password length, social providers. */
export function useLoginOptions() {
  return useQuery<LoginOptions>({
    queryKey: ["login-options"],
    queryFn: async () => {
      const res = await fetch(`${ISSUER}/login-options`)
      if (!res.ok) throw new Error("Could not load sign-in options")
      return res.json()
    },
    staleTime: 5 * 60_000,
  })
}

/**
 * The authorize path the auth-service continues to after a successful form post. It arrives in the URL
 * when the auth-service sent the user here; a user who opened the page directly gets a fresh one.
 */
export function useReturnTo(fromUrl: string | null) {
  const [fresh, setFresh] = useState<string | null>(null)
  useEffect(() => {
    if (!fromUrl) void authorizePath("/").then(setFresh)
  }, [fromUrl])
  return fromUrl ?? fresh
}

/** `path` on the auth-service, or on this app when it starts with "/", with return_to carried along. */
export function withReturn(path: string, returnTo: string | null) {
  const url = path.startsWith("/") ? path : `${ISSUER}/${path}`
  return returnTo ? `${url}?${new URLSearchParams({ return_to: returnTo })}` : url
}

// The auth-service sends codes, never text, so a crafted link cannot put its own words on these pages.
const messages: Record<string, string> = {
  invalid_credentials: "Invalid email or password.",
  invalid_email: "Enter a valid email address.",
  signup_disabled: "Sign-up is currently disabled.",
  link_expired: "This link is invalid or has expired.",
  social_cancelled: "Sign-in was cancelled.",
  social_expired: "Sign-in session expired. Please try again.",
  social_failed: "The provider did not complete the sign-in.",
  social_unverified: "Your account with this provider has no verified email address.",
  no_account: "No account exists for this email address.",
  suspended: "This account is suspended.",
  check_email: "If that address is new, we sent a verification link. You can sign in now.",
  email_verified: "Your email address is verified. You can sign in now.",
  password_updated: "Your password was changed and all other sessions were signed out.",
  reset_sent: "If an account exists for that address, a reset link is on its way.",
}

export function authMessage(code: string | null, minPasswordLength = 8): string | null {
  if (!code) return null
  if (code === "weak_password") return `Password must be between ${minPasswordLength} and 256 characters.`
  return Object.hasOwn(messages, code) ? messages[code] : "Something went wrong. Please try again."
}

const socialLabels: Record<string, string> = { google: "Google", github: "GitHub" }
export const socialLabel = (provider: string) => socialLabels[provider] ?? provider
