"use client"

import { useId } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card"
import { Field, FieldLabel, FieldError } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import { authErrorMessage } from "@/features/auth/hooks"
import { useChangePassword } from "@/features/settings/hooks"

const ChangePasswordSchema = z
    .object({
        currentPassword: z.string().min(1, "Current password is required."),
        newPassword: z.string().min(8, "New password must be at least 8 characters long."),
        confirmNewPassword: z.string().min(1, "Please confirm your new password."),
        revokeOtherSessions: z.boolean(),
    })
    .refine((v) => v.newPassword === v.confirmNewPassword, {
        path: ["confirmNewPassword"],
        message: "New passwords do not match.",
    })
    .refine((v) => v.currentPassword !== v.newPassword, {
        path: ["newPassword"],
        message: "New password must be different from your current password.",
    })

type ChangePasswordValues = z.infer<typeof ChangePasswordSchema>

export function ChangePasswordForm() {
    const revokeOtherSessionsId = useId()
    const changePassword = useChangePassword()

    const {
        register,
        handleSubmit,
        reset,
        setValue,
        watch,
        formState: { errors },
    } = useForm<ChangePasswordValues>({
        resolver: zodResolver(ChangePasswordSchema),
        defaultValues: {
            currentPassword: "",
            newPassword: "",
            confirmNewPassword: "",
            revokeOtherSessions: true,
        },
    })

    const revokeOtherSessions = watch("revokeOtherSessions")

    function onSubmit(values: ChangePasswordValues) {
        changePassword.mutate(
            {
                currentPassword: values.currentPassword,
                newPassword: values.newPassword,
                revokeOtherSessions: values.revokeOtherSessions,
            },
            {
                onSuccess: () => {
                    toast.success("Password updated successfully.")
                    reset()
                },
                onError: (err) => toast.error(authErrorMessage(err)),
            }
        )
    }

    return (
        <Card>
            <CardHeader>
                <CardTitle>Change password</CardTitle>
                <CardDescription>Update your account password.</CardDescription>
            </CardHeader>
            <CardContent>
                <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4">
                    <Field>
                        <FieldLabel htmlFor="currentPassword">Current password</FieldLabel>
                        <Input
                            id="currentPassword"
                            type="password"
                            autoComplete="current-password"
                            {...register("currentPassword")}
                        />
                        {errors.currentPassword && (
                            <FieldError>{errors.currentPassword.message}</FieldError>
                        )}
                    </Field>

                    <Field>
                        <FieldLabel htmlFor="newPassword">New password</FieldLabel>
                        <Input
                            id="newPassword"
                            type="password"
                            autoComplete="new-password"
                            {...register("newPassword")}
                        />
                        {errors.newPassword ? (
                            <FieldError>{errors.newPassword.message}</FieldError>
                        ) : (
                            <p className="text-sm text-muted-foreground">Must be at least 8 characters.</p>
                        )}
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

                    <div className="flex items-start gap-3">
                        <Checkbox
                            id={revokeOtherSessionsId}
                            checked={revokeOtherSessions}
                            onCheckedChange={(checked) => setValue("revokeOtherSessions", Boolean(checked))}
                        />
                        <div className="grid gap-1 leading-none">
                            <FieldLabel htmlFor={revokeOtherSessionsId} className="cursor-pointer select-none">
                                Sign out other devices
                            </FieldLabel>
                            <p className="text-sm text-muted-foreground">
                                Recommended. This will revoke other active sessions after changing your password.
                            </p>
                        </div>
                    </div>

                    <div>
                        <Button type="submit" disabled={changePassword.isPending}>
                            {changePassword.isPending ? "Updating…" : "Update password"}
                        </Button>
                    </div>
                </form>
            </CardContent>
        </Card>
    )
}
