# nof0

nof1.ai like frontend.

> 本项目为 nof1.ai 的前端仿盘与快速验证界面，用于演示：
> - 实时账户价值曲线（/account-totals 全量 + 增量）
> - 模型排行榜（/leaderboard）
> - 持仓/成交/分析等标签页（/positions, /trades, /analytics）
> - 模型详情与统一配色（model_id → 友好名称/颜色）

## 更好的基准（A Better Benchmark）

Alpha Arena 是一个用于衡量 AI 投资能力的全新基准。每个模型在真实市场中、用真金白银进行交易，起始资金为 $10,000，且使用相同的提示词与输入数据。

我们的目标是让基准更贴近真实世界；而市场天然适合作为测试场景——它们动态、对抗、开放、且充满不确定性，能以静态基准无法企及的方式挑战 AI。市场，是对智能的终极检验。

问题来了：投资是否需要新的模型架构，还是通用 LLM 已足够？让我们一起验证。

### 参赛选手
- Claude 4.5 Sonnet
- DeepSeek V3.1 Chat
- Gemini 2.5 Pro
- GPT 5
- Grok 4
- Qwen 3 Max

### 比赛规则
- 起始资金：每个模型获得 $10,000 实盘资金
- 市场：Hyperliquid 加密永续合约
- 目标：最大化风险调整后收益
- 透明度：所有模型输出与对应交易全公开
- 自主性：AI 必须自主产出 alpha、控制仓位、择时与风控
- 赛季时长：第一季至 2025-11-03 17:00（美东）

## 开发提示
- 右侧 Tab 可切换：持仓、模型详情、成交、分析、README.md（支持 `?tab=readme`）
- 代理路由：`/api/nof1/*` → `https://nof1.ai/api/*`
