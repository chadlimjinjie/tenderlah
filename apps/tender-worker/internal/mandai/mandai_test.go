package mandai

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
	if len(recs) != 16 {
		t.Fatalf("records = %d, want 16", len(recs))
	}

	r := recs[0] // first row in the fixture
	if r.Source != "mandai" || r.SourceID != "22b4e640-f2e1-468b-bbc2-b51f985729d2" {
		t.Errorf("source/id = %q/%q", r.Source, r.SourceID)
	}
	if r.ReferenceNo != "WRS-ZOO-Q26-06-01911" {
		t.Errorf("ref = %q", r.ReferenceNo)
	}
	if r.Agency != "Singapore Zoological Gardens" {
		t.Errorf("agency = %q", r.Agency)
	}
	if r.TenderType != "RFQ" {
		t.Errorf("type = %q", r.TenderType)
	}
	if r.PublishedDate == nil || r.ClosingDate == nil {
		t.Fatalf("dates not parsed: %v %v", r.PublishedDate, r.ClosingDate)
	}
	if got := r.ClosingDate.Format("2006-01-02 15:04"); got != "2026-07-06 14:00" {
		t.Errorf("closing = %q", got)
	}
	if r.Title == "" || r.Description == "" {
		t.Errorf("empty title/description: %+v", r)
	}
}

func TestParseDetail(t *testing.T) {
	body, err := os.ReadFile("testdata/detail.html")
	if err != nil {
		t.Fatal(err)
	}
	d := parseDetail(body)
	if d.VendorCriteria == "" {
		t.Fatal("vendor criteria not parsed")
	}
	if !strings.Contains(d.VendorCriteria, "1KG") {
		t.Errorf("criteria = %q", d.VendorCriteria)
	}
}
