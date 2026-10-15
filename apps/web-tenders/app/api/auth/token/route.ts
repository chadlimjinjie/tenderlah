import { NextResponse, type NextRequest } from "next/server"

import { exchangeToken, isJson, RT_COOKIE } from "@/lib/auth/service"

// ponytail: mirrors the tenant's refresh_token_ttl_days default; an expired token just fails the refresh.
const RT_MAX_AGE = 14 * 24 * 60 * 60

/** Redeems an authorization code ({ code, code_verifier, redirect_uri }) or, with an empty body, the refresh cookie. */
export async function POST(req: NextRequest) {
  if (!isJson(req)) return NextResponse.json({ error: "Unsupported request" }, { status: 415 })
  const { code, code_verifier, redirect_uri } = await req.json().catch(() => ({}))
  let params: Record<string, string>
  if (typeof code === "string" && typeof code_verifier === "string" && typeof redirect_uri === "string") {
    params = { grant_type: "authorization_code", code, code_verifier, redirect_uri }
  } else {
    const refresh = req.cookies.get(RT_COOKIE)?.value
    if (!refresh) return NextResponse.json({ error: "Signed out" }, { status: 401 })
    params = { grant_type: "refresh_token", refresh_token: refresh }
  }

  const { ok, body } = await exchangeToken(params)
  if (!ok) {
    const res = NextResponse.json({ error: body.error_description ?? body.error ?? "Sign-in failed" }, { status: 401 })
    res.cookies.delete({ name: RT_COOKIE, path: "/api/auth" })
    return res
  }
  const res = NextResponse.json({ access_token: body.access_token, id_token: body.id_token, expires_in: body.expires_in })
  if (body.refresh_token) {
    res.cookies.set(RT_COOKIE, body.refresh_token, {
      httpOnly: true,
      sameSite: "lax",
      secure: req.nextUrl.protocol === "https:",
      path: "/api/auth",
      maxAge: RT_MAX_AGE,
    })
  }
  return res
}
