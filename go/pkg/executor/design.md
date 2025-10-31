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

- [ ]  **OITopData** - 持仓量增长数据

  ```go
  type OITopData struct {
    Rank              int     // OI Top 排名
    OIDeltaPercent    float64 // 持仓量变化百分比（1小时）
    OIDeltaValue      float64 // 持仓量变化价值 (USD)
    PriceDeltaPercent float64 // 价格变化百分比
    NetLong           float64 // 净多仓
    NetShort          float64 // 净空仓
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
    CandidateCoins   []CandidateCoin         // 候选币种列表
    MarketDataMap    map[string]*MarketData  // 市场数据映射
    OITopDataMap     map[string]*OITopData   // OI Top 数据映射
    Performance      *PerformanceMetrics     // 历史表现分析
    BTCETHLeverage   int                     // BTC/ETH 杠杆倍数
    AltcoinLeverage  int                     // 山寨币杠杆倍数
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

- [ ]  **PerformanceMetrics** - 性能指标结构

  ```go
  type PerformanceMetrics struct {
    SharpeRatio      float64 // 夏普比率
    TotalTrades      int     // 总交易次数
    WinRate          float64 // 胜率
    AvgHoldingTime   int     // 平均持仓时间（分钟）
    RecentTradesRate float64 // 最近交易频率（笔/小时）
}
  ```



## 核心功能实现

### Phase 2: 市场数据获取 (market_data.go)

#### Priority: 🔴 High

- [ ]  **fetchMarketDataForContext** - 获取市场数据并填充上下文

- **输入**: `ctx *Context`

- **输出**: `error`

- **功能**:

    - 优先获取持仓币种市场数据（必须）

    - 根据账户状态动态获取候选币种数据

    - **加载 OI Top 数据**

    - **流动性过滤（持仓价值 < 15M USD 跳过，现有持仓除外）**

- **实现要点**:

    - **单个币种失败不影响整体流程**

    - **记录详细日志（流动性过滤、获取失败等）**

    - **计算持仓价值 = 持仓量 × 当前价格**

- [ ]  **calculateMaxCandidates** - 计算最大候选币种数量

- **输入**: `ctx *Context`

- **输出**: `int`

- **逻辑**:返回候选池全部币种数量（候选池已在 manager 中筛选）

- [ ]  **filterByLiquidity** - 流动性过滤函数

- **输入**: `symbol string, positionValue float64, isExistingPosition bool`

- **输出**:`bool` (是否通过过滤)

- **规则**:

    - **持仓价值 < 15M USD → 跳过**

    - **现有持仓必须保留（需决策是否平仓）**



### Phase 3: Prompt 生成 (prompt.go)

#### Priority: 🔴 High

- [ ]  **buildSystemPrompt** - 构建系统提示词（固定规则，可缓存）

- **输入**: `accountEquity float64`

- **输出**: `string`

- **内容包含**:

    - **核心目标：最大化夏普比率**

    - **硬约束：风险回报比 ≥ 3:1、最多 3 个持仓、保证金使用率 ≤ 90%**

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

    - **候选币种（含完整市场数据、OI Top 信息）**

    - **夏普比率反馈（当前值、历史表现）**

- **实现要点**:

    - **使用 `market.Format()` 输出完整市场数据**

    - **计算持仓时长（当前时间 - 入场时间）**

    - **标注币种来源（ai500、oi_top 或双重信号）**

- [ ]  **formatPositionInfo** - 格式化持仓信息

- **输入**: `position PositionInfo, currentTime time.Time`

- **输出**: `string`

- **格式**:包含符号、方向、入场价、当前价、盈亏、持仓时长

- [ ]  **formatCandidateInfo** - 格式化候选币种信息

- **输入**: `coin CandidateCoin, marketData *MarketData, oiData *OITopData`

- **输出**: `string`

- **格式**:包含完整市场数据、OI 信息、来源标签



### Phase 4: AI 集成 (executor.go)

#### Priority: 🔴 High

- [ ]  **GetFullDecision** - 主入口函数

- **输入**: `ctx *Context`

- **输出**: `(*FullDecision, error)`

- **流程**:

    1. 调用 `fetchMarketDataForContext(ctx)` 获取市场数据

    2. 调用 `buildSystemPrompt(ctx.Account.TotalEquity)` 构建系统提示词

    3. 调用 `buildUserPrompt(ctx)` 构建用户提示词

    4. 调用 `mcp.CallWithMessages(systemPrompt, userPrompt)` 获取 AI 响应

    5. 调用 `parseFullDecisionResponse(aiResponse, ctx)` 解析响应

    6. 返回 `FullDecision` 结构

- [ ]  **callAIAPI** - AI API 调用封装

- **输入**: `systemPrompt string, userPrompt string`

- **输出**: `(string, error)`

- **功能**:

    - **使用 MCP (Model Context Protocol) 调用 Claude**

    - **处理 API 错误和重试逻辑**

    - **记录请求和响应日志**



### Phase 5: 响应解析 (parser.go)

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



### Phase 6: 决策验证 (validator.go)

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



### Phase 7: 工具函数 (utils.go)

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

### Phase 8: 风险控制 (validator.go 扩展)

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

### Phase 9: 性能反馈 (context.go 扩展)

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

### Phase 10: 单元测试

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



### Phase 11: 集成测试

#### Priority: 🟡 Medium

- [ ]  **完整决策流程测试**

- **[ ]  空仓状态下的首次决策**

- **[ ]  有持仓状态下的决策（持有/平仓/新开仓）**

- **[ ]  满仓状态下的决策（只能平仓或持有）**

- [ ]  **AI API 集成测试**

- **[ ]  MCP API 调用成功**

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

### Phase 12: 接口定义

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
    BTCETHLeverage  int     // BTC/ETH 杠杆倍数
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

1. **Phase 1-2**：基础数据结构 + 市场数据获取（建立数据基础）

2. **Phase 3-4**：Prompt 生成 + AI 集成（实现核心决策流程）

3. **Phase 5-6**：响应解析 + 决策验证（确保输出质量）

4. **Phase 7-8**：工具函数 + 风险控制（完善细节）

5. **Phase 9**：性能反馈机制（优化策略）

6. **Phase 10-11**：测试（保证质量）

7. **Phase 12**：接口定义（对接 Manager）



## 参考资料

- **决策引擎产品文档**：详细技术规范和设计理念

- **MCP API 文档**：AI 调用接口说明

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
