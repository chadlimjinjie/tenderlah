package gebiz

import "testing"

func TestParseItem(t *testing.T) {
	link := "https://www.gebiz.gov.sg/ptn/opportunity/opportunityDetails.xhtml?code=NATHQ0ETQ26000002&status=RELEASED&type=TQ&OPPORTUNITY_ID=1000000000000533356&origin=rss"
	desc := "ITQ: NATHQ0ETQ26000002 | Published Date: 17/06/2026 | Closing Date: 26/06/2026 13:00:00 | Calling Entity: Ministry of National Development-Ministry Headquarter |"

	got := parseItem("Provision of Services", link, desc)

	if got.OpportunityID != "1000000000000533356" {
		t.Errorf("OpportunityID = %q", got.OpportunityID)
	}
	if got.ReferenceNo != "NATHQ0ETQ26000002" {
		t.Errorf("ReferenceNo = %q", got.ReferenceNo)
	}
	if got.Status != "RELEASED" {
		t.Errorf("Status = %q", got.Status)
	}
	if got.CallingEntity != "Ministry of National Development-Ministry Headquarter" {
		t.Errorf("CallingEntity = %q", got.CallingEntity)
	}
	if got.PublishedDate == nil || got.PublishedDate.Format("2006-01-02") != "2026-06-17" {
		t.Errorf("PublishedDate = %v", got.PublishedDate)
	}
	if got.ClosingDate == nil || got.ClosingDate.Format("2006-01-02 15:04:05") != "2026-06-26 13:00:00" {
		t.Errorf("ClosingDate = %v", got.ClosingDate)
	}
}

func TestParseItemFallbackKey(t *testing.T) {
	// No OPPORTUNITY_ID in link -> fall back to reference no, never empty.
	got := parseItem("X", "https://example.com/x", "ITT: REF123 |")
	if got.OpportunityID != "REF123" {
		t.Errorf("fallback OpportunityID = %q, want REF123", got.OpportunityID)
	}
	if got.TenderType != "ITT" {
		t.Errorf("TenderType = %q, want ITT", got.TenderType)
	}
}

func TestCategoryFromURL(t *testing.T) {
	got := CategoryFromURL("https://www.gebiz.gov.sg/rss/Professional_Services-CREATE_BO_FEED.xml")
	if got != "Professional Services" {
		t.Errorf("CategoryFromURL = %q", got)
	}
}
