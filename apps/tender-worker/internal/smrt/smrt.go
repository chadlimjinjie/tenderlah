// Package smrt scrapes SMRT's tender-notices page. Each accordion widget embeds a
// JSON `data-model` attribute with the tender list, so no DOM walking is needed.
//
// ponytail: reads SMRT's embedded data-model JSON; brittle to their CMS swap.
package smrt

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/midtone/tender-worker/internal/store"
)

const listURL = "https://www.smrt.com.sg/tender-notices/"

var (
	// data-model lives on the accordion widget: widget-type='AccordionWithImage' data-model="{…}"
	dataModelRe = regexp.MustCompile(`widget-type='AccordionWithImage'\s+data-model="([^"]*)"`)
	refRe       = regexp.MustCompile(`(?i)\bfor\s+(\S.*\S|\S)$`)
)

type details struct {
	Contacts    []store.Contact `json:"contacts"`
	Description string          `json:"description"`
}

type item struct {
	Title       string   `json:"title"`
	Project     string   `json:"project"`
	PublishedOn string   `json:"publishedOn"`
	ClosingDate string   `json:"closingDate"`
	Persons     []string `json:"persons"`
	Description string   `json:"description"`
}

// Fetch downloads the listing page and parses its tenders.
func Fetch(ctx context.Context) ([]store.Record, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MidtoneTenderBot/1.0)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("smrt: fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("smrt: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, err
	}
	return parse(body), nil
}

// parse extracts records from the page HTML.
func parse(body []byte) []store.Record {
	var out []store.Record
	for _, m := range dataModelRe.FindAllSubmatch(body, -1) {
		var model struct {
			Items []item `json:"items"`
		}
		if err := json.Unmarshal([]byte(html.UnescapeString(string(m[1]))), &model); err != nil {
			continue
		}
		for _, it := range model.Items {
			ref := parseRef(it.Title)
			if ref == "" {
				continue
			}
			out = append(out, store.Record{
				Source:         "smrt",
				SourceID:       ref,
				ReferenceNo:    ref,
				Title:          strings.TrimSpace(it.Project),
				Link:           listURL,
				Agency:         "SMRT",
				PublishedDate:  parseTime(it.PublishedOn),
				ClosingDate:    parseTime(it.ClosingDate),
				Description:    store.StripHTML(it.Description),
				Details:        details{Contacts: parsePersons(it.Persons), Description: store.StripHTML(it.Description)},
				DetailsFetched: true,
			})
		}
	}
	return out
}

func parseRef(title string) string {
	if m := refRe.FindStringSubmatch(title); m != nil {
		return strings.TrimSpace(m[1])
	}
	return strings.TrimSpace(title)
}

func parseTime(s string) *time.Time {
	if t, err := time.ParseInLocation("2006-01-02T15:04:05", s, store.SGZone); err == nil {
		return &t
	}
	return nil
}

func parsePersons(persons []string) []store.Contact {
	var out []store.Contact
	for _, p := range persons {
		f := strings.Split(p, ";")
		c := store.Contact{}
		if len(f) > 0 {
			c.Name = strings.TrimSpace(f[0])
		}
		if len(f) > 1 {
			c.Email = strings.TrimSpace(f[1])
		}
		if len(f) > 2 {
			c.Phone = strings.TrimSpace(f[2])
		}
		if c.Name != "" || c.Email != "" {
			out = append(out, c)
		}
	}
	return out
}
