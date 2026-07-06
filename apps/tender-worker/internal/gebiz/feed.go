// Package gebiz fetches GeBIZ tender-opportunity RSS feeds and parses each
// <item> into a Tender ready to upsert into Postgres.
package gebiz

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/midtone/tender-worker/internal/store"
)

// Tender is one parsed RSS item. Date fields are pointers so a parse failure
// leaves them null without dropping the row.
type Tender struct {
	OpportunityID string
	ReferenceNo   string
	TenderType    string
	Title         string
	Link          string
	Category      string
	CallingEntity string
	PublishedDate *time.Time
	ClosingDate   *time.Time
	Status        string
	Description   string
}

// XML shape: rss > channel > item{title,link,guid,description}.
type rssDoc struct {
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
		} `xml:"item"`
	} `xml:"channel"`
}

// CategoryFromURL pulls "Professional_Services" out of
// ".../rss/Professional_Services-CREATE_BO_FEED.xml".
func CategoryFromURL(feedURL string) string {
	last := feedURL
	if i := strings.LastIndex(feedURL, "/"); i >= 0 {
		last = feedURL[i+1:]
	}
	return strings.ReplaceAll(
		strings.TrimSuffix(strings.SplitN(last, "-CREATE_BO_FEED", 2)[0], "-"),
		"_", " ")
}

// Fetch downloads and parses one feed into Tenders, tagging each with category.
func Fetch(ctx context.Context, feedURL string) ([]Tender, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gebiz: %s returned %d", feedURL, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, err
	}

	var doc rssDoc
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("gebiz: parse %s: %w", feedURL, err)
	}

	category := CategoryFromURL(feedURL)
	out := make([]Tender, 0, len(doc.Channel.Items))
	for _, it := range doc.Channel.Items {
		t := parseItem(it.Title, it.Link, it.Description)
		t.Category = category
		out = append(out, t)
	}
	return out, nil
}

// parseItem builds a Tender from an item's title/link/description. Exposed shape
// kept small; link query params + the pipe-delimited description supply the rest.
func parseItem(title, link, description string) Tender {
	t := Tender{Title: strings.TrimSpace(title), Link: strings.TrimSpace(link), Description: strings.TrimSpace(description)}

	if u, err := url.Parse(t.Link); err == nil {
		q := u.Query()
		t.OpportunityID = q.Get("OPPORTUNITY_ID")
		t.ReferenceNo = q.Get("code")
		t.TenderType = q.Get("type")
		t.Status = q.Get("status")
	}

	// Description: "ITQ: REF | Published Date: DD/MM/YYYY | Closing Date: DD/MM/YYYY HH:MM:SS | Calling Entity: ... |"
	for i, seg := range strings.Split(t.Description, "|") {
		key, val, ok := strings.Cut(seg, ":")
		if !ok {
			continue
		}
		key, val = strings.TrimSpace(key), strings.TrimSpace(val)
		switch {
		case i == 0: // leading "ITQ: REF" — key is the tender type label, val the ref
			if t.TenderType == "" {
				t.TenderType = key
			}
			if t.ReferenceNo == "" {
				t.ReferenceNo = val
			}
		case strings.EqualFold(key, "Published Date"):
			if d, err := time.ParseInLocation("02/01/2006", val, store.SGZone); err == nil {
				t.PublishedDate = &d
			}
		case strings.EqualFold(key, "Closing Date"):
			if d, err := time.ParseInLocation("02/01/2006 15:04:05", val, store.SGZone); err == nil {
				t.ClosingDate = &d
			}
		case strings.EqualFold(key, "Calling Entity"):
			t.CallingEntity = val
		}
	}

	// Fallback dedup key if the link lacked OPPORTUNITY_ID.
	if t.OpportunityID == "" {
		if t.ReferenceNo != "" {
			t.OpportunityID = t.ReferenceNo
		} else {
			t.OpportunityID = t.Link
		}
	}
	return t
}
