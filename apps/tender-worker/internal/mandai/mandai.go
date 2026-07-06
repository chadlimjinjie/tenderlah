// Package mandai scrapes the Mandai Wildlife Group e-procurement portal hosted on
// SESAMi (https://sesami.online/mandai/businessOpportunities.jsp). The listing is
// a plain server-rendered HTML table — one <tr> per opportunity carrying every
// list field plus a documentID UUID. The per-opportunity detail page is fetched
// by POSTing that documentID to businessOpportunityView.jsp (there is no GET URL),
// which adds the "Vendor Criteria / Site Showround / Briefing" block.
//
// ponytail: pure stdlib + regex, same shape as the singtel/sats packages. No JS or
// challenge to solve — the page is static HTML.
package mandai

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/midtone/tender-worker/internal/store"
)

const (
	baseURL   = "https://sesami.online/mandai/"
	listURL   = baseURL + "businessOpportunities.jsp"
	detailURL = baseURL + "businessOpportunityView.jsp"
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)

var (
	rowSep   = "<tr>"
	cellRe   = regexp.MustCompile(`(?s)<td valign="top"[^>]*>(.*?)</td>`)
	docIDRe  = regexp.MustCompile(`viewDetail\('([0-9a-fA-F-]+)'\)`)
	dateTime = []string{"2 Jan 2006 15:04", "02 Jan 2006 15:04"}

	// Detail page: the cell after the "Vendor Criteria…" header cell.
	criteriaRe = regexp.MustCompile(`(?s)Vendor Criteria[^<]*</td>\s*<td[^>]*>(.*?)</td>`)
)

type details struct {
	VendorCriteria string `json:"vendor_criteria,omitempty"`
}

// Fetch downloads the listing, parses each opportunity, then enriches each with its
// POST-only detail page (vendor criteria / briefing notes).
func Fetch(ctx context.Context) ([]store.Record, error) {
	client := &http.Client{}

	body, err := get(ctx, client, listURL)
	if err != nil {
		return nil, err
	}

	out := parse(body)
	for i := range out {
		d, err := fetchDetail(ctx, client, out[i].SourceID)
		if err != nil {
			log.Printf("mandai: detail %s: %v", out[i].SourceID, err)
			continue // keep list-level record; criteria stays unfetched
		}
		out[i].Details = d
		out[i].DetailsFetched = true
	}
	return out, nil
}

// parse extracts list records from the opportunities table.
func parse(body []byte) []store.Record {
	var out []store.Record
	for row := range strings.SplitSeq(string(body), rowSep) {
		dm := docIDRe.FindStringSubmatch(row)
		if dm == nil {
			continue // header row / non-opportunity chunk
		}
		docID := dm[1]

		cells := cellRe.FindAllStringSubmatch(row, -1)
		if len(cells) < 7 {
			continue
		}
		c := func(i int) string { return store.StripHTML(cells[i][1]) }

		desc := c(4)
		out = append(out, store.Record{
			Source:        "mandai",
			SourceID:      docID,
			ReferenceNo:   c(1),
			Title:         desc,
			Link:          listURL, // detail is POST-only; no per-tender GET URL
			Category:      c(3),
			Agency:        c(0),
			TenderType:    c(2),
			Description:   desc,
			PublishedDate: parseDateTime(c(5)),
			ClosingDate:   parseDateTime(c(6)),
		})
	}
	return out
}

// fetchDetail POSTs the documentID and parses the vendor-criteria block.
func fetchDetail(ctx context.Context, client *http.Client, docID string) (details, error) {
	// ponytail: 30s knob so a hung detail page can't wedge the poll.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, detailURL,
		strings.NewReader(url.Values{"documentID": {docID}}.Encode()))
	if err != nil {
		return details{}, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return details{}, fmt.Errorf("mandai: detail %s: %w", docID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return details{}, fmt.Errorf("mandai: detail %s status %d", docID, resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return details{}, err
	}
	return parseDetail(b), nil
}

func parseDetail(body []byte) details {
	var d details
	if m := criteriaRe.FindSubmatch(body); m != nil {
		d.VendorCriteria = store.StripHTML(string(m[1]))
	}
	return d
}

// get does a GET with a browser UA and returns the response bytes (size-capped).
func get(ctx context.Context, client *http.Client, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mandai: fetch %s: %w", u, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mandai: %s status %d", u, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 5<<20))
}

// parseDateTime parses "26 Jun 2026 14:00" at SG time; nil on failure.
func parseDateTime(s string) *time.Time {
	s = strings.TrimSpace(s)
	for _, layout := range dateTime {
		if t, err := time.ParseInLocation(layout, s, store.SGZone); err == nil {
			return &t
		}
	}
	return nil
}
