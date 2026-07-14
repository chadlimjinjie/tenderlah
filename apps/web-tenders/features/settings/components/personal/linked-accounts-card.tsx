"use client"

import * as React from "react"
import { toast } from "sonner"
import { SiGoogle } from "@icons-pack/react-simple-icons"
import { Unlink, AlertCircle, Linkedin } from "lucide-react"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Badge } from "@/components/ui/badge"
import {
    AlertDialog,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { authErrorMessage } from "@/features/auth/hooks"
import { useLinkedAccounts, useUnlinkAccount } from "@/features/settings/hooks"
import type { LinkedAccount } from "@/lib/api/generated/model"

const PROVIDER_META: Record<string, { label: string; icon: React.ReactNode }> = {
    google: { label: "Google", icon: <SiGoogle className="h-4 w-4" /> },
    linkedin: { label: "LinkedIn", icon: <Linkedin className="h-4 w-4" /> },
}

function ProviderIcon({ providerId }: { providerId: string }) {
    const meta = PROVIDER_META[providerId]
    if (meta) return <>{meta.icon}</>
    return <AlertCircle className="h-4 w-4 text-muted-foreground" />
}

function providerLabel(providerId: string) {
    return PROVIDER_META[providerId]?.label ?? providerId
}

export function LinkedAccountsCard() {
    const [unlinkTarget, setUnlinkTarget] = React.useState<LinkedAccount | null>(null)

    const { data: accounts, isPending, isError } = useLinkedAccounts()
    const unlinkMutation = useUnlinkAccount()

    const socialAccounts = accounts?.filter((a) => a.providerId !== "credential") ?? []
    const hasPassword = accounts?.some((a) => a.providerId === "credential") ?? false

    function handleUnlink(account: LinkedAccount) {
        unlinkMutation.mutate(account.providerId, {
            onSuccess: () => {
                toast.success(`${providerLabel(account.providerId)} disconnected.`)
                setUnlinkTarget(null)
            },
            onError: (err) => {
                toast.error(authErrorMessage(err))
                setUnlinkTarget(null)
            },
        })
    }

    return (
        <>
            <Card>
                <CardHeader>
                    <CardTitle>Linked accounts</CardTitle>
                    <CardDescription>Third-party accounts connected for sign-in.</CardDescription>
                </CardHeader>
                <CardContent className="space-y-3">
                    {isPending && (
                        <>
                            <Skeleton className="h-12 w-full" />
                            <Skeleton className="h-12 w-full" />
                        </>
                    )}

                    {isError && (
                        <p className="text-sm text-destructive">Failed to load accounts.</p>
                    )}

                    {!isPending && !isError && socialAccounts.length === 0 && (
                        <p className="text-sm text-muted-foreground">No third-party accounts connected.</p>
                    )}

                    {socialAccounts.map((account) => (
                        <div
                            key={account.id}
                            className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2"
                        >
                            <div className="flex items-center gap-2.5">
                                <ProviderIcon providerId={account.providerId} />
                                <div className="text-sm">
                                    <div className="font-medium">{providerLabel(account.providerId)}</div>
                                    <div className="text-xs text-muted-foreground">
                                        Connected {new Date(account.createdAt).toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" })}
                                    </div>
                                </div>
                            </div>
                            <div className="flex items-center gap-2">
                                <Badge variant="secondary" className="text-xs">Connected</Badge>
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-7 w-7 text-muted-foreground hover:text-destructive"
                                    onClick={() => setUnlinkTarget(account)}
                                    disabled={!hasPassword && socialAccounts.length === 1}
                                    title={!hasPassword && socialAccounts.length === 1 ? "Cannot disconnect your only sign-in method" : "Disconnect"}
                                >
                                    <Unlink className="h-3.5 w-3.5" />
                                </Button>
                            </div>
                        </div>
                    ))}
                </CardContent>
            </Card>

            <AlertDialog open={!!unlinkTarget} onOpenChange={(open) => { if (!open) setUnlinkTarget(null) }}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Disconnect {providerLabel(unlinkTarget?.providerId ?? "")}?</AlertDialogTitle>
                        <AlertDialogDescription>
                            You will no longer be able to sign in with {providerLabel(unlinkTarget?.providerId ?? "")}. Make sure you have another sign-in method available.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <Button
                            variant="destructive"
                            onClick={() => unlinkTarget && handleUnlink(unlinkTarget)}
                            disabled={unlinkMutation.isPending}
                        >
                            {unlinkMutation.isPending ? "Disconnecting…" : "Disconnect"}
                        </Button>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </>
    )
}
