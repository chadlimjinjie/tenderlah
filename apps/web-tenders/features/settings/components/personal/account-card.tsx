"use client"

import { CheckCircle2, MailWarning } from "lucide-react"

import { useSession } from "@/features/auth/hooks"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

export function AccountCard() {
    const { data: user } = useSession()

    return (
        <Card>
            <CardHeader>
                <CardTitle>Account</CardTitle>
                <CardDescription>Your signed-in account details.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
                <div className="text-sm">
                    <div className="text-muted-foreground">Email</div>
                    <div className="font-medium">{user?.email ?? "—"}</div>
                </div>
                {user?.emailVerified ? (
                    <div className="flex items-center gap-2 text-sm text-emerald-600 dark:text-emerald-400">
                        <CheckCircle2 className="h-4 w-4" />
                        <span>Email verified</span>
                    </div>
                ) : (
                    <div className="flex items-center gap-2 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-sm text-amber-700 dark:text-amber-300">
                        <MailWarning className="h-4 w-4 shrink-0" />
                        <span>Your email address has not been verified. Check your inbox for the verification link.</span>
                    </div>
                )}
            </CardContent>
        </Card>
    )
}
