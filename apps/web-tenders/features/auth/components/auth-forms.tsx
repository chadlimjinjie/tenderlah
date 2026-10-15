"use client"

import { useState } from "react"
import Link from "next/link"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { resetPasswordUrl, signIn, signUp } from "@/lib/auth/oidc"

const linkClass = "text-foreground underline underline-offset-4"

function AuthCard({
  title,
  description,
  submit,
  onSubmit,
  notice,
  children,
  footer,
}: {
  title: string
  description: string
  submit: string
  /** Resolves when done; a thrown error's message is shown on the card. */
  onSubmit: (form: FormData) => Promise<void>
  notice?: string | null
  children: React.ReactNode
  footer: React.ReactNode
}) {
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setPending(true)
    setError(null)
    try {
      await onSubmit(new FormData(e.currentTarget))
    } catch (err) {
      setError(err instanceof Error ? err.message : "Something went wrong. Please try again.")
      setPending(false)
    }
  }

  return (
    <div className="flex flex-1 flex-col items-center justify-center p-8">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>{title}</CardTitle>
          <CardDescription>{description}</CardDescription>
        </CardHeader>
        <form onSubmit={handleSubmit}>
          <CardContent className="flex flex-col gap-4">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
            {notice && (
              <Alert role="status">
                <AlertDescription>{notice}</AlertDescription>
              </Alert>
            )}
            {children}
          </CardContent>
          <CardFooter className="mt-6 flex flex-col gap-3">
            <Button type="submit" className="w-full" disabled={pending}>
              {pending ? "Please wait…" : submit}
            </Button>
            {footer}
          </CardFooter>
        </form>
      </Card>
    </div>
  )
}

function AuthField({
  label,
  aside,
  ...props
}: React.ComponentProps<typeof Input> & { label: string; name: string; aside?: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <Label htmlFor={props.name}>{label}</Label>
        {aside}
      </div>
      <Input id={props.name} required {...props} />
    </div>
  )
}

const text = (form: FormData, name: string) => String(form.get(name) ?? "")

export function SignInForm({ notice }: { notice?: string | null }) {
  return (
    <AuthCard
      title="Sign in"
      description="Enter your email and password to continue."
      submit="Sign in"
      notice={notice}
      onSubmit={(form) => signIn(text(form, "email"), text(form, "password"))}
      footer={
        <p className="text-muted-foreground text-center text-sm">
          Don&apos;t have an account?{" "}
          <Link href="/sign-up" className={linkClass}>
            Sign up
          </Link>
        </p>
      }
    >
      <AuthField label="Email" name="email" type="email" autoComplete="username" placeholder="you@example.com" />
      <AuthField
        label="Password"
        name="password"
        type="password"
        autoComplete="current-password"
        aside={
          <a href={resetPasswordUrl()} className="text-muted-foreground text-sm underline-offset-4 hover:underline">
            Forgot password?
          </a>
        }
      />
    </AuthCard>
  )
}

export function SignUpForm() {
  const [registered, setRegistered] = useState(false)

  async function submit(form: FormData) {
    const [email, password] = [text(form, "email"), text(form, "password")]
    await signUp(text(form, "name"), email, password)
    // a new account can sign in straight away; an address that was already registered falls through to sign-in
    await signIn(email, password).catch(() => setRegistered(true))
  }

  if (registered) {
    return <SignInForm notice="If that address is new, we sent a verification link. Sign in to continue." />
  }
  return (
    <AuthCard
      title="Create account"
      description="Sign up to save tenders and get recommendations."
      submit="Create account"
      onSubmit={submit}
      footer={
        <p className="text-muted-foreground text-center text-sm">
          Already have an account?{" "}
          <Link href="/sign-in" className={linkClass}>
            Sign in
          </Link>
        </p>
      }
    >
      <AuthField label="Name" name="name" autoComplete="name" />
      <AuthField label="Email" name="email" type="email" autoComplete="username" placeholder="you@example.com" />
      <AuthField label="Password" name="password" type="password" autoComplete="new-password" />
    </AuthCard>
  )
}
