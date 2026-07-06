package sit

import (
	"os"
	"testing"
)

func TestParse(t *testing.T) {
	body, err := os.ReadFile("testdata/list.html")
	if err != nil {
		t.Fatal(err)
	}
	recs := parse(body)
	if len(recs) != 4 {
		t.Fatalf("got %d records, want 4", len(recs))
	}

	r0 := recs[0]
	if r0.Source != "sit" || r0.Agency != "Singapore Institute of Technology" {
		t.Errorf("source/agency: %q/%q", r0.Source, r0.Agency)
	}
	if r0.SourceID != "TO2026004" || r0.ReferenceNo != "TO2026004" {
		t.Errorf("ref: %q (want TO2026004, trimmed/uppercased)", r0.SourceID)
	}
	if r0.Link != "https://www.singaporetech.edu.sg/procurement-opportunities/to2026004" {
		t.Errorf("link: %q", r0.Link)
	}
	if !contains(r0.Title, "MANAGED SECURITY SERVICES") {
		t.Errorf("title: %q", r0.Title)
	}
	if r0.PublishedDate == nil || r0.PublishedDate.UTC().Format("2006-01-02 15:04") != "2026-06-12 12:00" {
		t.Errorf("published: %v", r0.PublishedDate)
	}
	if r0.ClosingDate == nil || r0.ClosingDate.UTC().Format("2006-01-02 15:04") != "2026-07-10 08:00" {
		t.Errorf("closing: %v", r0.ClosingDate)
	}

	// All four refs present, in order.
	want := []string{"TO2026004", "TO2026012", "TO2026011", "TO2026010"}
	for i, w := range want {
		if recs[i].SourceID != w {
			t.Errorf("rec %d: got %q want %q", i, recs[i].SourceID, w)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
