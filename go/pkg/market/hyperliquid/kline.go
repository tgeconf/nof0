package hyperliquid

import (
	"context"
	"fmt"
	"sort"
	"time"
)

var intervalDurations = map[string]time.Duration{
	"3m": 3 * time.Minute,
	"4h": 4 * time.Hour,
}

// GetKlines fetches OHLCV data for the given interval.
func (c *Client) GetKlines(ctx context.Context, symbol string, interval string, limit int) ([]Kline, error) {
	duration, ok := intervalDurations[interval]
	if !ok {
		return nil, fmt.Errorf("hyperliquid: unsupported interval %q", interval)
	}
	if limit <= 0 {
		return nil, fmt.Errorf("hyperliquid: limit must be positive")
	}

	canonical, err := c.canonicalSymbolFor(ctx, symbol)
	if err != nil {
		return nil, err
	}
	endTime := time.Now().UTC()
	startTime := endTime.Add(-duration * time.Duration(limit+10))

	var response CandleResponse
	request := InfoRequest{
		Type: "candleSnapshot",
		Req: CandleSnapshotRequest{
			Coin:      canonical,
			Interval:  interval,
			StartTime: startTime.UnixMilli(),
			EndTime:   endTime.UnixMilli(),
		},
	}

	if err := c.doRequest(ctx, request, &response); err != nil {
		return nil, err
	}
	if len(response) == 0 {
		return nil, fmt.Errorf("hyperliquid: empty kline response for %s %s", canonical, interval)
	}

	klines := make([]Kline, 0, len(response))
	for _, item := range response {
		klines = append(klines, Kline{
			OpenTime:  item.T,
			Open:      item.O,
			High:      item.H,
			Low:       item.L,
			Close:     item.C,
			Volume:    item.V,
			CloseTime: item.TClose,
		})
	}

	sort.Slice(klines, func(i, j int) bool {
		return klines[i].OpenTime < klines[j].OpenTime
	})

	if len(klines) > limit {
		klines = klines[len(klines)-limit:]
	}

	return klines, nil
}
