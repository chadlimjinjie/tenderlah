"use client"

import { useEffect, useRef, useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { useQueryClient } from "@tanstack/react-query"

import { Button } from "@/components/ui/button"
import { sessionKey } from "@/features/auth/hooks"
import { handleCallback } from "@/lib/auth/oidc"

/** Lands here from the auth-service's authorize redirect and redeems the code for tokens. */
export function AuthCallback() {
  const router = useRouter()
  const qc = useQueryClient()
  const [error, setError] = useState<string | null>(null)
  // the code is single-use; guard against StrictMode's double effect
  const attempted = useRef(false)

  useEffect(() => {
    if (attempted.current) return
    attempted.current = true
    handleCallback(new URLSearchParams(window.location.search))
      .then((returnTo) => {
        qc.invalidateQueries({ queryKey: sessionKey })
        router.replace(returnTo)
      })
      .catch((err: Error) => setError(err.message))
  }, [qc, router])

  if (!error) return null
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-4 p-8 text-center">
      <p className="text-sm text-destructive">{error}</p>
      <Button asChild>
        <Link href="/sign-in">Try again</Link>
      </Button>
    </div>
  )
}
