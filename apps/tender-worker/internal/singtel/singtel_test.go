package singtel

import (
	"strings"
	"testing"
)

// Mirrors Singtel's embedded Tender component: datamodel is HTML-entity-encoded JSON
// with currentTenders + pastTenders; we only take currentTenders.
const fixture = `<div class="lux-component-container" component="Tender" datamodel="{` +
	`&quot;currentTenderLabel&quot;:&quot;Current Tenders&quot;,` +
	`&quot;currentTenders&quot;:[{` +
	`&quot;tenderDesc&quot;:&quot;REPLACEMENT OF FIRE ALARM SYSTEM AT GEYLANG EXCHANGE&quot;,` +
	`&quot;tenderNumber&quot;:&quot;PRQ022516&quot;,` +
	`&quot;tenderPagePath&quot;:&quot;https://www.singtel.com/about-us/tenders/prq022516&quot;,` +
	`&quot;issueDate&quot;:&quot;28/05/2026&quot;,` +
	`&quot;closingDate&quot;:&quot;02/07/2026&quot;}],` +
	`&quot;pastTenders&quot;:[{&quot;tenderNumber&quot;:&quot;PRQ000001&quot;}]}"></div>`

func TestParse(t *testing.T) {
	recs := parse([]byte(fixture))
	if len(recs) != 1 {
		t.Fatalf("records = %d, want 1 (pastTenders must be ignored)", len(recs))
	}
	r := recs[0]
	if r.Source != "singtel" || r.SourceID != "PRQ022516" {
		t.Errorf("source/id = %q/%q", r.Source, r.SourceID)
	}
	if r.Title != "REPLACEMENT OF FIRE ALARM SYSTEM AT GEYLANG EXCHANGE" {
		t.Errorf("title = %q", r.Title)
	}
	if r.Link != "https://www.singtel.com/about-us/tenders/prq022516" {
		t.Errorf("link = %q", r.Link)
	}
	if r.PublishedDate == nil || r.PublishedDate.Format("02/01/2006") != "28/05/2026" {
		t.Errorf("issue date not parsed: %v", r.PublishedDate)
	}
	if r.ClosingDate == nil || r.ClosingDate.Format("02/01/2006") != "02/07/2026" {
		t.Errorf("closing date not parsed: %v", r.ClosingDate)
	}
}

// Real per-tender detail block: ux-heading-text datamodel is HTML-entity-encoded JSON
// whose "text" value is itself HTML.
const detailFixture = `<div><ux-heading-text datamodel="{&quot;text&quot;:&quot;` +
	`<p><b>ELIGIBILITY</b> All BCA Registered Contractors grade L3</p> ` +
	`<p><b>TENDER CLOSING DATE / TIME</b> 2/7/2026 / 12:00 PM (Singapore Time)</p> ` +
	`<p>email Samuel Ong at samuel.ong@singtel.com Tung Weng Foo at tgwgfo@singtel.com ` +
	`Wong Shioong Chai at shioongchai.wong@singtel.com</p>` +
	`&quot;,&quot;topSpacing&quot;:&quot;none&quot;}"></ux-heading-text></div>`

func TestParseDetail(t *testing.T) {
	d := parseDetail([]byte(detailFixture))
	if !strings.Contains(d.Description, "ELIGIBILITY") || !strings.Contains(d.Description, "TENDER CLOSING DATE") {
		t.Errorf("description missing key sections: %q", d.Description)
	}
	if strings.Contains(d.Description, "<p>") {
		t.Errorf("description still has HTML tags: %q", d.Description)
	}
	if len(d.Contacts) != 3 {
		t.Fatalf("contacts = %d, want 3: %+v", len(d.Contacts), d.Contacts)
	}
	if d.Contacts[0].Name != "Samuel Ong" || d.Contacts[0].Email != "samuel.ong@singtel.com" {
		t.Errorf("first contact = %+v", d.Contacts[0])
	}
}
