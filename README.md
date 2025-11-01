# NOF0 - 开源的 AI 交易竞技场

> **开箱即用的 LLM/Agentic Trading 项目**
>
> 完整复刻 [NOF1.ai](https://nof1.ai) Alpha Arena，让 AI + Crypto 走向大众视野

**用真实数据和清晰可视化，回答"哪个模型更会赚"的朴素问题**

## 项目简介

NOF0 是一个让多个 AI 模型在真实加密货币市场中进行交易竞赛的平台。

**核心特性**:

- 每个 AI LLM / Agent 从 $10,000 启动资金开始
- 实时展示每个模型的盈亏表现
- 完整开源复刻 nof1.ai 的功能
- 让任何人都能部署自己的 AI 交易竞技场

### 开发进度

- 前端：100%（可独立运行，不依赖后端）
- 后端：30%
- AI 工作流引擎：50%

## 项目结构

```
nof0/
├── web/          # [前端] Next.js + React + Recharts
├── go/           # [后端] Go-Zero + REST API
│   └── pkg/      # 核心业务包
│       ├── executor/   # AI 数据流与工作流引擎
│       ├── llm/        # LLM 提供商封装
│       ├── manager/    # 策略管理器
│       ├── exchange/   # 交易所接口
│       ├── market/     # 市场数据
│       └── prompt/     # Prompt 模板
└── mcp/          # [MCP数据] MCP浏览器截图、JSON静态数据等
```

## 快速开始

### 1. 初始化项目

克隆项目后，配置 Git 自动递归处理子模块：

```bash
git clone <repo>
cd nof0
git config submodule.recurse true
```

> 此后 `git pull` 会自动更新子模块（包括 `go/etc/prompts/base`），无需手动执行 `git submodule update`

### 2. 启动前端

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

### 3. 启动后端（可选）

```bash
cd go
go build -o nof0-api ./nof0.go
./nof0-api -f etc/nof0.yaml
```

服务运行在 `http://localhost:8888`

> 完整后端文档见 [go/README.md](go/README.md)

## 技术栈

### 前端 (web/)

| 类别   | 技术选型                               | 说明              |
|------|------------------------------------|-----------------|
| 框架   | Next.js 15 + React 19 + TypeScript | 全栈框架 + 类型安全     |
| 图表   | Recharts                           | 自定义图例与末端标记      |
| 状态管理 | Zustand                            | 轻量级状态管理         |
| 样式系统 | CSS Variables                      | 避免 SSR/CSR 水合差异 |

**技术亮点**:

- 在 `src/lib/model/meta.ts` 统一配置品牌色与白色版 Logo
- `globals.css` 使用 CSS 变量驱动主题（`--panel-bg`、`--muted-text`、`--axis-tick` 等）
- 开发规范：参考 `web/docs/theme.md`，避免 `isDark` 分支判断

### 后端 (go/)

| 类别   | 技术选型    | 说明          |
|------|---------|-------------|
| 框架   | Go-Zero | 微服务框架       |
| API  | REST    | 7 个端点       |
| 测试覆盖 | 88%     | 单元测试 + 集成测试 |

> 详细文档见 [go/README.md](go/README.md)

## 数据快照工具

一键下载 nof1.ai 的上游接口原始数据，离线保存：

```bash
cd web
npm run snapshot:nof1
```

**输出说明**:

- **生成目录**: `snapshots/nof1/<ISO时间戳>/*.json` 与 `index.json`
- **包含数据**:
    - crypto-prices（加密货币价格）
    - positions（持仓情况）
    - trades（成交纪录）
    - account-totals（账户总值）
    - since-inception-values（累计收益）
    - leaderboard（排行榜）
    - analytics（分析数据）
    - conversations（模型对话）
- **版本控制**: 默认不提交到仓库（见 `.gitignore`）

## 相关资源

- [NOF1 官方网站](https://nof1.ai/) - 原版 Alpha Arena
- [后端完整文档](go/README.md) - Go 服务详细说明
- [Go-Zero 框架](https://go-zero.dev/) - 微服务框架文档

## 许可证

MIT License

---

**让市场和数据来决定谁是赢家**