package hyperliquid

import "errors"

var (
	// ErrFeatureUnavailable indicates a feature stub has not been implemented yet.
	ErrFeatureUnavailable = errors.New("hyperliquid: feature unavailable")
)
