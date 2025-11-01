package hyperliquid

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/dnaeon/go-vcr/recorder"
	"github.com/stretchr/testify/assert"
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
		err := os.MkdirAll(filepath.Dir(cassette), 0o755)
		assert.NoError(t, err, "mkdir cassettes dir should succeed")
	}

	r, err := recorder.New(cassette)
	assert.NoError(t, err, "recorder.New should not error")
	assert.NotNil(t, r, "recorder should not be nil")
	defer func() { _ = r.Stop() }()

	httpClient := &http.Client{Transport: r}
	client := NewClient(WithHTTPClient(httpClient))
	ctx := context.Background()
	info, err := client.GetMarketInfo(ctx, "btc")
	assert.NoError(t, err, "GetMarketInfo should not error")
	assert.NotNil(t, info, "info should not be nil")
	assert.NotEmpty(t, info.Symbol, "symbol should not be empty")
	assert.Greater(t, info.MidPrice, 0.0, "mid price should be positive")
}
