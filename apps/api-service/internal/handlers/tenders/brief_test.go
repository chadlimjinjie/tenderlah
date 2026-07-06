package tenders

import "testing"

// parseBrief must handle clean JSON and a ```json-fenced response without panicking.
func TestParseBrief(t *testing.T) {
	cases := []string{
		`{"summary":"Supply of office chairs","scope":["Deliver 100 chairs"],"requirements":["GeBIZ registered"],"documents":["Price schedule"],"complexity":"Low","riskFlags":["Tight 5-day closing"]}`,
		"```json\n{\"summary\":\"x\",\"scope\":[],\"requirements\":[],\"documents\":[],\"complexity\":\"Medium\",\"riskFlags\":[]}\n```",
	}
	for i, raw := range cases {
		b, err := parseBrief(raw)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if b.Summary == "" || b.Complexity == "" {
			t.Fatalf("case %d: empty fields: %+v", i, b)
		}
	}
	if _, err := parseBrief("not json"); err == nil {
		t.Fatal("expected error on garbage input")
	}
}
