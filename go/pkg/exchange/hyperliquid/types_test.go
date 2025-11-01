package hyperliquid

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetaAndAssetCtxsResponse_UnmarshalJSON(t *testing.T) {
	t.Run("object_format", func(t *testing.T) {
		jsonData := `{
			"universe": [
				{"name": "BTC", "szDecimals": 5, "maxLeverage": 20.0, "marginTableId": 1, "onlyIsolated": false, "isDelisted": false},
				{"name": "ETH", "szDecimals": 4, "maxLeverage": 15.0, "marginTableId": 2, "onlyIsolated": false, "isDelisted": false}
			],
			"assetCtxs": [
				{
					"funding": "0.0001",
					"openInterest": "1000000",
					"prevDayPx": "49000.0",
					"dayNtlVlm": "500000000",
					"dayBaseVlm": "10000",
					"premium": "0.0002",
					"oraclePx": "50000.0",
					"markPx": "50001.0",
					"midPx": "50002.0",
					"impactPxs": ["49900.0", "50100.0"]
				},
				{
					"funding": "0.0002",
					"openInterest": "2000000",
					"prevDayPx": "2900.0",
					"dayNtlVlm": "300000000",
					"dayBaseVlm": "100000",
					"premium": "0.0003",
					"oraclePx": "3000.0",
					"markPx": "3001.0",
					"midPx": "3002.0",
					"impactPxs": ["2990.0", "3010.0"]
				}
			]
		}`

		var resp MetaAndAssetCtxsResponse
		err := json.Unmarshal([]byte(jsonData), &resp)
		assert.NoError(t, err)
		assert.Len(t, resp.Universe, 2)
		assert.Len(t, resp.AssetCtxs, 2)

		assert.Equal(t, "BTC", resp.Universe[0].Name)
		assert.Equal(t, 5, resp.Universe[0].SzDecimals)
		assert.Equal(t, 20.0, resp.Universe[0].MaxLeverage)

		assert.Equal(t, "ETH", resp.Universe[1].Name)
		assert.Equal(t, 4, resp.Universe[1].SzDecimals)

		assert.Equal(t, "50001.0", resp.AssetCtxs[0].MarkPx)
		assert.Equal(t, "50002.0", resp.AssetCtxs[0].MidPx)
		assert.Equal(t, "3001.0", resp.AssetCtxs[1].MarkPx)
	})

	t.Run("array_format_legacy", func(t *testing.T) {
		jsonData := `[
			{
				"universe": [
					{"name": "BTC", "szDecimals": 5, "maxLeverage": 20.0, "marginTableId": 1, "onlyIsolated": false, "isDelisted": false}
				]
			},
			[
				{
					"funding": "0.0001",
					"openInterest": "1000000",
					"prevDayPx": "49000.0",
					"dayNtlVlm": "500000000",
					"dayBaseVlm": "10000",
					"premium": "0.0002",
					"oraclePx": "50000.0",
					"markPx": "50001.0",
					"midPx": "50002.0",
					"impactPxs": ["49900.0", "50100.0"]
				}
			]
		]`

		var resp MetaAndAssetCtxsResponse
		err := json.Unmarshal([]byte(jsonData), &resp)
		assert.NoError(t, err)
		assert.Len(t, resp.Universe, 1)
		assert.Len(t, resp.AssetCtxs, 1)

		assert.Equal(t, "BTC", resp.Universe[0].Name)
		assert.Equal(t, "50001.0", resp.AssetCtxs[0].MarkPx)
	})

	t.Run("array_format_no_assetCtxs", func(t *testing.T) {
		jsonData := `[
			{
				"universe": [
					{"name": "BTC", "szDecimals": 5, "maxLeverage": 20.0, "marginTableId": 1, "onlyIsolated": false, "isDelisted": false}
				]
			}
		]`

		var resp MetaAndAssetCtxsResponse
		err := json.Unmarshal([]byte(jsonData), &resp)
		assert.NoError(t, err)
		assert.Len(t, resp.Universe, 1)
		assert.Empty(t, resp.AssetCtxs)
	})

	t.Run("invalid_json", func(t *testing.T) {
		jsonData := `invalid json`

		var resp MetaAndAssetCtxsResponse
		err := json.Unmarshal([]byte(jsonData), &resp)
		assert.Error(t, err)
	})

	t.Run("empty_array", func(t *testing.T) {
		jsonData := `[]`

		var resp MetaAndAssetCtxsResponse
		err := json.Unmarshal([]byte(jsonData), &resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty payload")
	})

	t.Run("empty_object", func(t *testing.T) {
		jsonData := `{}`

		var resp MetaAndAssetCtxsResponse
		err := json.Unmarshal([]byte(jsonData), &resp)
		// Empty object should try object format first, which won't have any data
		// This will result in the fallback to array format, which will fail
		assert.Error(t, err)
	})

	t.Run("object_with_only_universe", func(t *testing.T) {
		jsonData := `{
			"universe": [
				{"name": "BTC", "szDecimals": 5, "maxLeverage": 20.0, "marginTableId": 1, "onlyIsolated": false, "isDelisted": false}
			]
		}`

		var resp MetaAndAssetCtxsResponse
		err := json.Unmarshal([]byte(jsonData), &resp)
		assert.NoError(t, err)
		assert.Len(t, resp.Universe, 1)
		assert.Empty(t, resp.AssetCtxs)
	})

	t.Run("object_with_only_assetCtxs", func(t *testing.T) {
		jsonData := `{
			"assetCtxs": [
				{"markPx": "50001.0", "midPx": "50002.0"}
			]
		}`

		var resp MetaAndAssetCtxsResponse
		err := json.Unmarshal([]byte(jsonData), &resp)
		assert.NoError(t, err)
		assert.Empty(t, resp.Universe)
		assert.Len(t, resp.AssetCtxs, 1)
	})

	t.Run("array_format_invalid_universe", func(t *testing.T) {
		jsonData := `[
			{"invalid": "data"},
			[]
		]`

		var resp MetaAndAssetCtxsResponse
		err := json.Unmarshal([]byte(jsonData), &resp)
		// Should succeed but universe will be empty
		assert.NoError(t, err)
		assert.Empty(t, resp.Universe)
	})
}
