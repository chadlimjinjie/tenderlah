import Link from "next/link"

import { Button } from "@/components/ui/button"
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card"

/** Passwords are managed by the auth-service; changing one goes through an emailed reset link. */
export function ChangePasswordForm() {
    return (
        <Card>
            <CardHeader>
                <CardTitle>Change password</CardTitle>
                <CardDescription>We&apos;ll email you a link to set a new password.</CardDescription>
            </CardHeader>
            <CardContent>
                <Button asChild variant="outline">
                    <Link href="/forgot-password">Reset password</Link>
                </Button>
            </CardContent>
        </Card>
    )
}
