import Axios, { isAxiosError } from "axios"

export const API_BASE = process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8080"

// auth-service issues an httponly session cookie; `withCredentials` sends it on
// every request through the gateway.
export const axiosInstance = Axios.create({
  baseURL: API_BASE,
  withCredentials: true,
})

// Backend services (e.g. api-server-go) authenticate with a JWT Bearer token,
// not the session cookie. We exchange the session for a short-lived JWT at
// /api/auth/token and cache it until shortly before it expires.
const TOKEN_PATH = "/api/auth/token"
// auth-service mints tokens with a 15m TTL; refresh a minute early.
const TOKEN_TTL_MS = 14 * 60 * 1000

let cachedToken: string | null = null
let tokenExpiry = 0
let inflight: Promise<string | null> | null = null

async function fetchToken(): Promise<string | null> {
  try {
    const { data } = await axiosInstance.get<{ token: string }>(TOKEN_PATH)
    cachedToken = data.token
    tokenExpiry = Date.now() + TOKEN_TTL_MS
    return cachedToken
  } catch (err) {
    // Signed out (401) or token endpoint unavailable — proceed without a bearer.
    if (isAxiosError(err) && err.response?.status === 401) return null
    return null
  } finally {
    inflight = null
  }
}

/** Returns a valid JWT, reusing the cached one until it nears expiry. */
export async function getAuthToken(): Promise<string | null> {
  if (cachedToken && Date.now() < tokenExpiry) return cachedToken
  if (!inflight) inflight = fetchToken()
  return inflight
}

/** Clear the cached token (e.g. on sign-out). */
export function clearAuthToken() {
  cachedToken = null
  tokenExpiry = 0
}

axiosInstance.interceptors.request.use(async (config) => {
  // auth-service's session cookie must be first-party for cross-site
  // front-ends (e.g. tenderlah.sg, a different registrable domain from the
  // gateway) — mobile Safari/WebKit blocks third-party cookies outright.
  // Route auth calls through this app's own /api/auth/* rewrite
  // (next.config.mjs) instead of straight to the gateway.
  if (config.url?.startsWith("/api/auth")) config.baseURL = ""

  // Don't try to attach a bearer to the token-exchange call itself.
  if (config.url === TOKEN_PATH) return config
  const token = await getAuthToken()
  if (token) {
    config.headers = config.headers ?? {}
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

export const customAxiosInstance = <T>(
  config: Parameters<typeof axiosInstance.request>[0]
): Promise<T> => {
  return axiosInstance.request<T>(config).then(({ data }) => data)
}
