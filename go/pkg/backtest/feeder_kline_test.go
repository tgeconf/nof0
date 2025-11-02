package backtest

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCSVKlineFeeder(t *testing.T) {
	data := []byte("ts,close\n1,100\n2,101\n3,102\n")
	feeder, err := NewCSVKlineFeeder("BTC", bytes.NewReader(data))
	assert.NoError(t, err, "NewCSVKlineFeeder should not error")
	assert.NotNil(t, feeder, "feeder should not be nil")

	ctx := context.Background()

	snap, ok, err := feeder.Next(ctx, "BTC")
	assert.NoError(t, err, "Next1 should not error")
	assert.True(t, ok, "Next1 should return ok=true")
	assert.Equal(t, float64(100), snap.Price.Last, "first price should be 100")

	snap, ok, err = feeder.Next(ctx, "BTC")
	assert.NoError(t, err, "Next2 should not error")
	assert.True(t, ok, "Next2 should return ok=true")
	assert.Equal(t, float64(101), snap.Price.Last, "second price should be 101")
	assert.InDelta(t, 0.01, snap.Change.OneHour, 1e-9, "OneHour change should equal fractional 1%%")

	_, ok, err = feeder.Next(ctx, "BTC")
	assert.NoError(t, err, "Next3 should not error")
	assert.True(t, ok, "Next3 should return ok=true")

	_, ok, err = feeder.Next(ctx, "BTC")
	assert.NoError(t, err, "Next4 should not error")
	assert.False(t, ok, "Next4 should return ok=false at EOF")
}
