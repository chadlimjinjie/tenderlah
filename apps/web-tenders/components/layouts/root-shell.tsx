import Link from "next/link"
import { RootNavbar } from "@/components/layouts/root-navbar"

/**
 * Public chrome for signed-out visitors: a top navbar over the page content.
 * Used by the `(root)` layout and by `SharedLayoutClient` when there is no
 * session.
 */
export function RootShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-svh flex-col">
      <RootNavbar />
      <main className="flex flex-1 flex-col">{children}</main>
      <footer className="text-muted-foreground flex flex-wrap items-center justify-center gap-x-4 gap-y-1 border-t p-4 text-center text-xs">
        <Link href="/privacy" className="underline">
          Privacy Policy &amp; Cookie Notice
        </Link>
        <Link href="/contact" className="underline">
          Contact Us
        </Link>
      </footer>
    </div>
  )
}
