import { Suspense } from "react"

import { ForgotPasswordForm } from "@/features/auth/components/auth-forms"

export default function ForgotPasswordPage() {
  return (
    <Suspense>
      <ForgotPasswordForm />
    </Suspense>
  )
}
