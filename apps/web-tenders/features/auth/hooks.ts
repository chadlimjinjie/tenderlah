"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { isAxiosError } from "axios"

import {
  getGetMeQueryKey,
  getMe,
  requestPasswordReset,
  resetPassword,
  sendVerificationEmail,
  signIn,
  signOut,
  signUp,
} from "@/lib/api/generated/auth/auth"
import { axiosInstance, clearAuthToken } from "@/lib/api/axios-instance"
import type {
  AuthUser,
  RequestPasswordResetRequest,
  ResetPasswordRequest,
  SignInRequest,
  SignUpRequest,
} from "@/lib/api/generated/model"

export const sessionKey = getGetMeQueryKey()

/** Extract the human-readable message from an auth-service error response. */
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
    queryFn: async ({ signal }) => {
      try {
        return await getMe(signal)
      } catch (err) {
        if (isAxiosError(err) && err.response?.status === 401) return null
        throw err
      }
    },
    staleTime: 30_000,
  })
}

export function useSignIn() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: SignInRequest) => signIn(data),
    onSuccess: (user) => qc.setQueryData(sessionKey, user),
  })
}

export function useSignUp() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: SignUpRequest) => signUp(data),
    onSuccess: (user) => qc.setQueryData(sessionKey, user),
  })
}

export function useSignOut() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => signOut(),
    onSuccess: () => {
      clearAuthToken()
      qc.setQueryData(sessionKey, null)
    },
  })
}

/**
 * Redeems a one-time code from a cross-site OAuth redirect (see
 * app/(root)/auth/callback) for a session. This is a direct fetch, not part
 * of any redirect chain, so the resulting session cookie isn't subject to
 * the bounce-tracking cookie purging that blocks setting it during the
 * OAuth callback itself. Not part of the generated client — like /api/auth/token,
 * this is internal plumbing with no OpenAPI entry.
 */
export function useExchangeCode() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (code: string) => {
      const { data } = await axiosInstance.post<AuthUser>("/api/auth/exchange", { code })
      return data
    },
    onSuccess: (user) => qc.setQueryData(sessionKey, user),
  })
}

/** Re-send the email-verification link to the signed-in user. */
export function useSendVerificationEmail() {
  return useMutation({
    mutationFn: () => sendVerificationEmail(),
  })
}

/** Request a password-reset email. Always resolves (no account enumeration). */
export function useRequestPasswordReset() {
  return useMutation({
    mutationFn: (data: RequestPasswordResetRequest) => requestPasswordReset(data),
  })
}

/** Reset the password using a token from the emailed link. */
export function useResetPassword() {
  return useMutation({
    mutationFn: (data: ResetPasswordRequest) => resetPassword(data),
  })
}
