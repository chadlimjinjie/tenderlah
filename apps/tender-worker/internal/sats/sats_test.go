package sats

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
	"testing"
)

// Trimmed real AEM markup: two accordion items — one normal, one with the tricky
// "Closing on 10 Jul" wording and a /content/.html slug link.
const listFixture = `
<div class="cmp-accordion">
  <div class="cmp-accordion__item" id="a-1">
    <span class="cmp-accordion__title">CW2606I380 - Supply and Delivery of PAX Female Shoes</span>
    <div class="cmp-text">
      <p><a href="/tenders/CW2606I380"><strong>CW2606I380 - Supply and Delivery of PAX Female Shoes</strong></a></p>
      <p><strong>RFQ Closing Date: 15 July 2026, 1200hrs (Singapore Time)</strong></p>
    </div>
  </div>
  <div class="cmp-accordion__item" id="a-2">
    <span class="cmp-accordion__title">5 Years Mid Life Refurbishment Program</span>
    <div class="cmp-text">
      <p><a href="/content/sats/corporate/en/tenders/ct2506t032.html">Refurbishment</a></p>
      <p>Closing on 10 Jul 2025 at 1500 hrs SG time</p>
    </div>
  </div>
</div>`

const detailFixture = `
<nav class="cmp-breadcrumb">home</nav>
<div class="cmp-text"><p>SATS is seeking to refresh the current female footwear design for operational use.</p></div>
<div class="cmp-text"><p>Please email <a href="mailto:Shermaine_cheongsm@sats.com.sg">Shermaine</a> and <a href="mailto:Shermaine_cheongsm@sats.com.sg">again</a>.</p></div>
<div class="button download document">
  <a class="cmp-button" href="/content/dam/sats/corporate/docs/tenders/Ariba%20Form.docx.downloadasset.docx">form</a>
</div>
<div class="cmp-experiencefragment--footer"><div class="cmp-text"><p>footer junk should be ignored</p></div></div>`

func TestParse(t *testing.T) {
	recs := parse([]byte(listFixture))
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}

	r0 := recs[0]
	if r0.SourceID != "CW2606I380" || r0.ReferenceNo != "CW2606I380" {
		t.Errorf("ref: got %q", r0.SourceID)
	}
	if r0.Source != "sats" || r0.Agency != "SATS" {
		t.Errorf("source/agency: got %q/%q", r0.Source, r0.Agency)
	}
	if r0.Link != "https://www.sats.com.sg/tenders/CW2606I380" {
		t.Errorf("link: got %q", r0.Link)
	}
	if r0.ClosingDate == nil || r0.ClosingDate.Format("2006-01-02") != "2026-07-15" {
		t.Errorf("closing: got %v", r0.ClosingDate)
	}

	// Ref comes from the URL slug (title has no code); .html + /content prefix stripped;
	// "Closing on 10 Jul" must parse the day as 10, not 0.
	r1 := recs[1]
	if r1.SourceID != "CT2506T032" {
		t.Errorf("slug ref: got %q", r1.SourceID)
	}
	if r1.Link != "https://www.sats.com.sg/tenders/ct2506t032" {
		t.Errorf("normalized link: got %q", r1.Link)
	}
	if r1.ClosingDate == nil || r1.ClosingDate.Format("2006-01-02") != "2025-07-10" {
		t.Errorf("closing: got %v", r1.ClosingDate)
	}
}

func TestParseDetail(t *testing.T) {
	d := parseDetail([]byte(detailFixture))

	if len(d.Contacts) != 1 || d.Contacts[0].Email != "shermaine_cheongsm@sats.com.sg" {
		t.Errorf("contacts: got %+v", d.Contacts)
	}
	if len(d.Attachments) != 1 || d.Attachments[0] != "https://www.sats.com.sg/content/dam/sats/corporate/docs/tenders/Ariba%20Form.docx.downloadasset.docx" {
		t.Errorf("attachments: got %+v", d.Attachments)
	}
	if want := "female footwear design"; !contains(d.Description, want) {
		t.Errorf("description missing %q: got %q", want, d.Description)
	}
	if contains(d.Description, "footer junk") {
		t.Errorf("description leaked footer content: %q", d.Description)
	}
}

// TestSolveCrypto checks the proof-of-work: every answer must hash to 0 mod its
// difficulty (re-derived here with big.Int, independent of the running-modulus
// generator), and difficulty must increment per answer (count=2 exercises that).
func TestSolveCrypto(t *testing.T) {
	ch := challenge{
		Token:      "tok",
		Timestamp:  1781838703,
		Nonce:      "2e94d2e9793bd3c0e3bd",
		Difficulty: 500, // small so the test is fast
		Count:      2,
	}
	answers, err := solveCrypto("C57D7FE7D8E01C68F3EA0B402CB8B297", ch)
	if err != nil {
		t.Fatal(err)
	}
	if len(answers) != 2 {
		t.Fatalf("got %d answers, want 2", len(answers))
	}
	for i, ans := range answers {
		diff := ch.Difficulty + int64(i) // each answer is one harder
		input := fmt.Sprintf("%s%d%s%d%s", "C57D7FE7D8E01C68F3EA0B402CB8B297", ch.Timestamp, ch.Nonce, diff, ans)
		sum := sha256.Sum256([]byte(input))
		mod := new(big.Int).Mod(new(big.Int).SetBytes(sum[:]), big.NewInt(diff))
		if mod.Sign() != 0 {
			t.Errorf("answer %d (%q): hash mod %d = %s, want 0", i, ans, diff, mod)
		}
	}
}

func TestParseChallenge(t *testing.T) {
	raw := `{"token":"AAQtok","timestamp":1781838703,"nonce":"abc","difficulty":5000,"count":1,"verify_url":"https://www.sats.com.sg/x/y"}`
	page := `<iframe challenge="` + base64.StdEncoding.EncodeToString([]byte(raw)) + `" data-duration=10 src="/a/b"></iframe>`
	ch, dur, err := parseChallenge([]byte(page))
	if err != nil {
		t.Fatal(err)
	}
	if ch.Token != "AAQtok" || ch.Difficulty != 5000 || ch.Count != 1 || dur != 10 {
		t.Errorf("parsed: %+v dur=%d", ch, dur)
	}
	if ch.VerifyURL != "https://www.sats.com.sg/x/y" {
		t.Errorf("verify_url: %q", ch.VerifyURL)
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
