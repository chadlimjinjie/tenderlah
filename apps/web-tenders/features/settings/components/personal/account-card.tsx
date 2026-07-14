"use client"

import { useEffect } from "react"
import { CheckCircle2, MailWarning } from "lucide-react"
import { toast } from "sonner"

import { authErrorMessage, useSendVerificationEmail, useSession } from "@/features/auth/hooks"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

export function AccountCard() {
    const { data: user } = useSession()
    const sendVerification = useSendVerificationEmail()

    // The verify-email link bounces back here with ?verified=1|0. Read it from
    // the URL directly (rather than useSearchParams) to avoid a Suspense
    // boundary requirement at build time, then strip it so it fires once.
    useEffect(() => {
        const params = new URLSearchParams(window.location.search)
        const verified = params.get("verified")
        if (verified !== "1" && verified !== "0") return
        if (verified === "1") {
            toast.success("Email verified.")
        } else {
            toast.error("That verification link is invalid or has expired.")
        }
        params.delete("verified")
        const query = params.toString()
        window.history.replaceState(null, "", window.location.pathname + (query ? `?${query}` : ""))
    }, [])

    function handleResend() {
        sendVerification.mutate(undefined, {
            onSuccess: () => toast.success("Verification email sent. Check your inbox."),
            onError: (err) => toast.error(authErrorMessage(err)),
        })
    }

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
                    <div className="space-y-2 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-sm text-amber-700 dark:text-amber-300">
                        <div className="flex items-center gap-2">
                            <MailWarning className="h-4 w-4 shrink-0" />
                            <span>Your email address has not been verified.</span>
                        </div>
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={handleResend}
                            disabled={sendVerification.isPending}
                        >
                            {sendVerification.isPending ? "Sending…" : "Resend verification email"}
                        </Button>
                    </div>
                )}
            </CardContent>
        </Card>
    )
}
