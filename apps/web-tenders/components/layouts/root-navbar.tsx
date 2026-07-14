"use client"

import Link from "next/link"

import { Button } from "@/components/ui/button"
import { useSession, useSignOut } from "@/features/auth/hooks"

export function RootNavbar() {
  const { data: user, isPending } = useSession()
  const signOut = useSignOut()

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="mx-auto flex h-14 w-full max-w-6xl items-center gap-6 px-4 lg:px-6">
        <Link href="/" className="flex items-center gap-2 font-semibold">
          <img src="/mark.svg" alt="" className="size-6 rounded-md" />
          <span>
            Tender<span className="text-primary">Lah</span>
          </span>
        </Link>

        <div className="ml-auto flex items-center gap-2">
          {isPending ? null : user ? (
            <>
              <Button asChild variant="ghost" size="sm">
                <Link href="/dashboard">Dashboard</Link>
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => signOut.mutate()}
                disabled={signOut.isPending}
              >
                Sign out
              </Button>
            </>
          ) : (
            <>
              <Button asChild variant="ghost" size="sm">
                <Link href="/sign-in">Sign in</Link>
              </Button>
              <Button asChild size="sm">
                <Link href="/sign-up">Sign up</Link>
              </Button>
            </>
          )}
        </div>
      </div>
    </header>
  )
}
