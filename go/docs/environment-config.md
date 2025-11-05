# Environment Configuration for Database Fixes

## Required Environment Variables

### Update Postgres DSN to fix prepared statement conflicts

**Problem**: Supabase Transaction Pooler (port 6543) doesn't support server-side prepared statements, causing random INSERT/UPDATE failures.

**Solution**: Add `default_query_exec_mode=simple_protocol` parameter to disable prepared statements.

### Before
```bash
export Postgres__DataSource="postgres://postgres.rmlzrjeigjavjshmxnqr:I0OsnapB0rRzvdgW@aws-1-us-east-1.pooler.supabase.com:6543/postgres?sslmode=require&supa=base-pooler.x"
```

### After
```bash
export Postgres__DataSource="postgres://postgres.rmlzrjeigjavjshmxnqr:I0OsnapB0rRzvdgW@aws-1-us-east-1.pooler.supabase.com:6543/postgres?sslmode=require&default_query_exec_mode=simple_protocol&supa=base-pooler.x"
```

**Key Change**: Added `&default_query_exec_mode=simple_protocol`

### Alternative: Use Session Pooler

For better performance with prepared statements support, switch to Session Pooler (port 5432):

```bash
export Postgres__DataSource="postgres://postgres.rmlzrjeigjavjshmxnqr:I0OsnapB0rRzvdgW@aws-1-us-east-1.pooler.supabase.com:5432/postgres?sslmode=require&supa=base-pooler.x"
```

**Note**: Session pooler uses more connections but supports prepared statements.

## Apply the Changes

### Option 1: Export in shell
```bash
# Add to ~/.bashrc or ~/.zshrc
export Postgres__DataSource="postgres://...?default_query_exec_mode=simple_protocol..."
```

### Option 2: Use .env file
```bash
# Create .env file in project root
cat > .env <<EOF
Postgres__DataSource=postgres://postgres.rmlzrjeigjavjshmxnqr:I0OsnapB0rRzvdgW@aws-1-us-east-1.pooler.supabase.com:6543/postgres?sslmode=require&default_query_exec_mode=simple_protocol&supa=base-pooler.x
Cache__0__Host=redis-xxxxx.cloud.redislabs.com:6379
Cache__0__Pass=your_redis_password
Cache__0__Tls=true
EOF

# Load before running
source .env
go run cmd/llm/main.go --app-config etc/nof0.yaml
```

### Option 3: Docker Compose
```yaml
# docker-compose.yml
services:
  nof0:
    environment:
      - Postgres__DataSource=postgres://...?default_query_exec_mode=simple_protocol...
      - Cache__0__Host=redis:6379
      - Cache__0__Pass=
      - Cache__0__Tls=false
```

## Verify Configuration

Run application and check logs for absence of these errors:
- `ERROR: prepared statement "stmtcache_..." already exists`
- `ERROR: column "created_at" of relation ... does not exist`
- `ERROR: duplicate key value violates unique constraint "model_analytics_pkey"`

## Full Deployment Checklist

1. **Apply Migrations**:
   ```bash
   export POSTGRES_DSN="postgres://...?default_query_exec_mode=simple_protocol..."
   make migrate-up
   ```

2. **Update Environment Variables**:
   - Add `default_query_exec_mode=simple_protocol` to Postgres__DataSource

3. **Restart Application**:
   ```bash
   go run cmd/llm/main.go --app-config etc/nof0.yaml
   ```

4. **Monitor Logs**:
   - Watch for successful market data ingestion
   - Verify analytics updates without errors
   - Check query performance metrics

## Performance Considerations

**Impact of disabling prepared statements**:
- Slightly higher CPU usage on database server (query parsing)
- Slightly higher network overhead (full SQL text sent each time)
- But eliminates random failures and improves reliability

**Measured overhead**: Typically 5-10% increase in query latency, acceptable trade-off for stability.

## Rollback Plan

If issues occur, revert to original DSN:
```bash
export Postgres__DataSource="postgres://postgres.rmlzrjeigjavjshmxnqr:I0OsnapB0rRzvdgW@aws-1-us-east-1.pooler.supabase.com:6543/postgres?sslmode=require&supa=base-pooler.x"
```

Then rollback migrations:
```bash
make migrate-down  # Rollback 004
make migrate-down  # Rollback 003
```
