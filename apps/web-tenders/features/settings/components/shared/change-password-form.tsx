import { Button } from "@/components/ui/button"
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card"
import { resetPasswordUrl } from "@/lib/auth/oidc"

/** Passwords are managed by the auth-service; changing one goes through its emailed reset link. */
export function ChangePasswordForm() {
    return (
        <Card>
            <CardHeader>
                <CardTitle>Change password</CardTitle>
                <CardDescription>We&apos;ll email you a link to set a new password.</CardDescription>
            </CardHeader>
            <CardContent>
                <Button asChild variant="outline">
                    <a href={resetPasswordUrl()}>Reset password</a>
                </Button>
            </CardContent>
        </Card>
    )
}
