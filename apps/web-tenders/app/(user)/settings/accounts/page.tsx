import type { Metadata } from "next"

import { LinkedAccountsCard } from "@/features/settings/components/personal/linked-accounts-card"

export const metadata: Metadata = {
    title: "Accounts",
    description: "Manage your linked third-party accounts.",
}

export default function AccountsSettingsPage() {
    return (
        <div className="grid gap-4 md:max-w-xl">
            <LinkedAccountsCard />
        </div>
    )
}
