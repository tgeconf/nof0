package hyperliquid

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/dnaeon/go-vcr/recorder"
)

// This test uses go-vcr to record/replay a real GetMarketInfo call.
// It skips by default if cassette is absent and RECORD_CASSETTES != 1.
func TestClient_GetMarketInfo_Recorded(t *testing.T) {
	cassette := filepath.Join("testdata", "cassettes", "hyperliquid_info.yaml")
	if _, err := os.Stat(cassette); os.IsNotExist(err) {
		if os.Getenv("RECORD_CASSETTES") != "1" {
			t.Skipf("cassette missing; set RECORD_CASSETTES=1 to record: %s", cassette)
		}
		// Ensure parent directory exists for recording
		if err := os.MkdirAll(filepath.Dir(cassette), 0o755); err != nil {
			t.Fatalf("mkdir cassettes dir: %v", err)
		}
	}

	r, err := recorder.New(cassette)
	if err != nil {
		t.Fatalf("recorder.New: %v", err)
	}
	defer func() { _ = r.Stop() }()

	httpClient := &http.Client{Transport: r}
	client := NewClient(WithHTTPClient(httpClient))
	ctx := context.Background()
	info, err := client.GetMarketInfo(ctx, "btc")
	if err != nil {
		t.Fatalf("GetMarketInfo: %v", err)
	}
	if info.Symbol == "" {
		t.Fatalf("empty symbol in response")
	}
	if info.MidPrice <= 0 {
		t.Fatalf("non-positive mid price: %v", info.MidPrice)
	}
}
