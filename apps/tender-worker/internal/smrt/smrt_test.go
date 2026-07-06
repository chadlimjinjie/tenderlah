package smrt

import "testing"

// Mirrors SMRT's embedded accordion: data-model is HTML-entity-encoded JSON.
const fixture = `<div widget-type='AccordionWithImage' data-model="{&quot;items&quot;:[` +
	`{&quot;title&quot;:&quot;Expression of Interest (EOI) for M264700013&quot;,` +
	`&quot;project&quot;:&quot;5-Year BMU Servicing&quot;,` +
	`&quot;publishedOn&quot;:&quot;2026-06-18T10:43:28&quot;,` +
	`&quot;closingDate&quot;:&quot;2026-06-25T12:00:00&quot;,` +
	`&quot;persons&quot;:[&quot;Reuben George;reuben@smrt.com.sg;97105240&quot;,&quot;Kevin Tan;kevin@smrt.com.sg;65548324&quot;],` +
	`&quot;description&quot;:&quot;<p><b>Intro</b> text</p>&quot;}` +
	`]}"></div>`

func TestParse(t *testing.T) {
	recs := parse([]byte(fixture))
	if len(recs) != 1 {
		t.Fatalf("records = %d, want 1", len(recs))
	}
	r := recs[0]
	if r.Source != "smrt" || r.SourceID != "M264700013" {
		t.Errorf("source/id = %q/%q", r.Source, r.SourceID)
	}
	if r.Title != "5-Year BMU Servicing" {
		t.Errorf("title = %q", r.Title)
	}
	if r.PublishedDate == nil || r.ClosingDate == nil {
		t.Errorf("dates not parsed: %v %v", r.PublishedDate, r.ClosingDate)
	}
	if !r.DetailsFetched {
		t.Errorf("DetailsFetched should be true")
	}
	d, ok := r.Details.(details)
	if !ok || len(d.Contacts) != 2 || d.Contacts[0].Email != "reuben@smrt.com.sg" {
		t.Errorf("details = %+v", r.Details)
	}
	if d.Description != "Intro text" {
		t.Errorf("description = %q", d.Description)
	}
}
