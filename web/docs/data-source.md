# 前端数据获取说明

## 📊 数据获取流程

前端数据通过以下流程获取：

```
前端组件 → API Hooks → Next.js API 路由代理 → 上游 API → 返回数据
```

## 🔄 数据流架构

### 1. 前端组件层

前端组件使用自定义 Hooks 获取数据：

```typescript
// 示例：排行榜数据
import { useLeaderboard } from '@/lib/api/hooks/useLeaderboard';

function LeaderboardComponent() {
  const { rows, isLoading } = useLeaderboard();
  // ...
}
```

### 2. API Hooks 层

所有数据获取都通过 `web/src/lib/api/hooks/` 下的 Hooks：

| Hook | 数据源 | 说明 |
|------|--------|------|
| `useLeaderboard` | `/api/nof1/leaderboard` | 排行榜数据 |
| `useTrades` | `/api/nof1/trades` | 交易记录 |
| `useAccountTotals` | `/api/nof1/account-totals` | 账户总览 |
| `useCryptoPrices` | `/api/nof1/crypto-prices` | 加密货币价格 |
| `usePositions` | `/api/nof1/positions` | 持仓情况 |
| `useSinceInception` | `/api/nof1/since-inception-values` | 累计收益曲线 |
| `useAnalytics` | `/api/nof1/analytics` | 分析数据 |
| `useConversations` | `/api/nof1/conversations` | 模型对话 |

### 3. Next.js API 路由代理

所有 API 请求都通过 Next.js API 路由代理（`web/src/app/api/nof1/[...path]/route.ts`）：

- **路径**: `/api/nof1/*`
- **功能**: 
  - 转发请求到上游 API
  - 处理 CORS
  - 设置缓存策略
  - 时间对齐优化

### 4. 上游 API 配置

上游 API 地址通过环境变量配置：

**默认值**: `https://nof1.ai/api` (NOF1 官方 API)

**配置方式**:

1. **开发环境** (`.env.local`):
   ```bash
   # 使用本地后端
   NEXT_PUBLIC_NOF1_API_BASE_URL=http://localhost:8888
   NOF1_API_BASE_URL=http://localhost:8888
   ```

2. **生产环境**:
   ```bash
   # 使用自定义后端
   NEXT_PUBLIC_NOF1_API_BASE_URL=https://your-api.com/api
   NOF1_API_BASE_URL=https://your-api.com/api
   ```

## 🎯 三种数据源模式

### 模式 1: 使用 NOF1 官方 API（默认）

**无需配置**，前端会自动使用 `https://nof1.ai/api`

```bash
# 直接启动前端
cd web
npm install
npm run dev
```

访问 `http://localhost:3000` 即可看到 NOF1 的实时数据。

### 模式 2: 使用本地后端

**步骤**:

1. 启动本地后端：
   ```bash
   cd go
   go build -o nof0-api ./nof0.go
   ./nof0-api -f etc/nof0.yaml
   ```
   后端运行在 `http://localhost:8888`

2. 配置前端环境变量：
   
   创建 `web/.env.local`:
   ```bash
   NEXT_PUBLIC_NOF1_API_BASE_URL=http://localhost:8888
   NOF1_API_BASE_URL=http://localhost:8888
   ```

3. 重启前端：
   ```bash
   cd web
   npm run dev
   ```

### 模式 3: 使用自定义后端

设置环境变量指向你的后端地址：

```bash
NEXT_PUBLIC_NOF1_API_BASE_URL=https://your-backend.com/api
NOF1_API_BASE_URL=https://your-backend.com/api
```

## 📡 API 端点映射

前端请求的路径会映射到后端 API：

| 前端请求 | 后端 API | 说明 |
|---------|---------|------|
| `/api/nof1/crypto-prices` | `/api/crypto-prices` | 实时价格 |
| `/api/nof1/leaderboard` | `/api/leaderboard` | 排行榜 |
| `/api/nof1/trades` | `/api/trades` | 交易记录 |
| `/api/nof1/account-totals` | `/api/account-totals` | 账户总览 |
| `/api/nof1/positions` | `/api/positions` | 持仓情况 |
| `/api/nof1/since-inception-values` | `/api/since-inception-values` | 累计收益 |
| `/api/nof1/analytics` | `/api/analytics` | 分析数据 |
| `/api/nof1/conversations` | `/api/conversations` | 模型对话 |

## ⚡ 缓存策略

Next.js API 路由会根据数据类型设置不同的缓存策略：

| 数据类型 | 缓存时间 | 说明 |
|---------|---------|------|
| `crypto-prices` | 5秒 | 高频变化数据 |
| `account-totals` | 10秒 | 实时账户数据 |
| `positions` | 10秒 | 实时持仓数据 |
| `trades` | 10秒 | 交易记录 |
| `conversations` | 30秒 | 对话数据 |
| `leaderboard` | 60秒 | 排行榜 |
| `analytics` | 300秒 | 分析数据 |
| `since-inception-values` | 600秒 | 历史曲线 |

## 🔍 调试数据源

### 查看当前使用的 API 地址

在浏览器控制台运行：

```javascript
// 查看环境变量
console.log('API Base URL:', process.env.NEXT_PUBLIC_NOF1_API_BASE_URL || 'https://nof1.ai/api');
```

### 检查网络请求

1. 打开浏览器开发者工具 (F12)
2. 切换到 Network 标签
3. 查看 `/api/nof1/*` 的请求
4. 检查请求的 `target` 字段，确认实际请求的上游地址

### 测试本地后端

```bash
# 测试后端是否正常
curl http://localhost:8888/api/leaderboard
curl http://localhost:8888/api/crypto-prices
```

## 📝 代码位置

- **API 客户端**: `web/src/lib/api/client.ts`
- **API 端点定义**: `web/src/lib/api/nof1.ts`
- **Next.js 代理**: `web/src/app/api/nof1/[...path]/route.ts`
- **数据 Hooks**: `web/src/lib/api/hooks/*.ts`

## 💡 常见问题

### Q: 前端显示 "Request failed" 错误

**A**: 检查：
1. 后端是否正在运行
2. 环境变量配置是否正确
3. 后端 API 路径是否匹配

### Q: 如何切换到本地数据？

**A**: 
1. 确保本地后端运行在 `http://localhost:8888`
2. 创建 `web/.env.local` 并设置环境变量
3. 重启前端开发服务器

### Q: 数据更新不及时？

**A**: 
- 检查缓存策略设置
- 查看 Network 面板确认请求频率
- 某些数据（如 `since-inception-values`）缓存时间较长

---

**总结**: 前端通过 Next.js API 路由代理获取数据，默认使用 NOF1 官方 API，可以通过环境变量切换到本地后端或自定义后端。

