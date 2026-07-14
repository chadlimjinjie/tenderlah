import { JetBrains_Mono, Plus_Jakarta_Sans } from "next/font/google"
import type { Metadata, Viewport } from "next"

import "@/app/globals.css"
import { RootProviders } from "@/components/providers/root-providers"
import { Toaster } from "@/components/ui/sonner"
import { cn } from "@/lib/utils"
import { SITE_URL } from "@/lib/site"

const description =
  "TenderLah is Singapore's B2B opportunities platform — discover, track, and get AI-matched tenders and business opportunities from across the market, with analytics and personalised recommendations."

export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  applicationName: "TenderLah",
  title: {
    default: "TenderLah",
    template: "%s | TenderLah",
  },
  description,
  keywords: [
    "Singapore tenders",
    "business opportunities",
    "B2B opportunities",
    "tender discovery",
    "procurement",
    "bids",
  ],
  alternates: { canonical: "/" },
  openGraph: {
    siteName: "TenderLah",
    type: "website",
    description,
  },
  twitter: {
    card: "summary_large_image",
    description,
  },
}

export const viewport: Viewport = {
  themeColor: "#ED2939",
}

const fontSans = Plus_Jakarta_Sans({
  subsets: ["latin"],
  weight: ["400", "500", "600", "700", "800"],
  variable: "--font-sans",
})

const fontMono = JetBrains_Mono({
  subsets: ["latin"],
  weight: ["400", "500", "600"],
  variable: "--font-mono",
})

export default function AppLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html
      lang="en"
      suppressHydrationWarning
      className={cn("antialiased", fontMono.variable, "font-sans", fontSans.variable)}
    >
      <body>
        <RootProviders>{children}</RootProviders>
        <Toaster />
      </body>
    </html>
  )
}
