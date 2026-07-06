package emservices

import "testing"

// fixture mirrors the real Elementor `html_items` tag shapes (captured from
// page 1): a <span> agency, an <h3> title wrapping an <a>, h2 date label/value
// pairs, and the eligibility line — trimmed of the heavy inline CSS.
const fixture = `<div class="ue-grid-item" data-link="https://www.emservices.com.sg/tender/electrical-rewiring/">
  <div class="elementor-widget-container">
    <span class="elementor-heading-title elementor-size-default">HOLLAND-BUKIT PANJANG TOWN COUNCIL</span>
  </div>
  <div class="elementor-widget-container">
    <h3 class="elementor-heading-title elementor-size-default"><a href="https://x/notice.pdf" target="_blank">ELECTRICAL REWIRING AT 13 BLOCKS  (CONTRACT PERIOD &#8211; 15 MONTHS)  (&#9993; : CYCL-26-016-HBP@emservices.com.sg)</a></h3>
  </div>
  <span class="elementor-button-text">Download</span>
  <h2 class="elementor-heading-title elementor-size-default">ADVERTISEMENT DATE: </h2>
  <h2 class="elementor-heading-title elementor-size-default">03/07/2026</h2>
  <h2 class="elementor-heading-title elementor-size-default">CLOSING DATE: </h2>
  <h2 class="elementor-heading-title elementor-size-default">28/07/2026</h2>
  <h2 class="elementor-heading-title elementor-size-default">ELIGIBILITY &amp; FINANCIAL GRADES: </h2>
  <h2 class="elementor-heading-title elementor-size-default">Electrical Engineering (ME05) - $5 Million (L3) &amp; Above</h2>
  <span class="elementor-button-text">Download</span>
<div class="ue-grid-item" data-link="https://www.emservices.com.sg/tender/outstanding-works/">
  <span class="elementor-heading-title elementor-size-default">TANJONG PAGAR TOWN COUNCIL</span>
  <h3 class="elementor-heading-title elementor-size-default"><a href="https://x/notice2.pdf">OUTSTANDING WORKS FOR A&amp;A PROJECTS  (&#9993; : AA-26-013-TP@emservices.com.sg)</a></h3>
  <h2 class="elementor-heading-title elementor-size-default">ADVERTISEMENT DATE: </h2>
  <h2 class="elementor-heading-title elementor-size-default">03/07/2026</h2>
  <h2 class="elementor-heading-title elementor-size-default">CLOSING DATE: </h2>
  <h2 class="elementor-heading-title elementor-size-default">28/07/2026</h2>
  <h2 class="elementor-heading-title elementor-size-default">ELIGIBILITY &amp; FINANCIAL GRADES: </h2>
  <h2 class="elementor-heading-title elementor-size-default">General Building (CW01) - $1.6 Million (C2) &amp; Above</h2>`

func TestParseItems(t *testing.T) {
	recs := parseItems(fixture)
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}

	r := recs[0]
	if r.Source != "emservices" {
		t.Errorf("Source = %q, want emservices", r.Source)
	}
	if r.ReferenceNo != "CYCL-26-016-HBP" || r.SourceID != "CYCL-26-016-HBP" {
		t.Errorf("ref = %q / %q, want CYCL-26-016-HBP", r.ReferenceNo, r.SourceID)
	}
	if r.Agency != "HOLLAND-BUKIT PANJANG TOWN COUNCIL" {
		t.Errorf("Agency = %q", r.Agency)
	}
	if r.Link != "https://www.emservices.com.sg/tender/electrical-rewiring/" {
		t.Errorf("Link = %q", r.Link)
	}
	// title keeps the body but drops the trailing "(✉ : ref@…)" suffix
	if want := "ELECTRICAL REWIRING AT 13 BLOCKS (CONTRACT PERIOD – 15 MONTHS)"; r.Title != want {
		t.Errorf("Title = %q, want %q", r.Title, want)
	}
	if r.Description != "Electrical Engineering (ME05) - $5 Million (L3) & Above" {
		t.Errorf("Description = %q", r.Description)
	}
	if r.PublishedDate == nil || r.PublishedDate.Format("2006-01-02") != "2026-07-03" {
		t.Errorf("PublishedDate = %v, want 2026-07-03", r.PublishedDate)
	}
	if r.ClosingDate == nil || r.ClosingDate.Format("2006-01-02") != "2026-07-28" {
		t.Errorf("ClosingDate = %v, want 2026-07-28", r.ClosingDate)
	}
	if r.Status != "open" {
		t.Errorf("Status = %q, want open", r.Status)
	}

	if recs[1].ReferenceNo != "AA-26-013-TP" {
		t.Errorf("second ref = %q, want AA-26-013-TP", recs[1].ReferenceNo)
	}
}
