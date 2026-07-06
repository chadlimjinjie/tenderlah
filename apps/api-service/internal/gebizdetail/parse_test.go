package gebizdetail

import "testing"

// Mirrors the real GeBIZ JSF markup: contact values sit as text after an icon div,
// items are delimited by an "Item No. N" orange label + a title + label/value pairs.
const fixture = `<html><body>
<div class="formSectionHeader1_TEXT">WHO TO CONTACT</div>
<div class="formOutputText_HIDDEN-LABEL outputText_TITLE-BLACK">TAN TIONG LEI</div>
<div><div style="padding-right:5px;display:inline-block;"><div style="background-image: url('/ptn/desktop/images/imageicons/fontawesome/envelope_20_434343.png');"></div></div>TTIONGLE@dsta.gov.sg</div>
<div><div style="display:inline-block;"><div style="background-image: url('/ptn/desktop/images/imageicons/fontawesome/phone_20_434343.png');"></div></div>68194165</div>
<div><div style="display:inline-block;"><div style="background-image: url('/ptn/desktop/images/imageicons/fontawesome/location_20.png');"></div></div>1 Depot Road, Singapore 109679</div>
<div class="formSectionHeader1_TEXT">AWARDING AGENCY</div>
<div class="formSectionHeader1_TEXT">ITEMS TO RESPOND</div>
<span class="outputText_LABEL-ORANGE">Item No. 1<br/>Mandatory to Bid: No</span>
<div class="formOutputText_HIDDEN-LABEL outputText_TITLE-BLACK">VALVE SHUTOFF</div>
<label><span>Unit of Measurement</span></label>
<div class="formOutputText_VALUE-DIV">Each</div>
<label><span>Required Quantity</span></label>
<div class="formOutputText_VALUE-DIV">1.0</div>
<div class="formSectionHeader2_TEXT">PART NO. INFORMATION</div>
<label><span>NSN</span></label>
<div class="formOutputText_VALUE-DIV">3291268-1</div>
</body></html>`

func TestParse(t *testing.T) {
	d, err := Parse([]byte(fixture))
	if err != nil {
		t.Fatal(err)
	}

	if d.Contact.Name != "TAN TIONG LEI" {
		t.Errorf("name = %q", d.Contact.Name)
	}
	if d.Contact.Email != "TTIONGLE@dsta.gov.sg" {
		t.Errorf("email = %q", d.Contact.Email)
	}
	if d.Contact.Phone != "68194165" {
		t.Errorf("phone = %q", d.Contact.Phone)
	}
	if d.Contact.Address == "" {
		t.Errorf("address empty")
	}

	if len(d.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(d.Items))
	}
	it := d.Items[0]
	if it.No != "1" || it.Description != "VALVE SHUTOFF" {
		t.Errorf("item = %+v", it)
	}
	// Unit of Measurement, Required Quantity, NSN (the sub-section attr counts too).
	if len(it.Attributes) != 3 {
		t.Errorf("attributes = %d (%+v)", len(it.Attributes), it.Attributes)
	}
}
