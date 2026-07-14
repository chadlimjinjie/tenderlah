import type { Metadata } from "next"

export const metadata: Metadata = {
  title: "Privacy Policy & Cookie Notice",
  description:
    "How TenderLah collects, uses, and protects your personal data, your rights, and how to contact our Data Protection Officer.",
}

// TODO: before launch, create and monitor the privacy@tenderlah.sg inbox (alias/forwarder).
const ENTITY = "Chad Lim"
const DPO_EMAIL = "chadlimjinjie@gmail.com"
const LAST_UPDATED = "30 June 2026"

export default function PrivacyPage() {
  return (
    <div className="mx-auto w-full max-w-3xl flex-1 px-6 py-12">
      <h1 className="text-3xl font-semibold tracking-tight">
        Privacy Policy &amp; Cookie Notice
      </h1>
      <p className="text-muted-foreground mt-2 text-sm">Last updated: {LAST_UPDATED}</p>

      <div className="mt-8 space-y-8 text-sm leading-relaxed">
        <section className="space-y-2">
          <h2 className="text-lg font-medium">1. Who we are</h2>
          <p>
            {ENTITY} operates the TenderLah platform, which helps Singapore
            businesses discover and track tenders and business opportunities, and
            is the data controller responsible for your personal data. This
            notice explains what we collect, why, and the rights you have under
            Singapore&apos;s Personal Data Protection Act (PDPA) and, where it
            applies to you, the EU/UK General Data Protection Regulation (GDPR).
          </p>
        </section>

        <section className="space-y-2">
          <h2 className="text-lg font-medium">2. What we collect</h2>
          <ul className="ml-5 list-disc space-y-1">
            <li>
              <strong>Account details</strong> — your name, email address, and
              password (stored only in hashed form), provided when you sign up or
              sign in.
            </li>
            <li>
              <strong>Company profile</strong> — organisation and business name,
              description, industry, company size, geographic coverage, preferred
              contract size, and an optional logo, provided during onboarding.
            </li>
            <li>
              <strong>Usage data</strong> — pages you visit and basic analytics
              events, collected through cookies only if you accept analytics (see
              section 6).
            </li>
          </ul>
        </section>

        <section className="space-y-2">
          <h2 className="text-lg font-medium">3. Why we use it &amp; legal basis</h2>
          <p>
            We use account and company-profile data to provide the service,
            authenticate you, and personalise tender matches — on the basis of
            performing our contract with you (GDPR Art. 6(1)(b)) and the consent
            you give us under the PDPA. We use analytics data, where you consent,
            to understand and improve how the site is used (GDPR Art. 6(1)(a) /
            PDPA consent). We do not sell your personal data.
          </p>
        </section>

        <section className="space-y-2">
          <h2 className="text-lg font-medium">4. Who we share it with</h2>
          <p>
            We use <strong>PostHog</strong> as our analytics processor. If you
            accept analytics cookies, your usage data is processed by PostHog on
            our behalf. We do not otherwise disclose your personal data except
            where required by law.
          </p>
        </section>

        <section className="space-y-2">
          <h2 className="text-lg font-medium">5. International transfers</h2>
          <p>
            Our analytics processor (PostHog) currently stores data on servers in
            the United States. Where this involves transferring personal data out
            of Singapore or the EU/UK, we rely on the processor&apos;s contractual
            safeguards (data-processing agreement and standard contractual
            clauses) to ensure a comparable standard of protection, as required by
            the PDPA Transfer Limitation Obligation and GDPR Chapter V.
          </p>
        </section>

        <section className="space-y-2">
          <h2 className="text-lg font-medium">6. Cookies &amp; analytics</h2>
          <p>
            We use a small number of <strong>strictly necessary cookies</strong>{" "}
            to keep you signed in and remember interface preferences; these are
            required for the site to work and are always on.
          </p>
          <p>
            <strong>Analytics cookies</strong> (PostHog) are optional and load{" "}
            <em>only after you accept</em> them in the cookie banner. If you
            decline, no analytics cookies are set and no usage data is collected.
            You can change your mind at any time by clearing the{" "}
            <code>ph_consent</code> value in your browser storage, which brings
            the banner back.
          </p>
        </section>

        <section className="space-y-2">
          <h2 className="text-lg font-medium">7. How long we keep it</h2>
          <p>
            We keep your account and company-profile data for as long as your
            account is active. When you close your account, or when the data is no
            longer needed for the purpose it was collected, we delete or anonymise
            it in line with the PDPA Retention Limitation Obligation.
          </p>
        </section>

        <section className="space-y-2">
          <h2 className="text-lg font-medium">8. Your rights</h2>
          <p>
            Subject to the PDPA and (where applicable) the GDPR, you can:
          </p>
          <ul className="ml-5 list-disc space-y-1">
            <li>access a copy of the personal data we hold about you;</li>
            <li>correct inaccurate or incomplete data;</li>
            <li>withdraw consent (including for analytics) at any time;</li>
            <li>
              request deletion of your data, and — under the GDPR — request a
              portable copy of it.
            </li>
          </ul>
          <p>
            To exercise any of these, contact our Data Protection Officer (section
            9). We will respond within the timeframes the applicable law requires.
          </p>
        </section>

        <section className="space-y-2">
          <h2 className="text-lg font-medium">9. Contact our DPO</h2>
          <p>
            For any privacy question or to exercise your rights, contact our Data
            Protection Officer at{" "}
            <a className="underline" href={`mailto:${DPO_EMAIL}`}>
              {DPO_EMAIL}
            </a>
            .
          </p>
        </section>

        <section className="space-y-2">
          <h2 className="text-lg font-medium">10. Changes to this notice</h2>
          <p>
            We may update this notice from time to time. The &quot;last
            updated&quot; date above reflects the most recent change.
          </p>
        </section>
      </div>
    </div>
  )
}
