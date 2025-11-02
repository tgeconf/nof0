package backtest

import (
	"context"
	"encoding/csv"
	"io"
	"os"
	"strconv"

	"nof0-api/pkg/market"
)

// CSVKlineFeeder reads a 2-column CSV (ts,close) and emits snapshots by close.
// The timestamp column is optional and ignored for now.
type CSVKlineFeeder struct {
	symbol string
	closes []float64
	idx    int
}

// NewCSVKlineFeederFromFile constructs a CSV feeder from a file path.
func NewCSVKlineFeederFromFile(symbol, path string) (*CSVKlineFeeder, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return NewCSVKlineFeeder(symbol, f)
}

// NewCSVKlineFeeder constructs a CSV feeder from an io.Reader.
func NewCSVKlineFeeder(symbol string, r io.Reader) (*CSVKlineFeeder, error) {
	cr := csv.NewReader(r)
	records, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}
	var closes []float64
	for i, rec := range records {
		if len(rec) == 0 {
			continue
		}
		// Skip header if first row has non-numeric in column 0/1
		if i == 0 {
			if _, err0 := strconv.ParseFloat(rec[0], 64); err0 != nil {
				if len(rec) > 1 {
					if _, err1 := strconv.ParseFloat(rec[1], 64); err1 != nil {
						continue
					}
					// parse column 1 as close
					v, _ := strconv.ParseFloat(rec[1], 64)
					closes = append(closes, v)
					continue
				}
				continue
			}
		}
		// Use last column as close
		col := rec[len(rec)-1]
		if v, err := strconv.ParseFloat(col, 64); err == nil {
			closes = append(closes, v)
		}
	}
	return &CSVKlineFeeder{symbol: symbol, closes: closes}, nil
}

// Next returns the next snapshot built from the close series.
func (f *CSVKlineFeeder) Next(ctx context.Context, symbol string) (*market.Snapshot, bool, error) {
	if f.idx >= len(f.closes) {
		return nil, false, nil
	}
	px := f.closes[f.idx]
	f.idx++
	var oneHour, fourHour float64 // fractional change ratios (0.01 == +1%)
	if f.idx >= 2 {
		prev := f.closes[f.idx-2]
		if prev != 0 {
			oneHour = (px - prev) / prev
			fourHour = oneHour
		}
	}
	return &market.Snapshot{
		Symbol:     symbol,
		Price:      market.PriceInfo{Last: px},
		Change:     market.ChangeInfo{OneHour: oneHour, FourHour: fourHour},
		Indicators: market.IndicatorInfo{EMA: map[string]float64{"EMA2": px}},
	}, true, nil
}
