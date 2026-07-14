import { AuthGuard } from "@/features/auth/components/auth-guard"
import { UserShell } from "@/components/layouts/user-shell"

export default function UserLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <AuthGuard>
      <UserShell>{children}</UserShell>
    </AuthGuard>
  )
}
