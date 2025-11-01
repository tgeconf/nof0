package hyperliquid

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCalculatePriceChange tests the calculatePriceChange helper.
func TestCalculatePriceChange(t *testing.T) {
	tests := []struct {
		name          string
		currentPrice  float64
		previousPrice float64
		wantChange    float64
	}{
		{
			name:          "positive change",
			currentPrice:  110,
			previousPrice: 100,
			wantChange:    10.0,
		},
		{
			name:          "negative change",
			currentPrice:  90,
			previousPrice: 100,
			wantChange:    -10.0,
		},
		{
			name:          "zero previous price",
			currentPrice:  100,
			previousPrice: 0,
			wantChange:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculatePriceChange(tt.currentPrice, tt.previousPrice)
			assert.InDelta(t, tt.wantChange, result, 1e-9)
		})
	}
}

// TestPriceAt tests the priceAt helper.
func TestPriceAt(t *testing.T) {
	klines := []Kline{
		{Close: 100},
		{Close: 110},
		{Close: 120},
	}

	tests := []struct {
		name      string
		klines    []Kline
		stepsBack int
		wantPrice float64
	}{
		{
			name:      "valid steps back",
			klines:    klines,
			stepsBack: 1,
			wantPrice: 110,
		},
		{
			name:      "empty klines",
			klines:    []Kline{},
			stepsBack: 1,
			wantPrice: 0,
		},
		{
			name:      "steps back exceeds length",
			klines:    klines,
			stepsBack: 10,
			wantPrice: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := priceAt(tt.klines, tt.stepsBack)
			assert.Equal(t, tt.wantPrice, result)
		})
	}
}

// TestLastN tests the lastN helper.
func TestLastN(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		count  int
		want   []float64
	}{
		{
			name:   "normal case",
			values: []float64{1, 2, 3, 4, 5},
			count:  3,
			want:   []float64{3, 4, 5},
		},
		{
			name:   "count exceeds length",
			values: []float64{1, 2},
			count:  5,
			want:   []float64{1, 2},
		},
		{
			name:   "empty values",
			values: []float64{},
			count:  3,
			want:   []float64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lastN(tt.values, tt.count)
			assert.Equal(t, tt.want, result)
		})
	}
}

// TestLatestNonNaN tests the latestNonNaN helper.
func TestLatestNonNaN(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		want   float64
		isNaN  bool
	}{
		{
			name:   "last value is valid",
			values: []float64{1.0, 2.0, 3.0},
			want:   3.0,
			isNaN:  false,
		},
		{
			name:   "last values are NaN",
			values: []float64{1.0, 2.0, math.NaN(), math.NaN()},
			want:   2.0,
			isNaN:  false,
		},
		{
			name:   "all values are NaN",
			values: []float64{math.NaN(), math.NaN()},
			isNaN:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := latestNonNaN(tt.values)
			if tt.isNaN {
				assert.True(t, math.IsNaN(result))
			} else {
				assert.Equal(t, tt.want, result)
			}
		})
	}
}
