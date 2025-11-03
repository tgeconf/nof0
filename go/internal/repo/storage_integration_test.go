//go:build integration
// +build integration

package repo_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/stores/cache"

	appconfig "nof0-api/internal/config"
	"nof0-api/internal/svc"
)

func newIntegrationServiceContext(t *testing.T) *svc.ServiceContext {
	t.Helper()
	cfg := appconfig.MustLoad()
	return svc.NewServiceContext(*cfg, cfg.MainPath())
}

func TestPostgresConnectivity(t *testing.T) {
	svcCtx := newIntegrationServiceContext(t)
	db := requirePostgres(t, svcCtx)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var one int
	err := db.QueryRowContext(ctx, "SELECT 1").Scan(&one)
	assert.NoError(t, err, "postgres connectivity check failed")
	assert.Equal(t, 1, one, "postgres returned unexpected value")
}

func TestRedisConnectivity(t *testing.T) {
	svcCtx := newIntegrationServiceContext(t)
	cacheClient := requireCache(t, svcCtx)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	key := fmt.Sprintf("nof0:integration:%d", time.Now().UnixNano())
	const payload = "ok"

	err := cacheClient.SetWithExpireCtx(ctx, key, payload, 10*time.Second)
	assert.NoError(t, err, "cache set failed")
	defer cacheClient.DelCtx(context.Background(), key)

	var value string
	err = cacheClient.GetCtx(ctx, key, &value)
	assert.NoError(t, err, "cache get failed")
	assert.Equal(t, payload, value, "cache value mismatch")
}

func requirePostgres(t *testing.T, svcCtx *svc.ServiceContext) *sql.DB {
	t.Helper()
	if svcCtx.DBConn == nil {
		t.Skip("Postgres not configured (DBConn nil)")
	}
	raw, err := svcCtx.DBConn.RawDB()
	if err != nil {
		t.Fatalf("failed to obtain postgres handle: %v", err)
	}
	return raw
}

func requireCache(t *testing.T, svcCtx *svc.ServiceContext) cache.Cache {
	t.Helper()
	if svcCtx.Cache == nil {
		t.Skip("cache not configured (Cache nil)")
	}
	return svcCtx.Cache
}
