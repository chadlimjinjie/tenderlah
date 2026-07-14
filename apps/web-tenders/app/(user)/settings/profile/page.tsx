import type { Metadata } from "next"

import { AccountCard } from "@/features/settings/components/personal/account-card"
import { AppearanceCard } from "@/features/settings/components/personal/appearance-card"
import { BusinessProfileCard } from "@/features/settings/components/personal/business-profile-card"

export const metadata: Metadata = {
    title: "Profile",
    description: "Your account and appearance settings.",
}

export default function ProfileSettingsPage() {
    return (
        <div className="grid gap-4 md:grid-cols-2">
            <AccountCard />
            <AppearanceCard />
            <BusinessProfileCard />
        </div>
    )
}
