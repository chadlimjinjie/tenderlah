// Package ahtc scrapes AHTC's (Aljunied-Hougang Town Council) tender-advertisements
// page. The page renders its list client-side via a WordPress admin-ajax call that
// needs a per-page nonce, so we scrape the nonce from the page, POST for each page of
// results, and parse the returned HTML cards.
//
// ponytail: scrapes a WP nonce + parses rendered ahtc-tender cards by regex; brittle
// to AHTC's markup, the known ceiling. Upgrade to goquery only if the markup churns.
package ahtc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/midtone/tender-worker/internal/store"
)

const (
	listURL = "https://www.ahtc.sg/tender-advertisements/"
	ajaxURL = "https://www.ahtc.sg/wp-admin/admin-ajax.php"
	maxPage = 20 // ponytail: cap so a markup change can't loop forever
)

var (
	// The nonce sits right after the action in the page's inline ajax script.
	nonceRe = regexp.MustCompile(`ahtc_filter_tenders"[\s\S]{0,80}?nonce:\s*"([0-9a-f]+)"`)
	cardRe  = regexp.MustCompile(`ahtc-tender-card`)
	noRe    = regexp.MustCompile(`(?s)class="ahtc-tender-no">\s*(.*?)\s*</div>`)
	catRe   = regexp.MustCompile(`(?s)class="ahtc-tender-category">\s*(.*?)\s*</div>`)
	titleRe = regexp.MustCompile(`(?s)class="ahtc-tender-title">(.*?)</h3>`)
	openRe  = regexp.MustCompile(`(?s)class="ahtc-tender-open">.*?</span>\s*([^<]+?)\s*</div>`)
	closeRe = regexp.MustCompile(`(?s)class="ahtc-tender-close">.*?</span>\s*([^<]+?)\s*</div>`)
	linkRe  = regexp.MustCompile(`ahtc-tender-download-btn"\s+href="([^"]+)"`)
	refCutR = regexp.MustCompile(`(?i)^contract ref:\s*`)
)

// Fetch scrapes the listing nonce, then pages through the ajax results.
func Fetch(ctx context.Context) ([]store.Record, error) {
	page, err := get(ctx, listURL, nil)
	if err != nil {
		return nil, err
	}
	m := nonceRe.FindSubmatch(page)
	if m == nil {
		return nil, fmt.Errorf("ahtc: nonce not found on listing page")
	}
	nonce := string(m[1])

	var out []store.Record
	for p := 1; p <= maxPage; p++ {
		form := url.Values{
			"action":      {"ahtc_filter_tenders"},
			"nonce":       {nonce},
			"page":        {fmt.Sprint(p)},
			"category_id": {"0"},
		}
		body, err := get(ctx, ajaxURL, strings.NewReader(form.Encode()))
		if err != nil {
			return out, err
		}
		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				HTML string `json:"html"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return out, fmt.Errorf("ahtc: decode page %d: %w", p, err)
		}
		recs := parseCards(resp.Data.HTML)
		if len(recs) == 0 {
			break // "No tenders found." — past the last page
		}
		out = append(out, recs...)
	}
	return out, nil
}

// get does a GET (body nil) or form POST and returns the response bytes.
func get(ctx context.Context, u string, body io.Reader) ([]byte, error) {
	method := http.MethodGet
	if body != nil {
		method = http.MethodPost
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MidtoneTenderBot/1.0)")
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ahtc: fetch %s: %w", u, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ahtc: %s status %d", u, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 5<<20))
}

// parseCards splits the rendered grid into cards and extracts each tender.
func parseCards(htmlStr string) []store.Record {
	parts := cardRe.Split(htmlStr, -1)
	var out []store.Record
	for _, c := range parts[1:] { // parts[0] is before the first card
		ref := refCutR.ReplaceAllString(text(noRe, c), "")
		if ref == "" {
			continue
		}
		out = append(out, store.Record{
			Source:        "ahtc",
			SourceID:      ref,
			ReferenceNo:   ref,
			Title:         text(titleRe, c),
			Category:      text(catRe, c),
			Link:          first(linkRe, c),
			Agency:        "AHTC",
			PublishedDate: parseDate(text(openRe, c)),
			ClosingDate:   parseDate(text(closeRe, c)),
		})
	}
	return out
}

// text pulls a capture group and cleans HTML out of it.
func text(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return store.StripHTML(m[1])
}

func first(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// parseDate handles AHTC's "18 Jun 2026" dates (parsed at SG midnight).
func parseDate(s string) *time.Time {
	if t, err := time.ParseInLocation("02 Jan 2006", strings.TrimSpace(s), store.SGZone); err == nil {
		return &t
	}
	return nil
}
