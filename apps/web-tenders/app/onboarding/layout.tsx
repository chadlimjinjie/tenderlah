import { AuthGuard } from "@/features/auth/components/auth-guard"

export default function OnboardingLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return <AuthGuard>{children}</AuthGuard>
}
