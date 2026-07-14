import { SettingsSubNav } from "@/features/settings/components/settings-sub-nav"

export default function SettingsLayout({ children }: { children: React.ReactNode }) {
    return (
        <div className="flex flex-1 flex-col">
            <div className="@container/main flex flex-1 flex-col gap-2">
                <div className="flex flex-col gap-6 px-4 py-4 md:py-6 lg:px-6">
                    <div>
                        <h1 className="text-lg font-medium">Settings</h1>
                        <p className="text-sm text-muted-foreground">
                            Manage your preferences and account security.
                        </p>
                    </div>
                    <SettingsSubNav />
                    {children}
                </div>
            </div>
        </div>
    )
}
