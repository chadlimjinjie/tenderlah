import { Suspense } from "react"

import { SignUpForm } from "@/features/auth/components/auth-forms"

export default function SignUpPage() {
  return (
    <Suspense>
      <SignUpForm />
    </Suspense>
  )
}
