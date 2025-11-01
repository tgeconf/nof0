package hyperliquid

import (
	"bytes"
	"context"
	"encoding/binary"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	mathhex "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	"nof0-api/pkg/exchange"
)

func TestCanonicalAssetKey(t *testing.T) {
	tests := map[string]string{
		"btc":   "BTC",
		"  Eth": "ETH",
		"kPEPE": "KPEPE",
		"":      "",
		"   ":   "",
		"DOT\n": "DOT",
	}
	for input, expected := range tests {
		require.Equalf(t, expected, canonicalAssetKey(input), "canonicalAssetKey(%q)", input)
	}
}

func TestBuildPlaceOrderAction(t *testing.T) {
	order := exchange.Order{
		Asset:      3,
		IsBuy:      true,
		LimitPx:    "123.45",
		Sz:         "0.25",
		ReduceOnly: false,
		OrderType: exchange.OrderType{
			Limit: &exchange.LimitOrderType{TIF: "Gtc"},
		},
		Cloid: "order-1",
	}
	action, err := buildPlaceOrderAction([]exchange.Order{order})
	require.NoError(t, err)
	require.Equal(t, ActionTypeOrder, action.Type)
	require.Equal(t, "na", action.Grouping)
	require.Len(t, action.Orders, 1)

	payload := action.Orders[0]
	require.Equal(t, order.Asset, payload.Asset)
	require.Equal(t, order.IsBuy, payload.IsBuy)
	require.Equal(t, order.LimitPx, payload.LimitPx)
	require.Equal(t, order.Sz, payload.Sz)
	require.NotNil(t, payload.OrderType.Limit)
	require.Equal(t, order.OrderType.Limit.TIF, payload.OrderType.Limit.TIF)
	require.Equal(t, order.Cloid, payload.Cloid)
}

func TestBuildPlaceOrderActionWithBuilder(t *testing.T) {
	order := exchange.Order{
		Asset:      3,
		IsBuy:      true,
		LimitPx:    "123.45",
		Sz:         "0.25",
		ReduceOnly: false,
		OrderType: exchange.OrderType{
			Limit: &exchange.LimitOrderType{TIF: "Gtc"},
		},
		Builder:  &exchange.BuilderInfo{Name: "builder-addr", FeeBps: 25},
		Grouping: "bundle",
	}
	action, err := buildPlaceOrderAction([]exchange.Order{order})
	require.NoError(t, err)
	require.Equal(t, "bundle", action.Grouping)
	require.NotNil(t, action.Builder)
	require.Equal(t, "builder-addr", action.Builder.Builder)
	require.Equal(t, 25, action.Builder.Fee)
}

func TestBuildPlaceOrderActionGroupingMismatch(t *testing.T) {
	orders := []exchange.Order{
		{
			Asset:   1,
			IsBuy:   true,
			LimitPx: "1",
			Sz:      "1",
			OrderType: exchange.OrderType{
				Limit: &exchange.LimitOrderType{TIF: "Gtc"},
			},
			Grouping: "g1",
		},
		{
			Asset:   1,
			IsBuy:   false,
			LimitPx: "2",
			Sz:      "1",
			OrderType: exchange.OrderType{
				Limit: &exchange.LimitOrderType{TIF: "Gtc"},
			},
			Grouping: "g2",
		},
	}
	_, err := buildPlaceOrderAction(orders)
	require.Error(t, err)
}

func TestBuildEIP712Message(t *testing.T) {
	order := exchange.Order{
		Asset:   1,
		IsBuy:   true,
		LimitPx: "50000.0",
		Sz:      "0.001",
		OrderType: exchange.OrderType{
			Limit: &exchange.LimitOrderType{TIF: "Gtc"},
		},
	}
	action, err := buildPlaceOrderAction([]exchange.Order{order})
	require.NoError(t, err)

	nonce := int64(1700000000000)
	digest, err := buildEIP712Message(action, nonce, "", true)
	require.NoError(t, err)
	require.Len(t, digest, 32)

	expected := computeReferenceDigest(t, action, nonce, "", true)
	require.Equal(t, expected, digest)
}

func TestSignActionDeterministic(t *testing.T) {
	order := exchange.Order{
		Asset:   2,
		IsBuy:   false,
		LimitPx: "3200.5",
		Sz:      "1.5",
		OrderType: exchange.OrderType{
			Limit: &exchange.LimitOrderType{TIF: "Ioc"},
		},
		Cloid: "abc123",
	}
	action, err := buildPlaceOrderAction([]exchange.Order{order})
	require.NoError(t, err)

	const keyHex = "0x4c0883a69102937d6231471b5dbb6204fe5129617082796fe3f6a4ab2ed5f8d2"
	signer, err := NewPrivateKeySigner(keyHex)
	require.NoError(t, err)

	nonce := int64(1700000005000)
	req, err := signAction(action, signer, nonce, "", "", true)
	require.NoError(t, err)
	require.Equal(t, nonce, req.Nonce)
	require.Equal(t, action, req.Action)
	require.Equal(t, "", req.VaultAddress)

	expectedDigest := computeReferenceDigest(t, action, nonce, "", true)
	sigBytes, err := crypto.Sign(expectedDigest, signer.privateKey)
	require.NoError(t, err)

	require.Equal(t, "0x"+common.Bytes2Hex(sigBytes[:32]), req.Signature.R)
	require.Equal(t, "0x"+common.Bytes2Hex(sigBytes[32:64]), req.Signature.S)
	require.Equal(t, int(sigBytes[64])+27, req.Signature.V)
}

func computeReferenceDigest(t *testing.T, action Action, nonce int64, vault string, isMainnet bool) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.UseCompactInts(true)
	require.NoError(t, enc.Encode(action))
	msgpackBytes := convertStr16ToStr8(buf.Bytes())

	var nonceBytes [8]byte
	binary.BigEndian.PutUint64(nonceBytes[:], uint64(nonce))

	payload := append(msgpackBytes, nonceBytes[:]...)
	if vault == "" {
		payload = append(payload, 0x00)
	} else {
		require.True(t, common.IsHexAddress(vault))
		payload = append(payload, 0x01)
		payload = append(payload, common.HexToAddress(vault).Bytes()...)
	}

	connectionID := crypto.Keccak256(payload)

	source := "a"
	if !isMainnet {
		source = "b"
	}
	chainID := int64(1337)

	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"Agent": {
				{Name: "source", Type: "string"},
				{Name: "connectionId", Type: "bytes32"},
			},
		},
		PrimaryType: "Agent",
		Domain: apitypes.TypedDataDomain{
			Name:              "Exchange",
			Version:           "1",
			ChainId:           mathhex.NewHexOrDecimal256(chainID),
			VerifyingContract: verifyingContractHex,
		},
		Message: map[string]interface{}{
			"source":       source,
			"connectionId": connectionID,
		},
	}

	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	require.NoError(t, err)
	messageHash, err := typedData.HashStruct("Agent", typedData.Message)
	require.NoError(t, err)

	return crypto.Keccak256(append(append([]byte{0x19, 0x01}, domainSeparator...), messageHash...))
}

func TestValidateOrder(t *testing.T) {
	err := validateOrder(exchange.Order{
		Asset:   -1,
		LimitPx: "100",
		Sz:      "1",
	})
	require.Error(t, err)

	err = validateOrder(exchange.Order{
		Asset:   1,
		LimitPx: "0",
		Sz:      "1",
	})
	require.Error(t, err)

	err = validateOrder(exchange.Order{
		Asset:   1,
		LimitPx: "10",
		Sz:      "0",
	})
	require.Error(t, err)
}

func TestConvertOrder_Trigger(t *testing.T) {
	// Build a trigger order: market take-profit at 25000
	ord := exchange.Order{
		Asset:      1,
		IsBuy:      false,
		Sz:         "0.1",
		TriggerPx:  "25000",
		OrderType:  exchange.OrderType{Trigger: &exchange.TriggerOrderType{IsMarket: true, Tpsl: "tp"}},
		ReduceOnly: true,
	}
	payload, err := convertOrder(ord)
	require.NoError(t, err)
	require.Nil(t, payload.OrderType.Limit)
	require.NotNil(t, payload.OrderType.Trigger)
	require.Equal(t, "25000", payload.OrderType.Trigger.TriggerPx)
	require.Equal(t, "tp", payload.OrderType.Trigger.Tpsl)
	require.True(t, payload.OrderType.Trigger.IsMarket)
	// top-level trigger fields should be empty when nested under orderType.trigger
	require.Equal(t, "", payload.TriggerPx)
	require.Equal(t, "", payload.TriggerRel)
}

func TestValidateOrder_TriggerOnly(t *testing.T) {
	ord := exchange.Order{
		Asset:     1,
		IsBuy:     true,
		Sz:        "0.05",
		TriggerPx: "123.45",
		OrderType: exchange.OrderType{Trigger: &exchange.TriggerOrderType{IsMarket: true, Tpsl: "sl"}},
	}
	err := validateOrder(ord)
	require.NoError(t, err)
	_, err = buildPlaceOrderAction([]exchange.Order{ord})
	require.NoError(t, err)
}

func TestIsZeroDecimal(t *testing.T) {
	require.True(t, isZeroDecimal("0"))
	require.True(t, isZeroDecimal("-0.000"))
	require.True(t, isZeroDecimal(" 0.00000 "))
	require.False(t, isZeroDecimal("0.01"))
	require.False(t, isZeroDecimal("-1"))
}

func TestTrimSign(t *testing.T) {
	require.Equal(t, "1.25", trimSign("-1.25"))
	require.Equal(t, "0.01", trimSign("+0.01"))
	require.Equal(t, "2", trimSign("--2"))
	require.Equal(t, "3", trimSign("  - +3"))
}

func TestComputeCloseLimit(t *testing.T) {
	buy := computeCloseLimit("100", true)
	buyVal, err := strconv.ParseFloat(buy, 64)
	require.NoError(t, err)
	require.InEpsilon(t, 100*(1+closePriceSlippage), buyVal, 1e-9)

	sell := computeCloseLimit("100", false)
	sellVal, err := strconv.ParseFloat(sell, 64)
	require.NoError(t, err)
	require.InEpsilon(t, 100*(1-closePriceSlippage), sellVal, 1e-9)

	fallback := computeCloseLimit("", false)
	require.Equal(t, defaultAggressiveSellLimit, fallback)
}

func TestBuildCloseOrder(t *testing.T) {
	pos := exchange.Position{Szi: "0.5"}
	mark := "200"
	order, ok, err := buildCloseOrder(12, mark, pos)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 12, order.Asset)
	require.False(t, order.IsBuy)
	sellVal, err := strconv.ParseFloat(order.LimitPx, 64)
	require.NoError(t, err)
	require.InEpsilon(t, 200*(1-closePriceSlippage), sellVal, 1e-9)
	require.Equal(t, "0.5", order.Sz)
	require.True(t, order.ReduceOnly)
	require.Equal(t, "Ioc", order.OrderType.Limit.TIF)

	short := exchange.Position{Szi: "-1.25"}
	order, ok, err = buildCloseOrder(3, mark, short)
	require.NoError(t, err)
	require.True(t, ok)
	require.True(t, order.IsBuy)
	buyVal, err := strconv.ParseFloat(order.LimitPx, 64)
	require.NoError(t, err)
	require.InEpsilon(t, 200*(1+closePriceSlippage), buyVal, 1e-9)
	require.Equal(t, "1.25", order.Sz)

	flat := exchange.Position{Szi: "0"}
	_, ok, err = buildCloseOrder(1, mark, flat)
	require.NoError(t, err)
	require.False(t, ok)

	// Invalid mark should fall back to aggressive levels
	invalidMark := ""
	order, ok, err = buildCloseOrder(5, invalidMark, short)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, defaultAggressiveBuyLimit, order.LimitPx)
}

func TestBuildCancelByCloidAction(t *testing.T) {
	action, err := buildCancelByCloidAction([]CancelByCloid{{Asset: 5, Cloid: "abc"}})
	require.NoError(t, err)
	require.Equal(t, ActionTypeCancelByCloid, action.Type)
	require.Len(t, action.Cancels, 1)
	require.Equal(t, 5, action.Cancels[0].Asset)
	require.Equal(t, "abc", action.Cancels[0].Cloid)
}

func TestBuildModifyAction(t *testing.T) {
	oid := int64(42)
	req := ModifyOrderRequest{
		Oid: &oid,
		Order: exchange.Order{
			Asset:   1,
			IsBuy:   true,
			LimitPx: "100",
			Sz:      "1",
			OrderType: exchange.OrderType{
				Limit: &exchange.LimitOrderType{TIF: "Gtc"},
			},
		},
	}
	action, err := buildModifyAction(req)
	require.NoError(t, err)
	require.Equal(t, ActionTypeModify, action.Type)
	require.Equal(t, oid, action.Oid)
	require.Equal(t, "100", action.Order.LimitPx)

	req = ModifyOrderRequest{
		Cloid: "cl-123",
		Order: exchange.Order{
			Asset:   1,
			IsBuy:   false,
			LimitPx: "90",
			Sz:      "1",
			OrderType: exchange.OrderType{
				Limit: &exchange.LimitOrderType{TIF: "Gtc"},
			},
		},
	}
	action, err = buildModifyAction(req)
	require.NoError(t, err)
	require.Equal(t, "cl-123", action.Oid)
}

func TestFormatSizeAndIOCMarket(t *testing.T) {
	// Build a client with fake signer; only utility methods exercised (no HTTP)
	c, err := NewClient("0x4c0883a69102937d6231471b5dbb6204fe5129617082796fe3f6a4ab2ed5f8d2", true)
	require.NoError(t, err)
	// inject asset directory cache directly to avoid HTTP
	c.assetMu.Lock()
	c.assetIndex = map[string]int{"BTC": 0}
	c.assetInfo = map[string]AssetInfo{"BTC": {
		Name: "BTC", SzDecimals: 3, Index: 0, MidPx: "50000", MarkPx: "50010",
	}}
	c.assetMu.Unlock()

	sz, err := c.FormatSize(context.Background(), "BTC", 0.12349)
	require.NoError(t, err)
	require.Equal(t, "0.123", sz)

	// Verify RoundPriceToSigFigs
	p := RoundPriceToSigFigs(50000*1.01, 5)
	require.NotEmpty(t, p)
}

func TestClientOptionsDefaults(t *testing.T) {
	// defaults
	c, err := NewClient("0x4c0883a69102937d6231471b5dbb6204fe5129617082796fe3f6a4ab2ed5f8d2", true)
	require.NoError(t, err)
	require.Equal(t, 5, c.priceSigFigs)
	require.Equal(t, 0.0, c.defaultSlippage)

	// overrides
	c2, err := NewClient("0x4c0883a69102937d6231471b5dbb6204fe5129617082796fe3f6a4ab2ed5f8d2", true,
		WithPriceSigFigs(4), WithDefaultSlippage(0.02))
	require.NoError(t, err)
	require.Equal(t, 4, c2.priceSigFigs)
	require.InDelta(t, 0.02, c2.defaultSlippage, 1e-12)
}
