// Package emservices scrapes EM Services' town-council tenders. The listing is
// served by an Elementor "Unlimited Elements" AJAX endpoint that returns JSON
// with an HTML fragment in `html_items`; each tender is one `.ue-grid-item` div.
// Paged via `ucpage`; we loop until an empty page. Only the open-tenders grid is
// scraped (the awarded "Tender Result" grid is intentionally left out).
//
// ponytail: reads EM Services' Elementor AJAX fragment; brittle to a CMS swap or
// a changed layoutid/elid — re-capture the URL from the tenders page if it breaks.
package emservices

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/midtone/tender-worker/internal/store"
)

// baseURL is the open-tenders AJAX feed; %d is the 1-based page. The layoutid/elid
// and addelids select the open-tenders grid, ordered by advertisement date desc.
const baseURL = "https://www.emservices.com.sg/tenders/?ucfrontajaxaction=getfiltersdata" +
	"&layoutid=289&elid=71ca4f0&ucpage=%d" +
	"&ucorderby=meta&ucorderby_meta=advertisement_date&ucorderby_metatype=date&ucorderdir=desc" +
	"&addelids=f6bfad4,f6bfad4,f6bfad4"

const maxPages = 50 // safety cap; the loop normally stops on the first empty page

var (
	linkRe   = regexp.MustCompile(`data-link="([^"]*)"`)
	agencyRe = regexp.MustCompile(`<span class="elementor-heading-title[^"]*">([^<]+)</span>`)
	titleRe  = regexp.MustCompile(`(?s)<h3[^>]*class="elementor-heading-title[^"]*">(.*?)</h3>`)
	refRe    = regexp.MustCompile(`([A-Za-z0-9-]+)@emservices\.com\.sg`)
	advRe    = regexp.MustCompile(`ADVERTISEMENT DATE:\s*(\d{2}/\d{2}/\d{4})`)
	closeRe  = regexp.MustCompile(`CLOSING DATE:\s*(\d{2}/\d{2}/\d{4})`)
	eligRe   = regexp.MustCompile(`ELIGIBILITY & FINANCIAL GRADES:\s*(.+?)(?:\s+Download|$)`)
	// trailing "(✉ : REF@emservices.com.sg)" tacked onto the title
	titleRefSuffixRe = regexp.MustCompile(`\s*\([^()]*@emservices\.com\.sg\)\s*$`)
)

// Fetch pages through the open-tenders feed until an empty page, returning one
// Record per tender.
func Fetch(ctx context.Context) ([]store.Record, error) {
	var out []store.Record
	for page := 1; page <= maxPages; page++ {
		frag, err := getFragment(ctx, fmt.Sprintf(baseURL, page))
		if err != nil {
			if page == 1 {
				return nil, err
			}
			break // a mid-run page error: keep what we have
		}
		if strings.TrimSpace(frag) == "" {
			break
		}
		recs := parseItems(frag)
		if len(recs) == 0 {
			break
		}
		out = append(out, recs...)
	}
	return out, nil
}

// parseItems splits the fragment on the per-tender delimiter and extracts one
// Record per block. "ue-grid-item" appears only as the item class, so it's a
// safe split token.
func parseItems(fragment string) []store.Record {
	var out []store.Record
	blocks := strings.Split(fragment, "ue-grid-item")
	for _, b := range blocks[1:] { // blocks[0] is preamble before the first item
		text := store.StripHTML(b)

		ref := ""
		if m := refRe.FindStringSubmatch(text); m != nil {
			ref = strings.TrimSpace(m[1])
		}
		if ref == "" {
			continue // no stable id → skip
		}

		title := ""
		if m := titleRe.FindStringSubmatch(b); m != nil {
			title = titleRefSuffixRe.ReplaceAllString(store.StripHTML(m[1]), "")
		}
		agency := ""
		if m := agencyRe.FindStringSubmatch(b); m != nil {
			agency = store.StripHTML(m[1])
		}
		link := ""
		if m := linkRe.FindStringSubmatch(b); m != nil {
			link = store.StripHTML(m[1])
		}
		desc := ""
		if m := eligRe.FindStringSubmatch(text); m != nil {
			desc = strings.TrimSpace(m[1])
		}

		out = append(out, store.Record{
			Source:        "emservices",
			SourceID:      ref,
			ReferenceNo:   ref,
			Title:         title,
			Link:          link,
			Agency:        agency,
			Status:        "open",
			Description:   desc,
			PublishedDate: parseDate(advRe, text),
			ClosingDate:   parseDate(closeRe, text),
		})
	}
	return out
}

// parseDate applies re to text and parses the captured DD/MM/YYYY at SG midnight.
func parseDate(re *regexp.Regexp, text string) *time.Time {
	m := re.FindStringSubmatch(text)
	if m == nil {
		return nil
	}
	if t, err := time.ParseInLocation("02/01/2006", m[1], store.SGZone); err == nil {
		return &t
	}
	return nil
}

// getFragment GETs a page and returns the JSON `html_items` fragment.
func getFragment(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MidtoneTenderBot/1.0)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("emservices: fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("emservices: %s status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return "", err
	}
	var payload struct {
		HTMLItems string `json:"html_items"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("emservices: decode %s: %w", url, err)
	}
	return payload.HTMLItems, nil
}
