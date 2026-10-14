// OIDC public client (auth code + PKCE) for chadlimjinjie's auth-service.
// Sign-in, sign-up, social and password reset all happen on its hosted pages;
// this module only starts the redirect, redeems the code and keeps tokens fresh.

export const ISSUER = process.env.NEXT_PUBLIC_AUTH_ISSUER ?? "http://localhost:9080/auth/t/tenderlah"
const CLIENT_ID = process.env.NEXT_PUBLIC_AUTH_CLIENT_ID ?? "web-tenders"
const SCOPE = "openid profile email offline_access"

// ponytail: refresh/id tokens in sessionStorage are XSS-readable and end with the tab;
// move to a BFF with an httponly cookie if that matters.
const REFRESH_KEY = "auth.refresh_token"
const ID_KEY = "auth.id_token"
const FLOW_KEY = "auth.flow"

export interface AuthUser {
  id: string
  email: string
  name: string
  emailVerified: boolean
}

let accessToken: string | null = null
let accessExpiry = 0
let inflight: Promise<string | null> | null = null

const redirectUri = () => `${window.location.origin}/auth/callback`

function b64url(bytes: Uint8Array): string {
  return btoa(String.fromCharCode(...bytes)).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "")
}

const randomString = () => b64url(crypto.getRandomValues(new Uint8Array(32)))

/** Path of the authorize request, with a fresh PKCE verifier and state stored for the callback. */
async function authorizePath(returnTo: string): Promise<string> {
  const verifier = randomString()
  const state = randomString()
  const challenge = b64url(new Uint8Array(await crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier))))
  sessionStorage.setItem(FLOW_KEY, JSON.stringify({ verifier, state, returnTo }))
  const q = new URLSearchParams({
    response_type: "code",
    client_id: CLIENT_ID,
    redirect_uri: redirectUri(),
    scope: SCOPE,
    state,
    code_challenge: challenge,
    code_challenge_method: "S256",
  })
  return `${new URL(ISSUER).pathname}/authorize?${q}`
}

export async function signIn(returnTo = "/") {
  window.location.assign(new URL(await authorizePath(returnTo), ISSUER).href)
}

export async function signUp(returnTo = "/") {
  const q = new URLSearchParams({ return_to: await authorizePath(returnTo) })
  window.location.assign(`${ISSUER}/sign-up?${q}`)
}

async function requestTokens(params: Record<string, string>) {
  const res = await fetch(`${ISSUER}/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({ client_id: CLIENT_ID, ...params }),
  })
  const body = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(body.error_description ?? body.error ?? "Sign-in failed")
  accessToken = body.access_token
  // refresh a minute early
  accessExpiry = Date.now() + (body.expires_in - 60) * 1000
  if (body.refresh_token) sessionStorage.setItem(REFRESH_KEY, body.refresh_token)
  if (body.id_token) sessionStorage.setItem(ID_KEY, body.id_token)
}

/** Redeems the authorization code; returns the path to continue to. */
export async function handleCallback(params: URLSearchParams): Promise<string> {
  const flow = JSON.parse(sessionStorage.getItem(FLOW_KEY) ?? "null")
  sessionStorage.removeItem(FLOW_KEY)
  const error = params.get("error")
  if (error) throw new Error(params.get("error_description") ?? error)
  const code = params.get("code")
  if (!flow || !code || params.get("state") !== flow.state) throw new Error("Invalid sign-in response")
  await requestTokens({
    grant_type: "authorization_code",
    code,
    redirect_uri: redirectUri(),
    code_verifier: flow.verifier,
  })
  return flow.returnTo
}

function clearTokens() {
  accessToken = null
  accessExpiry = 0
  sessionStorage.removeItem(REFRESH_KEY)
  sessionStorage.removeItem(ID_KEY)
}

/** A valid access token, refreshing it when near expiry; null when signed out. */
export async function getAccessToken(): Promise<string | null> {
  if (typeof window === "undefined") return null
  if (accessToken && Date.now() < accessExpiry) return accessToken
  const refreshToken = sessionStorage.getItem(REFRESH_KEY)
  if (!refreshToken) return null
  inflight ??= requestTokens({ grant_type: "refresh_token", refresh_token: refreshToken })
    .then(() => accessToken)
    .catch(() => {
      clearTokens()
      return null
    })
    .finally(() => {
      inflight = null
    })
  return inflight
}

/** The signed-in user from the ID token, or null. */
export async function getUser(): Promise<AuthUser | null> {
  if (!(await getAccessToken())) return null
  const idToken = sessionStorage.getItem(ID_KEY)
  if (!idToken) return null
  const payload = idToken.split(".")[1].replace(/-/g, "+").replace(/_/g, "/")
  const claims = JSON.parse(new TextDecoder().decode(Uint8Array.from(atob(payload), (c) => c.charCodeAt(0))))
  return {
    id: claims.sub,
    email: claims.email ?? "",
    name: claims.name || claims.email || "",
    emailVerified: !!claims.email_verified,
  }
}

/** Ends the local and the auth-service session. */
export function signOut() {
  const idToken = sessionStorage.getItem(ID_KEY)
  clearTokens()
  const q = new URLSearchParams({ client_id: CLIENT_ID, post_logout_redirect_uri: `${window.location.origin}/` })
  if (idToken) q.set("id_token_hint", idToken)
  window.location.assign(`${ISSUER}/end-session?${q}`)
}

/** Hosted page for setting a new password via an emailed link. */
export const resetPasswordUrl = () => `${ISSUER}/forgot-password`
