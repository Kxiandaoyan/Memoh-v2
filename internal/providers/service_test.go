package providers

import "testing"

func TestResolveUpdatedAPIKey(t *testing.T) {
	t.Parallel()

	existing := "sk-1234567890abcdef"
	masked := maskAPIKey(existing)

	t.Run("nil update keeps existing", func(t *testing.T) {
		t.Parallel()
		if got := resolveUpdatedAPIKey(existing, nil); got != existing {
			t.Fatalf("expected existing key, got %q", got)
		}
	})

	t.Run("masked update keeps existing", func(t *testing.T) {
		t.Parallel()
		if got := resolveUpdatedAPIKey(existing, &masked); got != existing {
			t.Fatalf("expected existing key, got %q", got)
		}
	})

	t.Run("new key replaces existing", func(t *testing.T) {
		t.Parallel()
		next := "sk-new-secret"
		if got := resolveUpdatedAPIKey(existing, &next); got != next {
			t.Fatalf("expected new key, got %q", got)
		}
	})

	t.Run("empty update clears key", func(t *testing.T) {
		t.Parallel()
		empty := ""
		if got := resolveUpdatedAPIKey(existing, &empty); got != empty {
			t.Fatalf("expected empty key, got %q", got)
		}
	})
}
