import { NextResponse, type NextRequest } from "next/server"

import { isJson, passwordSignIn, SID_COOKIE } from "@/lib/auth/service"

export async function POST(req: NextRequest) {
  if (!isJson(req)) return NextResponse.json({ error: "Unsupported request" }, { status: 415 })
  const { email, password, authorize } = await req.json().catch(() => ({}))
  if (typeof email !== "string" || typeof password !== "string" || typeof authorize !== "string") {
    return NextResponse.json({ error: "Enter your email and password." }, { status: 400 })
  }
  const result = await passwordSignIn(req, email, password, authorize)
  if (!result) return NextResponse.json({ error: "Invalid email or password." }, { status: 401 })
  const res = NextResponse.json({ redirect: result.redirect })
  res.cookies.set(SID_COOKIE, result.sid, {
    httpOnly: true,
    sameSite: "lax",
    secure: req.nextUrl.protocol === "https:",
    path: "/api/auth",
    maxAge: result.maxAge,
  })
  return res
}
