import { fileURLToPath } from "url"
import path from "path"

const __dirname = path.dirname(fileURLToPath(import.meta.url))

const gatewayUrl = process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8080"

/** @type {import('next').NextConfig} */
const nextConfig = {
  turbopack: {
    root: path.resolve(__dirname, "../.."),
  },
  // Proxy auth calls through this app's own origin so the session cookie is
  // first-party. web-tenders' production domain (tenderlah.sg) is a different
  // registrable domain from the auth-service/gateway, so a cookie set directly
  // cross-site is blocked outright by mobile Safari/WebKit's third-party
  // cookie policy — proxying makes the browser see a same-origin request.
  async rewrites() {
    return [{ source: "/api/auth/:path*", destination: `${gatewayUrl}/api/auth/:path*` }]
  },
}

export default nextConfig
