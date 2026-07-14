import type { Metadata } from "next"

import { ChangePasswordForm } from "@/features/settings/components/shared/change-password-form"
import { SessionsCard } from "@/features/settings/components/personal/sessions-card"

export const metadata: Metadata = {
    title: "Security",
    description: "Manage your password and sessions.",
}

export default function SecuritySettingsPage() {
    return (
        <div className="grid gap-4 md:grid-cols-2">
            <ChangePasswordForm />
            <SessionsCard />
        </div>
    )
}
