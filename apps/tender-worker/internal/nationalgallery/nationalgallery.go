// Package nationalgallery pulls National Gallery Singapore's tender notices. The
// tenders page embeds a third-party "flexsolver" iframe whose JS loads the list from
// a plain JSON API — so this is an API client, not an HTML scraper. We hit that API
// directly and map each record to store.Record.
package nationalgallery

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
	apiBase = "https://apim.sg.workato.com/nationalgallerysg/rfp-v1"
	apiURL  = apiBase + "/rfps/query/tender"
	// Public client token shipped in the page's JS (flexsolver assets/main.js) — not a secret.
	apiToken = "7255ee905f79c172c71f720c04bccff3709b776bee4ba0d9bbd7a9e7e6ad5705"
	pageURL  = "https://www.nationalgallery.sg/sg/en/about/business-opportunities/tenders.html"
)

type apiTender struct {
	ReferenceNumber string `json:"reference_number"`
	Title           string `json:"title"`
	Details         string `json:"rfp_tender_details"`
	ClosingDatetime string `json:"closing_datetime"`
	RecordID        string `json:"record_id"`
	Attachment      string `json:"attachment"`
}

type details struct {
	RecordID      string `json:"record_id"`
	Attachment    string `json:"attachment,omitempty"`
	AttachmentURL string `json:"attachment_url,omitempty"`
}

// Fetch queries the tender API and maps each record to a store.Record.
func Fetch(ctx context.Context) ([]store.Record, error) {
	body, err := get(ctx, apiURL)
	if err != nil {
		return nil, err
	}
	return parse(body)
}

// parse decodes the API body into records.
func parse(body []byte) ([]store.Record, error) {
	var resp struct {
		Data []apiTender `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("nationalgallery: decode: %w", err)
	}
	out := make([]store.Record, 0, len(resp.Data))
	for _, t := range resp.Data {
		if t.ReferenceNumber == "" {
			continue // no stable id
		}
		d := details{RecordID: t.RecordID, Attachment: t.Attachment}
		if t.RecordID != "" && t.Attachment != "" {
			d.AttachmentURL = apiBase + "/rfps/" + t.RecordID + "/attachment"
		}
		out = append(out, store.Record{
			Source:         "nationalgallery",
			SourceID:       t.ReferenceNumber,
			ReferenceNo:    t.ReferenceNumber,
			Title:          t.Title,
			Link:           pageURL,
			Agency:         "National Gallery Singapore",
			ClosingDate:    parseTime(t.ClosingDatetime),
			Description:    t.Details,
			Details:        d,
			DetailsFetched: true,
		})
	}
	return out, nil
}

// parseTime parses the API's ISO8601 closing time (absolute UTC); nil on failure.
func parseTime(s string) *time.Time {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return &t
	}
	return nil
}

// get does a GET with the api-token header and returns the body (UA + size cap).
func get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MidtoneTenderBot/1.0)")
	req.Header.Set("api-token", apiToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nationalgallery: fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nationalgallery: %s status %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 5<<20))
}
