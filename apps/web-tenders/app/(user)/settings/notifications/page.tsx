import type { Metadata } from "next"

import { NotificationsCard } from "@/features/settings/components/personal/notifications-card"

export const metadata: Metadata = {
    title: "Notifications",
    description: "Control how and when you are notified.",
}

export default function NotificationsSettingsPage() {
    return (
        <div className="grid gap-4 md:max-w-xl">
            <NotificationsCard />
        </div>
    )
}
