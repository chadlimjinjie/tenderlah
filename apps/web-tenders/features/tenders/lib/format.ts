// ponytail: tender dates are ISO strings; locale-format them, drop the time noise.
export const fmtDate = (s?: string | null) =>
  s ? new Date(s).toLocaleDateString() : "—"

// AI milestone dates are sometimes a relative phrase ("4 weeks after award") instead of
// an ISO date when the LLM can't infer an exact one — only locale-format real dates.
export const fmtMilestoneDate = (s: string) => {
  const d = new Date(s)
  return isNaN(d.getTime()) ? s : d.toLocaleDateString()
}

// ponytail: ids are ASCII (source:source_id) and can contain '/'; base64url makes a
// URL-safe single path segment so the gin /:id and Next [id] routes match unchanged.
export const encodeTenderId = (id: string) =>
  btoa(id).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "")

// Whole calendar days until closingDate, in the viewer's local timezone.
export const daysUntilClosing = (closingDate?: string | null) => {
  if (!closingDate) return null
  const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate())
  const ms = startOfDay(new Date(closingDate)).getTime() - startOfDay(new Date()).getTime()
  return Math.round(ms / (24 * 60 * 60 * 1000))
}

export const isClosingUrgent = (closingDate?: string | null) => {
  const days = daysUntilClosing(closingDate)
  return days != null && days >= 0 && days <= 7
}
