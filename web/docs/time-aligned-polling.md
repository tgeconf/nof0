# Time-Aligned Polling Strategy

## Overview

Time-aligned polling ensures all clients request data at the same clock boundaries (e.g., every 10 seconds at `:00`, `:10`, `:20`, etc.). This maximizes Vercel Edge cache hit rates and dramatically reduces Fast Data Transfer costs.

## Cost Savings

**Before time alignment:**
- 100 concurrent users polling every 10s
- Each user hits origin independently
- Result: **100 origin transfers per interval**

**After time alignment:**
- 100 concurrent users polling at aligned 10s boundaries
- First request hits origin, subsequent requests hit Edge cache
- Result: **1 origin transfer per interval**
- **Savings: 99% reduction in Fast Data Transfer costs**

## How It Works

### Client-Side Alignment

1. **Calculate next boundary**: When a hook initializes, calculate the time until the next 10s boundary
   - Current time: 10:00:03.456
   - Next boundary: 10:00:10.000
   - Wait time: 6.544s

2. **Align all requests**: All clients wait until the next boundary before making their first request

3. **Synchronized polling**: After the first aligned request, SWR continues polling at regular 10s intervals, maintaining alignment

### Server-Side Caching

The Edge route (`/api/nof1/[...path]/route.ts`) is configured with matching cache headers:

```typescript
// For 10s client polling endpoints
Cache-Control: public, max-age=5, s-maxage=10, stale-while-revalidate=40
```

- `max-age=5`: Browser can cache for 5s
- `s-maxage=10`: Edge cache for 10s (matches alignment interval)
- `stale-while-revalidate=40`: Serve stale data while revalidating

## Usage

### Default Behavior (Time Alignment Enabled)

All hooks using `activityAwareRefresh()` now have time alignment enabled by default:

```typescript
// web/src/lib/api/hooks/useCryptoPrices.ts
export function useCryptoPrices() {
  const { data, error, isLoading } = useSWR<PricesResponse>(
    endpoints.cryptoPrices(),
    fetcher,
    {
      ...activityAwareRefresh(10_000), // Automatically aligns to 10s boundaries
    },
  );
  // ...
}
```

### Disabling Time Alignment

To disable time alignment for a specific hook (revert to old behavior):

```typescript
{
  ...activityAwareRefresh(10_000, {
    enableTimeAlignment: false // Disable alignment
  }),
}
```

### Adding Jitter (Advanced)

If you're concerned about thundering herd on the origin server, add a small jitter:

```typescript
{
  ...activityAwareRefresh(10_000, {
    alignmentJitter: 100 // Add 0-100ms random jitter
  }),
}
```

Note: Adding jitter reduces cache hit rates, so only use if necessary.

## Affected Endpoints

The following endpoints have time-aligned polling with 10s intervals:

- `/api/nof1/crypto-prices` - Real-time crypto prices
- `/api/nof1/account-totals` - Account values and P&L
- `/api/nof1/positions` - Open positions
- `/api/nof1/trades` - Trade history
- All other endpoints using `activityAwareRefresh(10_000)`

## Debugging

### Browser Console Debug Mode

Enable debug mode in your browser console:

```javascript
// Enable debug logging
window.__TIME_ALIGNMENT_DEBUG__ = true;

// Check alignment status
window.__DEBUG_ALIGNMENT_INFO__();
// Returns:
// {
//   enabled: true,
//   currentTime: "2025-10-28T12:16:35.000Z",
//   nextBoundary: "2025-10-28T12:16:40.000Z",
//   waitMs: 5000,
//   alignmentInterval: 10000
// }
```

### Test Script

Run the test script to verify alignment behavior:

```bash
cd web
npx tsx scripts/test-time-alignment.ts
```

Expected output:
```
[Client 1] Started at 12:16:34.785Z, next request in 5215ms at 12:16:40.000Z
[Client 2] Started at 12:16:35.869Z, next request in 4131ms at 12:16:40.000Z
...
[Client 1] ✓ Request sent at 12:16:40.001Z, drift: 1ms (good)
[Client 2] ✓ Request sent at 12:16:40.002Z, drift: 2ms (good)
```

All clients should request at the same boundary with <100ms drift.

## Monitoring

### Key Metrics to Monitor

1. **Cache Hit Rate**: Should increase to 90-99% for time-aligned endpoints
2. **Origin Requests**: Should decrease dramatically (90-99% reduction)
3. **Fast Data Transfer**: Should see significant cost reduction
4. **Latency**: Should remain unchanged (data freshness within 10s)

### Vercel Analytics

Check your Vercel dashboard for:
- Edge cache hit rates
- Origin transfer volumes
- Fast Data Transfer costs

## Trade-offs

### Pros ✅
- **95-99% reduction in Fast Data Transfer costs**
- **Minimal code changes** (enabled by default)
- **No user-facing changes** (data freshness within 10s)
- **Scalable** (more users = more savings)

### Cons ⚠️
- **Increased initial delay**: First request may wait up to 10s
  - Mitigation: Users typically won't notice 0-10s delay on page load
- **Data freshness reduced slightly**: Data may be up to 10s old
  - Mitigation: Still acceptable for most trading/monitoring use cases
- **Clock synchronization dependency**: Requires accurate client clocks
  - Mitigation: Uses `Date.now()` which is sufficient (no NTP needed)

## Future Improvements

1. **Adaptive alignment intervals**: Adjust based on data volatility
2. **Smart jitter**: Add jitter only when origin is under load
3. **Per-endpoint tuning**: Different alignment intervals for different data types
4. **Background revalidation**: Prefetch data before alignment boundary

## Related Files

- `web/src/lib/api/hooks/timeAligned.ts` - Core alignment logic
- `web/src/lib/api/hooks/activityAware.ts` - Integration with activity tracking
- `web/src/lib/api/hooks/debugTimeAlignment.ts` - Debug utilities
- `web/src/app/api/nof1/[...path]/route.ts` - Server-side caching
- `web/scripts/test-time-alignment.ts` - Test script
