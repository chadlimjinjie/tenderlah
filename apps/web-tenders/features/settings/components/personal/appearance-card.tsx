import { ModeToggle } from "@/components/mode-toggle";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export function AppearanceCard() {
    return (
        <Card>
            <CardHeader>
                <CardTitle>Appearance</CardTitle>
                <CardDescription>Theme and display preferences.</CardDescription>
            </CardHeader>
            <CardContent>
                <div className="flex items-center justify-between gap-4">
                    <div className="text-sm">
                        <div className="font-medium">Theme</div>
                        <div className="text-muted-foreground">Light, dark, or system.</div>
                    </div>
                    <ModeToggle />
                </div>
            </CardContent>
        </Card>
    );
}
