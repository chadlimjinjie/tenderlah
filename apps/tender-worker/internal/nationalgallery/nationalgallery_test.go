package nationalgallery

import "testing"

// Captured live API body (two records).
const fixture = `{"data":[
  {"reference_number":"TG/ITT/2026/0012","title":"INVITATION TO TENDER FOR SOUTHEAST ASIA GALLERIES REVAMP","rfp_tender_details":"The Gallery invites Vendor Proposal.","closing_datetime":"2026-07-09T08:00:00.000000+00:00","record_id":"a13ea6fa-1635-4792-b8d7-32b9fa9d5374","attachment":"SEAG_Tender Doc & NDA.zip"},
  {"reference_number":"TG/ITT/2026/0013","title":"Digital Advertising and Analytics Services","rfp_tender_details":"","closing_datetime":"2026-07-17T08:00:00.000000+00:00","record_id":"f5e7febf-15df-46a2-972d-db94673bc863","attachment":""}
]}`

func TestParse(t *testing.T) {
	recs, err := parse([]byte(fixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}

	r0 := recs[0]
	if r0.Source != "nationalgallery" || r0.Agency != "National Gallery Singapore" {
		t.Errorf("source/agency: %q/%q", r0.Source, r0.Agency)
	}
	if r0.SourceID != "TG/ITT/2026/0012" || r0.ReferenceNo != "TG/ITT/2026/0012" {
		t.Errorf("ref: %q", r0.SourceID)
	}
	if r0.Link != pageURL || !r0.DetailsFetched {
		t.Errorf("link/detailsFetched: %q/%v", r0.Link, r0.DetailsFetched)
	}
	if r0.ClosingDate == nil || r0.ClosingDate.UTC().Format("2006-01-02 15:04") != "2026-07-09 08:00" {
		t.Errorf("closing: %v", r0.ClosingDate)
	}
	d0, ok := r0.Details.(details)
	if !ok || d0.AttachmentURL != apiBase+"/rfps/a13ea6fa-1635-4792-b8d7-32b9fa9d5374/attachment" {
		t.Errorf("details: %+v", r0.Details)
	}

	// Second record has no attachment → no attachment URL.
	d1 := recs[1].Details.(details)
	if d1.AttachmentURL != "" {
		t.Errorf("expected no attachment URL, got %q", d1.AttachmentURL)
	}
}
