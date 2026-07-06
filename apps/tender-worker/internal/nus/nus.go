// Package nus scrapes NUS's business-opportunities listing. The ProjectListing
// endpoint is an AJAX handler: a POST with form fields `status` and `page` returns
// an HTML fragment (a <table> of opportunities + a pagination block), not JSON.
// Parsed with the standard library, same shape as the mediacorp package.
//
// We POST status=Open so only open opportunities come back, and walk every page:
// the pagination block's highest data-page-number is the last page (out-of-range
// pages clamp to it rather than erroring), so we fetch 1..max.
package nus

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/midtone/tender-worker/internal/store"
)

const listURL = "https://www.nus.edu.sg/suppliers/business-opportunities/ProjectListing/"

// ponytail: cap pages so a malformed pagination block can't loop forever.
const maxPages = 50

var (
	rowRe      = regexp.MustCompile(`(?s)<th scope="row">([^<]+)</th>(.*?)</tr>`)
	linkRe     = regexp.MustCompile(`<a href="([^"]+)"`)
	titleRe    = regexp.MustCompile(`(?s)text-decoration-none">(.*?)(?:<img|</a>)`)
	catRe      = regexp.MustCompile(`(?s)<b>Category:</b>\s*([^<]*)`)
	pubRe      = regexp.MustCompile(`class="second-td">([^<]*)</td>`)
	closeRe    = regexp.MustCompile(`class="third-td">([^<]*)</td>`)
	statusRe   = regexp.MustCompile(`class="fourth-td">([^<]*)</td>`)
	pageNumRe  = regexp.MustCompile(`data-page-number="(\d+)"`)
	dateLayout = "02/01/2006 15:04"
)

// Fetch POSTs each page of open opportunities and parses every row.
func Fetch(ctx context.Context) ([]store.Record, error) {
	body, err := post(ctx, 1)
	if err != nil {
		return nil, err
	}
	out := parse(body)
	last := lastPage(body)
	for p := 2; p <= last && p <= maxPages; p++ {
		body, err := post(ctx, p)
		if err != nil {
			return nil, err
		}
		out = append(out, parse(body)...)
	}
	return out, nil
}

// parse extracts open-tender records from the listing fragment.
func parse(body []byte) []store.Record {
	var out []store.Record
	for _, m := range rowRe.FindAllStringSubmatch(string(body), -1) {
		ref := strings.TrimSpace(m[1])
		cells := m[2]
		if !strings.EqualFold(text(statusRe, cells), "Open") {
			continue // belt-and-braces: we only requested Open
		}
		if ref == "" {
			continue // no stable id
		}
		out = append(out, store.Record{
			Source:        "nus",
			SourceID:      ref,
			ReferenceNo:   ref,
			Title:         text(titleRe, cells),
			Link:          first(linkRe, cells),
			Category:      text(catRe, cells),
			Agency:        "National University of Singapore",
			TenderType:    "ITQ/ITT",
			Status:        "Open",
			PublishedDate: parseDate(text(pubRe, cells)),
			ClosingDate:   parseDate(text(closeRe, cells)),
		})
	}
	return out
}

// lastPage returns the highest page number in the pagination block (1 if none).
func lastPage(body []byte) int {
	last := 1
	for _, m := range pageNumRe.FindAllStringSubmatch(string(body), -1) {
		if n, err := strconv.Atoi(m[1]); err == nil && n > last {
			last = n
		}
	}
	return last
}

// post submits the form for one page and returns the HTML fragment (size-capped).
func post(ctx context.Context, page int) ([]byte, error) {
	form := url.Values{"status": {"Open"}, "page": {strconv.Itoa(page)}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, listURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nus: fetch page %d: %w", page, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nus: page %d status %d", page, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 5<<20))
}

// parseDate parses "DD/MM/YYYY HH:MM" at SG time; returns nil if absent/unparseable.
func parseDate(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if t, err := time.ParseInLocation(dateLayout, s, store.SGZone); err == nil {
		return &t
	}
	return nil
}

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
