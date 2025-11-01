# Executor 执行器模块

**Executor 执行器模块**是新系统的决策大脑，负责整合市场数据、账户信息、持仓状态，并通过 AI 模型生成交易决策。

## 核心职责

- 🔍 **数据整合**：收集并组织市场数据、账户状态、持仓信息

- 🤖 **AI 决策**：调用大语言模型生成交易决策

- ⚖️ **风险验证**：严格验证决策的风险控制规则

- 📊 **性能优化**：以夏普比率为目标持续优化策略

## 设计目标

- **最大化夏普比率**（平均收益 / 收益波动率）

- **质量优于数量**（严格开仓标准，避免过度交易）

- **多空平衡**（消除做多偏见，做空是核心工具）

- **透明可追溯**（完整记录思维链和决策过程）



## 目录结构

```plaintext
executor/
├── executor.go           # 主入口和核心流程
├── context.go           # 上下文构建
├── prompt.go            # Prompt 生成
├── validator.go         # 决策验证
├── parser.go            # AI 响应解析
├── types.go             # 数据结构定义
├── market_data.go       # 市场数据获取
└── utils.go             # 工具函数
```



## 核心数据结构实现

### Phase 1: 基础数据结构 (types.go)

#### Priority: 🔴 High

- [ ]  **PositionInfo** - 持仓信息结构

  ```go
  type PositionInfo struct {
    Symbol           string  // 交易对符号
    Side             string  // "long" 或 "short"
    EntryPrice       float64 // 入场价格
    MarkPrice        float64 // 当前标记价格
    Quantity         float64 // 持仓数量
    Leverage         int     // 杠杆倍数
    UnrealizedPnL    float64 // 未实现盈亏 (USD)
    UnrealizedPnLPct float64 // 未实现盈亏百分比
    LiquidationPrice float64 // 强平价格
    MarginUsed       float64 // 已用保证金
    UpdateTime       int64   // 更新时间戳 (毫秒)
}
  ```

- [ ]  **AccountInfo** - 账户信息结构

  ```go
  type AccountInfo struct {
    TotalEquity      float64 // 账户净值（含未实现盈亏）
    AvailableBalance float64 // 可用余额
    TotalPnL         float64 // 总盈亏 (USD)
    TotalPnLPct      float64 // 总盈亏百分比
    MarginUsed       float64 // 已用保证金
    MarginUsedPct    float64 // 保证金使用率 (%)
    PositionCount    int     // 当前持仓数量
}
  ```

- [ ]  **CandidateCoin** - 候选币种结构

  ```go
  type CandidateCoin struct {
    Symbol  string   // 币种符号
    Sources []string // 来源标签: ["ai500"] 或 ["oi_top"] 或两者
}
  ```

- [ ]  **Context** - 交易上下文结构

  ```go
  type Context struct {
    CurrentTime      string                  // 当前时间
    RuntimeMinutes   int                     // 系统运行时长（分钟）
    CallCount        int                     // 决策周期计数
    Account          AccountInfo             // 账户信息
    Positions        []PositionInfo          // 当前持仓列表
    CandidateCoins   []CandidateCoin             // 候选币种列表（Manager 预筛）
    MarketDataMap    map[string]*market.Snapshot // 市场快照映射（复用 market.Provider）
    OpenInterestMap  map[string]*OpenInterest    // OI 扩展数据（可选）
    Performance      *PerformanceView            // 历史表现概览（Manager 提供只读视图）
    MajorCoinLeverage   int                        // BTC/ETH 杠杆倍数（来自配置）
    AltcoinLeverage  int                        // 山寨币杠杆倍数（来自配置）
  }
  ```

- [ ]  **Decision** - 单个交易决策结构

  ```go
  type Decision struct {
    Symbol          string  // 交易对符号
    Action          string  // 动作类型
    Leverage        int     // 杠杆倍数（开仓时必填）
    PositionSizeUSD float64 // 仓位大小 (USD，开仓时必填)
    StopLoss        float64 // 止损价格（开仓时必填）
    TakeProfit      float64 // 止盈价格（开仓时必填）
    Confidence      int     // 信心度 0-100（开仓建议 ≥ 75）
    RiskUSD         float64 // 最大美元风险
    Reasoning       string  // 决策理由
}
  ```

- **注意**:Action 类型包括 `open_long`, `open_short`, `close_long`, `close_short`, `hold`, `wait`

- [ ]  **FullDecision** - 完整决策输出结构

  ```go
  type FullDecision struct {
    UserPrompt string     // 发送给 AI 的输入 prompt
    CoTTrace   string     // 思维链分析 (Chain of Thought)
    Decisions  []Decision // 具体决策列表
    Timestamp  time.Time  // 决策时间戳
}
  ```

- [ ]  **PerformanceView** - 来自 Manager 的性能数据只读视图

  ```go
  type PerformanceView struct {
    SharpeRatio      float64
    WinRate          float64
    TotalTrades      int
    RecentTradesRate float64
    UpdatedAt        time.Time
}
  ```


## 核心功能实现

## 依赖关系概览

- **市场数据**：通过 `market.Provider` 获取，并直接使用 `*market.Snapshot`。Executor 不再维护独立的 MarketData 结构，只负责做轻量过滤和聚合。
- **性能指标**：由 Manager 汇总成 `PerformanceView`，注入到上下文中，Executor 只读使用。
- **交易账户信息**：从 `exchange.Provider` 返回的 `AccountState`、`Position` 经过 Manager 归一化后传入。
- **大模型客户端**：遵循 `llm.LLMClient` 接口 via 依赖注入。


## 实施阶段（MVP 聚焦）

### Phase 0：核心类型与接口 (types.go, executor.go)
- [ ] 定义 `Context` / `Decision` / `FullDecision` 数据结构
- [ ] 定义 `PerformanceView`、`OpenInterest` 等辅助类型
- [ ] 声明 `Executor` 接口（`GetFullDecision` / `UpdatePerformance` / `GetConfig`）并规划依赖注入点

### Phase 1：上下文组装 (context.go)
- [ ] `BuildContext`：整合账户、持仓、候选币、实时配置
- [ ] `fetchMarketSnapshots`：批量调用 `market.Provider`，生成 `map[string]*market.Snapshot`
- [ ] `mergeOpenInterest`：在 Manager 提供扩展数据时合并为 `OpenInterestMap`
- [ ] `attachPerformanceView`：把 Manager 的 `PerformanceView` 注入上下文
- [ ] `filterCandidates`：保留流动性符合要求的候选币，并确保现有持仓不过滤

### Phase 2：Prompt 生成与 LLM 调用 (prompt.go, executor.go)
- [ ] `buildSystemPrompt`（缓存静态规则）
- [ ] `buildUserPrompt`：使用 `market.Snapshot`、持仓、候选币信息生成上下文文本
- [ ] `callLLM`：通过 `llm.LLMClient` 调用模型，处理超时/重试/日志
- [ ] `sanitizeResponse`：在进入解析前做基础清洗（去除 BOM、截断异常字符）

### Phase 3：响应解析与验证 (parser.go, validator.go)
- [ ] `parseFullDecisionResponse`：输出 `FullDecision`，包含 prompt、CoT、决策列表
- [ ] `extractCoTTrace` / `extractDecisions`：容错处理缺失字段、json 修复
- [ ] `validateDecisions`：检查仓位数量、杠杆、仓位大小、风险回报、保证金占用等硬约束
- [ ] `enrichDecisions`：补全缺失价格或信心度、转换单位

### Phase 4：决策输出与反馈 (executor.go, utils.go)
- [ ] `assembleDecision`：结合上下文与解析结果返回 `FullDecision`
- [ ] 记录核心日志和指标（决策耗时、提示词长度、模型 ID 等）
- [ ] `UpdatePerformance`：接受 Manager 推送的最新绩效视图，更新缓存
- [ ] 对接 Manager：确认数据格式、错误返回语义
### Backlog（下一阶段再实现）
- 夏普率反馈、交易频率分析
- 更复杂的 OI Top/成交量过滤
- 性能 profiling 与缓存策略优化
- 深度集成测试 / 压测工具

    - **做空激励：多空平衡理念**

    - **交易频率认知：每小时 0.1-0.2 笔是优秀标准**

    - **开仓标准：信心度 ≥ 75，多维度交叉验证**

    - **夏普比率自我进化机制**

    - **决策流程说明**

    - **输出格式规范（思维链 + JSON 数组）**

- [ ]  **buildUserPrompt** - 构建用户提示词（动态数据）

- **输入**: `ctx *Context`

- **输出**: `string`

- **内容包含**:

    - **系统状态（当前时间、周期计数、运行时长）**

    - **BTC 市场概况（价格、趋势、技术指标）**

    - **账户状态（净值、可用余额、保证金使用率）**

    - **当前持仓（含持仓时长、盈亏情况）**

    - **候选币种（含市场快照、OpenInterest 扩展）**

    - **夏普比率反馈（当前值、历史表现）**

- **实现要点**:

    - **直接使用 `market.Snapshot` 提供的字段构造输出**

    - **计算持仓时长（当前时间 - 入场时间）**

    - **标注币种来源（ai500、oi_top 或双重信号）**

- [ ]  **formatPositionInfo** - 格式化持仓信息

- **输入**: `position PositionInfo, currentTime time.Time`

- **输出**: `string`

- **格式**:包含符号、方向、入场价、当前价、盈亏、持仓时长

- [ ]  **formatCandidateInfo** - 格式化候选币种信息

- **输入**: `coin CandidateCoin, snapshot *market.Snapshot, oiData *OpenInterest`

- **输出**: `string`

- **格式**:包含完整市场数据、OI 信息、来源标签



### Phase 2 细化：AI 集成 (executor.go)

#### Priority: 🔴 High

- [ ]  **GetFullDecision** - 主入口函数

- **输入**: `ctx *Context`

- **输出**: `(*FullDecision, error)`

- **流程**:

    1. 调用 `fetchMarketSnapshots(ctx)` 完成市场数据补全

    2. 调用 `buildSystemPrompt(ctx.Account.TotalEquity)` 构建系统提示词

    3. 调用 `buildUserPrompt(ctx)` 构建用户提示词

    4. 调用 `callLLM(ctx, systemPrompt, userPrompt)` 获取模型响应

    5. 调用 `parseFullDecisionResponse(aiResponse, ctx)` 解析响应

    6. 返回 `FullDecision` 结构

- [ ]  **callLLM** - LLM 调用封装

- **输入**: `ctx context.Context, systemPrompt string, userPrompt string`

- **输出**: `(string, error)`

- **功能**:

    - **通过 `llm.LLMClient` 发起调用，支持同步或流式扩展**

    - **应用超时、重试、熔断等容错策略**

    - **记录请求与响应摘要日志，脱敏敏感字段**



### Phase 3 细化：响应解析 (parser.go)

#### Priority: 🔴 High

- [ ]  **parseFullDecisionResponse** - 解析 AI 完整响应

- **输入**: `response string, ctx *Context`

- **输出**: `(*FullDecision, error)`

- **步骤**:

    1. 调用 `extractCoTTrace(response)` 提取思维链

    2. 调用 `extractDecisions(response)` 提取决策列表

    3. 调用 `validateDecisions(decisions, ctx)` 验证决策

    4. 构建并返回 `FullDecision` 结构

- [ ]  **extractCoTTrace** - 提取思维链分析

- **输入**: `response string`

- **输出**: `string`

- **逻辑**:提取 JSON 数组之前的文本内容

- [ ]  **extractDecisions** - 提取 JSON 决策列表

- **输入**: `response string`

- **输出**: `([]Decision, error)`

- **步骤**:

    1. 查找 JSON 数组起始 `[`

    2. 使用 `findMatchingBracket()` 找到结束 `]`

    3. 调用 `fixMissingQuotes()` 修复常见格式错误

    4. 反序列化为 `[]Decision`

- [ ]  **fixMissingQuotes** - 修复 JSON 格式错误

- **输入**: `jsonStr string`

- **输出**: `string`

- **处理**:替换中文引号为英文引号（`"` → `"`，`"` → `"`）

- [ ]  **findMatchingBracket** - 查找匹配的右括号

- **输入**: `s string, start int`

- **输出**: `int`

- **逻辑**:使用深度计数器匹配嵌套括号



### Phase 3 细化：决策验证 (validator.go)

#### Priority: 🔴 High

- [ ]  **validateDecisions** - 批量验证决策列表

- **输入**: `decisions []Decision, ctx *Context`

- **输出**: `error`

- **功能**:遍历所有决策，调用 `validateDecision()` 逐个验证

- [ ]  **validateDecision** - 验证单个决策

- **输入**: `d *Decision, accountEquity float64, btcEthLeverage, altcoinLeverage int`

- **输出**: `error`

- **验证项**:

    1. **Action 有效性**：检查是否在允许的动作列表中

    2. **开仓参数完整性**：开仓时必填字段检查

    3. **杠杆验证**：范围检查（山寨币 ≤ 20x，BTC/ETH ≤ 50x）

    4. **仓位价值验证**：

        - **山寨币：0.8x ~ 1.5x 账户净值**

        - **BTC/ETH：5x ~ 10x 账户净值**

    5. **止损止盈合理性**：

        - **做多：止损 < 止盈**

        - **做空：止损 > 止盈**

    6. **风险回报比验证**：必须 ≥ 3.0:1

- [ ]  **calculateRiskRewardRatio** - 计算风险回报比

- **输入**: `decision *Decision, currentPrice float64`

- **输出**: `(float64, error)`

- **计算公式**:

  ```plaintext
  做多：
风险% = (入场价 - 止损价) / 入场价 × 100
收益% = (止盈价 - 入场价) / 入场价 × 100
做空：
风险% = (止损价 - 入场价) / 入场价 × 100
收益% = (入场价 - 止盈价) / 入场价 × 100
风险回报比 = 收益% / 风险%
```

- [ ]  **validatePositionSize** - 验证仓位大小

- **输入**: `symbol string, positionSizeUSD float64, accountEquity float64`

- **输出**: `error`

- **规则**:

    - **山寨币：最多 1.5x 账户净值**

    - **BTC/ETH：最多 10x 账户净值**

    - 容差 1%（避免浮点数精度问题）

- [ ]  **validateLeverage** - 验证杠杆倍数

- **输入**: `symbol string, leverage int, btcEthLeverage, altcoinLeverage int`

- **输出**: `error`

- **规则**:

    - **山寨币：1 ~ altcoinLeverage (默认 20x)**

    - **BTC/ETH：1 ~ btcEthLeverage (默认 50x)**



### Phase 4 细化：工具函数 (utils.go)

#### Priority: 🟡 Medium

- [ ]  **estimateEntryPrice** - 估算入场价格

- **输入**: `action string, stopLoss, takeProfit float64`

- **输出**: `float64`

- **逻辑**:假设在止损和止盈之间 20% 位置入场

- [ ]  **calculateHoldingTime** - 计算持仓时长

- **输入**: `entryTime, currentTime int64`

- **输出**:`int` (分钟)

- **逻辑**:(currentTime - entryTime) / 60000

- [ ]  **isBTCOrETH** - 判断是否为 BTC/ETH

- **输入**: `symbol string`

- **输出**: `bool`

- **逻辑**:检查 symbol 是否为 "BTCUSDT" 或 "ETHUSDT"

- [ ]  **formatUSD** - 格式化美元金额

- **输入**: `amount float64`

- **输出**: `string`

- **格式**:千位分隔符，保留 2 位小数

- [ ]  **formatPercentage** - 格式化百分比

- **输入**: `value float64`

- **输出**: `string`

- **格式**:带正负号，保留 2 位小数



## 风险控制规则实现

### Backlog：风险控制增强 (validator.go 扩展)

#### Priority: 🔴 High

- [ ]  **硬约束规则常量定义**

  ```go
  const (
    MinRiskRewardRatio = 3.0   // 最低风险回报比
    MaxPositions       = 3     // 最多持仓数量
    MaxMarginUsagePct  = 90.0  // 最大保证金使用率
    MinLiquidityUSD    = 15000000 // 最低流动性 15M USD

    // 仓位限制
    AltcoinMinPositionMultiple = 0.8  // 山寨币最小仓位倍数
    AltcoinMaxPositionMultiple = 1.5  // 山寨币最大仓位倍数
    BTCETHMinPositionMultiple  = 5.0  // BTC/ETH 最小仓位倍数
    BTCETHMaxPositionMultiple  = 10.0 // BTC/ETH 最大仓位倍数
)
  ```

- [ ]  **checkMaxPositions** - 检查持仓数量限制

- **输入**: `currentPositions int, newOpenActions int`

- **输出**: `error`

- **规则**:当前持仓 + 新开仓 ≤ 3

- [ ]  **checkMarginUsage** - 检查保证金使用率

- **输入**: `account AccountInfo, newPositionMargin float64`

- **输出**: `error`

- **规则**:(已用保证金 + 新仓位保证金) / 账户净值 ≤ 90%

- [ ]  **calculateRequiredMargin** - 计算所需保证金

- **输入**: `positionSizeUSD float64, leverage int`

- **输出**: `float64`

- **公式**:positionSizeUSD / leverage



## 夏普比率优化机制

### Backlog：性能反馈 (context.go 扩展)

#### Priority: 🟡 Medium

- [ ]  **buildSharpeRatioFeedback** - 构建夏普比率反馈文本

- **输入**: `performance *PerformanceMetrics`

- **输出**: `string`

- **内容**:

    - **当前夏普比率值**

    - **根据夏普比率给出策略建议：**

        - **< -0.5：停止交易，深度反思**

        - **-0.5 ~ 0：严格控制，只做高信心度交易**

        - **0 ~ 0.7：维持当前策略**

        -

      > 0.7：可适度扩大仓位

- [ ]  **analyzeTradingFrequency** - 分析交易频率

- **输入**: `performance *PerformanceMetrics`

- **输出**: `string`

- **评估标准**:

    - **优秀：每小时 0.1-0.2 笔**

    - **过度：每小时 > 2 笔**

    - **建议最佳持仓时间：30-60 分钟**



## 测试清单

### Backlog：单元测试清单

#### Priority: 🟡 Medium

- [ ]  **数据结构测试**

- **[ ]  PositionInfo 序列化/反序列化**

- **[ ]  AccountInfo 计算逻辑**

- **[ ]  Decision 验证逻辑**

- [ ]  **Prompt 生成测试**

- **[ ]  buildSystemPrompt 输出格式**

- **[ ]  buildUserPrompt 数据完整性**

- **[ ]  不同账户状态下的 prompt 变化**

- [ ]  **解析器测试**

- **[ ]  extractCoTTrace 边界情况**

- **[ ]  extractDecisions JSON 格式错误处理**

- **[ ]  fixMissingQuotes 中文引号替换**

- [ ]  **验证器测试**

- **[ ]  风险回报比计算准确性**

- **[ ]  仓位限制验证**

- **[ ]  杠杆限制验证**

- **[ ]  边界值测试（容差处理）**

- [ ]  **市场数据获取测试**

- **[ ]  流动性过滤逻辑**

- **[ ]  单个币种失败不影响整体**

- **[ ]  持仓币种优先级**



### Backlog：集成测试清单

#### Priority: 🟡 Medium

- [ ]  **完整决策流程测试**

- **[ ]  空仓状态下的首次决策**

- **[ ]  有持仓状态下的决策（持有/平仓/新开仓）**

- **[ ]  满仓状态下的决策（只能平仓或持有）**

- [ ]  **AI API 集成测试**

- **[ ]  LLM API 调用成功**

- **[ ]  API 超时处理**

- **[ ]  API 错误响应处理**

- [ ]  **风险控制测试**

- **[ ]  超过最大持仓数量时拒绝开仓**

- **[ ]  保证金不足时拒绝开仓**

- **[ ]  风险回报比不足时拒绝决策**

- [ ]  **性能测试**

- **[ ]  大量候选币种时的处理速度**

- **[ ]  Prompt 生成性能**

- **[ ]  决策验证性能**



## 与 Manager 模块的集成点

### Backlog：接口定义补充

#### Priority: 🔴 High

- [ ]  **Executor 接口定义**

  ```go
  type Executor interface {
    // 获取完整决策
    GetFullDecision(ctx *Context) (*FullDecision, error)

    // 更新性能指标（供 Manager 回调）
    UpdatePerformance(metrics *PerformanceMetrics)

    // 获取当前配置
    GetConfig() *ExecutorConfig
}
  ```

- [ ]  **ExecutorConfig 配置结构**

  ```go
  type ExecutorConfig struct {
    MajorCoinLeverage  int     // BTC/ETH 杠杆倍数
    AltcoinLeverage int     // 山寨币杠杆倍数
    MinConfidence   int     // 最低信心度
    MinRiskReward   float64 // 最低风险回报比
    MaxPositions    int     // 最多持仓数量
}
  ```

- [ ]  **Manager 需要提供的数据**

- 账户信息（AccountInfo）

- 当前持仓列表（[]PositionInfo）

- 候选币种列表（[]CandidateCoin）

- **市场数据（通过 market 包获取）**

- **性能指标（PerformanceMetrics）**

- [ ]  **Executor 返回给 Manager 的数据**

- 完整决策（FullDecision）

- 思维链分析（CoTTrace）

- 决策列表（[]Decision）

- **决策时间戳**



## 实现注意事项

### 代码质量要求

- [ ]  **错误处理**

- **所有外部调用必须有错误处理**

- **错误信息要详细且可追溯**

- **区分可恢复错误和致命错误**

- [ ]  **日志记录**

- **关键步骤记录 INFO 日志**

- **异常情况记录 WARN 日志**

- **错误记录 ERROR 日志**

- **流动性过滤、决策验证失败等记录详细信息**

- [ ]  **性能优化**

- **System Prompt 可缓存（固定规则）**

- **市场数据批量获取**

- **避免重复计算**

- [ ]  **代码规范**

- **遵循 Go 语言命名规范**

- **函数注释清晰（输入、输出、功能）**

- **复杂逻辑添加行内注释**



## 实施顺序建议

1. **Phase 0-1**：核心类型、上下文构建、Provider 接入

2. **Phase 2**：Prompt 生成与 LLM 调用

3. **Phase 3**：响应解析与风险验证

4. **Phase 4**：决策输出、性能反馈

5. **Backlog**：性能优化、测试矩阵、接口扩展



## 参考资料

- **决策引擎产品文档**：详细技术规范和设计理念

- **LLM API 文档**：AI 调用接口说明

- **Market 包文档**：市场数据格式和获取方法

- **Hyperliquid API 文档**：交易所接口参考



## 完成标准

- **[ ]  所有核心功能实现并通过单元测试**

- **[ ]  集成测试覆盖主要场景**

- **[ ]  错误处理完善，日志清晰**

- **[ ]  代码注释完整，文档齐全**

- **[ ]  性能满足要求（3 分钟决策周期内完成）**

- **[ ]  与 Manager 模块接口对接成功**

- **[ ]  实际运行验证（至少 24 小时无崩溃）**
- [ ]  **OpenInterest** - 附加 OI 数据（仅在 Manager 提供时使用）

  ```go
  type OpenInterest struct {
    Latest        float64
    Delta1hPct    float64
    DeltaValueUSD float64
    Source        string
}
  ```