// Server-side bridge from our own sign-in/sign-up pages to the auth-service's form endpoints.
// Those endpoints only accept posts carrying the auth-service's own Origin, which a browser on our
// origin can never send, so route handlers make the calls instead and hand back an authorization code.
// ponytail: depends on undocumented auth-service behaviour (pages.rs: 303/401 replies, the
// auth_session cookie, the <p class="error"> markup); switch if it ever gets a JSON login API.

import { CLIENT_ID, ISSUER } from "@/lib/auth/oidc"

const issuer = new URL(ISSUER)
/** The auth-service PUBLIC_URL; its form guard rejects posts with any other Origin. */
const PUBLIC_ORIGIN = process.env.AUTH_PUBLIC_URL ?? issuer.origin
/** Where this server reaches the gateway, when that differs from the browser-facing origin. */
const BASE = (process.env.AUTH_INTERNAL_URL ?? issuer.origin) + issuer.pathname
const AUTH_COOKIE = "auth_session"

/** Our own cookie holding the auth-service session, so sign-out can end it. */
export const SID_COOKIE = "auth_sid"
/** Our httpOnly cookie holding the refresh token, so it survives the tab and stays out of JS. */
export const RT_COOKIE = "auth_rt"

/** The user's browser details, so the auth-service records them on the session instead of ours. */
function headersFor(req: Request, sid?: string): Record<string, string> {
  const h: Record<string, string> = { Origin: PUBLIC_ORIGIN }
  for (const name of ["user-agent", "x-forwarded-for"]) {
    const v = req.headers.get(name)
    if (v) h[name] = v
  }
  if (sid) h.Cookie = `${AUTH_COOKIE}=${sid}`
  return h
}

/** Our route handlers only take JSON, which other sites cannot post without a CORS preflight we never allow. */
export const isJson = (req: Request) => req.headers.get("content-type")?.startsWith("application/json") ?? false

/**
 * Signs in with email and password and runs the authorize request the browser prepared (it holds the
 * PKCE verifier). Returns the callback URL carrying the code, or null for bad credentials.
 */
export async function passwordSignIn(req: Request, email: string, password: string, authorize: string) {
  // only this tenant's authorize endpoint, so the server cannot be pointed anywhere else
  if (!authorize.startsWith(`${issuer.pathname}/authorize?`)) throw new Error("invalid authorize path")
  const res = await fetch(`${BASE}/sign-in?${new URLSearchParams({ return_to: authorize })}`, {
    method: "POST",
    headers: headersFor(req),
    body: new URLSearchParams({ email, password }),
    redirect: "manual",
  })
  if (res.status === 401) return null
  const cookie = res.headers.getSetCookie().find((c) => c.startsWith(`${AUTH_COOKIE}=`))
  if (res.status !== 303 || !cookie) throw new Error(`auth-service sign-in returned ${res.status}`)
  const sid = cookie.slice(AUTH_COOKIE.length + 1).split(";")[0]
  const maxAge = Number(/Max-Age=(\d+)/i.exec(cookie)?.[1] ?? 0)

  const auth = await fetch(BASE + authorize.slice(issuer.pathname.length), {
    headers: headersFor(req, sid),
    redirect: "manual",
  })
  const redirect = auth.headers.get("location")
  if (!redirect) throw new Error(`auth-service authorize returned ${auth.status}`)
  return { redirect, sid, maxAge }
}

const entities: Record<string, string> = { "&amp;": "&", "&lt;": "<", "&gt;": ">", "&quot;": '"', "&#x27;": "'", "&#39;": "'" }

/** Creates an account; returns the auth-service's validation message, or null when accepted. */
export async function passwordSignUp(req: Request, name: string, email: string, password: string) {
  const res = await fetch(`${BASE}/sign-up`, {
    method: "POST",
    headers: headersFor(req),
    body: new URLSearchParams({ name, email, password }),
    redirect: "manual",
  })
  if (res.ok) return null
  if (res.status !== 400) throw new Error(`auth-service sign-up returned ${res.status}`)
  const msg = /<p class="error" role="alert">([^<]*)<\/p>/.exec(await res.text())?.[1]
  return msg ? msg.replace(/&(amp|lt|gt|quot|#x27|#39);/g, (e) => entities[e]) : "Could not create the account."
}

/** Calls the token endpoint for this client; the body is the OAuth JSON reply or error. */
export async function exchangeToken(params: Record<string, string>) {
  const res = await fetch(`${BASE}/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({ client_id: CLIENT_ID, ...params }),
  })
  return { ok: res.ok, body: await res.json().catch(() => ({})) }
}

/** Ends the auth-service session, which also revokes this client's refresh tokens. */
export async function endSession(req: Request, sid: string) {
  await fetch(`${BASE}/end-session?${new URLSearchParams({ client_id: CLIENT_ID })}`, {
    headers: headersFor(req, sid),
    redirect: "manual",
  })
}
