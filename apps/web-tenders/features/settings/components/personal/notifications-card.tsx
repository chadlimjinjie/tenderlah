import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

export function NotificationsCard() {
    return (
        <Card>
            <CardHeader>
                <CardTitle>Notifications</CardTitle>
                <CardDescription>Control how and when you are notified.</CardDescription>
            </CardHeader>
            <CardContent className="text-sm text-muted-foreground">
                No notification settings yet.
            </CardContent>
        </Card>
    )
}
