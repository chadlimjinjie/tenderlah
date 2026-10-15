import { NextResponse, type NextRequest } from "next/server"

import { isJson, passwordSignUp } from "@/lib/auth/service"

export async function POST(req: NextRequest) {
  if (!isJson(req)) return NextResponse.json({ error: "Unsupported request" }, { status: 415 })
  const { name, email, password } = await req.json().catch(() => ({}))
  if (typeof name !== "string" || typeof email !== "string" || typeof password !== "string") {
    return NextResponse.json({ error: "Fill in all fields." }, { status: 400 })
  }
  const error = await passwordSignUp(req, name, email, password)
  return error ? NextResponse.json({ error }, { status: 400 }) : NextResponse.json({ ok: true })
}
