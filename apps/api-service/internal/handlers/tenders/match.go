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

// TenderMatch is the AI-generated, per-company fit assessment for one tender.
// Unlike BidBrief/DeliveryTimeline this is not supplier-agnostic, so it's
// stored per (user, tender) in the tender_match table, not on the tender row.
type TenderMatch struct {
	Score               int      `json:"score"`    // 0-100
	Priority            string   `json:"priority"` // HIGH | MEDIUM | LOW
	FitReasons          []string `json:"fitReasons"`
	MissingRequirements []string `json:"missingRequirements"`
	Risks               []string `json:"risks"`
}

const matchSystem = `You are a Singapore government procurement analyst helping a supplier decide whether to bid on a tender, ` +
	`given that supplier's company profile and track record. Read the tender and the company profile and produce a factual fit assessment. ` +
	`Do not invent capabilities or requirements that are not present. Score 0-100 reflecting how well the company's profile/products/services/certifications/past projects match what the tender asks for. ` +
	`Use this scoring rubric for priority: score >= 70 is HIGH, 40-69 is MEDIUM, below 40 is LOW. ` +
	`Respond with valid JSON only, matching this shape: ` +
	`{"score": number, "priority": "HIGH"|"MEDIUM"|"LOW", "fitReasons": string[], "missingRequirements": string[], "risks": string[]}. ` +
	`fitReasons should call out concrete reasons the company is/isn't a good fit (industry match, certifications held, relevant past projects). ` +
	`missingRequirements should list certifications/registrations/experience the tender seems to require that aren't evidenced in the company profile. ` +
	`risks should call out bid risks (tight deadline, narrow specs, broad scope, contract size mismatch). Use [] when a section has nothing.`

// generateMatch builds the prompt from the tender row and the caller's onboarding
// profile, then asks the LLM for a structured fit assessment. Returns the parsed
// match and its JSON.
func generateMatch(ctx context.Context, client *llm.Client, t db.Tender, onboarding db.UserOnboarding) (TenderMatch, string, error) {
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

	fmt.Fprintf(&b, "\nCompany profile:\n")
	fmt.Fprintf(&b, "Organisation: %s\n", onboarding.OrgName)
	fmt.Fprintf(&b, "Industry: %s\n", onboarding.Industry)
	fmt.Fprintf(&b, "Account type: %s\n", onboarding.AccountType)
	if onboarding.BusinessDescription != nil && *onboarding.BusinessDescription != "" {
		fmt.Fprintf(&b, "Business description: %s\n", *onboarding.BusinessDescription)
	}
	if onboarding.CompanySize != nil && *onboarding.CompanySize != "" {
		fmt.Fprintf(&b, "Company size: %s\n", *onboarding.CompanySize)
	}
	if onboarding.GeographicCoverage != nil && *onboarding.GeographicCoverage != "" {
		fmt.Fprintf(&b, "Geographic coverage: %s\n", *onboarding.GeographicCoverage)
	}
	if onboarding.PreferredContractSize != nil && *onboarding.PreferredContractSize != "" {
		fmt.Fprintf(&b, "Preferred contract size: %s\n", *onboarding.PreferredContractSize)
	}

	raw, err := client.GenerateJSON(ctx, matchSystem, b.String())
	if err != nil {
		return TenderMatch{}, "", err
	}
	match, err := parseMatch(raw)
	if err != nil {
		return TenderMatch{}, "", err
	}
	out, err := json.Marshal(match)
	if err != nil {
		return TenderMatch{}, "", err
	}
	return match, string(out), nil
}

// generateMatchJSON is generateMatch plus the per-field JSON encodings needed to
// store fitReasons/missingRequirements/risks as separate jsonb columns.
func generateMatchJSON(ctx context.Context, client *llm.Client, t db.Tender, onboarding db.UserOnboarding) (match TenderMatch, fitReasonsJSON string, missingJSON string, risksJSON string, err error) {
	match, _, err = generateMatch(ctx, client, t, onboarding)
	if err != nil {
		return TenderMatch{}, "", "", "", err
	}
	fitReasons, err := json.Marshal(match.FitReasons)
	if err != nil {
		return TenderMatch{}, "", "", "", err
	}
	missing, err := json.Marshal(match.MissingRequirements)
	if err != nil {
		return TenderMatch{}, "", "", "", err
	}
	risks, err := json.Marshal(match.Risks)
	if err != nil {
		return TenderMatch{}, "", "", "", err
	}
	return match, string(fitReasons), string(missing), string(risks), nil
}

// parseMatch unmarshals model output, tolerating a ```json fenced block.
func parseMatch(raw string) (TenderMatch, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(strings.TrimSpace(raw), "```")
	}
	var match TenderMatch
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &match); err != nil {
		return TenderMatch{}, err
	}
	return match, nil
}
