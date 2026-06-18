package loops

import "testing"

func TestVersionFallback(t *testing.T) {
	// when running from this module's own test binary, loops-go is the main
	// module and won't appear in BuildInfo.Deps, so readVersion must return
	// the "dev" fallback.
	if got := readVersion(); got != "dev" {
		t.Errorf("readVersion() = %q, want %q", got, "dev")
	}
	if Version == "" {
		t.Error("Version is empty")
	}
}
