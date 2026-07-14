"use client"

import { useEffect } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { AlertTriangle, ArrowLeft, ExternalLink, Lock, Sparkles } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"
import { Tender, TenderBrief, TenderContact, TenderDeliveryTimeline } from "@/lib/api/generated/model"
import { useSession } from "@/features/auth/hooks"
import {
  useTender,
  useGenerateTenderBrief,
  useGenerateTenderDeliveryTimeline,
} from "@/features/tenders/hooks/use-tenders"
import { FavouriteButton } from "@/features/tenders/components/favourite-button"
import { fmtDate, fmtMilestoneDate } from "@/features/tenders/lib/format"

interface TenderViewPageProps {
  id: string
}

function Field({
  label,
  value,
  mono = false,
}: {
  label: string
  value: string | null | undefined
  mono?: boolean
}) {
  return (
    <div className="min-w-0">
      <p className="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</p>
      <p className={cn("text-sm break-words", mono && "font-mono tabular-nums")}>{value || "—"}</p>
    </div>
  )
}

function ContactCard({ c }: { c: TenderContact }) {
  return (
    <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
      <Field label="Name" value={c.name} />
      <div className="min-w-0">
        <p className="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">Email</p>
        <p className="text-sm break-words">
          {c.email ? (
            <a href={`mailto:${c.email}`} className="underline-offset-4 hover:underline">
              {c.email}
            </a>
          ) : (
            "—"
          )}
        </p>
      </div>
      <Field label="Phone" value={c.phone} />
      <Field label="Address" value={c.address} />
    </div>
  )
}

function BulletList({ label, items }: { label: string; items?: string[] }) {
  if (!items || items.length === 0) return null
  return (
    <div>
      <p className="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</p>
      <ul className="list-disc space-y-1 pl-5 text-sm">
        {items.map((it, i) => (
          <li key={i}>{it}</li>
        ))}
      </ul>
    </div>
  )
}

function BriefCard({ brief }: { brief: TenderBrief }) {
  return (
    <div className="rounded-lg border bg-muted/30 p-5">
      <div className="mb-3 flex items-center gap-2">
        <Sparkles className="h-4 w-4 text-primary" />
        <h2 className="text-lg font-semibold">AI bid brief</h2>
        {brief.complexity && <Badge variant="secondary">{brief.complexity} complexity</Badge>}
      </div>
      {brief.summary && <p className="mb-4 text-sm">{brief.summary}</p>}
      {brief.riskFlags && brief.riskFlags.length > 0 && (
        <div className="mb-4 flex flex-col gap-2 sm:flex-row sm:flex-wrap">
          {brief.riskFlags.map((f, i) => (
            <Badge
              key={i}
              variant="outline"
              className="h-auto w-full items-start justify-start gap-1 whitespace-normal border-amber-500/50 py-1 text-left break-words text-amber-700 dark:text-amber-400 sm:w-auto"
            >
              <AlertTriangle className="mt-0.5 h-3 w-3 shrink-0" />
              {f}
            </Badge>
          ))}
        </div>
      )}
      <div className="grid gap-4 md:grid-cols-3">
        <BulletList label="Scope" items={brief.scope} />
        <BulletList label="Requirements" items={brief.requirements} />
        <BulletList label="Documents to submit" items={brief.documents} />
      </div>
      <p className="mt-4 text-xs text-muted-foreground">AI-generated from the tender listing — verify against the source page before bidding.</p>
    </div>
  )
}

function DeliveryTimelineCard({ timeline }: { timeline: TenderDeliveryTimeline }) {
  if (!timeline.milestones || timeline.milestones.length === 0) return null
  return (
    <div className="rounded-lg border bg-muted/30 p-5">
      <div className="mb-4 flex items-center gap-2">
        <Sparkles className="h-4 w-4 text-primary" />
        <h2 className="text-lg font-semibold">AI delivery timeline</h2>
      </div>
      <ol className="space-y-4 border-l-2 pl-4">
        {timeline.milestones.map((m, i) => (
          <li key={i} className="relative">
            <span className="absolute top-1.5 -left-[21px] h-2.5 w-2.5 rounded-full bg-primary" />
            <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{fmtMilestoneDate(m.date)}</p>
            <p className="text-sm font-semibold">{m.label}</p>
            {m.description && <p className="text-sm text-muted-foreground">{m.description}</p>}
          </li>
        ))}
      </ol>
      <p className="mt-4 text-xs text-muted-foreground">AI-generated estimate — verify against the source page before bidding.</p>
    </div>
  )
}

function DeliveryTimelineLockedCard() {
  return (
    <div className="relative overflow-hidden rounded-lg border bg-muted/30 p-5">
      <div aria-hidden className="pointer-events-none space-y-4 select-none blur-sm">
        <div className="flex items-center gap-2">
          <Sparkles className="h-4 w-4 text-primary" />
          <h2 className="text-lg font-semibold">AI delivery timeline</h2>
        </div>
        <div className="space-y-3 border-l-2 pl-4">
          <Skeleton className="h-10" />
          <Skeleton className="h-10" />
          <Skeleton className="h-10" />
        </div>
      </div>
      <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 bg-background/40 text-center">
        <Lock className="h-5 w-5 text-muted-foreground" />
        <p className="text-sm font-medium">Sign in to view the AI delivery timeline</p>
        <div className="flex gap-2">
          <Button size="sm" variant="outline" asChild>
            <Link href="/sign-up">Sign up</Link>
          </Button>
          <Button size="sm" asChild>
            <Link href="/sign-in">Sign in</Link>
          </Button>
        </div>
      </div>
    </div>
  )
}

function BriefLockedCard() {
  return (
    <div className="relative overflow-hidden rounded-lg border bg-muted/30 p-5">
      <div aria-hidden className="pointer-events-none space-y-4 select-none blur-sm">
        <div className="flex items-center gap-2">
          <Sparkles className="h-4 w-4 text-primary" />
          <h2 className="text-lg font-semibold">AI bid brief</h2>
          <Badge variant="secondary">Medium complexity</Badge>
        </div>
        <p className="text-sm">Sign in to see an AI-generated summary of scope, requirements, and risk flags for this tender.</p>
        <div className="grid gap-4 md:grid-cols-3">
          <Skeleton className="h-16" />
          <Skeleton className="h-16" />
          <Skeleton className="h-16" />
        </div>
      </div>
      <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 bg-background/40 text-center">
        <Lock className="h-5 w-5 text-muted-foreground" />
        <p className="text-sm font-medium">Sign in to view the AI bid brief</p>
        <div className="flex gap-2">
          <Button size="sm" variant="outline" asChild>
            <Link href="/sign-up">Sign up</Link>
          </Button>
          <Button size="sm" asChild>
            <Link href="/sign-in">Sign in</Link>
          </Button>
        </div>
      </div>
    </div>
  )
}

// Brief/timeline generation is a separate explicit action from fetching the tender, so
// trigger each once per tender for signed-in viewers as soon as it loads.
function useAutoGenerateAI(tender: Tender | undefined, enabled: boolean) {
  const brief = useGenerateTenderBrief()
  const timeline = useGenerateTenderDeliveryTimeline()

  useEffect(() => {
    if (!enabled || !tender) return
    if (!tender.aiBriefAt && !brief.isPending) brief.mutate({ id: tender.id })
    if (!tender.aiDeliveryTimelineAt && !timeline.isPending) timeline.mutate({ id: tender.id })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabled, tender?.id, tender?.aiBriefAt, tender?.aiDeliveryTimelineAt])

  return {
    briefPending: enabled && !tender?.aiBriefAt,
    timelinePending: enabled && !tender?.aiDeliveryTimelineAt,
  }
}

export default function TenderViewPage({ id }: TenderViewPageProps) {
  const router = useRouter()
  const { data: tender, isLoading, error } = useTender(id)
  const { data: user } = useSession()
  const { briefPending, timelinePending } = useAutoGenerateAI(tender, !!user)

  if (isLoading) {
    return (
      <div className="space-y-6 px-4 lg:px-6">
        <div className="flex items-center gap-3">
          <Skeleton className="h-9 w-9" />
          <Skeleton className="h-8 w-64" />
        </div>
        <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
          {Array.from({ length: 8 }).map((_, i) => <Skeleton key={i} className="h-12" />)}
        </div>
        <Skeleton className="h-24" />
      </div>
    )
  }

  if (error || !tender) {
    return (
      <div className="px-4 lg:px-6">
        <Button variant="ghost" size="sm" onClick={() => router.push("/tenders")}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Tenders
        </Button>
        <p className="mt-4 text-muted-foreground">Tender not found.</p>
      </div>
    )
  }

  const contacts = tender.details?.contacts ?? (tender.details?.contact ? [tender.details.contact] : [])
  const detailsPending = tender.source === "gebiz" && !tender.detailsFetchedAt

  return (
    <div className="space-y-6 px-4 lg:px-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex min-w-0 items-start gap-3">
          <Button variant="ghost" size="icon" className="shrink-0" onClick={() => router.push("/tenders")}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <h1 className="min-w-0 break-words text-xl font-semibold sm:text-2xl">{tender.title || "Untitled tender"}</h1>
            <Badge variant="outline">{tender.source.toUpperCase()}</Badge>
            {tender.source === "gebiz" && (
              <span className="text-muted-foreground text-xs">
                Source: GeBIZ, Government of Singapore
              </span>
            )}
            <FavouriteButton tenderId={tender.id} isFavourite={tender.isFavourite} />
          </div>
        </div>
        {/* ponytail: GeBIZ deep-linking needs prior DSTA permission — suppress the
            outbound link for gebiz until NEXT_PUBLIC_GEBIZ_LINKS_ENABLED=true. */}
        {tender.link &&
          (tender.source !== "gebiz" ||
          process.env.NEXT_PUBLIC_GEBIZ_LINKS_ENABLED === "true" ? (
            <Button variant="outline" size="sm" asChild className="shrink-0 self-start">
              <a href={tender.link} target="_blank" rel="noopener noreferrer">
                <ExternalLink className="mr-2 h-4 w-4" />
                Open source page
              </a>
            </Button>
          ) : (
            <p className="text-muted-foreground shrink-0 self-start text-xs">
              Search reference{" "}
              <span className="font-mono">{tender.referenceNo}</span> on GeBIZ
            </p>
          ))}
      </div>

      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        <Field label="Reference No" value={tender.referenceNo} mono />
        <Field label="Agency" value={tender.agency} />
        <Field label="Type" value={tender.tenderType} />
        <Field label="Category" value={tender.category} />
        <Field label="Published" value={fmtDate(tender.publishedDate)} mono />
        <Field label="Closing" value={fmtDate(tender.closingDate)} mono />
        <Field label="Last Seen" value={fmtDate(tender.lastSeenAt)} mono />
      </div>

      {user ? (
        <>
          {tender.aiBrief && <BriefCard brief={tender.aiBrief} />}
          {briefPending && <Skeleton className="h-32" />}
          {tender.aiDeliveryTimeline && <DeliveryTimelineCard timeline={tender.aiDeliveryTimeline} />}
          {timelinePending && <Skeleton className="h-32" />}
        </>
      ) : (
        <>
          <BriefLockedCard />
          <DeliveryTimelineLockedCard />
        </>
      )}

      {tender.description && (
        <div>
          <p className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">Description</p>
          <p className="whitespace-pre-wrap text-sm">{tender.description}</p>
        </div>
      )}

      {contacts.length > 0 && (
        <div>
          <h2 className="mb-3 text-lg font-semibold">Who to contact</h2>
          <div className="space-y-4">
            {contacts.map((c, i) => (
              <ContactCard key={i} c={c} />
            ))}
          </div>
        </div>
      )}

      {tender.details?.items && tender.details.items.length > 0 && (
        <div>
          <h2 className="mb-3 text-lg font-semibold">Items to respond</h2>
          <div className="space-y-4">
            {tender.details.items.map((item) => (
              <div key={item.no} className="rounded-md border p-4">
                <p className="mb-2 font-medium">
                  Item {item.no} — {item.description || "—"}
                </p>
                <dl className="grid grid-cols-2 gap-x-4 gap-y-1 md:grid-cols-3">
                  {item.attributes.map((a, i) => (
                    <div key={i} className="text-sm">
                      <dt className="text-xs uppercase tracking-wide text-muted-foreground">{a.label}</dt>
                      <dd>{a.value || "—"}</dd>
                    </div>
                  ))}
                </dl>
              </div>
            ))}
          </div>
        </div>
      )}

      {detailsPending && <Skeleton className="h-24" />}
    </div>
  )
}
