// Single source of truth for the public site URL; env override for non-prod, prod fallback baked in.
export const SITE_URL = process.env.NEXT_PUBLIC_SITE_URL ?? "https://tenderlah.sg"

// Server-side gateway base for SEO fetches (sitemap, generateMetadata).
export const GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8080"
