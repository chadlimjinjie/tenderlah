"use client"

import { useSession } from "@/features/auth/hooks"
import { UserShell } from "@/components/layouts/user-shell"
import { RootShell } from "@/components/layouts/root-shell"

/**
 * Layout for public `(shared)` pages. When the visitor is authenticated, render
 * the full app chrome (sidebar + header) so the page feels part of the signed-in
 * experience; otherwise fall back to the public `(root)` chrome (top navbar).
 * Unlike `AuthGuard`, signed-out users are never redirected — these pages stay
 * publicly accessible.
 */
export function SharedLayoutClient({ children }: { children: React.ReactNode }) {
  const { data: user } = useSession()

  if (user) {
    return <UserShell>{children}</UserShell>
  }

  return <RootShell>{children}</RootShell>
}
