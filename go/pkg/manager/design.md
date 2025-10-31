# Manager 模块设计

Manager 管理器模块是新系统的**编排层（Orchestration Layer）**，负责协调多个虚拟交易员（Virtual Traders）的运行、资源分配、风险管理和性能监控。

## 核心职责

- **虚拟交易员管理** - 创建、配置、启停多个独立的交易员实例
- **交易循环编排** - 协调各交易员的决策周期和执行顺序
- **交易所适配** - 统一的交易所接口，支持任意交易所接入
- **跨交易员监控** - 聚合性能指标、持仓状态、风险暴露
- **资源协调** - 在多个交易员间分配资金、管理冲突信号

## 核心创新：虚拟交易员（Virtual Trader）

### 传统架构问题

- Trader = 交易所（一对一绑定）
- 难以在同一交易所运行多种策略
- 配置变更需要修改代码

### 新架构优势

**Virtual Trader = Exchange + Prompt Template + Risk Parameters**

示例组合：
```plaintext
├─ Trader_A: Hyperliquid + 激进做空策略 + 20x杠杆
├─ Trader_B: Hyperliquid + 保守做多策略 + 10x杠杆
├─ Trader_C: Binance + 网格套利策略 + 5x杠杆
└─ Trader_D: Hyperliquid + BTC专注策略 + 50x杠杆
```

### 设计目的

- **同一交易所运行多个策略**
- **快速测试不同 Prompt 组合**
- **动态创建/删除交易员**
- **独立的风险参数和资金分配**

## 目录结构

```plaintext
manager/
├── manager.go              # 核心管理器
├── trader.go               # 虚拟交易员抽象
├── config.go               # 配置管理
├── orchestrator.go         # 交易循环编排
├── resource_allocator.go   # 资源分配器
├── conflict_resolver.go    # 冲突解决器
├── state_manager.go        # 状态持久化
├── monitor.go              # 性能监控
├── exchange/               # 交易所适配器
│   ├── adapter.go          # 统一接口定义
│   ├── hyperliquid.go      # Hyperliquid 实现
│   ├── binance.go          # Binance 实现
│   └── mock.go             # 模拟交易所（测试用）
└── manager_test.go         # 集成测试
```

## 核心组件实现清单

### 1. 虚拟交易员抽象层 (trader.go)

#### ☐ **P0** - 定义 VirtualTrader 结构体

```go
type VirtualTrader struct {
    ID              string                 // 唯一标识（如 "trader_aggressive_short"）
    Name            string                 // 显示名称
    Exchange        string                 // 交易所类型（hyperliquid/binance）
    ExchangeAdapter ExchangeAdapter        // 交易所适配器实例
    PromptTemplate  string                 // Prompt 模板路径或内容
    RiskParams      RiskParameters         // 风险参数配置
    ResourceAlloc   ResourceAllocation     // 资源分配
    State           TraderState            // 运行状态
    Performance     PerformanceMetrics     // 性能指标
    LastDecisionAt  time.Time              // 上次决策时间
    DecisionInterval time.Duration         // 决策间隔（如 3分钟）
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

**实现要点**：

- **每个字段的详细注释**
- **合理的默认值设置**
- **状态枚举定义（Running/Paused/Stopped/Error）**

#### ☐ **P0** - 定义 RiskParameters 结构体

```go
type RiskParameters struct {
    MaxPositions        int     // 最大持仓数量
    MaxPositionSizeUSD  float64 // 单币种最大仓位（USD）
    MaxMarginUsagePct   float64 // 最大保证金使用率（%）
    BTCETHLeverage      int     // BTC/ETH 杠杆倍数
    AltcoinLeverage     int     // 山寨币杠杆倍数
    MinRiskRewardRatio  float64 // 最小风险回报比（默认 3.0）
    MinConfidence       int     // 最小信心度（默认 75）
    StopLossEnabled     bool    // 是否启用止损
    TakeProfitEnabled   bool    // 是否启用止盈
}
```

**实现要点**：

- 验证函数 `Validate() error`

- 默认值生成函数 `DefaultRiskParameters()`

- 从配置文件加载

#### ☐ **P0** - 定义 ResourceAllocation 结构体

```go
type ResourceAllocation struct {
    AllocatedEquityUSD  float64 // 分配的账户净值（USD）
    AllocationPct       float64 // 占总账户的百分比
    CurrentEquityUSD    float64 // 当前实际净值
    AvailableBalanceUSD float64 // 可用余额
    MarginUsedUSD       float64 // 已用保证金
    UnrealizedPnLUSD    float64 // 未实现盈亏
}
```

**实现要点**：

- 实时更新机制

- 超限检测函数 `IsOverAllocated() bool`

#### ☐ **P1** - 实现 Trader 生命周期方法

```go
func (t *VirtualTrader) Start() error
func (t *VirtualTrader) Pause() error
func (t *VirtualTrader) Resume() error
func (t *VirtualTrader) Stop() error
func (t *VirtualTrader) IsActive() bool
```

**实现要点**：

- 状态转换验证（如 Stopped 不能 Pause）

- 线程安全（使用 sync.RWMutex）

- 状态变更日志记录

#### ☐ **P1** - 实现 Trader 决策触发逻辑

```go
func (t *VirtualTrader) ShouldMakeDecision() bool
func (t *VirtualTrader) RecordDecision(timestamp time.Time)
```

**实现要点**：

- 基于 `DecisionInterval` 判断

- 考虑交易所 API 限流

- 避免同一时刻多个 Trader 同时调用

### 2. 核心管理器 (manager.go)

#### ☐ **P0** - 定义 Manager 结构体

```go
type Manager struct {
    traders          map[string]*VirtualTrader // Trader ID -> Trader
    exchangeAdapters map[string]ExchangeAdapter // Exchange Type -> Adapter
    orchestrator     *Orchestrator
    resourceAllocator *ResourceAllocator
    conflictResolver *ConflictResolver
    stateManager     *StateManager
    monitor          *Monitor
    config           *Config
    mu               sync.RWMutex
    stopChan         chan struct{}
    wg               sync.WaitGroup
}
```



#### ☐ **P0** - 实现 InitializeManager

```go
func InitializeManager(configPath string) (*Manager, error)
```

**实现步骤**：

1. 加载配置文件（YAML/JSON）

2. 初始化交易所适配器

3. 加载已保存的 Trader 状态

4. 初始化子模块（orchestrator, allocator, resolver, monitor）

5. 启动监控协程

6. 返回 Manager 实例

**错误处理**：

- 配置文件不存在或格式错误

- 交易所连接失败

- 状态恢复失败

#### ☐ **P0** - 实现 RegisterTrader

```go
func (m *Manager) RegisterTrader(config TraderConfig) (*VirtualTrader, error)
```

**实现步骤**：

1. 验证配置有效性

2. 检查 Trader ID 唯一性

3. 创建 VirtualTrader 实例

4. 分配交易所适配器

5. 初始化资源分配

6. 加入 traders map

7. 持久化状态

8. 触发监控更新

**验证项**：

- ID 不能为空且唯一

- Exchange 类型必须已注册

- PromptTemplate 必须存在

- RiskParams 必须合法

- ResourceAllocation 不能超过总资金

#### ☐ **P1** - 实现 UnregisterTrader

```go
func (m *Manager) UnregisterTrader(traderID string) error
```

**实现步骤**：

1. 检查 Trader 是否存在

2. 停止 Trader（如果正在运行）

3. 平掉所有持仓（可选，根据配置）

4. 释放资源分配

5. 从 traders map 移除

6. 删除持久化状态

7. 触发监控更新

**安全检查**：

- 不能删除有持仓的 Trader（除非强制）

- 确认操作（防止误删）

#### ☐ **P0** - 实现 RunTradingLoop（主编排循环）

```go
func (m *Manager) RunTradingLoop(ctx context.Context) error
```

**实现逻辑**：

```go
for {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-time.After(1 * time.Second): // 每秒检查一次
        // 1. 遍历所有 Active Traders
        for _, trader := range m.GetActiveTraders() {
            if trader.ShouldMakeDecision() {
                // 2. 调用 Executor 获取决策
                decision, err := m.orchestrator.RequestDecision(trader)
                if err != nil {
                    log.Error("决策失败", trader.ID, err)
                    continue
                }

                // 3. 冲突检测和解决
                resolvedDecisions := m.conflictResolver.Resolve(decision, m.traders)

                // 4. 执行决策
                for _, d := range resolvedDecisions {
                    err := m.ExecuteDecision(trader, d)
                    if err != nil {
                        log.Error("执行失败", trader.ID, d.Symbol, err)
                    }
                }

                // 5. 更新 Trader 状态
                trader.RecordDecision(time.Now())
            }
        }

        // 6. 同步持仓和账户状态
        m.SyncAllPositions()

        // 7. 更新性能监控
        m.monitor.Update(m.traders)
    }
}
```

**关键点**：

- 使用 context 控制生命周期

- 错误不应中断主循环

- 每个 Trader 独立决策周期

- 避免 API 限流（错开调用时间）

#### ☐ **P0** - 实现 ExecuteDecision

```go
func (m *Manager) ExecuteDecision(trader *VirtualTrader, decision *Decision) error
```

**实现步骤**：

1. 验证决策合法性（风险参数）

2. 检查资源可用性（余额、保证金）

3. 调用交易所适配器执行

4. 记录执行结果

5. 更新持仓状态

6. 更新资源分配

7. 触发监控事件

**错误处理**：

- 余额不足

- 交易所 API 错误

- 网络超时

- 订单被拒绝

#### ☐ **P1** - 实现 SyncPositions

```go
func (m *Manager) SyncAllPositions() error
func (m *Manager) SyncTraderPositions(traderID string) error
```

**实现逻辑**：

1. 调用交易所 API 获取最新持仓

2. 更新 Trader 的 Performance 指标

3. 更新 ResourceAllocation

4. 检测异常（如持仓丢失）

5. 触发告警（如有必要）

**同步频率**：

- 全量同步：每 30 秒

- 单个 Trader：决策后立即同步

#### ☐ **P1** - 实现 MonitorPerformance

```go
func (m *Manager) MonitorPerformance() *AggregatePerformance
func (m *Manager) GetTraderPerformance(traderID string) *PerformanceMetrics
```

**性能指标**：

```go
type PerformanceMetrics struct {
    TotalPnLUSD       float64
    TotalPnLPct       float64
    SharpeRatio       float64
    WinRate           float64
    TotalTrades       int
    WinningTrades     int
    LosingTrades      int
    AvgWinUSD         float64
    AvgLossUSD        float64
    MaxDrawdownPct    float64
    CurrentDrawdownPct float64
    UpdatedAt         time.Time
}
```

**聚合性能**：

```go
type AggregatePerformance struct {
    TotalEquityUSD    float64
    TotalPnLUSD       float64
    AverageSharpe     float64
    BestTrader        string
    WorstTrader       string
    TraderCount       int
    ActiveTraderCount int
}
```



#### ☐ **P2** - 实现 Trader 配置热更新

```go
func (m *Manager) UpdateTraderConfig(traderID string, newConfig TraderConfig) error
```

**可更新项**：

- PromptTemplate（需重启决策）

- RiskParameters（立即生效）

- DecisionInterval（立即生效）

- ResourceAllocation（需验证）

**不可更新项**：

- ID（唯一标识）

- Exchange（需重建 Trader）

### 3. 交易循环编排器 (orchestrator.go)

#### ☐ **P0** - 定义 Orchestrator 结构体

```go
type Orchestrator struct {
    executorClient ExecutorClient // 与 Executor 模块通信
    decisionQueue  chan DecisionRequest
    resultQueue    chan DecisionResult
    workers        int // 并发 worker 数量
}
```



#### ☐ **P0** - 实现 RequestDecision

```go
func (o *Orchestrator) RequestDecision(trader *VirtualTrader) (*FullDecision, error)
```

**实现步骤**：

1. 构建决策请求（包含 Trader 配置、市场数据、持仓状态）

2. 发送到 Executor 模块（HTTP/gRPC/本地调用）

3. 等待决策结果（带超时）

4. 解析和验证决策

5. 返回 FullDecision

**超时处理**：

- 默认超时：30 秒

- 超时后返回错误，不影响其他 Trader

#### ☐ **P1** - 实现并发决策调度

```go
func (o *Orchestrator) StartWorkers(ctx context.Context)
```

**设计思路**：

- 使用 worker pool 并发处理多个 Trader 的决策请求

- 避免阻塞主循环

- 限制并发数（防止 API 限流）

#### ☐ **P1** - 实现决策优先级队列

```go
func (o *Orchestrator) PrioritizeRequests(requests []DecisionRequest) []DecisionRequest
```

**优先级规则**：

1. 有持仓的 Trader 优先（需要及时止损/止盈）

2. 夏普比率高的 Trader 优先

3. 资金分配大的 Trader 优先

4. 其他按 FIFO

### 4. 资源分配器 (resource_allocator.go)

#### ☐ **P0** - 定义 ResourceAllocator 结构体

```go
type ResourceAllocator struct {
    totalEquityUSD    float64
    allocatedEquityUSD float64
    reserveEquityUSD  float64 // 预留资金（不分配）
    allocationStrategy string  // "equal" / "performance_based" / "custom"
}
```



#### ☐ **P0** - 实现 AllocateResources

```go
func (ra *ResourceAllocator) AllocateResources(traders []*VirtualTrader) error
```

**分配策略**：

**1. 平均分配（Equal）**：

```plaintext
每个 Trader 分配 = 总资金 / Trader 数量
```

**2. 基于性能（Performance-Based）**：

```plaintext
分配权重 = Trader 夏普比率 / 所有 Trader 夏普比率之和
分配金额 = 总资金 × 分配权重
```

**3. 自定义（Custom）**：

```plaintext
从配置文件读取每个 Trader 的分配比例
```

**验证**：

- 总分配不能超过 `totalEquityUSD - reserveEquityUSD`

- 每个 Trader 至少分配最小金额（如 100 USD）

#### ☐ **P1** - 实现动态再平衡

```go
func (ra *ResourceAllocator) Rebalance(traders []*VirtualTrader) error
```

**触发条件**：

- 定期触发（如每小时）

- Trader 性能显著变化（夏普比率变化 > 0.2）

- 新增/删除 Trader

**再平衡逻辑**：

1. 计算新的分配方案

2. 对比当前分配

3. 调整资金（可能需要平仓部分持仓）

4. 更新 ResourceAllocation

#### ☐ **P1** - 实现资源使用监控

```go
func (ra *ResourceAllocator) CheckUtilization() *UtilizationReport
```

**报告内容**：

```go
type UtilizationReport struct {
    TotalEquityUSD       float64
    AllocatedEquityUSD   float64
    UtilizedEquityUSD    float64 // 实际使用（含持仓）
    ReserveEquityUSD     float64
    UtilizationPct       float64 // 使用率
    OverAllocatedTraders []string // 超限 Trader
}
```



### 5. 冲突解决器 (conflict_resolver.go)

#### ☐ **P0** - 定义 ConflictResolver 结构体

```go
type ConflictResolver struct {
    resolutionStrategy string // "first_come" / "highest_confidence" / "aggregate"
}
```



#### ☐ **P0** - 实现 Resolve 方法

```go
func (cr *ConflictResolver) Resolve(
    newDecision *FullDecision,
    allTraders map[string]*VirtualTrader,
) []*Decision
```

**冲突场景**：

**场景 1：同一币种，不同方向**

```plaintext
Trader_A: BTCUSDT open_long
Trader_B: BTCUSDT open_short
```

**解决策略**：

- **first_come**：保留先到的决策

- **highest_confidence**：保留信心度高的

- **aggregate**：取消双方决策（信号矛盾）

**场景 2：同一币种，相同方向**

```plaintext
Trader_A: BTCUSDT open_long (1000 USD)
Trader_B: BTCUSDT open_long (1500 USD)
```

**解决策略**：

- **first_come**：只执行先到的

- **aggregate**：合并仓位（总计 2500 USD，分配到两个 Trader）

**场景 3：资源不足**

```plaintext
可用余额: 500 USD
Trader_A: 需要 400 USD
Trader_B: 需要 300 USD
```

**解决策略**：

- **highest_confidence**：优先执行信心度高的

- **proportional**：按比例缩减仓位

#### ☐ **P1** - 实现冲突检测

```go
func (cr *ConflictResolver) DetectConflicts(
    decisions []*Decision,
    traders map[string]*VirtualTrader,
) []Conflict
```

**冲突类型**：

```go
type Conflict struct {
    Type        string   // "direction" / "resource" / "duplicate"
    Symbol      string
    TraderIDs   []string
    Decisions   []*Decision
    Severity    string   // "high" / "medium" / "low"
}
```



#### ☐ **P1** - 实现冲突日志记录

```go
func (cr *ConflictResolver) LogConflict(conflict Conflict)
```

**日志格式**：

```plaintext
[CONFLICT] 类型:方向冲突 币种:BTCUSDT 严重性:高
  Trader_A: open_long (信心度:85)
  Trader_B: open_short (信心度:78)
  解决方案: 保留 Trader_A 决策（信心度更高）
```



### 6. 状态持久化 (state_manager.go)

#### ☐ **P0** - 定义 StateManager 结构体

```go
type StateManager struct {
    storageBackend string // "file" / "redis" / "database"
    storagePath    string
}
```



#### ☐ **P0** - 实现状态保存

```go
func (sm *StateManager) SaveState(manager *Manager) error
```

**保存内容**：

```go
type ManagerState struct {
    Traders          map[string]*VirtualTrader
    ResourceAllocator *ResourceAllocator
    LastSyncTime     time.Time
    Version          string
}
```

**存储格式**：

- JSON 文件（开发阶段）

- Redis（生产环境）

- PostgreSQL（长期存储）

#### ☐ **P0** - 实现状态恢复

```go
func (sm *StateManager) LoadState() (*ManagerState, error)
```

**恢复步骤**：

1. 读取状态文件/数据库

2. 验证版本兼容性

3. 重建 VirtualTrader 实例

4. 重新连接交易所适配器

5. 同步最新持仓状态

6. 验证数据完整性

**错误处理**：

- 状态文件损坏 → 使用默认配置

- 版本不兼容 → 迁移数据结构

- 交易所连接失败 → 重试机制

#### ☐ **P1** - 实现增量状态更新

```go
func (sm *StateManager) UpdateTraderState(trader *VirtualTrader) error
```

**触发时机**：

- Trader 配置变更

- 决策执行后

- 持仓状态变化

- 性能指标更新

**优化**：

- 批量更新（减少 I/O）

- 异步写入（不阻塞主循环）

- 写入队列（防止丢失）

#### ☐ **P2** - 实现状态快照

```go
func (sm *StateManager) CreateSnapshot(label string) error
func (sm *StateManager) RestoreSnapshot(label string) error
```

**用途**：

- 重大配置变更前备份

- 测试新策略前保存快照

- 灾难恢复

### 7. 性能监控 (monitor.go)

#### ☐ **P0** - 定义 Monitor 结构体

```go
type Monitor struct {
    metricsStore   MetricsStore
    alerter        Alerter
    updateInterval time.Duration
}
```



#### ☐ **P0** - 实现实时监控

```go
func (m *Monitor) Update(traders map[string]*VirtualTrader) error
```

**监控指标**：

**系统级**：

- 总账户净值

- 总未实现盈亏

- 总保证金使用率

- Active Trader 数量

- 决策成功率

**Trader 级**：

- 夏普比率

- 总盈亏（USD 和 %）

- 胜率

- 最大回撤

- 持仓数量

- 最近决策时间

#### ☐ **P1** - 实现告警机制

```go
func (m *Monitor) CheckAlerts(traders map[string]*VirtualTrader) []Alert
```

**告警规则**：

**高优先级**：

- 保证金使用率 > 90%

- 单个 Trader 亏损 > 20%

- 总账户亏损 > 15%

- 交易所连接断开

**中优先级**：

- 夏普比率 < -0.5（持续 1 小时）

- Trader 长时间无决策（> 30 分钟）

- 资源分配超限

**低优先级**：

- 决策失败率 > 10%

- API 调用延迟 > 5 秒

#### ☐ **P1** - 实现性能报告生成

```go
func (m *Monitor) GenerateReport(period string) *PerformanceReport
```

**报告周期**：

- 实时（最近 1 小时）

- 每日

- 每周

- 每月

**报告内容**：

```go
type PerformanceReport struct {
    Period            string
    TotalPnLUSD       float64
    TotalPnLPct       float64
    AverageSharpe     float64
    TotalTrades       int
    WinRate           float64
    BestTrader        TraderSummary
    WorstTrader       TraderSummary
    TopSymbols        []SymbolPerformance
    TraderBreakdown   []TraderPerformance
    GeneratedAt       time.Time
}
```



#### ☐ **P2** - 实现可视化仪表板接口

```go
func (m *Monitor) GetDashboardData() *DashboardData
```

**仪表板数据**：

- 实时净值曲线

- 各 Trader 盈亏分布

- 持仓热力图

- 决策频率统计

- 告警历史

**输出格式**：

- JSON（供前端调用）

- Prometheus Metrics（供 Grafana）

### 8. 交易所适配器接口 (exchange/adapter.go)

#### ☐ **P0** - 定义 ExchangeAdapter 接口

```go
type ExchangeAdapter interface {
    // 连接管理
    Connect(config ExchangeConfig) error
    Disconnect() error
    IsConnected() bool

    // 账户信息
    GetAccountInfo() (*AccountInfo, error)
    GetPositions() ([]PositionInfo, error)

    // 市场数据
    GetMarketData(symbol string) (*MarketData, error)
    GetMarketDataBatch(symbols []string) (map[string]*MarketData, error)

    // 交易执行
    OpenPosition(order *OpenPositionOrder) (*OrderResult, error)
    ClosePosition(order *ClosePositionOrder) (*OrderResult, error)
    ModifyPosition(order *ModifyPositionOrder) (*OrderResult, error)

    // 订单管理
    GetOrder(orderID string) (*Order, error)
    CancelOrder(orderID string) error

    // 元数据
    GetExchangeInfo() *ExchangeInfo
    GetTradingFees(symbol string) (*TradingFees, error)
}
```



#### ☐ **P0** - 定义通用数据结构

**OpenPositionOrder**：

```go
type OpenPositionOrder struct {
    Symbol          string
    Side            string  // "long" / "short"
    Leverage        int
    PositionSizeUSD float64
    StopLoss        float64
    TakeProfit      float64
    OrderType       string  // "market" / "limit"
    LimitPrice      float64 // 限价单价格（可选）
}
```

**ClosePositionOrder**：

```go
type ClosePositionOrder struct {
    Symbol      string
    Side        string
    Quantity    float64 // 平仓数量（0 = 全平）
    OrderType   string  // "market" / "limit"
    LimitPrice  float64
}
```

**OrderResult**：

```go
type OrderResult struct {
    OrderID       string
    Status        string  // "filled" / "partial" / "rejected"
    FilledQty     float64
    AvgPrice      float64
    Fee           float64
    Timestamp     time.Time
    ErrorMessage  string
}
```



#### ☐ **P0** - 实现 Hyperliquid 适配器 (exchange/hyperliquid.go)

```go
type HyperliquidAdapter struct {
    client    *hyperliquid.Client
    apiKey    string
    apiSecret string
    testnet   bool
}
```

**实现方法**：

- 复用现有 `hyperliquid/` 包的代码

- 适配到统一接口

- 添加错误处理和重试机制

#### ☐ **P1** - 实现 Binance 适配器 (exchange/binance.go)

```go
type BinanceAdapter struct {
    client    *binance.Client
    apiKey    string
    apiSecret string
    testnet   bool
}
```

**实现方法**：

- 复用现有 `binance/` 包的代码

- 适配到统一接口

- 处理 Binance 特有的限流规则

#### ☐ **P2** - 实现 Mock 适配器 (exchange/mock.go)

```go
type MockAdapter struct {
    positions      []PositionInfo
    accountBalance float64
    marketData     map[string]*MarketData
}
```

**用途**：

- 单元测试

- 集成测试

- 策略回测（未来扩展）

**实现要点**：

- 模拟订单执行延迟

- 模拟部分成交

- 模拟 API 错误

### 9. 配置管理 (config.go)

#### ☐ **P0** - 定义配置结构

```go
type Config struct {
    Manager    ManagerConfig
    Traders    []TraderConfig
    Exchanges  map[string]ExchangeConfig
    Monitoring MonitoringConfig
}

type ManagerConfig struct {
    TotalEquityUSD      float64
    ReserveEquityPct    float64
    AllocationStrategy  string
    RebalanceInterval   time.Duration
    StateStorageBackend string
    StateStoragePath    string
}

type TraderConfig struct {
    ID                string
    Name              string
    Exchange          string
    PromptTemplate    string
    DecisionInterval  time.Duration
    RiskParams        RiskParameters
    AllocationPct     float64
    AutoStart         bool
}

type ExchangeConfig struct {
    Type      string
    APIKey    string
    APISecret string
    Testnet   bool
    Timeout   time.Duration
}

type MonitoringConfig struct {
    UpdateInterval  time.Duration
    AlertWebhook    string
    MetricsExporter string // "prometheus" / "influxdb"
}
```



#### ☐ **P0** - 实现配置加载

```go
func LoadConfig(path string) (*Config, error)
```

**支持格式**：

- YAML（推荐）

- JSON

- TOML

**示例 YAML**：

```yaml
manager:
  total_equity_usd: 10000
  reserve_equity_pct: 10
  allocation_strategy: performance_based
  rebalance_interval: 1h
  state_storage_backend: file
  state_storage_path: ./data/state.json

traders:
  - id: trader_aggressive_short
    name: 激进做空策略
    exchange: hyperliquid
    prompt_template: ./prompts/aggressive_short.txt
    decision_interval: 3m
    risk_params:
      max_positions: 3
      altcoin_leverage: 20
      min_confidence: 80
    allocation_pct: 30
    auto_start: true

  - id: trader_conservative_long
    name: 保守做多策略
    exchange: hyperliquid
    prompt_template: ./prompts/conservative_long.txt
    decision_interval: 5m
    risk_params:
      max_positions: 2
      altcoin_leverage: 10
      min_confidence: 85
    allocation_pct: 30
    auto_start: true

exchanges:
  hyperliquid:
    type: hyperliquid
    api_key: ${HYPERLIQUID_API_KEY}
    api_secret: ${HYPERLIQUID_API_SECRET}
    testnet: false
    timeout: 30s

monitoring:
  update_interval: 10s
  alert_webhook: https://hooks.slack.com/services/xxx
  metrics_exporter: prometheus
```



#### ☐ **P1** - 实现配置验证

```go
func (c *Config) Validate() error
```

**验证项**：

- 总资金分配 ≤ 100%

- Trader ID 唯一性

- Exchange 配置完整性

- Prompt 文件存在性

- 风险参数合理性

#### ☐ **P2** - 实现配置热加载

```go
func (c *Config) Reload() error
```

**支持热加载项**：

- Trader 配置（需重启 Trader）

- 监控配置（立即生效）

- 告警规则（立即生效）

**不支持热加载项**：

- 交易所配置（需重启 Manager）

- 状态存储配置（需重启）

## 多交易员协调机制

### ☐ **P0** - 实现全局风险限制

```go
func (m *Manager) CheckGlobalRiskLimits() error
```

**全局限制**：

- 总保证金使用率 ≤ 85%

- 总持仓数量 ≤ 10

- 单币种总仓位 ≤ 总资金 × 50%

- 总未实现亏损 ≤ 总资金 × 20%

**超限处理**：

- 暂停所有新开仓

- 触发告警

- 自动平掉部分亏损仓位（可选）

### ☐ **P1** - 实现 Trader 间隔离

```go
func (m *Manager) IsolateTrader(traderID string, reason string) error
```

**隔离场景**：

- Trader 连续亏损（夏普比率 < -1.0）

- Trader 决策频繁失败

- Trader 违反风险规则

**隔离措施**：

- 停止新开仓

- 保留现有持仓（或强制平仓）

- 释放资源分配

- 记录隔离原因

### ☐ **P1** - 实现 Trader 性能排名

```go
func (m *Manager) RankTraders() []TraderRanking
```

**排名指标**：

```go
type TraderRanking struct {
    TraderID      string
    Rank          int
    SharpeRatio   float64
    TotalPnLPct   float64
    WinRate       float64
    Score         float64 // 综合评分
}
```

**评分公式**：

```plaintext
Score = 0.5 × SharpeRatio + 0.3 × TotalPnLPct + 0.2 × WinRate
```

**用途**：

- 动态资源分配

- 决策优先级

- 性能报告

## 集成测试策略

### ☐ **P0** - 单 Trader 基础测试

```go
func TestSingleTraderBasicFlow(t *testing.T)
```

**测试场景**：

1. 注册 Trader

2. 启动 Trader

3. 触发决策

4. 执行开仓

5. 同步持仓

6. 执行平仓

7. 验证性能指标

### ☐ **P0** - 多 Trader 并发测试

```go
func TestMultiTraderConcurrency(t *testing.T)
```

**测试场景**：

1. 注册 3 个 Trader

2. 同时启动

3. 并发决策

4. 验证无资源竞争

5. 验证决策隔离

### ☐ **P1** - 冲突解决测试

```go
func TestConflictResolution(t *testing.T)
```

**测试场景**：

- 同一币种反向信号

- 同一币种同向信号

- 资源不足场景

- 验证解决策略正确性

### ☐ **P1** - 状态恢复测试

```go
func TestStateRecovery(t *testing.T)
```

**测试场景**：

1. 运行 Manager 并执行交易

2. 保存状态

3. 模拟崩溃（停止 Manager）

4. 恢复状态

5. 验证 Trader 配置

6. 验证持仓状态

7. 验证资源分配

### ☐ **P2** - 压力测试

```go
func TestManagerStressTest(t *testing.T)
```

**测试场景**：

- 10 个 Trader 同时运行

- 高频决策（每分钟）

- 模拟 API 延迟

- 模拟网络错误

- 验证系统稳定性

## 与 Executor 模块集成

### ☐ **P0** - 定义通信接口

```go
type ExecutorClient interface {
    RequestDecision(ctx context.Context, req *DecisionRequest) (*DecisionResponse, error)
    Ping() error
}

type DecisionRequest struct {
    TraderID       string
    PromptTemplate string
    MarketData     map[string]*MarketData
    Positions      []PositionInfo
    Account        AccountInfo
    RiskParams     RiskParameters
}

type DecisionResponse struct {
    CoTTrace   string
    Decisions  []Decision
    Timestamp  time.Time
    ErrorMsg   string
}
```



### ☐ **P0** - 实现本地调用（同进程）

```go
type LocalExecutorClient struct {
    executor *executor.Executor
}

func (c *LocalExecutorClient) RequestDecision(ctx context.Context, req *DecisionRequest) (*DecisionResponse, error) {
    // 直接调用 executor 包的函数
    return c.executor.GetFullDecision(req)
}
```



### ☐ **P1** - 实现 HTTP 调用（跨进程）

```go
type HTTPExecutorClient struct {
    baseURL    string
    httpClient *http.Client
}

func (c *HTTPExecutorClient) RequestDecision(ctx context.Context, req *DecisionRequest) (*DecisionResponse, error) {
    // POST /api/v1/decision
    // Body: JSON(req)
    // Response: JSON(DecisionResponse)
}
```



### ☐ **P2** - 实现 gRPC 调用（高性能）

```go
type GRPCExecutorClient struct {
    conn   *grpc.ClientConn
    client pb.ExecutorServiceClient
}
```

**优势**：

- 更低延迟

- 二进制协议

- 流式响应（未来扩展）

## 监控和告警集成

### ☐ **P1** - 实现 Prometheus Exporter

```go
func (m *Monitor) ExposeMetrics(port int)
```

**暴露指标**：

```plaintext
# HELP nof1_manager_total_equity_usd Total account equity in USD
# TYPE nof1_manager_total_equity_usd gauge
nof1_manager_total_equity_usd 10000.0

# HELP nof1_trader_sharpe_ratio Trader Sharpe Ratio
# TYPE nof1_trader_sharpe_ratio gauge
nof1_trader_sharpe_ratio{trader_id="trader_aggressive_short"} 0.65

# HELP nof1_trader_total_pnl_usd Trader total PnL in USD
# TYPE nof1_trader_total_pnl_usd gauge
nof1_trader_total_pnl_usd{trader_id="trader_aggressive_short"} 250.0
```



### ☐ **P1** - 实现 Webhook 告警

```go
func (m *Monitor) SendAlert(alert Alert) error
```

**告警格式（Slack）**：

```json
{
  "text": "🚨 高优先级告警",
  "attachments": [
    {
      "color": "danger",
      "fields": [
        {"title": "类型", "value": "保证金使用率过高", "short": true},
        {"title": "Trader", "value": "trader_aggressive_short", "short": true},
        {"title": "当前值", "value": "92%", "short": true},
        {"title": "阈值", "value": "90%", "short": true}
      ]
    }
  ]
}
```



## 实现优先级说明

### P0 - 核心功能（必须实现）

- Manager 基础结构和生命周期

- VirtualTrader 抽象和管理

- 交易循环编排

- 交易所适配器接口

- 状态持久化

- 基础监控

### P1 - 重要功能（尽快实现）

- 冲突解决机制

- 资源动态分配

- 告警系统

- 多交易所支持

- 性能报告

### P2 - 增强功能（后续迭代）

- 配置热加载

- 可视化仪表板

- gRPC 通信

- 压力测试

- 高级分析

## 开发建议

### 1. 开发顺序

```plaintext
阶段 1: 核心框架（1-2 周）
  ├─ Manager 结构 + 配置管理
  ├─ VirtualTrader 抽象
  ├─ Mock 交易所适配器
  └─ 基础测试

阶段 2: 交易循环（1 周）
  ├─ Orchestrator 实现
  ├─ 与 Executor 集成
  └─ 决策执行流程

阶段 3: 资源管理（1 周）
  ├─ ResourceAllocator
  ├─ ConflictResolver
  └─ 多 Trader 协调

阶段 4: 监控告警（1 周）
  ├─ Monitor 实现
  ├─ Prometheus 集成
  └─ Webhook 告警

阶段 5: 生产就绪（1-2 周）
  ├─ 真实交易所适配器
  ├─ 状态持久化
  ├─ 错误处理完善
  └─ 集成测试
```



### 2. 代码规范

- 所有公开函数必须有详细注释

- 错误处理不能忽略

- 使用 context 控制生命周期

- 线程安全（使用 sync.RWMutex）

- 日志分级（Debug/Info/Warn/Error）

### 3. 测试要求

- 单元测试覆盖率 ≥ 80%

- 每个核心函数必须有测试

- 使用 Mock 隔离外部依赖

- 集成测试覆盖主要场景

### 4. 文档要求

- [README.md](http://README.md)（模块概述）

- [API.md](http://API.md)（接口文档）

- [CONFIG.md](http://CONFIG.md)（配置说明）

- [EXAMPLES.md](http://EXAMPLES.md)（使用示例）

## 参考资料

### 现有代码（仅供参考，不直接复用）

- `auto_trader.go` - 交易循环逻辑

- `trader_manager.go` - Trader 管理思路

- `hyperliquid/trader.go` - 交易所集成方式

- `decision/` - 决策数据结构

### 设计文档

- [nof1.ai](http://nof1.ai) 决策引擎产品文档（本文档）

- Executor 模块 TODO List（待创建）

### 技术选型

- Go 1.21+

- YAML 配置（[github.com/spf13/viper）](http://github.com/spf13/viper%EF%BC%89)

- Prometheus 监控（[github.com/prometheus/client_golang）](http://github.com/prometheus/client_golang%EF%BC%89)

- 日志库（[github.com/sirupsen/logrus）](http://github.com/sirupsen/logrus%EF%BC%89)
