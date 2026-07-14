import type { Metadata } from "next"

export const metadata: Metadata = {
  title: "Contact Us",
  description: "Get in touch with the TenderLah team on WhatsApp.",
}

const WHATSAPP_DISPLAY = "+65 8113 6098"
const WHATSAPP_LINK = "https://wa.me/6581136098"

export default function ContactPage() {
  return (
    <div className="mx-auto w-full max-w-3xl flex-1 px-6 py-12">
      <h1 className="text-3xl font-semibold tracking-tight">Contact Us</h1>
      <p className="text-muted-foreground mt-2 text-sm">
        We&apos;d love to hear from you.
      </p>

      <div className="mt-8 space-y-4 text-sm leading-relaxed">
        <p>
          Message us on WhatsApp at{" "}
          <a className="underline" href={WHATSAPP_LINK}>
            {WHATSAPP_DISPLAY}
          </a>{" "}
          and we&apos;ll get back to you as soon as we can.
        </p>
      </div>
    </div>
  )
}
