"use client"

import Link from "next/link"
import {
  BarChart3,
  BellRing,
  Clock,
  Database,
  Search,
  Sparkles,
} from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { InfoCard } from "@/features/landing/components/info-card"
import { useSession } from "@/features/auth/hooks"

const cards = [
  {
    icon: Database,
    title: "Trusted data sources",
    description:
      "Aggregated from Singapore government procurement portals and public tender boards — deduplicated into one clean feed.",
  },
  {
    icon: Clock,
    title: "Fresh twice a day",
    description:
      "New tenders ingested every weekday at 10am and 3pm (SGT), so opportunities never go stale on you.",
  },
  {
    icon: Sparkles,
    title: "AI matching",
    description:
      "Personalised recommendations and opportunity scores surface the tenders your business can actually win.",
  },
  {
    icon: Search,
    title: "Search everything",
    description:
      "Every open tender and business opportunity across the market, searchable in one place.",
  },
  {
    icon: BellRing,
    title: "Never miss a deadline",
    description:
      "Save opportunities to favourites and keep an eye on closing dates before they pass.",
  },
  {
    icon: BarChart3,
    title: "Market analytics",
    description:
      "Understand the market with analytics on tenders, buyers, and award history.",
  },
]

// ponytail: static — regenerate if the scrapers in apps/tender-worker/internal change.
const sources = [
  { name: "EM Services", note: "Town council tenders" },
  { name: "AHTC", note: "Aljunied-Hougang Town Council" },
  { name: "Mandai Wildlife Group", note: "" },
  { name: "Mediacorp", note: "" },
  { name: "National Gallery Singapore", note: "" },
  { name: "NUS", note: "National University of Singapore" },
  { name: "SATS", note: "" },
  { name: "Singtel", note: "" },
  { name: "SIT", note: "Singapore Institute of Technology" },
  { name: "SMRT", note: "" },
  { name: "Tuas Power", note: "" },
]

export default function LandingPage() {
  const { data: user, isPending } = useSession()

  return (
    <div className="mx-auto w-full max-w-6xl px-4 lg:px-6">
      {/* Hero */}
      <section className="flex flex-col items-center gap-5 py-16 text-center md:py-24">
        <h1 className="text-4xl font-semibold tracking-tight md:text-5xl">
          Tender<span className="text-primary">Lah</span>
        </h1>
        <p className="text-muted-foreground max-w-xl text-base md:text-lg">
          Singapore&apos;s B2B platform to discover, track, and get AI-matched
          tenders and business opportunities from across the market — with
          analytics, all in one place.
        </p>

        {isPending ? null : user ? (
          <div className="flex flex-col items-center gap-3">
            <p className="text-sm">
              Signed in as <span className="font-medium">{user.email}</span>
            </p>
            <Button asChild size="lg">
              <Link href="/dashboard">Go to dashboard</Link>
            </Button>
          </div>
        ) : (
          <div className="flex gap-3">
            <Button asChild size="lg">
              <Link href="/tenders">View Tenders</Link>
            </Button>
            <Button asChild size="lg" variant="outline">
              <Link href="/sign-up">Sign up</Link>
            </Button>
          </div>
        )}
      </section>

      {/* Info cards */}
      <section className="grid gap-4 pb-16 sm:grid-cols-2 md:pb-24 lg:grid-cols-3">
        {cards.map((card) => (
          <InfoCard key={card.title} {...card} />
        ))}
      </section>

      {/* Data sources */}
      <section className="flex flex-col gap-6 pb-16 md:pb-24">
        <div className="flex flex-col gap-2 text-center">
          <h2 className="text-2xl font-semibold tracking-tight md:text-3xl">
            Where our tenders come from
          </h2>
          <p className="text-muted-foreground text-sm md:text-base">
            We pull from {sources.length} Singapore procurement sources into one
            feed.
          </p>
        </div>
        <div className="grid gap-3 sm:grid-cols-2 md:grid-cols-3">
          {sources.map((source) => (
            <div
              key={source.name}
              className="bg-card flex flex-col gap-0.5 rounded-lg border px-4 py-3"
            >
              <span className="font-medium">{source.name}</span>
              {source.note ? (
                <span className="text-muted-foreground text-sm">
                  {source.note}
                </span>
              ) : null}
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}
