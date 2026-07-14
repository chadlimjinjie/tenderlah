"use client"

import { useQuery } from "@tanstack/react-query"
import { WrenchIcon } from "lucide-react"

import { useSession } from "@/features/auth/hooks"
import {
  getMaintenanceScope,
  getGetMaintenanceScopeQueryKey,
} from "@/lib/api/generated/maintenance/maintenance"

/** The maintenance scope this frontend reads, toggled from the admin console. */
const SCOPE = "tenders"

const DEFAULT_MESSAGE =
  "We’re currently performing scheduled maintenance. We’ll be back shortly — thanks for your patience."

/**
 * MaintenanceGate blocks the public site with a full-screen notice while the
 * global maintenance flag is on. Admins are exempt (so they can keep working)
 * and instead see a thin banner above the app. The flag is polled so toggling
 * it in the admin console takes effect without a reload.
 *
 * It fails open: while the flag is loading or the request errors, children
 * render normally so a transient backend hiccup never locks everyone out.
 */
export function MaintenanceGate({ children }: { children: React.ReactNode }) {
  const { data: maintenance } = useQuery({
    queryKey: getGetMaintenanceScopeQueryKey(SCOPE),
    queryFn: ({ signal }) => getMaintenanceScope(SCOPE, signal),
    staleTime: 30_000,
    refetchInterval: 60_000,
    refetchOnWindowFocus: true,
  })
  const { data: session } = useSession()

  if (!maintenance?.enabled) return <>{children}</>

  const isAdmin = session?.role === "admin"
  const message = maintenance.message?.trim() || DEFAULT_MESSAGE

  if (isAdmin) {
    return (
      <>
        <div className="sticky top-0 z-50 flex items-center justify-center gap-2 bg-amber-500/90 px-4 py-1.5 text-center text-sm font-medium text-amber-950">
          <WrenchIcon className="size-4 shrink-0" />
          Maintenance mode is ON — visitors see the maintenance screen. You have admin
          access.
        </div>
        {children}
      </>
    )
  }

  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-6 px-6 text-center">
      <div className="flex size-16 items-center justify-center rounded-full bg-muted text-muted-foreground">
        <WrenchIcon className="size-8" />
      </div>
      <div className="flex max-w-md flex-col gap-2">
        <h1 className="text-2xl font-semibold tracking-tight">Down for maintenance</h1>
        <p className="text-muted-foreground">{message}</p>
      </div>
    </main>
  )
}
