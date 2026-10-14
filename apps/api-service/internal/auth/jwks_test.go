package auth

import "testing"

func TestRoleFromScope(t *testing.T) {
	if r := roleFromScope("openid email admin"); r == nil || *r != "admin" {
		t.Fatalf("want admin, got %v", r)
	}
	for _, scope := range []string{"", "openid profile", "administrator"} {
		if r := roleFromScope(scope); r != nil {
			t.Fatalf("scope %q: want nil, got %q", scope, *r)
		}
	}
}
