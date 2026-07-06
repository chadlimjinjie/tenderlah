package tuaspower

import (
	"os"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	body, err := os.ReadFile("testdata/active.json")
	if err != nil {
		t.Fatal(err)
	}
	recs, err := parse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}

	r0 := recs[0]
	if r0.Source != "tuaspower" || r0.Agency != "Tuas Power" {
		t.Errorf("source/agency: %q/%q", r0.Source, r0.Agency)
	}
	if r0.SourceID != "40008201" || r0.ReferenceNo != "40008201" {
		t.Errorf("ref: %q", r0.SourceID)
	}
	if r0.Link != "https://etender.tuaspower.com.sg/tendernotice/9ef9c350-d9df-4a14-b54f-2a3b5bc8c234" {
		t.Errorf("link: %q", r0.Link)
	}
	if r0.Status != "ACTIVE" {
		t.Errorf("status: %q", r0.Status)
	}
	if !strings.Contains(r0.Title, "KAESER SCREW AIR COMPRESSORS") || strings.Contains(r0.Title, "<") {
		t.Errorf("title not stripped/expected: %q", r0.Title)
	}
	if r0.ClosingDate == nil || r0.ClosingDate.UTC().Format("2006-01-02 15:04") != "2026-07-10 11:00" {
		t.Errorf("closing: %v", r0.ClosingDate)
	}
	if r0.PublishedDate == nil || r0.PublishedDate.UTC().Format("2006-01-02") != "2026-06-19" {
		t.Errorf("published: %v", r0.PublishedDate)
	}

	// HTML entities decoded in the second record's subject.
	if !strings.Contains(recs[1].Title, "SPARES & CONSUMABLES") {
		t.Errorf("rec1 title (entity decode?): %q", recs[1].Title)
	}
}
