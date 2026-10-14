import { Suspense } from "react"

import { ResetPasswordForm } from "@/features/auth/components/auth-forms"

export default function ResetPasswordPage() {
  return (
    <Suspense>
      <ResetPasswordForm />
    </Suspense>
  )
}
