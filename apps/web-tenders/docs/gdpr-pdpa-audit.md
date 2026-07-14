# GDPR + PDPA Audit — `apps/web-tenders`

**Date:** 2026-06-30
**Scope:** `apps/web-tenders` (client-only Next.js frontend, prod domain `tenderlah.sg`)
**Framing:** EU/UK GDPR + ePrivacy primary; Singapore PDPA differences flagged inline.
Because the product is Singapore-facing (`tenderlah.sg`), **PDPA is likely the actual
governing law**; GDPR applies only insofar as EU/UK data subjects are targeted or monitored.

---

## 1. Scope & method

**Audited (this frontend repo only):**
- Client-side personal-data collection (auth + onboarding forms, uploads).
- Analytics / third-party SDKs and their configuration.
- Client storage (localStorage, sessionStorage, cookies) and token handling.
- Outbound network egress to non-first-party domains.
- Presence of transparency, consent, erasure, and access/export surfaces.

**Not audited (out of scope — needs separate backend review):**
backend/gateway data handling, retention periods, encryption-at-rest, sub-processor
agreements (incl. the PostHog DPA), breach process, and DPO appointment. These live
outside this repo. **No finding here should be read as backend assurance.**

**Method:** full read of `app/`, `features/`, `components/`, `lib/`, `package.json`, and
env files. Key claims spot-checked against source (cited inline as `path:line`).

---

## 2. Data inventory (mini-ROPA)

| Data | Fields | Where collected / shown | Sent to |
|---|---|---|---|
| **User account PII** | name, email, password | `features/auth/components/sign-up-form.tsx`, `sign-in-form.tsx`, password reset/change forms | `POST /api/auth/*` via gateway |
| **Session identity** | `AuthUser{id, email, name, emailVerified, role}` | `GET /api/auth/me`, shown in `components/nav/nav-user.tsx`, `features/settings/.../account-card.tsx` | gateway; cached in-memory (TanStack Query) |
| **Company profile** | orgName, businessName, businessDescription, industry, companySize, geographicCoverage, preferredContractSize | `features/onboarding/components/onboarding-form.tsx`, `features/settings/.../business-profile-card.tsx` | `POST /api/users/onboarding` |
| **Logo upload** | image file (may carry EXIF/personal data) | `features/onboarding/components/logo-picker.tsx` | `POST /api/users/onboarding/logo` |
| **Third-party PII** | tender contact **name, email, phone, address** (GeBIZ officers) | `features/tenders/components/tender-view-page.tsx:44-61` (ContactCard) | read-only display from `GET /api/tenders/{id}` |
| **Analytics** | `$pageview` + full URL (path **and** query string) | `components/providers/posthog-provider.tsx:17` | **PostHog US** (`us.i.posthog.com`) |

**Sole third-party data processor:** PostHog (`posthog-js`). No Google Analytics, Sentry,
Meta pixel, Segment, Mixpanel, Hotjar, Clarity, or Datadog exist in `package.json` or the
tree.

---

## 3. Findings

Ordered by severity. Each lists evidence, the GDPR mapping, the PDPA mapping, and a
recommendation.

### HIGH

#### H1 — No consent gate; PostHog cookie + tracking fire on first page load — ✅ RESOLVED 2026-06-30
**Original evidence:** `posthog-provider.tsx` called `posthog.init(...)` unconditionally in
production, setting a tracking cookie (`ph_<key>_posthog`) and capturing pageviews before
any consent. No banner/CMP existed.
- **GDPR / ePrivacy:** ePrivacy Directive Art. 5(3) requires **prior** consent for
  non-essential (analytics) cookies. **PDPA:** Consent Obligation (s.13).
- **Fix applied:** a consent gate was added.
  `components/cookie-consent-banner.tsx` (new) shows a bottom banner with Accept/Decline and
  persists the choice in `localStorage` (`ph_consent`).
  `components/providers/posthog-provider.tsx` now calls `posthog.init` **only** when
  `consent === "granted"` (guarded by `posthog.__loaded`); `PostHogPageView` renders only
  when granted. On Decline (or undecided), PostHog never loads — no cookie, no capture. The
  banner shows in production while consent is undecided.
- **Residual:** the banner now links to the privacy page (added with H2). Consent is binary
  site-wide (no per-category granularity), which is adequate given PostHog is the only
  processor.

#### H2 — No privacy policy, cookie notice, or terms page — ✅ RESOLVED 2026-06-30
**Original evidence:** no `app/privacy`, `app/terms`, or `app/cookie-policy` route existed;
no policy link anywhere.
- **GDPR:** Arts. 13-14 transparency. **PDPA:** Openness Obligation (s.11-12) requires a
  written policy + published DPO business contact; Notification Obligation (s.20).
- **Fix applied:** added `app/(shared)/privacy/page.tsx` — a privacy policy + cookie notice
  covering controller identity, data collected, purposes & legal basis, processors
  (PostHog), international transfer (US), retention, data-subject rights, and a **DPO
  contact**. Linked from the consent banner ("Learn more") and from a footer on every public
  page (`components/layouts/root-shell.tsx`).
- **Controller / DPO:** set to **Chad Lim** (sole developer — a natural person is a valid
  controller/DPO under PDPA & GDPR; no registered company needed) with DPO contact
  `privacy@tenderlah.sg`.
- **Residual (before launch):** create and monitor the `privacy@tenderlah.sg` inbox (alias
  or forwarder) — flagged with a `TODO` in the file. The retention and transfer-safeguard
  statements assert practices the backend must actually implement (see out-of-scope note);
  align the wording once those are confirmed.

#### H3 — No right to erasure (account deletion)
**Evidence:** no delete-account UI, route, hook, or API. `lib/api/generated/users/users.ts`
exposes only `getUserOnboarding`, `completeUserOnboarding`, and `uploadOnboardingLogo` —
no DELETE. The only DELETE in the client is OAuth-provider **unlink**
(`DELETE /api/auth/accounts/{provider}`), which is not account deletion.
- **GDPR:** Art. 17 right to erasure — no self-serve or documented manual path visible.
- **PDPA:** no general "right to be forgotten", **but** Retention Limitation (s.25)
  requires ceasing retention once the purpose ends, and from mid-2025 PDPC guidance expects
  honouring withdrawal-of-consent + deletion on request. A manual process must at least
  exist.
- **Recommendation:** add a delete-account flow (or document a manual erasure SOP if
  backend support is pending). Confirm a backend erasure endpoint exists.

#### H4 — No data access / portability (export)
**Evidence:** no export/download-my-data feature anywhere in the app.
- **GDPR:** Art. 15 (access) + Art. 20 (portability).
- **PDPA:** Access & Correction Obligation (s.21-22) — must provide an individual access
  to their personal data on request.
- **Recommendation:** add a "download my data" action, or a documented manual access
  process meeting the PDPA response timeline.

### MEDIUM

#### M1 — PostHog autocapture ON (default); session recording not explicitly disabled
**Evidence:** `posthog-provider.tsx:27-31` sets neither `autocapture: false` nor
`disable_session_recording`. PostHog's defaults: autocapture **on** (records clicks,
change events, element text/labels), session recording follows the project setting. The
onboarding and settings forms collect business/company data; autocapture records
surrounding labels/structure even though `<input>` values are masked by default.
- **GDPR:** data-minimisation, Art. 5(1)(c). **PDPA:** Purpose Limitation (s.18).
- **Mitigating:** `person_profiles: "identified_only"` (line 29) and **no
  `posthog.identify()` call exists anywhere** → events are effectively anonymous today.
- **Recommendation:** set `autocapture: false` (or a `property_denylist`), and explicitly
  `disable_session_recording: true` unless deliberately enabled with consent. Verify the
  project-side session-recording toggle.

#### M2 — US data residency (EU→US transfer)
**Evidence:** `NEXT_PUBLIC_POSTHOG_HOST=https://us.i.posthog.com` (`.env.local:2`).
- **GDPR:** Chapter V transfer rules — EU→US needs a transfer mechanism (PostHog's DPA /
  SCCs / DPF). **PDPA:** Transfer Limitation Obligation (s.26) — comparable protection for
  overseas transfers.
- **Recommendation:** either move to PostHog EU cloud (`eu.i.posthog.com`) or confirm the
  PostHog DPA + SCCs are in place and documented in the transfer record.

#### M3 — Full URLs (path + query) sent to PostHog
**Evidence:** `posthog-provider.tsx:14-17` builds `window.origin + pathname + "?" + search`
and sends it as `$current_url`. Any identifier or personal value in a query string is
exported to PostHog. (Also note `capture_pageview: true` on line 30 means pageviews may be
captured twice — once automatically, once manually.)
- **GDPR/PDPA:** data-minimisation / purpose limitation.
- **Recommendation:** strip or denylist query params before capture; resolve the
  double-pageview (drop manual capture or set `capture_pageview: false`).

#### M4 — Third-party PII displayed without documented lawful basis / source
**Evidence:** `tender-view-page.tsx:44-61` renders tender contact name, email (as
`mailto:`), phone, and address — personal data of agency officers scraped from GeBIZ.
- **GDPR:** Arts. 6 + 14 (data not obtained from the subject) — basis and source notice.
- **PDPA:** publicly-available-data exception (s.17 / Second Schedule) may apply, but the
  source and basis should be documented.
- **Recommendation:** document source (GeBIZ) and lawful basis in the ROPA / privacy
  policy. App is processor of display only; the controller decision sits backend.

### LOW / HYGIENE

#### L1 — PostHog project key committed in `.env.local`
**Evidence:** `apps/web-tenders/.env.local` contains
`NEXT_PUBLIC_POSTHOG_KEY=phc_...` and `NEXT_PUBLIC_POSTHOG_HOST=...`. The key is a public
client-side key by design (low secrecy risk), but a committed `.env.local` is bad hygiene
and the vars are undocumented in `.env.example`.
- **Recommendation:** confirm `.env.local` is gitignored (it is, per `.gitignore`); add the
  two PostHog vars to `.env.example` as documentation; rotate only if it was ever committed.

#### L2 — Stray duplicate env file
**Evidence:** a nested `apps/web-tenders/apps/web-tenders/.env.local` exists with the same
PostHog keys.
- **Recommendation:** delete the nested copy; it is a path accident.

#### L3 — Plaintext-HTTP dev default for the gateway
**Evidence:** `lib/api/axios-instance.ts:3` and `features/auth/components/social-sign-in.tsx`
default to `http://localhost:8080` when `NEXT_PUBLIC_GATEWAY_URL` is unset.
- **Recommendation:** dev-only; confirm prod build always injects an HTTPS
  `NEXT_PUBLIC_GATEWAY_URL` (credentials in transit otherwise).

#### L4 — Auth credentials proxied through the app origin
**Evidence:** `next.config.mjs:18-20` rewrites `/api/auth/:path*` to the gateway, so
email/password/name transit `tenderlah.sg` infrastructure before the gateway. This is **by
design** for first-party session cookies.
- **Recommendation:** no fix needed; document the data flow in the ROPA so the origin's
  role as a transit point is explicit.

---

## 4. Good practices observed

- **No PII in `localStorage`/`sessionStorage`** — confirmed none anywhere. The only
  JS-set cookie is `sidebar_state` (functional UI preference, `components/ui/sidebar.tsx`),
  which is strictly-necessary and needs no consent.
- **JWT kept in memory only** (`lib/api/axios-instance.ts` `cachedToken`), cleared on
  sign-out; **httpOnly session cookie** set by the backend (not JS-readable).
- **Analytics effectively anonymous today:** `person_profiles: "identified_only"` and no
  `identify()` call.
- **Analytics gated to production builds** (no tracking in dev) **and behind a consent
  banner** (H1 fix) — PostHog loads only after explicit Accept.
- **Anti-enumeration** on password reset (constant confirmation message); client-side
  password minimum length (8).
- **Single, well-known processor** (PostHog) — small, auditable third-party surface.

---

## 5. Prioritised remediation backlog

**Quick wins (low effort, high value)**
1. ~~**H1** — gate `posthog.init` behind consent.~~ ✅ Done 2026-06-30.
2. **M1 / M3** — set `autocapture: false`, `disable_session_recording: true`; strip query
   params and fix the double-pageview. One file: `posthog-provider.tsx`.
3. **L2** — delete the stray nested `.env.local`.
4. **L1** — add PostHog vars to `.env.example`; confirm gitignore.
5. **M2** — switch PostHog to EU cloud, or file the DPA/SCC reference.

**Larger items (need backend + legal)**
5. ~~**H2** — publish privacy policy + cookie notice + DPO contact page; footer links.~~
   ✅ Done 2026-06-30 — controller/DPO set (Chad Lim / privacy@tenderlah.sg); **only open
   item is creating + monitoring that inbox.**
6. **H3** — account-deletion flow (needs a backend erasure endpoint).
7. **H4** — data export/access flow (needs a backend endpoint) or documented manual SOP.
8. **M4** — document tender-contact source & lawful basis in the ROPA.

**Note:** items 5-7 are **PDPA-mandatory** for a Singapore product (privacy policy + DPO
are required; access requests must be honoured), independent of GDPR exposure. With H1
(consent gate) and H2 (privacy policy page) now resolved, **H3/H4 (erasure + access/export)
are the highest remaining priority** — both need backend endpoints, so they pair with a
backend review. Before launch, replace the privacy-page placeholders (legal entity + DPO
email).

---

*This audit covers the `apps/web-tenders` frontend only. A separate review of the gateway,
auth-service, and `api-server-go` is required for retention, encryption, sub-processor, and
breach-handling obligations.*
