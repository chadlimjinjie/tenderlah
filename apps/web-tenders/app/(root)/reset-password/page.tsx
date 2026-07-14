import type { Metadata } from "next"

import { ResetPasswordForm } from "@/features/auth/components/reset-password-form"

export const metadata: Metadata = {
  title: "Reset password",
  description: "Set a new password for your account.",
}

export default async function ResetPasswordPage({
  searchParams,
}: {
  searchParams: Promise<{ token?: string }>
}) {
  const { token } = await searchParams

  return (
    <div className="flex flex-1 flex-col items-center justify-center p-8">
      <ResetPasswordForm token={token} />
    </div>
  )
}
