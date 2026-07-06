package mediacorp

import (
	"os"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	body, err := os.ReadFile("testdata/list.html")
	if err != nil {
		t.Fatal(err)
	}
	recs := parse(body)
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}

	r0 := recs[0]
	if r0.Source != "mediacorp" || r0.Agency != "Mediacorp Pte Ltd" {
		t.Errorf("source/agency: %q/%q", r0.Source, r0.Agency)
	}
	if r0.SourceID != "1000076605" || r0.ReferenceNo != "1000076605" {
		t.Errorf("ref: %q (want 1000076605)", r0.SourceID)
	}
	if r0.Link != "https://www.mediacorp.sg/tender-notices/tender-no-1000076605-11-june-2026-9am-189386" {
		t.Errorf("link: %q", r0.Link)
	}
	if !strings.Contains(r0.Title, "TENDER NO: 1000076605") {
		t.Errorf("title: %q", r0.Title)
	}
	if !strings.Contains(r0.Description, "Mediacorp Experience") {
		t.Errorf("description: %q", r0.Description)
	}
	if r0.ClosingDate == nil || r0.ClosingDate.UTC().Format("2006-01-02 15:04") != "2026-07-27 07:00" {
		t.Errorf("closing: %v", r0.ClosingDate)
	}

	if recs[1].SourceID != "1000076562" {
		t.Errorf("rec1 ref: %q", recs[1].SourceID)
	}
	if !strings.Contains(recs[1].Description, "Unified Monitoring & Operation Centre") {
		t.Errorf("rec1 description (entities decoded?): %q", recs[1].Description)
	}
}
