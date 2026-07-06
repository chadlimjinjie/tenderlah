// Package gebizdetail scrapes a GeBIZ opportunity detail page (the tender's `link`)
// into structured contact + line-item data. The page is server-rendered JSF, so a
// plain HTTP GET is enough — values sit in stable CSS classes / icon images.
//
// ponytail: class-name scrape, brittle to GeBIZ redesigns — swap to their API if one appears.
package gebizdetail

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

type Attr struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type Item struct {
	No          string `json:"no"`
	Description string `json:"description"`
	Attributes  []Attr `json:"attributes"`
}

type Contact struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
}

type Details struct {
	Contact Contact `json:"contact"`
	Items   []Item  `json:"items"`
}

// Fetch downloads the opportunity page and parses it into Details.
func Fetch(ctx context.Context, url string) (Details, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Details{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MidtoneTenderBot/1.0)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Details{}, fmt.Errorf("gebizdetail: fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return Details{}, fmt.Errorf("gebizdetail: fetch %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 3<<20))
	if err != nil {
		return Details{}, err
	}
	return Parse(body)
}

type tokKind int

const (
	tSection tokKind = iota
	tOrange
	tTitle
	tLabel
	tValue
	tIcon
)

type tok struct {
	kind tokKind
	text string
}

var (
	itemNoRe = regexp.MustCompile(`Item No\.\s*(\d+)`)
	emailRe  = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
)

// Parse turns the page HTML into Details.
func Parse(body []byte) (Details, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return Details{}, err
	}

	var toks []tok
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			class := attr(n, "class")
			style := attr(n, "style")
			switch {
			case n.Data == "label":
				toks = append(toks, tok{tLabel, text(n)})
				return
			case strings.Contains(class, "formSectionHeader"):
				toks = append(toks, tok{tSection, text(n)})
				return
			case strings.Contains(class, "outputText_LABEL-ORANGE"):
				toks = append(toks, tok{tOrange, text(n)})
				return
			case strings.Contains(class, "outputText_TITLE-BLACK"):
				toks = append(toks, tok{tTitle, text(n)})
				return
			case strings.Contains(class, "formOutputText_VALUE-DIV"):
				toks = append(toks, tok{tValue, text(n)})
				return
			case strings.Contains(style, "imageicons"):
				toks = append(toks, tok{tIcon, nearestText(n)})
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	d := Details{
		Contact: parseContact(toks),
		Items:   parseItems(toks),
	}
	if d.Contact.Email == "" {
		d.Contact.Email = emailRe.FindString(string(body))
	}
	return d, nil
}

// parseContact reads the tokens between the "WHO TO CONTACT" section header and the
// next section header: the first title is the name; icon values classify into
// email / phone / address.
func parseContact(toks []tok) Contact {
	start := sectionIndex(toks, "WHO TO CONTACT")
	if start < 0 {
		start = sectionIndex(toks, "CONTACT")
	}
	if start < 0 {
		return Contact{}
	}
	var c Contact
	var phones []string
	for _, t := range toks[start+1:] {
		if t.kind == tSection {
			break
		}
		switch t.kind {
		case tTitle:
			if c.Name == "" {
				c.Name = t.text
			}
		case tIcon:
			v := strings.TrimSpace(t.text)
			switch {
			case v == "":
			case strings.Contains(v, "@"):
				if c.Email == "" {
					c.Email = v
				}
			case isPhone(v):
				phones = append(phones, v)
			default:
				if c.Address == "" {
					c.Address = v
				}
			}
		}
	}
	c.Phone = strings.Join(phones, ", ")
	return c
}

// parseItems reads everything after the "ITEMS TO RESPOND" header. Each
// "Item No. N" label starts a new item; the following title is its description;
// label/value pairs (including the PART NO. / DELIVERY sub-sections) become attributes.
func parseItems(toks []tok) []Item {
	start := sectionIndex(toks, "ITEMS TO RESPOND")
	if start < 0 {
		return nil
	}
	var items []Item
	cur := -1
	pending := ""
	for _, t := range toks[start+1:] {
		switch t.kind {
		case tOrange:
			if m := itemNoRe.FindStringSubmatch(t.text); m != nil {
				items = append(items, Item{No: m[1]})
				cur = len(items) - 1
				pending = ""
			}
		case tTitle:
			if cur >= 0 && items[cur].Description == "" {
				items[cur].Description = t.text
			}
		case tLabel:
			pending = t.text
		case tValue:
			if cur >= 0 && pending != "" && t.text != "" {
				items[cur].Attributes = append(items[cur].Attributes, Attr{Label: pending, Value: t.text})
			}
			pending = ""
		}
	}
	return items
}

func sectionIndex(toks []tok, needle string) int {
	for i, t := range toks {
		if t.kind == tSection && strings.Contains(strings.ToUpper(t.text), needle) {
			return i
		}
	}
	return -1
}

// isPhone: digits (with spaces/dashes/parens) and at least 6 of them.
func isPhone(s string) bool {
	digits := 0
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == ' ' || r == '-' || r == '(' || r == ')' || r == '+' || r == ',':
		default:
			return false
		}
	}
	return digits >= 6
}

func attr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

// text concatenates all descendant text, collapsing whitespace.
func text(n *html.Node) string {
	var b strings.Builder
	var f func(*html.Node)
	f = func(x *html.Node) {
		if x.Type == html.TextNode {
			b.WriteString(x.Data)
			b.WriteString(" ")
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

// nearestText climbs from an icon node to the closest ancestor that carries text
// (the icon's value sits as a sibling text node a couple of levels up).
func nearestText(n *html.Node) string {
	for a, depth := n.Parent, 0; a != nil && depth < 5; a, depth = a.Parent, depth+1 {
		if t := text(a); t != "" {
			return t
		}
	}
	return ""
}
