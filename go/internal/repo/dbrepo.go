package repo

import (
	"context"
	"errors"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"nof0-api/internal/data"
	"nof0-api/internal/types"
)

// TTLs bundles cache durations in seconds.
type TTLs struct {
	Short  int
	Medium int
	Long   int
}

// DBRepo loads data from Postgres and caches responses via the go-zero cache layer.
// For resources not yet implemented in DB, it falls back to the file DataLoader.
type DBRepo struct {
	conn     sqlx.SqlConn
	cache    cache.Cache
	fallback *data.DataLoader
	ttls     TTLs
}

func NewDBRepo(conn sqlx.SqlConn, cache cache.Cache, fallback *data.DataLoader, ttls TTLs) *DBRepo {
	return &DBRepo{conn: conn, cache: cache, fallback: fallback, ttls: ttls}
}

// helper: get from redis into v
func (r *DBRepo) getCache(ctx context.Context, key string, v interface{}) (bool, error) {
	if r.cache == nil {
		return false, nil
	}
	if err := r.cache.GetCtx(ctx, key, v); err != nil {
		if r.cache.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// helper: set redis from v
func (r *DBRepo) setCache(ctx context.Context, key string, ttl int, v interface{}) {
	if r.cache == nil || ttl <= 0 {
		return
	}
	expire := time.Duration(ttl) * time.Second
	if err := r.cache.SetWithExpireCtx(ctx, key, v, expire); err != nil {
		logx.WithContext(ctx).Errorf("set cache %s: %v", key, err)
	}
}

// ================= Crypto Prices =================

type cryptoRow struct {
	Symbol    string  `db:"symbol"`
	Price     float64 `db:"price"`
	Timestamp int64   `db:"timestamp_ms"`
}

func (r *DBRepo) LoadCryptoPrices() (*types.CryptoPricesResponse, error) {
	ctx := context.Background()
	const key = "nof0:crypto_prices"
	var cached types.CryptoPricesResponse
	if ok, _ := r.getCache(ctx, key, &cached); ok {
		return &cached, nil
	}

	// Read from materialized view populated by migrations/importer
	const q = `SELECT symbol, price, timestamp_ms FROM v_crypto_prices_latest`

	var rows []cryptoRow
	if err := r.conn.QueryRowsCtx(ctx, &rows, q); err != nil {
		// Fallback to file data
		logx.WithContext(ctx).Errorf("db crypto_prices failed, falling back: %v", err)
		return r.fallback.LoadCryptoPrices()
	}

	resp := &types.CryptoPricesResponse{Prices: map[string]types.CryptoPrice{}, ServerTime: time.Now().UnixMilli()}
	for _, row := range rows {
		resp.Prices[row.Symbol] = types.CryptoPrice{Symbol: row.Symbol, Price: row.Price, Timestamp: row.Timestamp}
	}
	r.setCache(ctx, key, r.ttls.Short, resp)
	return resp, nil
}

// For the complex AccountTotals (nested positions, markers, etc.),
// we currently fall back to file data until a full schema is defined.
func (r *DBRepo) LoadAccountTotals() (*types.AccountTotalsResponse, error) {
	return r.fallback.LoadAccountTotals()
}

func (r *DBRepo) LoadTrades() (*types.TradesResponse, error) { return r.fallback.LoadTrades() }

// ======= Not yet DB-implemented: fallback to file loader =======

func (r *DBRepo) LoadSinceInception() (*types.SinceInceptionResponse, error) {
	return r.fallback.LoadSinceInception()
}

func (r *DBRepo) LoadLeaderboard() (*types.LeaderboardResponse, error) {
	return r.fallback.LoadLeaderboard()
}

func (r *DBRepo) LoadAnalytics() (*types.AnalyticsResponse, error) {
	return r.fallback.LoadAnalytics()
}

func (r *DBRepo) LoadModelAnalytics(modelId string) (*types.ModelAnalyticsResponse, error) {
	if modelId == "" {
		return nil, errors.New("modelId required")
	}
	return r.fallback.LoadModelAnalytics(modelId)
}

func (r *DBRepo) LoadPositions() (*types.PositionsResponse, error) {
	return r.fallback.LoadPositions()
}

func (r *DBRepo) LoadConversations() (*types.ConversationsResponse, error) {
	return r.fallback.LoadConversations()
}
