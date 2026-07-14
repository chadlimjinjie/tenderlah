"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { isAxiosError } from "axios"

import { useSession } from "@/features/auth/hooks"
import {
  completeUserOnboarding,
  getGetUserOnboardingQueryKey,
  getUserOnboarding,
  uploadOnboardingLogo,
} from "@/lib/api/generated/users/users"
import type { OnboardingRequest, UserOnboarding } from "@/lib/api/generated/model"

const onboardingKey = getGetUserOnboardingQueryKey()

/** The signed-in user's onboarding profile, or `null` if not yet onboarded. */
export function useOnboardingStatus() {
  const { data: user } = useSession()
  return useQuery<UserOnboarding | null>({
    queryKey: onboardingKey,
    queryFn: async ({ signal }) => {
      try {
        return await getUserOnboarding(signal)
      } catch (err) {
        if (isAxiosError(err) && err.response?.status === 404) return null
        throw err
      }
    },
    enabled: !!user,
    staleTime: 30_000,
  })
}

export function useCompleteOnboarding() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: OnboardingRequest) => completeUserOnboarding(data),
    onSuccess: (data) => qc.setQueryData(onboardingKey, data),
  })
}

export function useUploadOnboardingLogo() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (file: File) => uploadOnboardingLogo({ file }),
    onSuccess: (data) => qc.setQueryData(onboardingKey, data),
  })
}
