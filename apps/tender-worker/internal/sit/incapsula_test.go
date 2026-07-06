package sit

import (
	"os"
	"strings"
	"testing"
)

// TestComputeUtmvc proves the goja solver executes the real (captured) Incapsula
// utmvc challenge under browserEnvJS and emits a ___utmvc cookie. This is the offline
// runnable check for the solver mechanics. NOTE: it does not assert the fingerprint
// is *accepted* by Incapsula — that requires a live request from a non-blocked IP
// (see the ceiling on solveIncapsula).
func TestComputeUtmvc(t *testing.T) {
	js, err := os.ReadFile("testdata/incap_challenge.js")
	if err != nil {
		t.Skip("no captured challenge fixture")
	}
	cookie, err := computeUtmvc(string(js))
	if err != nil {
		t.Fatalf("computeUtmvc: %v", err)
	}
	if !strings.HasPrefix(cookie, "___utmvc=") {
		t.Fatalf("expected ___utmvc cookie, got %q", cookie)
	}
	if len(cookie) < 100 {
		t.Errorf("cookie suspiciously short (%d chars): %q", len(cookie), cookie)
	}
}
