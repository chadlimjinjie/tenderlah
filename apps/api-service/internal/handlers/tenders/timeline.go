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

// Milestone is one point on the AI-generated delivery timeline.
type Milestone struct {
	Label       string `json:"label"`       // e.g. "Contract award", "Delivery of Phase 1"
	Date        string `json:"date"`        // estimated ISO date (YYYY-MM-DD), always a concrete date, never a relative phrase
	Description string `json:"description"`
}

// DeliveryTimeline is the AI-generated, supplier-agnostic delivery schedule for one
// tender. Stored as the tender's ai_delivery_timeline jsonb and reused for every viewer.
// There is no cap on the number of milestones — as many as the tender implies.
type DeliveryTimeline struct {
	Milestones []Milestone `json:"milestones"`
}

const timelineSystem = `You are a Singapore government procurement analyst helping a supplier plan delivery against a tender. ` +
	`Read the tender and infer a realistic delivery timeline as a sequence of milestones, from contract award through final delivery/acceptance. ` +
	`Use the closing date as the basis for the award date when estimating. Every milestone must have a concrete estimated date in YYYY-MM-DD format — never a relative phrase (e.g. "4 weeks after award"). ` +
	`If the tender doesn't state an exact date for a milestone, estimate one using typical timelines for similar government procurement and express it as a calendar date. ` +
	`Do not invent milestones not implied by the tender. Respond with valid JSON only, matching this shape: ` +
	`{"milestones": [{"label": string, "date": string, "description": string}, ...]}. Order milestones chronologically. Use [] if nothing can be inferred.`

// generateTimeline builds the prompt from the tender row (+ already-scraped details)
// and asks the LLM for a structured delivery timeline. Returns the parsed timeline and its JSON.
func generateTimeline(ctx context.Context, client *llm.Client, t db.Tender) (DeliveryTimeline, string, error) {
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

	raw, err := client.GenerateJSON(ctx, timelineSystem, b.String())
	if err != nil {
		return DeliveryTimeline{}, "", err
	}
	timeline, err := parseTimeline(raw)
	if err != nil {
		return DeliveryTimeline{}, "", err
	}
	out, err := json.Marshal(timeline)
	if err != nil {
		return DeliveryTimeline{}, "", err
	}
	return timeline, string(out), nil
}

// parseTimeline unmarshals model output, tolerating a ```json fenced block.
func parseTimeline(raw string) (DeliveryTimeline, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(strings.TrimSpace(raw), "```")
	}
	var timeline DeliveryTimeline
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &timeline); err != nil {
		return DeliveryTimeline{}, err
	}
	return timeline, nil
}
