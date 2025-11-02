package manager

import (
	"strings"
	"testing"
	"time"
)

func TestBuildCloid_Format(t *testing.T) {
	now := time.Unix(0, 0)
	got := buildCloid("trader", "btc", "open_long", 0.123456, now)
	if !strings.HasPrefix(got, "0x") {
		t.Fatalf("expected cloid to have 0x prefix, got %s", got)
	}
	if len(got) != 34 {
		t.Fatalf("expected cloid length 34, got %d (%s)", len(got), got)
	}
}

func TestBuildCloid_DeterministicWithinMinute(t *testing.T) {
	base := time.Date(2025, time.November, 2, 9, 3, 30, 0, time.UTC)
	a := buildCloid("trader", "BTC", "open_long", 0.5, base)
	b := buildCloid("trader", "btc", "open_long", 0.5, base.Add(20*time.Second))
	if a != b {
		t.Fatalf("expected cloid to be deterministic within same minute: %s vs %s", a, b)
	}

	c := buildCloid("trader", "BTC", "open_long", 0.5, base.Add(time.Minute))
	if c == a {
		t.Fatalf("expected cloid to differ across minute buckets: %s vs %s", a, c)
	}
}
