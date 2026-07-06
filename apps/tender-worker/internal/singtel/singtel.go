// Package singtel scrapes Singtel's tenders page. The tender list is embedded as
// an HTML-entity-encoded JSON `datamodel` attribute on the Tender component, and each
// tender's full detail lives on its own page in a rich-text `ux-heading-text` block —
// both parsed with the standard library, same shape as the smrt package.
//
// ponytail: reads Singtel's embedded datamodel JSON; brittle to their CMS swap.
package singtel

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/midtone/tender-worker/internal/store"
)

const listURL = "https://www.singtel.com/about-us/tenders"

var (
	// The list lives on: <div ... component="Tender" datamodel="{…}">
	dataModelRe = regexp.MustCompile(`component="Tender"\s+datamodel="([^"]*)"`)
	// Detail body lives in one or more <ux-heading-text datamodel="{…}"> blocks.
	headingRe = regexp.MustCompile(`<ux-heading-text\s+datamodel="([^"]*)"`)
	// "Samuel Ong at samuel.ong@singtel.com" → name + email.
	contactRe = regexp.MustCompile(`([A-Z][A-Za-z.'-]+(?:\s+[A-Z][A-Za-z.'-]+)*)\s+at\s+([\w.+-]+@[\w.-]+\.[A-Za-z]{2,})`)
	emailRe   = regexp.MustCompile(`[\w.+-]+@[\w.-]+\.[A-Za-z]{2,}`)
)

type details struct {
	Contacts    []store.Contact `json:"contacts"`
	Description string          `json:"description"`
}

type item struct {
	TenderDesc     string `json:"tenderDesc"`
	TenderNumber   string `json:"tenderNumber"`
	TenderPagePath string `json:"tenderPagePath"`
	IssueDate      string `json:"issueDate"`
	ClosingDate    string `json:"closingDate"`
}

// Fetch downloads the tenders page, parses current tenders, then enriches each with
// its per-tender detail page.
func Fetch(ctx context.Context) ([]store.Record, error) {
	body, err := get(ctx, listURL)
	if err != nil {
		return nil, err
	}
	out := parse(body)
	for i := range out {
		link := out[i].Link
		if link == "" {
			continue
		}
		d, err := fetchDetail(ctx, link)
		if err != nil {
			log.Printf("singtel: detail %s: %v", out[i].SourceID, err)
			continue // keep list-level record; details stay unfetched
		}
		out[i].Details = d
		out[i].DetailsFetched = true
	}
	return out, nil
}

// parse extracts list records from the page HTML.
func parse(body []byte) []store.Record {
	m := dataModelRe.FindSubmatch(body)
	if m == nil {
		return nil
	}
	var model struct {
		CurrentTenders []item `json:"currentTenders"`
	}
	if err := json.Unmarshal([]byte(html.UnescapeString(string(m[1]))), &model); err != nil {
		return nil
	}
	var out []store.Record
	for _, it := range model.CurrentTenders {
		ref := strings.TrimSpace(it.TenderNumber)
		if ref == "" {
			continue
		}
		out = append(out, store.Record{
			Source:        "singtel",
			SourceID:      ref,
			ReferenceNo:   ref,
			Title:         strings.TrimSpace(it.TenderDesc),
			Link:          strings.TrimSpace(it.TenderPagePath),
			Agency:        "Singtel",
			PublishedDate: parseDate(it.IssueDate),
			ClosingDate:   parseDate(it.ClosingDate),
		})
	}
	return out
}

// fetchDetail downloads a single tender's page and parses its detail block.
func fetchDetail(ctx context.Context, url string) (details, error) {
	// ponytail: 30s knob so a hung detail page can't wedge the 6h poll.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	body, err := get(ctx, url)
	if err != nil {
		return details{}, err
	}
	return parseDetail(body), nil
}

// parseDetail extracts the description + contacts from a tender detail page.
func parseDetail(body []byte) details {
	var parts []string
	for _, m := range headingRe.FindAllSubmatch(body, -1) {
		var hb struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(html.UnescapeString(string(m[1]))), &hb); err != nil {
			continue
		}
		if t := store.StripHTML(hb.Text); t != "" {
			parts = append(parts, t)
		}
	}
	desc := strings.Join(parts, "\n")
	return details{Contacts: parseContacts(desc), Description: desc}
}

// parseContacts pulls "Name at email" pairs, falling back to bare emails.
func parseContacts(text string) []store.Contact {
	seen := map[string]bool{}
	var out []store.Contact
	for _, m := range contactRe.FindAllStringSubmatch(text, -1) {
		email := strings.ToLower(m[2])
		if seen[email] {
			continue
		}
		seen[email] = true
		out = append(out, store.Contact{Name: strings.TrimSpace(m[1]), Email: email})
	}
	for _, email := range emailRe.FindAllString(text, -1) {
		email = strings.ToLower(email)
		if seen[email] {
			continue
		}
		seen[email] = true
		out = append(out, store.Contact{Email: email})
	}
	return out
}

// get does a GET and returns the response bytes (UA + size cap).
func get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MidtoneTenderBot/1.0)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("singtel: fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("singtel: %s status %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 5<<20))
}

// parseDate handles Singtel's DD/MM/YYYY dates (parsed at SG midnight).
func parseDate(s string) *time.Time {
	if t, err := time.ParseInLocation("02/01/2006", strings.TrimSpace(s), store.SGZone); err == nil {
		return &t
	}
	return nil
}
