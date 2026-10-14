import { Suspense } from "react"

import { SignInForm } from "@/features/auth/components/auth-forms"

export default function SignInPage() {
  return (
    <Suspense>
      <SignInForm />
    </Suspense>
  )
}
