import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

export function SessionsCard() {
    return (
        <Card>
            <CardHeader>
                <CardTitle>Sessions</CardTitle>
                <CardDescription>Security actions for your account.</CardDescription>
            </CardHeader>
            <CardContent className="text-sm text-muted-foreground">
                Tip: when you change your password, you can choose to sign out
                other devices.
            </CardContent>
        </Card>
    )
}
