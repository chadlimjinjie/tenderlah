import type { Metadata } from "next"

import { RootShell } from "@/components/layouts/root-shell"

const description =
  "TenderLah helps Singapore businesses discover and track B2B tenders and business opportunities from across the market, with analytics, personalised recommendations and saved favourites."

export const metadata: Metadata = {
  title: "Singapore B2B Tender & Business Opportunities",
  description,
  openGraph: {
    title: "TenderLah",
    description,
  },
  twitter: {
    title: "TenderLah",
    description,
  },
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return <RootShell>{children}</RootShell>
}
