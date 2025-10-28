# Time-Aligned Polling Optimization

## 🎯 优化目标

减少 Vercel `/api/nof1/[...path]` 端点的 Fast Data Transfer 成本。

## ✅ 实施方案

**方案 1: 时间对齐的请求策略 (Time-Aligned Polling)**

所有客户端在时钟的整10秒时刻统一发起请求(如 `:00`, `:10`, `:20`),利用 Vercel Edge Cache 实现以下效果:
- 第一个请求触发回源
- 后续请求直接命中 CDN

## 📊 预期效果

### 成本节省
- **当前**: 100个用户 → 100次回源 → 100次 Fast Data Transfer
- **优化后**: 100个用户 → 1次回源 + 99次 CDN → **节省99%成本**

### 技术指标
- ✅ **漂移精度**: <10ms (测试显示1-9ms)
- ✅ **自动对齐**: 5个客户端从不同时间启动,全部对齐到同一边界
- ✅ **零配置**: 默认启用,所有现有 hooks 自动获得优化

## 🔧 实施细节

### 新增文件
1. **`web/src/lib/api/hooks/timeAligned.ts`**
   - 核心时间对齐逻辑
   - 计算下一个边界时间
   - 创建 SWR 兼容的 refreshInterval 函数

2. **`web/src/lib/api/hooks/debugTimeAlignment.ts`**
   - 开发者调试工具
   - 浏览器控制台集成
   - 对齐质量统计

3. **`web/scripts/test-time-alignment.ts`**
   - 自动化测试脚本
   - 验证多客户端对齐行为

4. **`web/docs/time-aligned-polling.md`**
   - 完整技术文档
   - 使用指南和最佳实践

### 修改文件
1. **`web/src/lib/api/hooks/activityAware.ts`**
   - 集成时间对齐功能
   - 默认启用 `enableTimeAlignment: true`
   - 保持向后兼容(可通过参数禁用)

2. **`web/src/app/api/nof1/[...path]/route.ts`**
   - 调整 `s-maxage` 为 10s(匹配客户端对齐间隔)
   - 优化高频端点缓存策略

## 🧪 测试结果

```bash
cd web && npx tsx scripts/test-time-alignment.ts
```

**结果**:
```
[Client 1] Started at 12:16:34.785Z, next request in 5215ms at 12:16:40.000Z
[Client 2] Started at 12:16:35.869Z, next request in 4131ms at 12:16:40.000Z
[Client 3] Started at 12:16:35.997Z, next request in 4003ms at 12:16:40.000Z
[Client 4] Started at 12:16:34.929Z, next request in 5071ms at 12:16:40.000Z
[Client 5] Started at 12:16:36.645Z, next request in 3355ms at 12:16:40.000Z

[Client 1] ✓ Request sent at 12:16:40.009Z, drift: 9ms (good)
[Client 2] ✓ Request sent at 12:16:40.002Z, drift: 2ms (good)
[Client 3] ✓ Request sent at 12:16:40.002Z, drift: 2ms (good)
[Client 4] ✓ Request sent at 12:16:40.001Z, drift: 1ms (good)
[Client 5] ✓ Request sent at 12:16:40.002Z, drift: 2ms (good)
```

✅ **所有客户端成功对齐到同一时间边界(12:16:40.000),漂移精度<10ms**

## 📈 影响范围

### 自动优化的端点(10秒对齐)
- `/api/nof1/crypto-prices` - 加密货币价格
- `/api/nof1/account-totals` - 账户总值
- `/api/nof1/positions` - 持仓信息
- `/api/nof1/trades` - 交易记录
- `/api/nof1/since-inception-values` - 累计收益

### 自动优化的端点(15秒对齐)
- `/api/nof1/leaderboard` - 排行榜
- `/api/nof1/analytics` - 分析数据

### 未变化的端点
- 历史数据端点(300-600秒缓存)保持原有策略

## 🎮 使用方式

### 默认行为(推荐)
```typescript
// 时间对齐已默认启用,无需修改代码
useSWR(key, fetcher, {
  ...activityAwareRefresh(10_000), // 自动对齐到10s边界
});
```

### 禁用时间对齐
```typescript
useSWR(key, fetcher, {
  ...activityAwareRefresh(10_000, {
    enableTimeAlignment: false, // 恢复旧行为
  }),
});
```

### 浏览器调试
```javascript
// 开发者控制台
window.__TIME_ALIGNMENT_DEBUG__ = true;
window.__DEBUG_ALIGNMENT_INFO__(); // 查看对齐状态
```

## 🔍 监控建议

部署后观察以下指标:

1. **Vercel Analytics**
   - Edge cache hit rate: 应提升至 90-99%
   - Origin requests: 应减少 90-99%
   - Fast Data Transfer: 应显著降低

2. **用户体验**
   - 页面加载时间: 应无明显变化
   - 数据新鲜度: 保持在10秒内

3. **错误率**
   - 应保持不变或降低(减少回源压力)

## ⚠️ 注意事项

1. **首次请求延迟**: 页面加载时,第一次请求可能延迟0-10秒
   - 大多数用户不会注意到这个延迟
   - 可通过预加载或骨架屏优化感知

2. **数据新鲜度**: 数据最多延迟10秒
   - 对于交易监控场景,10秒延迟是可接受的
   - 如需更高实时性,考虑 WebSocket 方案

3. **时钟同步**: 依赖客户端时钟准确性
   - 使用 `Date.now()` 已足够,无需 NTP
   - 即使时钟略有偏差,仍能获得大部分收益

## 🚀 后续优化方向

1. **方案 2: 智能缓存降级** - 根据数据变化频率自适应调整间隔
2. **方案 3: 批量更新接口** - 单次请求获取多个资源
3. **方案 4: WebSocket 推送** - 彻底替代轮询(需独立服务)

## 📚 参考文档

- [详细技术文档](web/docs/time-aligned-polling.md)
- [核心代码](web/src/lib/api/hooks/timeAligned.ts)
- [测试脚本](web/scripts/test-time-alignment.ts)
