package nus

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
	if len(recs) != 10 {
		t.Fatalf("got %d records, want 10", len(recs))
	}

	r0 := recs[0]
	if r0.Source != "nus" || r0.Agency != "National University of Singapore" {
		t.Errorf("source/agency: %q/%q", r0.Source, r0.Agency)
	}
	if r0.SourceID != "Doc3197396395" || r0.ReferenceNo != "Doc3197396395" {
		t.Errorf("ref: %q (want Doc3197396395)", r0.SourceID)
	}
	if r0.Status != "Open" {
		t.Errorf("status: %q (want Open)", r0.Status)
	}
	if r0.Link != "https://portal.us.bn.cloud.ariba.com/dashboard/public/appext/comsapsbncdiscoveryui#/RfxEvent/preview/1110017057?anId=ANONYMOUS" {
		t.Errorf("link: %q", r0.Link)
	}
	if r0.Title != "ITQ for the Provision of Delay Telecast Services for NUS Commencement Ceremony 2026 to the Centre for Teaching, Learning and Technology (CTLT), NUS" {
		t.Errorf("title: %q", r0.Title)
	}
	if r0.Category != "Video streaming system" {
		t.Errorf("category: %q", r0.Category)
	}
	if r0.PublishedDate == nil || r0.PublishedDate.Format("2006-01-02 15:04") != "2026-06-26 17:30" {
		t.Errorf("published: %v", r0.PublishedDate)
	}
	if r0.ClosingDate == nil || r0.ClosingDate.Format("2006-01-02 15:04") != "2026-07-02 16:00" {
		t.Errorf("closing: %v", r0.ClosingDate)
	}

	// Every parsed row must be Open with a doc id (we only requested Open).
	for _, r := range recs {
		if r.Status != "Open" || r.SourceID == "" {
			t.Errorf("bad row: %+v", r)
		}
	}
}

func TestLastPage(t *testing.T) {
	body, err := os.ReadFile("testdata/list.html")
	if err != nil {
		t.Fatal(err)
	}
	if got := lastPage(body); got != 3 {
		t.Errorf("lastPage = %d, want 3", got)
	}
}
