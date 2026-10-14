"use client"

import Link from "next/link"
import { useSearchParams } from "next/navigation"

import { Button } from "@/components/ui/button"
import { AuthCard, AuthField } from "@/features/auth/components/auth-card"
import { authMessage, socialLabel, useLoginOptions, useReturnTo, withReturn } from "@/features/auth/login-ui"
import { ISSUER } from "@/lib/auth/oidc"

const linkClass = "text-foreground underline underline-offset-4"

export function SignInForm() {
  const params = useSearchParams()
  const returnTo = useReturnTo(params.get("return_to"))
  const { data: options } = useLoginOptions()
  return (
    <AuthCard
      title="Sign in"
      description="Enter your email and password to continue."
      action={returnTo && withReturn("sign-in", returnTo)}
      submit="Sign in"
      error={authMessage(params.get("error"))}
      notice={authMessage(params.get("notice"))}
      footer={
        <>
          {options?.socials.map((p) => (
            <Button key={p} asChild variant="outline" className="w-full">
              <a href={withReturn(`social/${p}`, returnTo)}>Continue with {socialLabel(p)}</a>
            </Button>
          ))}
          {options?.sign_up_enabled !== false && (
            <p className="text-muted-foreground text-center text-sm">
              Don&apos;t have an account?{" "}
              <Link href={withReturn("/sign-up", params.get("return_to"))} className={linkClass}>
                Sign up
              </Link>
            </p>
          )}
        </>
      }
    >
      <AuthField label="Email" name="email" type="email" autoComplete="username" placeholder="you@example.com" />
      <AuthField
        label="Password"
        name="password"
        type="password"
        autoComplete="current-password"
        aside={
          <Link href="/forgot-password" className="text-muted-foreground text-sm underline-offset-4 hover:underline">
            Forgot password?
          </Link>
        }
      />
    </AuthCard>
  )
}

export function SignUpForm() {
  const params = useSearchParams()
  const returnTo = useReturnTo(params.get("return_to"))
  const { data: options } = useLoginOptions()
  const min = options?.password_min_length ?? 8
  const disabled = options?.sign_up_enabled === false
  return (
    <AuthCard
      title="Create account"
      description="Sign up to save tenders and get recommendations."
      action={returnTo && !disabled ? withReturn("sign-up", returnTo) : null}
      submit="Create account"
      error={authMessage(disabled ? "signup_disabled" : params.get("error"), min)}
      footer={
        <p className="text-muted-foreground text-center text-sm">
          Already have an account?{" "}
          <Link href={withReturn("/sign-in", params.get("return_to"))} className={linkClass}>
            Sign in
          </Link>
        </p>
      }
    >
      <AuthField label="Name" name="name" autoComplete="name" />
      <AuthField label="Email" name="email" type="email" autoComplete="username" placeholder="you@example.com" />
      <AuthField label="Password" name="password" type="password" autoComplete="new-password" minLength={min} maxLength={256} />
    </AuthCard>
  )
}

export function ForgotPasswordForm() {
  const params = useSearchParams()
  return (
    <AuthCard
      title="Reset password"
      description="We'll email you a link to set a new password."
      action={`${ISSUER}/forgot-password`}
      submit="Send reset link"
      error={authMessage(params.get("error"))}
      notice={authMessage(params.get("notice"))}
      footer={
        <Link href="/sign-in" className={`${linkClass} text-sm`}>
          Back to sign in
        </Link>
      }
    >
      <AuthField label="Email" name="email" type="email" autoComplete="username" placeholder="you@example.com" />
    </AuthCard>
  )
}

export function ResetPasswordForm() {
  const params = useSearchParams()
  const token = params.get("token")
  const { data: options } = useLoginOptions()
  const min = options?.password_min_length ?? 8
  return (
    <AuthCard
      title="Choose a new password"
      action={token && `${ISSUER}/reset-password?${new URLSearchParams({ token })}`}
      submit="Update password"
      error={authMessage(token ? params.get("error") : "link_expired", min)}
      footer={
        <Link href="/forgot-password" className={`${linkClass} text-sm`}>
          Request a new link
        </Link>
      }
    >
      <AuthField label="New password" name="password" type="password" autoComplete="new-password" minLength={min} maxLength={256} />
    </AuthCard>
  )
}
