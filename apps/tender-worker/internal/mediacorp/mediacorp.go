// Package mediacorp scrapes Mediacorp's tender-notices listing. It's a Drupal view:
// each tender is a views-row with a title+link, a brief description, and an ISO
// closing date. Parsed with the standard library, same shape as the sit package.
//
// Mediacorp sits behind Incapsula but serves this page to a plain browser-UA GET, so
// no challenge solver is needed. isChallenge guards the rare case where Incapsula
// blocks the worker's IP: Fetch then errors → the source is logged + skipped (0
// records). ponytail: if it starts challenging, sit.solveIncapsula is the upgrade path.
package mediacorp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/midtone/tender-worker/internal/store"
)

const (
	baseURL = "https://www.mediacorp.sg"
	listURL = baseURL + "/suppliers/tender-notices?status=1"
)

var (
	rowSep  = `class="views-row"`
	linkRe  = regexp.MustCompile(`href="(/tender-notices/[^"#]+)"`)
	titleRe = regexp.MustCompile(`(?s)views-field-title.*?<a[^>]*>(.*?)</a>`)
	briefRe = regexp.MustCompile(`(?s)views-field-field-brief.*?field-content">(.*?)</div>`)
	timeRe  = regexp.MustCompile(`datetime="([^"]+)"`)
	refRe   = regexp.MustCompile(`(?i)TENDER NO:\s*(\d+)`)
	nodeRe  = regexp.MustCompile(`-(\d+)$`) // trailing node id of the slug, fallback ref
)

// Fetch downloads the listing and parses each tender row.
func Fetch(ctx context.Context) ([]store.Record, error) {
	body, err := get(ctx, listURL)
	if err != nil {
		return nil, err
	}
	if isChallenge(body) {
		return nil, fmt.Errorf("mediacorp: incapsula challenge (worker IP blocked?)")
	}
	return parse(body), nil
}

// parse extracts tender records from the Drupal view rows.
func parse(body []byte) []store.Record {
	rows := strings.Split(string(body), rowSep)
	var out []store.Record
	for _, r := range rows[1:] { // rows[0] is everything before the first row
		link := ""
		if m := linkRe.FindStringSubmatch(r); m != nil {
			link = baseURL + m[1]
		}
		ref := ""
		if m := refRe.FindStringSubmatch(r); m != nil {
			ref = m[1]
		} else if m := nodeRe.FindStringSubmatch(link); m != nil {
			ref = m[1]
		}
		if ref == "" {
			continue // no stable id
		}
		var closing *time.Time
		if m := timeRe.FindStringSubmatch(r); m != nil {
			closing = parseTime(m[1])
		}
		out = append(out, store.Record{
			Source:      "mediacorp",
			SourceID:    ref,
			ReferenceNo: ref,
			Title:       text(titleRe, r),
			Link:        link,
			Agency:      "Mediacorp Pte Ltd",
			Description: text(briefRe, r),
			ClosingDate: closing,
		})
	}
	return out
}

// isChallenge reports whether body is the Incapsula stub rather than the real listing.
func isChallenge(body []byte) bool {
	return bytes.Contains(body, []byte("_Incapsula_Resource")) && !bytes.Contains(body, []byte("views-row"))
}

func parseTime(s string) *time.Time {
	if t, err := time.Parse(time.RFC3339, strings.TrimSpace(s)); err == nil {
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

// get does a GET with a browser UA and returns the body (size-capped).
func get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mediacorp: fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mediacorp: %s status %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 5<<20))
}
