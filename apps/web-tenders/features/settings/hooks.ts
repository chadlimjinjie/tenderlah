"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  changePassword,
  getListAccountsQueryKey,
  listAccounts,
  unlinkAccount,
} from "@/lib/api/generated/auth/auth"
import type { ChangePasswordRequest } from "@/lib/api/generated/model"

export const linkedAccountsKey = getListAccountsQueryKey()

/** The caller's linked accounts (credential + any OAuth providers). */
export function useLinkedAccounts() {
  return useQuery({
    queryKey: linkedAccountsKey,
    queryFn: ({ signal }) => listAccounts(signal),
  })
}

/** Disconnect an OAuth provider, then refresh the linked-accounts list. */
export function useUnlinkAccount() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (provider: string) => unlinkAccount(provider),
    onSuccess: () => qc.invalidateQueries({ queryKey: linkedAccountsKey }),
  })
}

/** Change the authenticated user's password. */
export function useChangePassword() {
  return useMutation({
    mutationFn: (data: ChangePasswordRequest) => changePassword(data),
  })
}
