// Package tuaspower pulls Tuas Power's active tender notices. The eTender portal is
// a Vue SPA that loads its list from a public JSON API; this hits that API directly
// (no auth, no HTML scraping). The portal host is behind Incapsula but the API host
// (prodapi-etender) serves JSON to a plain GET.
package tuaspower

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/midtone/tender-worker/internal/store"
)

const (
	apiURL  = "https://prodapi-etender.tuaspower.com.sg/tenders/active_tender_notices"
	portal  = "https://etender.tuaspower.com.sg"
	origin  = portal
)

// MongoDB extended-JSON wrappers used by the API.
type mdate struct {
	Date string `json:"$date"`
}

type apiTender struct {
	Guid         string `json:"guid"`
	TenderNumber string `json:"tendernumber"`
	Title        string `json:"title"`       // HTML — the full invitation
	Description  string `json:"description"` // HTML — the concise subject
	Status       string `json:"status"`
	OpenDate     mdate  `json:"tenderopendate"`
	CloseDate    mdate  `json:"tenderclosedate"`
}

// Fetch queries the active-tenders API and maps each record to a store.Record.
func Fetch(ctx context.Context) ([]store.Record, error) {
	body, err := get(ctx, apiURL)
	if err != nil {
		return nil, err
	}
	return parse(body)
}

// parse decodes the API array into records.
func parse(body []byte) ([]store.Record, error) {
	var items []apiTender
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("tuaspower: decode: %w", err)
	}
	out := make([]store.Record, 0, len(items))
	for _, t := range items {
		ref := t.TenderNumber
		if ref == "" {
			ref = t.Guid // fallback stable id
		}
		if ref == "" {
			continue
		}
		link := ""
		if t.Guid != "" {
			link = portal + "/tendernotice/" + t.Guid
		}
		out = append(out, store.Record{
			Source:        "tuaspower",
			SourceID:      ref,
			ReferenceNo:   ref,
			Title:         store.StripHTML(t.Description), // concise subject
			Link:          link,
			Agency:        "Tuas Power",
			Status:        t.Status,
			Description:   store.StripHTML(t.Title), // full invitation text
			PublishedDate: parseTime(t.OpenDate.Date),
			ClosingDate:   parseTime(t.CloseDate.Date),
		})
	}
	return out, nil
}

func parseTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t
	}
	return nil
}

// get does a GET with a browser UA + the portal Origin/Referer and returns the body.
func get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", origin)
	req.Header.Set("Referer", portal+"/")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tuaspower: fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tuaspower: %s status %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 5<<20))
}
