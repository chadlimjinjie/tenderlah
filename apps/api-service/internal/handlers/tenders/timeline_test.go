package tenders

import "testing"

// parseTimeline must handle clean JSON and a ```json-fenced response without panicking,
// and must not cap the number of milestones.
func TestParseTimeline(t *testing.T) {
	cases := []string{
		`{"milestones":[{"label":"Contract award","date":"2026-07-01","description":"Award notified"},{"label":"Delivery","date":"2026-08-01","description":"Final delivery"}]}`,
		"```json\n{\"milestones\":[{\"label\":\"Award\",\"date\":\"2026-07-01\",\"description\":\"x\"}]}\n```",
	}
	for i, raw := range cases {
		tl, err := parseTimeline(raw)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if len(tl.Milestones) == 0 {
			t.Fatalf("case %d: expected milestones: %+v", i, tl)
		}
	}
	if _, err := parseTimeline("not json"); err == nil {
		t.Fatal("expected error on garbage input")
	}
}
