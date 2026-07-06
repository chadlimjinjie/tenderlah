// Package sats scrapes SATS's tenders page. The "Live Tenders" list is an Adobe
// Experience Manager (AEM) accordion — each tender is a cmp-accordion__item whose
// title carries the reference code and whose panel links to a per-tender detail
// page; the detail page holds the description, contact emails, and document links.
// Parsed with the standard library, same shape as the singtel package.
//
// SATS sits behind Akamai Bot Manager's SEC-CPT "crypto" challenge: a plain GET
// returns a "Challenge Validation" interstitial, not tender HTML. solveChallenge
// (see below) replicates Akamai's SHA-256 proof-of-work in pure Go, waits the
// server-enforced duration, POSTs the answers to the verify_url for a cleared
// sec_cpt cookie, and re-fetches.
//
// ponytail: this is brittle by design — Akamai rotates paths/obfuscation and can
// change the algorithm at any time, and we only solve the SEC-CPT interstitial
// (not the _abck sensor). When it breaks, Fetch returns an error → the source is
// logged + skipped, yielding 0 records (no crash, others unaffected). Upgrade
// path if it stops working: a headless browser that lets Akamai solve its own JS.
package sats

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
	"time"

	"github.com/midtone/tender-worker/internal/store"
)

// A real browser UA — Akamai serves harder challenges (or refuses) to obvious bots.
const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

const (
	baseURL = "https://www.sats.com.sg"
	listURL = baseURL + "/tenders"
	// ponytail: cap chained Akamai challenge rounds so a never-clearing config can't
	// loop forever; each round costs the server-enforced ~10s wait.
	maxChallengeRounds = 6
)

var (
	// Each tender is a cmp-accordion__item; split the page on it.
	itemSep = `class="cmp-accordion__item"`
	titleRe = regexp.MustCompile(`(?s)class="cmp-accordion__title"[^>]*>(.*?)</span>`)
	// Detail link: /tenders/<slug>, optionally with the AEM /content prefix or .html.
	linkRe = regexp.MustCompile(`(?i)href="((?:https://www\.sats\.com\.sg)?/(?:content/sats/corporate/en/)?tenders/[^"#]+)"`)
	// Reference code, e.g. CW2606I380 / CT2605Z078 (two letters, 4 digits, letter, 3 digits).
	codeRe = regexp.MustCompile(`[A-Za-z]{2}\d{4}[A-Za-z]\d{3}`)
	// Free-text closing date: "Closing Date: 15 July 2026", "Closing on 10 Jul 2025", "1 st September 2025".
	closeRe = regexp.MustCompile(`(?i)Clos(?:ing|e)[^<\d]{0,40}(\d{1,2})\s*(?:st|nd|rd|th)?\s+([A-Za-z]{3,9})\s+(\d{4})`)

	// Detail page selectors.
	textRe   = regexp.MustCompile(`(?s)class="cmp-text"[^>]*>(.*?)</div>`)
	mailtoRe = regexp.MustCompile(`(?i)mailto:([\w.+-]+@[\w.-]+\.[A-Za-z]{2,})`)
	docRe    = regexp.MustCompile(`href="(/content/dam/[^"]+)"`)

	// Akamai SEC-CPT challenge: base64 JSON in challenge="…" + a server-enforced data-duration.
	challengeRe = regexp.MustCompile(`challenge="([^"]+)"`)
	durationRe  = regexp.MustCompile(`data-duration=(\d+)`)
)

type details struct {
	Contacts    []store.Contact `json:"contacts"`
	Description string          `json:"description"`
	Attachments []string        `json:"attachments,omitempty"`
}

// Fetch downloads the tenders page (clearing the Akamai challenge first if
// present), parses each tender, then enriches each with its per-tender detail
// page (description + contacts + attachments).
func Fetch(ctx context.Context) ([]store.Record, error) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	body, err := get(ctx, client, listURL)
	if err != nil {
		return nil, err
	}
	// Akamai chains SEC-CPT challenges: each solved round increments the sec_cpt
	// counter and the reload serves the next, until the page clears. Cap the rounds
	// so a never-clearing config (e.g. _abck sensor) can't loop forever.
	for round := 0; isChallenge(body); round++ {
		if round >= maxChallengeRounds {
			return nil, fmt.Errorf("sats: still challenged after %d rounds (abck sensor enforced?)", maxChallengeRounds)
		}
		if err := solveChallenge(ctx, client, body); err != nil {
			return nil, fmt.Errorf("sats: akamai challenge: %w", err)
		}
		if body, err = get(ctx, client, listURL); err != nil {
			return nil, err
		}
	}

	out := parse(body)
	for i := range out {
		link := out[i].Link
		if link == "" {
			continue
		}
		d, err := fetchDetail(ctx, client, link)
		if err != nil {
			log.Printf("sats: detail %s: %v", out[i].SourceID, err)
			continue // keep list-level record; details stay unfetched
		}
		out[i].Details = d
		out[i].Description = d.Description
		out[i].DetailsFetched = true
	}
	return out, nil
}

// parse extracts list records from the accordion on the page HTML.
func parse(body []byte) []store.Record {
	items := strings.Split(string(body), itemSep)
	var out []store.Record
	for _, c := range items[1:] { // items[0] is everything before the first item
		title := text(titleRe, c)
		link := normalizeLink(first(linkRe, c))
		_, seg, _ := strings.Cut(link, "/tenders/")
		// Prefer the code from the URL slug; fall back to the title.
		ref := codeRe.FindString(seg)
		if ref == "" {
			ref = codeRe.FindString(title)
		}
		if ref == "" {
			continue // no stable id — skip
		}
		ref = strings.ToUpper(ref)
		out = append(out, store.Record{
			Source:      "sats",
			SourceID:    ref,
			ReferenceNo: ref,
			Title:       title,
			Link:        link,
			Agency:      "SATS",
			ClosingDate: parseClosing(store.StripHTML(c)),
		})
	}
	return out
}

// fetchDetail downloads a single tender's page and parses its detail block.
func fetchDetail(ctx context.Context, client *http.Client, url string) (details, error) {
	// ponytail: 30s knob so a hung detail page can't wedge the poll.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	body, err := get(ctx, client, url)
	if err != nil {
		return details{}, err
	}
	return parseDetail(body), nil
}

// parseDetail extracts description + contacts + attachments from a detail page,
// scoped to the main content (between the breadcrumb and the footer fragment) so
// the global nav/footer text is ignored.
func parseDetail(body []byte) details {
	s := string(body)
	if a := strings.Index(s, "cmp-breadcrumb"); a >= 0 {
		if b := strings.Index(s[a:], "cmp-experiencefragment--footer"); b >= 0 {
			s = s[a : a+b]
		}
	}

	var parts []string
	for _, m := range textRe.FindAllStringSubmatch(s, -1) {
		if t := store.StripHTML(m[1]); len(t) > 15 {
			parts = append(parts, t)
		}
	}
	desc := strings.Join(parts, "\n")

	seen := map[string]bool{}
	var contacts []store.Contact
	for _, m := range mailtoRe.FindAllStringSubmatch(s, -1) {
		email := strings.ToLower(m[1])
		if seen[email] {
			continue
		}
		seen[email] = true
		contacts = append(contacts, store.Contact{Email: email})
	}

	dseen := map[string]bool{}
	var docs []string
	for _, m := range docRe.FindAllStringSubmatch(s, -1) {
		u := baseURL + m[1]
		if dseen[u] {
			continue
		}
		dseen[u] = true
		docs = append(docs, u)
	}

	return details{Contacts: contacts, Description: desc, Attachments: docs}
}

// normalizeLink turns any tender href into an absolute https URL, dropping the
// AEM /content prefix and the .html suffix.
func normalizeLink(href string) string {
	if href == "" {
		return ""
	}
	href = strings.TrimSuffix(href, ".html")
	href = strings.Replace(href, "/content/sats/corporate/en/", "/", 1)
	if strings.HasPrefix(href, "/") {
		href = baseURL + href
	}
	return href
}

// get does a GET on the (cookie-jar) client and returns the response bytes (UA + size cap).
func get(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	setBrowserHeaders(req)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sats: fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sats: %s status %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 5<<20))
}

// setBrowserHeaders mimics Chrome's request headers so Akamai's header fingerprint
// doesn't flag the request as a bot. (TLS/JA3 fingerprint is a separate matter.)
func setBrowserHeaders(req *http.Request) {
	h := req.Header
	h.Set("User-Agent", userAgent)
	h.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	h.Set("Accept-Language", "en-US,en;q=0.9")
	h.Set("sec-ch-ua", `"Chromium";v="124", "Google Chrome";v="124", "Not-A.Brand";v="99"`)
	h.Set("sec-ch-ua-mobile", "?0")
	h.Set("sec-ch-ua-platform", `"Windows"`)
	h.Set("Sec-Fetch-Dest", "document")
	h.Set("Sec-Fetch-Mode", "navigate")
	h.Set("Sec-Fetch-Site", "none")
	h.Set("Sec-Fetch-User", "?1")
	h.Set("Upgrade-Insecure-Requests", "1")
}

// --- Akamai SEC-CPT crypto challenge solver (see package doc) ---

type challenge struct {
	Token      string `json:"token"`
	Timestamp  int64  `json:"timestamp"`
	Nonce      string `json:"nonce"`
	Difficulty int64  `json:"difficulty"`
	Count      int    `json:"count"`
	VerifyURL  string `json:"verify_url"`
}

// isChallenge reports whether body is the Akamai "Challenge Validation" interstitial.
func isChallenge(body []byte) bool {
	return bytes.Contains(body, []byte("Challenge Validation"))
}

// solveChallenge runs the proof-of-work, waits the enforced duration, and POSTs the
// answers to the verify_url so the client's jar receives a cleared sec_cpt cookie.
func solveChallenge(ctx context.Context, client *http.Client, body []byte) error {
	ch, duration, err := parseChallenge(body)
	if err != nil {
		return err
	}

	// sec = the part of the sec_cpt cookie before the first '~' (set on the first GET).
	sec := ""
	if u, err := http.NewRequest(http.MethodGet, listURL, nil); err == nil {
		for _, c := range client.Jar.Cookies(u.URL) {
			if c.Name == "sec_cpt" {
				sec, _, _ = strings.Cut(c.Value, "~")
			}
		}
	}
	if sec == "" {
		return fmt.Errorf("no sec_cpt cookie")
	}

	answers, err := solveCrypto(sec, ch)
	if err != nil {
		return err
	}

	// The wait is enforced server-side and cannot be shortened.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(duration) * time.Second):
	}

	payload, _ := json.Marshal(struct {
		Token   string   `json:"token"`
		Answers []string `json:"answers"`
	}{ch.Token, answers})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ch.VerifyURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	setBrowserHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", listURL)
	req.Header.Set("Origin", baseURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("verify status %d", resp.StatusCode)
	}
	return nil
}

// parseChallenge extracts the SEC-CPT challenge JSON and the enforced wait duration.
func parseChallenge(body []byte) (challenge, int, error) {
	var ch challenge
	m := challengeRe.FindSubmatch(body)
	if m == nil {
		return ch, 0, fmt.Errorf("challenge attr not found")
	}
	raw, err := base64.StdEncoding.DecodeString(string(m[1]))
	if err != nil {
		return ch, 0, fmt.Errorf("challenge b64: %w", err)
	}
	if err := json.Unmarshal(raw, &ch); err != nil {
		return ch, 0, fmt.Errorf("challenge json: %w", err)
	}
	if ch.Difficulty <= 0 || ch.Count <= 0 || ch.Token == "" || ch.VerifyURL == "" {
		return ch, 0, fmt.Errorf("incomplete challenge")
	}
	duration := 0
	if d := durationRe.FindSubmatch(body); d != nil {
		fmt.Sscan(string(d[1]), &duration)
	}
	return ch, duration, nil
}

// solveCrypto finds `count` proof-of-work answers. For each answer it searches random
// candidates until sha256(sec+timestamp+nonce+difficulty+answer) ≡ 0 (mod difficulty),
// computed as a running modulus so no big integers are needed; difficulty increments
// per accepted answer (each is progressively harder), matching Akamai's algorithm.
func solveCrypto(sec string, ch challenge) ([]string, error) {
	answers := make([]string, 0, ch.Count)
	difficulty := ch.Difficulty
	buf := make([]byte, 8)
	for len(answers) < ch.Count {
		if _, err := rand.Read(buf); err != nil {
			return nil, err
		}
		answer := "0." + hex.EncodeToString(buf)
		input := fmt.Sprintf("%s%d%s%d%s", sec, ch.Timestamp, ch.Nonce, difficulty, answer)
		sum := sha256.Sum256([]byte(input))
		var out int64
		for _, b := range sum {
			out = (out<<8 | int64(b)) % difficulty
		}
		if out == 0 {
			answers = append(answers, answer)
			difficulty++
		}
	}
	return answers, nil
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

// parseClosing pulls the closing date out of free-text and parses it at SG midnight.
// Time-of-day ("1200hrs") is intentionally dropped — the closing date is what matters.
func parseClosing(s string) *time.Time {
	m := closeRe.FindStringSubmatch(s)
	if m == nil {
		return nil
	}
	v := m[1] + " " + m[2] + " " + m[3] // "15 July 2026" or "10 Jul 2025"
	for _, layout := range []string{"2 January 2006", "2 Jan 2006"} {
		if t, err := time.ParseInLocation(layout, v, store.SGZone); err == nil {
			return &t
		}
	}
	return nil
}
