package tenders

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/midtone/api-server/internal/db"
	"github.com/midtone/api-server/internal/llm"
)

// BidBrief is the AI-generated, supplier-agnostic decision brief for one tender.
// Stored as the tender's ai_brief jsonb and reused for every viewer.
type BidBrief struct {
	Summary      string   `json:"summary"`      // plain-English "what they're actually buying"
	Scope        []string `json:"scope"`        // scope as bullet points
	Requirements []string `json:"requirements"` // eligibility / mandatory certs / registrations
	Documents    []string `json:"documents"`    // submission documents the bidder must provide
	Complexity   string   `json:"complexity"`   // Low | Medium | High
	RiskFlags    []string `json:"riskFlags"`    // tight deadline, restrictive specs (possible incumbent), broad scope, etc.
}

const briefSystem = `You are a Singapore government procurement analyst helping a supplier decide whether to bid on a tender. ` +
	`Read the tender and produce a concise, factual bid brief. Do not invent requirements that are not present. ` +
	`Respond with valid JSON only, matching this shape: ` +
	`{"summary": string, "scope": string[], "requirements": string[], "documents": string[], "complexity": "Low"|"Medium"|"High", "riskFlags": string[]}. ` +
	`riskFlags should call out things a bidder should worry about: a tight closing window, unusually narrow/single-source-looking specifications (possible incumbent advantage), or an unusually broad scope. Use [] when a section has nothing.`

// generateBrief builds the prompt from the tender row (+ already-scraped details)
// and asks the LLM for a structured brief. Returns the parsed brief and its JSON.
func generateBrief(ctx context.Context, client *llm.Client, t db.Tender) (BidBrief, string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "Title: %s\n", t.Title)
	fmt.Fprintf(&b, "Buyer/Agency: %s\n", firstNonEmpty(t.Agency, t.Source))
	fmt.Fprintf(&b, "Category: %s\n", t.Category)
	fmt.Fprintf(&b, "Tender type: %s\n", t.TenderType)
	if t.ClosingDate != nil {
		fmt.Fprintf(&b, "Closing date: %s (today is %s)\n", t.ClosingDate.Format(time.RFC3339), time.Now().Format(time.RFC3339))
	}
	if t.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", t.Description)
	}
	if len(t.Details) > 0 && string(t.Details) != "null" {
		fmt.Fprintf(&b, "Line items and contact (JSON): %s\n", string(t.Details))
	}

	raw, err := client.GenerateJSON(ctx, briefSystem, b.String())
	if err != nil {
		return BidBrief{}, "", err
	}
	brief, err := parseBrief(raw)
	if err != nil {
		return BidBrief{}, "", err
	}
	out, err := json.Marshal(brief)
	if err != nil {
		return BidBrief{}, "", err
	}
	return brief, string(out), nil
}

// parseBrief unmarshals model output, tolerating a ```json fenced block.
func parseBrief(raw string) (BidBrief, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(strings.TrimSpace(raw), "```")
	}
	var brief BidBrief
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &brief); err != nil {
		return BidBrief{}, err
	}
	return brief, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
