import { SharedLayoutClient } from "@/components/layouts/shared-layout-client"

export default function SharedLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return <SharedLayoutClient>{children}</SharedLayoutClient>
}
