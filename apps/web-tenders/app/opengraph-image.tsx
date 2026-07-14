import { ImageResponse } from "next/og"

export const alt = "TenderLah — Singapore B2B tender & business opportunities"
export const size = { width: 1200, height: 630 }
export const contentType = "image/png"

export default function OpengraphImage() {
  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          flexDirection: "column",
          justifyContent: "center",
          padding: "80px",
          background: "#ED2939",
          color: "#FFFFFF",
          fontFamily: "sans-serif",
        }}
      >
        <div style={{ fontSize: 96, fontWeight: 800, letterSpacing: "-0.03em" }}>TenderLah</div>
        <div style={{ fontSize: 44, fontWeight: 500, marginTop: 24, lineHeight: 1.2, maxWidth: 900 }}>
          Singapore B2B tender &amp; business opportunities — discovery and analytics
        </div>
      </div>
    ),
    size,
  )
}
