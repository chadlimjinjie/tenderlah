import { OAuthCallback } from "@/features/auth/components/oauth-callback"

export default async function OAuthCallbackPage({
  searchParams,
}: {
  searchParams: Promise<{ code?: string }>
}) {
  const { code } = await searchParams

  return <OAuthCallback code={code} />
}
