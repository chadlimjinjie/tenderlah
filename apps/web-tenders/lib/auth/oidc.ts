// OIDC public client (auth code + PKCE) for chadlimjinjie's auth-service.
// Sign-in and sign-up use our own pages, bridged server-side by app/api/auth (lib/auth/service.ts);
// password reset stays on its hosted pages. This module redeems the code and keeps tokens fresh.

export const ISSUER = process.env.NEXT_PUBLIC_AUTH_ISSUER ?? "http://localhost:9080/auth/t/tenderlah"
export const CLIENT_ID = process.env.NEXT_PUBLIC_AUTH_CLIENT_ID ?? "web-tenders"
const SCOPE = "openid profile email offline_access"

// The refresh token lives in an httpOnly cookie set by app/api/auth/token; access and ID tokens
// stay in memory and are re-minted from it on each page load.
const FLOW_KEY = "auth.flow"

export interface AuthUser {
  id: string
  email: string
  name: string
  emailVerified: boolean
  /** "admin" when the access token's scope grants it, mirroring the API's role_from_scope. */
  role: "admin" | null
}

let accessToken: string | null = null
let idToken: string | null = null
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

async function postAuth(path: string, body: object) {
  const res = await fetch(`/api/auth/${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(data.error ?? "Something went wrong. Please try again.")
  return data
}

/** Signs in with email and password, then continues to the callback page that redeems the code. */
export async function signIn(email: string, password: string, returnTo = "/") {
  const { redirect } = await postAuth("sign-in", { email, password, authorize: await authorizePath(returnTo) })
  window.location.assign(redirect)
}

/** Creates the account; the auth-service emails a verification link and allows signing in right away. */
export async function signUp(name: string, email: string, password: string) {
  await postAuth("sign-up", { name, email, password })
}

/** Redeems a code, or with an empty body the refresh cookie, through our route handler. */
async function requestTokens(params: Record<string, string>) {
  const body = await postAuth("token", params)
  accessToken = body.access_token
  idToken = body.id_token
  // refresh a minute early
  accessExpiry = Date.now() + (body.expires_in - 60) * 1000
}

/** Redeems the authorization code; returns the path to continue to. */
export async function handleCallback(params: URLSearchParams): Promise<string> {
  const flow = JSON.parse(sessionStorage.getItem(FLOW_KEY) ?? "null")
  sessionStorage.removeItem(FLOW_KEY)
  const error = params.get("error")
  if (error) throw new Error(params.get("error_description") ?? error)
  const code = params.get("code")
  if (!flow || !code || params.get("state") !== flow.state) throw new Error("Invalid sign-in response")
  await requestTokens({ code, redirect_uri: redirectUri(), code_verifier: flow.verifier })
  return flow.returnTo
}

function clearTokens() {
  accessToken = null
  idToken = null
  accessExpiry = 0
}

/** A valid access token, refreshing it when near expiry; null when signed out. */
export async function getAccessToken(): Promise<string | null> {
  if (typeof window === "undefined") return null
  if (accessToken && Date.now() < accessExpiry) return accessToken
  // Refresh tokens are single-use and a replay revokes them all, so tabs sharing the cookie take turns.
  inflight ??= navigator.locks
    .request("auth.refresh", () => requestTokens({}))
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

function jwtClaims(token: string) {
  const payload = token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/")
  return JSON.parse(new TextDecoder().decode(Uint8Array.from(atob(payload), (c) => c.charCodeAt(0))))
}

/** The signed-in user from the ID token, or null. */
export async function getUser(): Promise<AuthUser | null> {
  const token = await getAccessToken()
  if (!token) return null
  if (!idToken) return null
  const claims = jwtClaims(idToken)
  // UI hint only; the API enforces admin itself.
  const scope = String(jwtClaims(token).scope ?? "")
  return {
    id: claims.sub,
    email: claims.email ?? "",
    name: claims.name || claims.email || "",
    emailVerified: !!claims.email_verified,
    role: scope.split(" ").includes("admin") ? "admin" : null,
  }
}

/** Ends the local and the auth-service session (revoking the refresh token). */
export async function signOut() {
  await postAuth("sign-out", {}).catch(() => {})
  clearTokens()
  window.location.assign("/")
}

/** Hosted page for setting a new password via an emailed link. */
export const resetPasswordUrl = () => `${ISSUER}/forgot-password`
