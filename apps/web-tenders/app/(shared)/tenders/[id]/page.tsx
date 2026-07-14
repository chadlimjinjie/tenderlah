import type { Metadata } from "next"

import TenderViewPage from "@/features/tenders/components/tender-view-page"
import { GATEWAY_URL } from "@/lib/site"

export const revalidate = 3600

export async function generateMetadata({
  params,
}: {
  params: Promise<{ id: string }>
}): Promise<Metadata> {
  const { id } = await params
  try {
    // Endpoint is public; id is base64url (backend decodeTenderID handles it).
    const res = await fetch(`${GATEWAY_URL}/api/tenders/${id}`, { next: { revalidate } })
    if (!res.ok) return { title: "Tender" }
    const tender = await res.json()
    const title: string = tender.title || "Tender"
    const raw: string = tender.description || [tender.agency, tender.category].filter(Boolean).join(" · ")
    const description = raw.length > 160 ? `${raw.slice(0, 157)}…` : raw
    return {
      title,
      description: description || undefined,
      alternates: { canonical: `/tenders/${id}` },
      openGraph: { title, description: description || undefined },
      twitter: { title, description: description || undefined },
    }
  } catch {
    return { title: "Tender" }
  }
}

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params
  return (
    <div className="flex flex-1 flex-col">
      <div className="@container/main flex flex-1 flex-col gap-2">
        <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
          <TenderViewPage id={id} />
        </div>
      </div>
    </div>
  )
}
