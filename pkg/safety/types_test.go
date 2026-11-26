package safety

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxRollDegrees != 25.0 {
		t.Fatalf("expected 25.0 max roll degrees, got %f", cfg.MaxRollDegrees)
	}
	if cfg.DebounceThreshold < 1 {
		t.Fatalf("expected debounce threshold >= 1, got %d", cfg.DebounceThreshold)
	}
}
