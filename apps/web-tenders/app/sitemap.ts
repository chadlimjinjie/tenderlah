import type { MetadataRoute } from "next"

import { encodeTenderId } from "@/features/tenders/lib/format"
import { GATEWAY_URL, SITE_URL } from "@/lib/site"

export const revalidate = 3600

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const staticRoutes: MetadataRoute.Sitemap = [
    { url: `${SITE_URL}/`, changeFrequency: "daily", priority: 1 },
    { url: `${SITE_URL}/tenders`, changeFrequency: "hourly", priority: 0.9 },
  ]

  // ponytail: bounded to the 1000 most-recently-published tenders; split into a
  // sitemap index (app/sitemap/[id].ts) if it grows past ~50k URLs.
  try {
    const res = await fetch(
      `${GATEWAY_URL}/api/tenders?page=1&pageSize=1000&sortBy=publishedDate&sortDir=desc`,
      { next: { revalidate } },
    )
    if (!res.ok) return staticRoutes
    const { items = [] } = (await res.json()) as {
      items?: { id: string; lastSeenAt?: string; publishedDate?: string | null }[]
    }
    const tenderRoutes: MetadataRoute.Sitemap = items.map((t) => ({
      url: `${SITE_URL}/tenders/${encodeTenderId(t.id)}`,
      lastModified: t.lastSeenAt ?? t.publishedDate ?? undefined,
      changeFrequency: "weekly",
      priority: 0.7,
    }))
    return [...staticRoutes, ...tenderRoutes]
  } catch {
    return staticRoutes
  }
}
