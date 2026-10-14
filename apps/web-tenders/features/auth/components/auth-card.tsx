"use client"

import { useEffect, useState } from "react"

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

/**
 * A native form post to the auth-service (not fetch): the navigation is first-party to it,
 * so it can set its session cookie, then it redirects on to authorize or back here with a code.
 */
export function AuthCard({
  title,
  description,
  action,
  submit,
  error,
  notice,
  children,
  footer,
}: {
  title: string
  description?: string
  /** Absolute auth-service URL; null while it is still being prepared. */
  action: string | null
  submit: string
  error?: string | null
  notice?: string | null
  children?: React.ReactNode
  footer?: React.ReactNode
}) {
  const [pending, setPending] = useState(false)
  // coming back with the browser's Back button restores this page from cache, still pending
  useEffect(() => {
    const reset = () => setPending(false)
    window.addEventListener("pageshow", reset)
    return () => window.removeEventListener("pageshow", reset)
  }, [])
  return (
    <div className="flex flex-1 flex-col items-center justify-center p-8">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>{title}</CardTitle>
          {description && <CardDescription>{description}</CardDescription>}
        </CardHeader>
        <form method="post" action={action ?? undefined} onSubmit={() => setPending(true)}>
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
            <Button type="submit" className="w-full" disabled={!action || pending}>
              {pending ? "Please wait…" : submit}
            </Button>
            {footer}
          </CardFooter>
        </form>
      </Card>
    </div>
  )
}

export function AuthField({
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
