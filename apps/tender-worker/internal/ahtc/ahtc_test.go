package ahtc

import "testing"

// Real card markup returned by the ahtc_filter_tenders ajax endpoint.
const fixture = `<div class="ahtc-tender-grid">
    <div class="ahtc-tender-card">
        <div class="ahtc-tender-no">
            CONTRACT REF: OT/0652/26        </div>
        <div class="ahtc-tender-category">
            Eunos (Aljunied GRC), Hougang (Hougang SMC)            </div>
        <h3 class="ahtc-tender-title">Expression Of Interest For Multi-Disciplinary Consultancy Services (Architect, C&amp;S, M&amp;E And QS)</h3>
        <div class="ahtc-tender-dates">
            <div class="ahtc-tender-open">
                <span><strong>Opening Date:</strong></span>
                18 Jun 2026            </div>
            <div class="ahtc-tender-close">
                <span><strong>Closing Date:</strong></span>
                08 Jul 2026            </div>
        </div>
        <div class="ahtc-tender-download-wrap"> <a class="ahtc-tender-download-btn" href="https://www.ahtc.sg/assets/2026/06/OT-0652-26.pdf" target="_blank"> Download </a> </div>
    </div>
</div>`

func TestParseCards(t *testing.T) {
	recs := parseCards(fixture)
	if len(recs) != 1 {
		t.Fatalf("records = %d, want 1", len(recs))
	}
	r := recs[0]
	if r.Source != "ahtc" || r.ReferenceNo != "OT/0652/26" {
		t.Errorf("source/ref = %q/%q (CONTRACT REF: must be stripped)", r.Source, r.ReferenceNo)
	}
	if r.Title != "Expression Of Interest For Multi-Disciplinary Consultancy Services (Architect, C&S, M&E And QS)" {
		t.Errorf("title = %q", r.Title)
	}
	if r.Category == "" {
		t.Errorf("category empty")
	}
	if r.Link != "https://www.ahtc.sg/assets/2026/06/OT-0652-26.pdf" {
		t.Errorf("link = %q", r.Link)
	}
	if r.PublishedDate == nil || r.PublishedDate.Format("02 Jan 2006") != "18 Jun 2026" {
		t.Errorf("opening date not parsed: %v", r.PublishedDate)
	}
	if r.ClosingDate == nil || r.ClosingDate.Format("02 Jan 2006") != "08 Jul 2026" {
		t.Errorf("closing date not parsed: %v", r.ClosingDate)
	}
}

// "No tenders found." past the last page must yield zero records.
func TestParseCardsEmpty(t *testing.T) {
	if recs := parseCards("<p>No tenders found.</p>"); len(recs) != 0 {
		t.Errorf("records = %d, want 0", len(recs))
	}
}
