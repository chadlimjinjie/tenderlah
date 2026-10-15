import { NextResponse, type NextRequest } from "next/server"

import { endSession, isJson, RT_COOKIE, SID_COOKIE } from "@/lib/auth/service"

export async function POST(req: NextRequest) {
  if (!isJson(req)) return NextResponse.json({ error: "Unsupported request" }, { status: 415 })
  const sid = req.cookies.get(SID_COOKIE)?.value
  if (sid) await endSession(req, sid)
  const res = NextResponse.json({ ok: true })
  res.cookies.delete({ name: SID_COOKIE, path: "/api/auth" })
  res.cookies.delete({ name: RT_COOKIE, path: "/api/auth" })
  return res
}
