"use client"

import { useEffect } from "react"

import { signIn, signUp } from "@/lib/auth/oidc"

/** Sends the browser to the auth-service's hosted sign-in or sign-up page. */
export function AuthRedirect({ to }: { to: "sign-in" | "sign-up" }) {
  useEffect(() => {
    void (to === "sign-up" ? signUp() : signIn())
  }, [to])
  return null
}
