"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useRouter } from "next/navigation"
import Link from "next/link"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Field, FieldLabel, FieldError } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { authErrorMessage, useResetPassword } from "@/features/auth/hooks"

const ResetPasswordSchema = z
  .object({
    newPassword: z.string().min(8, "Password must be at least 8 characters long."),
    confirmNewPassword: z.string().min(1, "Please confirm your new password."),
  })
  .refine((v) => v.newPassword === v.confirmNewPassword, {
    path: ["confirmNewPassword"],
    message: "Passwords do not match.",
  })

type ResetPasswordValues = z.infer<typeof ResetPasswordSchema>

export function ResetPasswordForm({ token }: { token?: string }) {
  const router = useRouter()
  const resetPassword = useResetPassword()

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<ResetPasswordValues>({
    resolver: zodResolver(ResetPasswordSchema),
    defaultValues: { newPassword: "", confirmNewPassword: "" },
  })

  function onSubmit(values: ResetPasswordValues) {
    if (!token) return
    resetPassword.mutate(
      { token, newPassword: values.newPassword },
      {
        onSuccess: () => {
          toast.success("Password reset. Please sign in with your new password.")
          router.push("/sign-in")
        },
        onError: (err) => toast.error(authErrorMessage(err)),
      }
    )
  }

  if (!token) {
    return (
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Invalid reset link</CardTitle>
          <CardDescription>
            This password reset link is missing or malformed. Request a new one.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button asChild className="w-full">
            <Link href="/forgot-password">Request a new link</Link>
          </Button>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className="w-full max-w-sm">
      <CardHeader>
        <CardTitle>Set a new password</CardTitle>
        <CardDescription>Choose a new password for your account.</CardDescription>
      </CardHeader>
      <form onSubmit={handleSubmit(onSubmit)}>
        <CardContent className="flex flex-col gap-4">
          <Field>
            <FieldLabel htmlFor="newPassword">New password</FieldLabel>
            <Input
              id="newPassword"
              type="password"
              autoComplete="new-password"
              {...register("newPassword")}
            />
            {errors.newPassword && <FieldError>{errors.newPassword.message}</FieldError>}
          </Field>
          <Field>
            <FieldLabel htmlFor="confirmNewPassword">Confirm new password</FieldLabel>
            <Input
              id="confirmNewPassword"
              type="password"
              autoComplete="new-password"
              {...register("confirmNewPassword")}
            />
            {errors.confirmNewPassword && (
              <FieldError>{errors.confirmNewPassword.message}</FieldError>
            )}
          </Field>
        </CardContent>
        <CardFooter className="mt-6">
          <Button type="submit" className="w-full" disabled={resetPassword.isPending}>
            {resetPassword.isPending ? "Resetting…" : "Reset password"}
          </Button>
        </CardFooter>
      </form>
    </Card>
  )
}
