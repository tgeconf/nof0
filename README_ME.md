# NOF0 - 开箱即用的Agentic Trading项目

> **终极目标**: 完整复刻 [NOF1.ai](https://nof1.ai) Alpha Arena，打造开源的AI交易竞技平台

让 AI + Crypto 走向大众视野：用真实数据和清晰可视化，回答"哪个模型更会赚"的朴素问题。


## 项目简介

NOF0 是一个让多个AI模型在真实加密货币市场中进行交易竞赛的平台。每个AI从$10,000起步，实时展示谁赚的多、谁亏的惨。本项目复刻 nof1.ai 的完整功能，让任何人都能部署自己的AI交易竞技场。

### 整体设计与运行流程

1. **后端初始化**：执行 `server/main.py` 时，程序使用 `server/nof0/config.py` 读取 `etc/nof0.yaml`，随后由 `server/nof0/app.py` 创建 FastAPI 应用并生成 `ServiceContext`。该上下文在启动阶段实例化 `server/nof0/data_loader.py` 中的 `DataLoader`（指向 `mcp/data` 目录）以及可选的 Postgres 连接池，为后续请求提供依赖。
2. **前端服务启动**：运行 `web` 目录下的 Next.js 应用时，`web/src/app/page.tsx` 渲染首屏布局（价格条、资产曲线、右侧标签页）。构建期间 Next 会预渲染主要静态结构，客户端完成 Hydration 后各面板开始按需获取数据。
3. **初始数据请求**：客户端挂载时，SWR 钩子（如 `web/src/lib/api/hooks/useAccountTotals.ts`、`useTrades.ts`）立即发起首个请求，以 `/api/nof1/*` 形式命中 `web/src/lib/api/nof1.ts` 定义的本地代理端点。若设置 `NEXT_PUBLIC_NOF1_API_BASE_URL` 指向自建后端，则这些请求会被重写到 `http://localhost:8888/api/*`。
4. **代理与缓存控制**：请求首先进入 Next Edge 函数 `web/src/app/api/nof1/[...path]/route.ts`，该函数根据 `NOF1_API_BASE_URL` 拼接上游地址，并对响应设置 `Cache-Control`、`s-maxage`、`stale-while-revalidate` 等头值，实现浏览器与 CDN 双层缓存。
5. **后端数据读取**：Edge 层将请求转发给 FastAPI 路由 `server/nof0/api/routes.py`。所有路由通过 `ServiceContext` 获取 `DataLoader`，再调用对应方法读取 JSON 快照、写入 `serverTime` 毫秒时间戳，并返回给客户端。若配置了 Postgres，后续可在此阶段改为查询实时库或缓存。
6. **轮询机制**：SWR 使用 `web/src/lib/api/hooks/activityAware.ts` 和 `timeAligned.ts` 计算刷新间隔。默认对可见页面启用 10 秒对齐轮询：`createTimeAlignedInterval` 让所有客户端在同一时刻（例如每个 10 秒边界）触发刷新；页面隐藏时则降频或暂停。某些数据（如账户总额）还会传入 `lastHourlyMarker` 参数，由 `useAccountTotals` 在浏览器端合并增量。
7. **数据回流与渲染**：FastAPI 返回的 JSON 被 Edge 层原样透传，SWR 收到响应后更新缓存并触发组件重渲染；各面板据此刷新图表、表格与模型对话。整个过程持续循环，直至用户关闭页面或切换 Tab，期间所有步骤都会重复执行，实现统一的时间对齐轮询链路。

## 项目愿景

### 终极目标
完整开源复刻 [NOF1.ai](https://nof1.ai) Alpha Arena

### 当前进度

- 前端：100%（可独立运行，不依赖后端）
- 后端：20%
- AI Agent：0%

## 项目结构

```
nof0/
├── web/          # [前端] Next.js + React + Recharts
├── server/       # [后端] FastAPI + Python REST API
├── mcp/          # [MCP数据] MCP浏览器截图、JSON静态数据等
└── agents/       # [AI引擎] (规划中)
```

## 快速开始

### 启动前端

```bash
cd web
npm install
npm run dev
```

访问 `http://localhost:3000`

**前端核心特性**:
- 账户总资产曲线
- 持仓情况
- 成交纪录
- 模型对话（Model Chat）
- 排行榜
- 模型详情

### 启动后端

```bash
cd server
python -m venv .venv
source .venv/bin/activate
pip install -e .
python main.py --config etc/nof0.yaml
```

服务运行在 `http://localhost:8888`

完整后端文档见 [server/README.md](server/README.md)

## 项目运行逻辑

### 前端数据流

1. 页面由 `web/src/app/page.tsx` 渲染主布局，嵌入价格、资产曲线和右侧标签页等客户端组件。当路由参数变化时（例如 `?tab=trades`），`web/src/components/tabs/RightTabs.tsx` 会切换相应面板。
2. 每个面板通过 SWR 钩子按需轮询数据：例如成交表使用 `web/src/lib/api/hooks/useTrades.ts` 每 10 秒刷新，资产曲线则在 `web/src/lib/api/hooks/useAccountTotals.ts` 中先拉取完整历史，再以 `lastHourlyMarker` 增量更新。刷新间隔由 `web/src/lib/api/hooks/activityAware.ts` 与 `web/src/lib/api/hooks/timeAligned.ts` 控制，可根据页面可见性和时间对齐自动调整频率。
3. 钩子统一调用 `web/src/lib/api/nof1.ts` 中定义的本地代理端点（如 `/api/nof1/trades`），再由 `web/src/lib/api/client.ts` 的 `fetcher` 发送请求。若设置 `NEXT_PUBLIC_NOF1_API_BASE_URL`，组件可以直接指向自定义后端。

### 边缘代理与缓存

- 所有前端请求都会命中 Next.js 边缘路由 `web/src/app/api/nof1/[...path]/route.ts`。该路由根据 `NOF1_API_BASE_URL`（默认 `https://nof1.ai/api`）拼接上游地址，向后端透传条件请求头，并统一设置 `Cache-Control`、`s-maxage` 等响应头，确保多客户端轮询时命中边缘缓存，减少真实源站压力。

### 后端服务链路

1. 启动 `server/main.py` 时会先解析 `--config` 参数，使用 `server/nof0/config.py` 载入 YAML 配置。配置包含监听端口、静态数据目录以及 Postgres/Redis 选项。
2. `server/nof0/app.py` 在应用生命周期内构建 `ServiceContext`，该上下文由 `server/nof0/service.py` 创建，负责实例化 `DataLoader`（热数据读取器）及可选的 Postgres 连接池。服务器关闭时会调用 `ServiceContext.close()` 释放资源。
3. API 路由集中在 `server/nof0/api/routes.py`，使用 FastAPI 将请求绑定到 `DataLoader`。所有响应都包含当前毫秒级时间戳（`server/nof0/data_loader.py` 内部添加的 `serverTime` 字段），用于前端判断热数据新鲜度与对齐缓存。

#### 接口与数据来源

- `/api/account-totals`：调用 `DataLoader.load_account_totals()` 读取 `account-totals.json`，动态回填 `serverTime`。前端首次加载完整历史，之后带上 `lastHourlyMarker` 参数增量拉取，热数据由 `useAccountTotals` 在浏览器端合并。
- `/api/analytics`：经由 `load_analytics()` 返回 `analytics.json` 的聚合指标，并注入 `serverTime`，供排行榜和模型详情使用。
- `/api/analytics/{model_id}`：先尝试读取 `analytics-{model_id}.json`，若不存在则回退到 `analytics.json` 中对应条目。该逻辑封装在 `DataLoader.load_model_analytics()`，确保单模型分析页可用。
- `/api/crypto-prices`：直接返回 `crypto-prices.json`，没有额外加工，主要给价格跑马灯等轻量组件消费。
- `/api/leaderboard`：读取 `leaderboard.json`，与前端 `/api/nof1/leaderboard` 缓存策略配合，10 秒对齐轮询。
- `/api/since-inception-values`：通过 `load_since_inception()` 提供长期绩效曲线，附带 `serverTime` 以便前端判断是否刷新。
- `/api/trades`：`load_trades()` 返回 `trades.json` 中的交易明细，同时封装为 `{ trades: [...], serverTime }`，保证最新成交在前端列表顶端出现。
- `/api/positions`：`load_positions()` 将 `positions.json` 中的 `accountTotals` 字段抽出，并附加 `serverTime`，用于前端仓位页实时刷新。
- `/api/conversations`：`load_conversations()` 包装 `conversations.json`，保持 MPC 产生的模型对话历史。当前实现读取静态快照，后续可换成数据库或流式接口。

#### 热数据与模型调用

- `DataLoader` 默认指向 `mcp/data` 中的本地快照，所有接口都以读文件方式提供热点数据，并通过毫秒时间戳提示前端缓存是否失效。
- 若在配置中提供 Postgres DSN，则 `ServiceContext` 会创建 `psycopg_pool.ConnectionPool`，为后续热数据写入或实时模型输出准备连接（目前只创建、不使用）。
- 实际模型推理尚未接入：当前对话与交易详情均来自静态 JSON。未来可在 Postgres 或 Redis 中落地模型产生的热数据，或改写 `DataLoader` 连接实时数据源。

#### Postgres 配置说明

- 默认配置文件位于 `server/etc/nof0.yaml`；`Postgres.DSN`、`Postgres.MaxOpen`、`Postgres.MaxIdle` 字段控制连接信息，`main.py` 在启动时自动加载。
- 如需自定义配置，可复制 `etc/nof0.yaml` 并使用 `python main.py --config path/to/custom.yaml`，或通过环境变量 `NOF0_CONFIG`/`NOF0_CONFIG_PATH` 指定配置路径。
- 当 DSN 非空时，`ServiceContext.from_config()` 会构建连接池供 API 或后台任务复用；可配合 `server/nof0/importer.py` 将 `mcp/data` 快照导入数据库：`python -m nof0.importer --dsn postgresql://user:pass@host:5432/db --data ../mcp/data --truncate`。
- 生产部署建议为 API 提供只读账号，写入操作由后续的行情/模型调度器承担，以简化权限管理。

当前版本主要通过 `web/scripts/snapshot-nof1.ts` 从官方接口抓取最新快照，再由后端读取这些快照对外提供统一 API。真正的实时模型请求（生成交易指令、更新仓位等）尚未实现。

## TODO（Agent化交易扩展计划）

1. **实时行情与账户快照落地**：在 `server/nof0/service.py` 增加行情轮询器，按照 `config.ttl` 配置周期调用交易所或上游 REST/WebSocket，将增量数据写入 Postgres（`importer.py` 中的 upsert 逻辑可复用）。同时提供事件广播（如使用 Redis Pub/Sub）供前端或策略端订阅。
2. **模型推理与仓位调度服务**：在 `agents/` 目录下新增 `controller` 子模块，负责自动化轮训。调度器读取最新账户与行情状态，调用 `LLM`（可封装在 `agents/llm_client.py`）生成操作建议。建议结构化输出（目标杠杆、买卖方向、置信度），并写入数据库的待执行队列。
3. **执行引擎与风控**：实现 `server/nof0` 内的执行 API（如 `/api/orders`），由后台工作线程读取队列，依据策略调用交易所 API 或模拟撮合。引入风控模块（仓位限额、滑点预估、止盈止损），并将订单、成交反馈写回数据库与缓存，使前端 `useTrades`、`usePositions` 可以实时响应。
4. **对话与决策可溯源**：扩展 `conversations` 数据管道，将 LLM 推理输入（市场特征、历史仓位）与输出（指令、理由）以消息形式写入数据库，通过新的 `/api/conversations/live` 接口流式返回，为审核和可解释性提供基础。

上述扩展需结合现有缓存策略：在边缘层继续使用 `/api/nof1/[...path]/route.ts` 统一代理，并根据新 TTL 调整 `TTL_BY_SEGMENT`，确保高频轮询的成本可控。

## 技术栈

### 前端 (web/)
- **框架**: Next.js 15 + React 19 + TypeScript
- **图表**: Recharts（自定义图例与末端标记）
- **状态**: Zustand
- **样式**: CSS Variables 主题系统（避免SSR/CSR水合差异）
- **状态**: 开发完毕

**技术亮点**:
- 在 `src/lib/model/meta.ts` 统一配置品牌色与白色版 Logo
- `globals.css` 使用 CSS 变量驱动主题（`--panel-bg`、`--muted-text`、`--axis-tick` 等）
- 开发规范：参考 `web/docs/theme.md`，避免 `isDark` 分支判断

### 后端 (server/)
- **框架**: FastAPI + Uvicorn
- **特性**: 7个REST端点、兼容 MCP JSON 数据、可选 Postgres 导入工具
- **状态**: 开发中

详细文档见 [server/README.md](server/README.md)

## 数据快照工具

一键下载 nof1.ai 的上游接口原始数据，离线保存：

```bash
cd web
npm run snapshot:nof1
```

**生成内容**:
- 生成目录：`snapshots/nof1/<ISO时间戳>/*.json` 与 `index.json`
- 已包含：crypto-prices、positions、trades、account-totals、since-inception-values、leaderboard、analytics、conversations
- 默认不提交到仓库（见 `.gitignore`）

## 相关资源

- [NOF1 官方网站](https://nof1.ai/)
- [后端完整文档](server/README.md)
- [FastAPI](https://fastapi.tiangolo.com/)

## 许可证

MIT License
