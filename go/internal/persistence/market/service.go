package marketpersist

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	gocache "github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	cachekeys "nof0-api/internal/cache"
	"nof0-api/internal/model"
	"nof0-api/pkg/market"
)

// Service implements market data persistence and caching hooks.
type Service struct {
	sqlConn          sqlx.SqlConn
	assetsModel      model.MarketAssetsModel
	assetCtxModel    model.MarketAssetCtxModel
	priceLatestModel model.PriceLatestModel
	cache            gocache.Cache
	ttl              cachekeys.TTLSet
}

// Config enumerates dependencies required to persist market data.
type Config struct {
	SQLConn          sqlx.SqlConn
	AssetsModel      model.MarketAssetsModel
	AssetCtxModel    model.MarketAssetCtxModel
	PriceLatestModel model.PriceLatestModel
	Cache            gocache.Cache
	TTL              cachekeys.TTLSet
}

// NewService wires a market persistence service. Returns nil when dependencies missing.
func NewService(cfg Config) market.Persistence {
	if cfg.SQLConn == nil {
		return nil
	}
	return &Service{
		sqlConn:          cfg.SQLConn,
		assetsModel:      cfg.AssetsModel,
		assetCtxModel:    cfg.AssetCtxModel,
		priceLatestModel: cfg.PriceLatestModel,
		cache:            cfg.Cache,
		ttl:              cfg.TTL,
	}
}

// UpsertAssets persists static metadata and refreshes Redis cache.
func (s *Service) UpsertAssets(ctx context.Context, provider string, assets []market.Asset) error {
	if s == nil || s.sqlConn == nil || len(assets) == 0 {
		return nil
	}
	stmt := `
INSERT INTO public.market_assets (
    provider, symbol, name, sz_decimals, max_leverage, only_isolated, margin_table_id, is_delisted, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()
)
ON CONFLICT (provider, symbol) DO UPDATE SET
    name = EXCLUDED.name,
    sz_decimals = EXCLUDED.sz_decimals,
    max_leverage = EXCLUDED.max_leverage,
    only_isolated = EXCLUDED.only_isolated,
    margin_table_id = EXCLUDED.margin_table_id,
    is_delisted = EXCLUDED.is_delisted,
    updated_at = NOW();`
	for _, asset := range assets {
		if strings.TrimSpace(asset.Symbol) == "" {
			continue
		}
		name := asset.Symbol
		if base := strings.TrimSpace(asset.Base); base != "" {
			name = base
		}
		metadata := asset.RawMetadata
		maxLev := nullFloatFromMeta(metadata, "maxLeverage")
		marginTbl := nullIntFromMeta(metadata, "marginTable", "margin_table_id")
		onlyIso := nullBoolFromMeta(metadata, "onlyIsolated", "only_isolated")
		precision := sql.NullInt64{}
		if asset.Precision > 0 {
			precision = sql.NullInt64{Int64: int64(asset.Precision), Valid: true}
		}
		isDelisted := !asset.IsActive
		if _, err := s.sqlConn.ExecCtx(ctx, stmt,
			provider,
			asset.Symbol,
			sql.NullString{String: name, Valid: name != ""},
			precision,
			maxLev,
			onlyIso,
			marginTbl,
			isDelisted,
		); err != nil {
			return err
		}
		s.cacheAsset(ctx, provider, asset)
	}
	return nil
}

// RecordSnapshot persists latest price/context data to Postgres + Redis.
func (s *Service) RecordSnapshot(ctx context.Context, provider string, snapshot *market.Snapshot) error {
	if s == nil || s.sqlConn == nil || snapshot == nil || strings.TrimSpace(snapshot.Symbol) == "" {
		return nil
	}
	now := time.Now().UTC()
	price := snapshot.Price.Last
	raw, _ := json.Marshal(snapshot)
	priceStmt := `
INSERT INTO public.price_latest (provider, symbol, price, ts_ms, raw, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
ON CONFLICT (provider, symbol) DO UPDATE SET
    price = EXCLUDED.price,
    ts_ms = EXCLUDED.ts_ms,
    raw = EXCLUDED.raw,
    updated_at = NOW();`
	if _, err := s.sqlConn.ExecCtx(ctx, priceStmt, provider, snapshot.Symbol, price, now.UnixMilli(), string(raw)); err != nil {
		return err
	}

	ctxStmt := `
INSERT INTO public.market_asset_ctx (
    provider, symbol, funding, open_interest, oracle_px, mark_px, mid_px, impact_pxs, prev_day_px, day_ntl_vlm, day_base_vlm, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, NULL, $5, NULL, NULL, NULL, NULL, NULL, NOW(), NOW()
)
ON CONFLICT (provider, symbol) DO UPDATE SET
    funding = EXCLUDED.funding,
    open_interest = EXCLUDED.open_interest,
    mark_px = EXCLUDED.mark_px,
    updated_at = NOW();`
	funding := sql.NullFloat64{}
	if snapshot.Funding != nil {
		funding = sql.NullFloat64{Float64: snapshot.Funding.Rate, Valid: true}
	}
	openInterest := sql.NullFloat64{}
	if snapshot.OpenInterest != nil {
		openInterest = sql.NullFloat64{Float64: snapshot.OpenInterest.Latest, Valid: true}
	}
	if _, err := s.sqlConn.ExecCtx(ctx, ctxStmt, provider, snapshot.Symbol, funding, openInterest, price); err != nil {
		return err
	}

	s.cachePrice(ctx, provider, snapshot.Symbol, price, now)
	s.cacheMarketCtx(ctx, provider, snapshot)
	s.updateCryptoPrices(ctx, provider, snapshot.Symbol, price)
	return nil
}

func (s *Service) cacheAsset(ctx context.Context, provider string, asset market.Asset) {
	if s.cache == nil {
		return
	}
	key := cachekeys.MarketAssetKey(provider, asset.Symbol)
	ttl := s.ttl.Duration(cachekeys.TTLLong)
	if ttl <= 0 {
		ttl = cachekeys.MarketAssetTTL(s.ttl)
	}
	payload := map[string]any{
		"symbol":     asset.Symbol,
		"base":       asset.Base,
		"quote":      asset.Quote,
		"precision":  asset.Precision,
		"is_active":  asset.IsActive,
		"metadata":   asset.RawMetadata,
		"updated_at": time.Now().UTC().UnixMilli(),
	}
	if err := s.cache.SetWithExpireCtx(ctx, key, payload, ttl); err != nil {
		logx.WithContext(ctx).Errorf("marketpersist: cache asset key=%s err=%v", key, err)
	}
}

func (s *Service) cachePrice(ctx context.Context, provider, symbol string, price float64, ts time.Time) {
	if s.cache == nil {
		return
	}
	ttl := cachekeys.PriceTTL(s.ttl)
	if ttl <= 0 {
		return
	}
	// Provider scoped key
	key := cachekeys.PriceLatestByProviderKey(provider, symbol)
	payload := map[string]any{
		"price": price,
		"ts":    ts.UnixMilli(),
	}
	if err := s.cache.SetWithExpireCtx(ctx, key, payload, ttl); err != nil {
		logx.WithContext(ctx).Errorf("marketpersist: cache price key=%s err=%v", key, err)
	}
	// Global key
	global := cachekeys.PriceLatestKey(symbol)
	if err := s.cache.SetWithExpireCtx(ctx, global, payload, ttl); err != nil {
		logx.WithContext(ctx).Errorf("marketpersist: cache price key=%s err=%v", global, err)
	}
}

func (s *Service) cacheMarketCtx(ctx context.Context, provider string, snapshot *market.Snapshot) {
	if s.cache == nil {
		return
	}
	ttl := cachekeys.MarketAssetCtxTTL(s.ttl)
	if ttl <= 0 {
		return
	}
	key := cachekeys.MarketAssetCtxKey(provider, snapshot.Symbol)
	ctxPayload := map[string]any{
		"price":        snapshot.Price.Last,
		"change":       snapshot.Change,
		"funding":      snapshot.Funding,
		"oi":           snapshot.OpenInterest,
		"indicators":   snapshot.Indicators,
		"timestamp_ms": time.Now().UTC().UnixMilli(),
	}
	if err := s.cache.SetWithExpireCtx(ctx, key, ctxPayload, ttl); err != nil {
		logx.WithContext(ctx).Errorf("marketpersist: cache ctx key=%s err=%v", key, err)
	}
}

func (s *Service) updateCryptoPrices(ctx context.Context, provider, symbol string, price float64) {
	if s.cache == nil {
		return
	}
	key := cachekeys.CryptoPricesKey()
	var payload map[string]float64
	if err := s.cache.GetCtx(ctx, key, &payload); err != nil && !s.cache.IsNotFound(err) {
		logx.WithContext(ctx).Errorf("marketpersist: load crypto prices key=%s err=%v", key, err)
		return
	}
	if payload == nil {
		payload = make(map[string]float64)
	}
	field := fmt.Sprintf("%s:%s", provider, symbol)
	payload[field] = price
	ttl := cachekeys.CryptoPricesTTL(s.ttl)
	if ttl <= 0 {
		return
	}
	if err := s.cache.SetWithExpireCtx(ctx, key, payload, ttl); err != nil {
		logx.WithContext(ctx).Errorf("marketpersist: cache crypto prices key=%s err=%v", key, err)
	}
}

func nullFloatFromMeta(meta map[string]any, keys ...string) sql.NullFloat64 {
	for _, key := range keys {
		if v, ok := meta[key]; ok {
			if f, conv := toFloat64(v); conv {
				return sql.NullFloat64{Float64: f, Valid: true}
			}
		}
	}
	return sql.NullFloat64{}
}

func nullIntFromMeta(meta map[string]any, keys ...string) sql.NullInt64 {
	for _, key := range keys {
		if v, ok := meta[key]; ok {
			switch t := v.(type) {
			case int:
				return sql.NullInt64{Int64: int64(t), Valid: true}
			case int64:
				return sql.NullInt64{Int64: t, Valid: true}
			case float64:
				return sql.NullInt64{Int64: int64(t), Valid: true}
			case json.Number:
				if val, err := t.Int64(); err == nil {
					return sql.NullInt64{Int64: val, Valid: true}
				}
			}
		}
	}
	return sql.NullInt64{}
}

func nullBoolFromMeta(meta map[string]any, keys ...string) sql.NullBool {
	for _, key := range keys {
		if v, ok := meta[key]; ok {
			switch t := v.(type) {
			case bool:
				return sql.NullBool{Bool: t, Valid: true}
			case string:
				if strings.EqualFold(t, "true") {
					return sql.NullBool{Bool: true, Valid: true}
				}
				if strings.EqualFold(t, "false") {
					return sql.NullBool{Bool: false, Valid: true}
				}
			}
		}
	}
	return sql.NullBool{}
}

func toFloat64(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(t, 64)
		return f, err == nil
	default:
		return 0, false
	}
}
