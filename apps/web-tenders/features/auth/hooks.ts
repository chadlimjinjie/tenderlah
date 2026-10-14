"use client"

import { useQuery } from "@tanstack/react-query"
import { isAxiosError } from "axios"

import { getUser, type AuthUser } from "@/lib/auth/oidc"

export const sessionKey = ["session"]

/** Extract the human-readable message from an API error response. */
export function authErrorMessage(err: unknown): string {
  if (isAxiosError(err)) {
    const data = err.response?.data as { error?: string } | undefined
    return data?.error ?? err.message
  }
  return err instanceof Error ? err.message : "Something went wrong"
}

/** Current authenticated user, or `null` when signed out. */
export function useSession() {
  return useQuery<AuthUser | null>({
    queryKey: sessionKey,
    queryFn: getUser,
    staleTime: 30_000,
  })
}

