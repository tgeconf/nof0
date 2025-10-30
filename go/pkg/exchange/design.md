
# Hyperliquid 交易与账户接口对接 TODO List (全新实现)

> **目标**: 使用大语言模型 + 浏览器 MCP 在 `exchange/` 模块下完成 Hyperliquid 交易与账户相关接口的全新实现
>
> **实现方式**: 干净方案，不涉及迁移，从零开始构建
>
> **参考文档**: [Hyperliquid API Docs - Exchange Endpoint](https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/exchange-endpoint)

---

## 📋 阶段一: 项目结构搭建

### ✅ Task 1.1: 创建基础目录结构

```plaintext
exchange/
├── hyperliquid/
│   ├── client.go          # HTTP 客户端封装
│   ├── types.go           # 数据结构定义
│   ├── auth.go            # 签名与认证逻辑
│   ├── order.go           # 订单管理
│   ├── position.go        # 仓位管理
│   ├── account.go         # 账户信息
│   ├── websocket.go       # WebSocket 连接 (暂不需要)
│   ├── utils.go           # 工具函数
│   └── client_test.go     # 单元测试
├── interface.go           # 交易接口定义
└── README.md              # 模块说明文档
```

**验收标准**:

- 目录结构清晰，职责分明
- 每个文件都有明确的功能定位
- 包含测试文件

---

## 📋 阶段二: 数据结构定义

### ✅ Task 2.1: 定义核心数据结构 (`types.go`)

```go
// OrderSide 订单方向
type OrderSide string

const (
    OrderSideBuy  OrderSide = "A" // Ask (买入)
    OrderSideSell OrderSide = "B" // Bid (卖出)
)

// OrderType 订单类型
type OrderType struct {
    Limit *LimitOrderType `json:"limit,omitempty"`
}

type LimitOrderType struct {
    Tif string `json:"tif"` // "Alo", "Ioc", "Gtc"
}

// Order 订单结构
type Order struct {
    Asset      int     `json:"asset"`      // 资产索引
    IsBuy      bool    `json:"isBuy"`      // 是否买入
    LimitPx    string  `json:"limitPx"`    // 限价
    Sz         string  `json:"sz"`         // 数量
    ReduceOnly bool    `json:"reduceOnly"` // 只减仓
    OrderType  OrderType `json:"orderType"`
    Cloid      string  `json:"cloid,omitempty"` // 客户端订单ID
}

// Position 仓位信息
type Position struct {
    Coin          string  `json:"coin"`
    EntryPx       string  `json:"entryPx"`       // 入场价格
    PositionValue string  `json:"positionValue"` // 仓位价值
    Szi           string  `json:"szi"`           // 仓位大小(带符号)
    UnrealizedPnl string  `json:"unrealizedPnl"` // 未实现盈亏
    ReturnOnEquity string `json:"returnOnEquity"` // ROE
    Leverage      Leverage `json:"leverage"`
    LiquidationPx string  `json:"liquidationPx,omitempty"` // 清算价格
}

type Leverage struct {
    Type  string `json:"type"`  // "cross" or "isolated"
    Value int    `json:"value"` // 杠杆倍数
}

// AccountState 账户状态
type AccountState struct {
    MarginSummary MarginSummary `json:"marginSummary"`
    CrossMarginSummary CrossMarginSummary `json:"crossMarginSummary"`
    AssetPositions []Position `json:"assetPositions"`
}

type MarginSummary struct {
    AccountValue      string `json:"accountValue"`      // 账户价值
    TotalMarginUsed   string `json:"totalMarginUsed"`   // 已用保证金
    TotalNtlPos       string `json:"totalNtlPos"`       // 总名义持仓
    TotalRawUsd       string `json:"totalRawUsd"`       // 总USD价值
}

type CrossMarginSummary struct {
    AccountValue    string `json:"accountValue"`
    TotalMarginUsed string `json:"totalMarginUsed"`
    TotalNtlPos     string `json:"totalNtlPos"`
    TotalRawUsd     string `json:"totalRawUsd"`
}

// OrderStatus 订单状态
type OrderStatus struct {
    Order    OrderInfo `json:"order"`
    Status   string    `json:"status"` // "open", "filled", "canceled", etc.
    StatusTimestamp int64 `json:"statusTimestamp"`
}

type OrderInfo struct {
    Coin       string `json:"coin"`
    Side       string `json:"side"`
    LimitPx    string `json:"limitPx"`
    Sz         string `json:"sz"`
    Oid        int64  `json:"oid"`        // 订单ID
    Timestamp  int64  `json:"timestamp"`
    OrigSz     string `json:"origSz"`     // 原始数量
    Cloid      string `json:"cloid,omitempty"`
}

// Fill 成交记录
type Fill struct {
    Coin      string `json:"coin"`
    Px        string `json:"px"`        // 成交价格
    Sz        string `json:"sz"`        // 成交数量
    Side      string `json:"side"`
    Time      int64  `json:"time"`
    StartPosition string `json:"startPosition"`
    Dir       string `json:"dir"`       // "Open Long", "Close Long", etc.
    ClosedPnl string `json:"closedPnl"`
    Hash      string `json:"hash"`
    Oid       int64  `json:"oid"`
    Crossed   bool   `json:"crossed"`
    Fee       string `json:"fee"`
    Tid       int64  `json:"tid"`       // 成交ID
}
```

**验收标准**:

- 所有字段都有清晰的注释
- 数据类型与 Hyperliquid API 文档一致
- JSON 标签正确

### ✅ Task 2.2: 定义 API 请求/响应结构

```go
// ExchangeRequest 交易请求基础结构
type ExchangeRequest struct {
    Action      Action      `json:"action"`
    Nonce       int64       `json:"nonce"`
    Signature   Signature   `json:"signature"`
    VaultAddress string     `json:"vaultAddress,omitempty"`
}

type Action struct {
    Type   string      `json:"type"`
    Orders []Order     `json:"orders,omitempty"`
    Cancels []Cancel   `json:"cancels,omitempty"`
    // ... 其他 action 类型
}

type Cancel struct {
    Asset int   `json:"asset"` // 资产索引
    Oid   int64 `json:"oid"`   // 订单ID
}

type Signature struct {
    R string `json:"r"`
    S string `json:"s"`
    V int    `json:"v"`
}

// InfoRequest Info API 请求
type InfoRequest struct {
    Type string      `json:"type"`
    User string      `json:"user,omitempty"`
}

// OrderResponse 下单响应
type OrderResponse struct {
    Status   string         `json:"status"` // "ok" or "err"
    Response OrderResponseData `json:"response"`
}

type OrderResponseData struct {
    Type string                 `json:"type"` // "order"
    Data OrderResponseDataDetail `json:"data"`
}

type OrderResponseDataDetail struct {
    Statuses []OrderStatusResponse `json:"statuses"`
}

type OrderStatusResponse struct {
    Resting *RestingOrder `json:"resting,omitempty"`
    Filled  *FilledOrder  `json:"filled,omitempty"`
    Error   string        `json:"error,omitempty"`
}

type RestingOrder struct {
    Oid int64 `json:"oid"`
}

type FilledOrder struct {
    TotalSz  string `json:"totalSz"`
    AvgPx    string `json:"avgPx"`
    Oid      int64  `json:"oid"`
}
```

**验收标准**:

- 结构与 Hyperliquid API 文档一致
- 支持所有需要的 Exchange 端点
- 包含完整的错误响应结构

---

## 📋 阶段三: 认证与签名实现

### ✅ Task 3.1: 实现签名逻辑 (`auth.go`)

```go
// Signer 签名器接口
type Signer interface {
    Sign(message []byte) (*Signature, error)
    GetAddress() string
}

// PrivateKeySigner 私钥签名器
type PrivateKeySigner struct {
    privateKey *ecdsa.PrivateKey
    address    string
}

// NewPrivateKeySigner 从私钥字符串创建签名器
func NewPrivateKeySigner(privateKeyHex string) (*PrivateKeySigner, error)

// Sign 对消息进行签名
func (s *PrivateKeySigner) Sign(message []byte) (*Signature, error)

// GetAddress 获取钱包地址
func (s *PrivateKeySigner) GetAddress() string

// signAction 对 Action 进行签名
func signAction(action Action, signer Signer, nonce int64, vaultAddress string) (*ExchangeRequest, error)
```

**实现要点**:

- 使用 EIP-712 签名标准
- 支持 secp256k1 椭圆曲线
- 正确构造签名消息的哈希
- Phantom agent 签名支持（可选）

**签名流程**:

1. 构造 Action 对象
2. 生成 nonce（当前时间戳毫秒）
3. 构造 EIP-712 结构化数据
4. 计算 Keccak256 哈希
5. 使用私钥签名
6. 返回 r, s, v 签名组件

**验收标准**:

- 签名格式符合 Hyperliquid 要求
- 能通过 API 验证
- 支持主账户和 Vault 账户

### ✅ Task 3.2: 实现 EIP-712 消息构造

```go
// buildEIP712Message 构造 EIP-712 消息
func buildEIP712Message(action Action, nonce int64, vaultAddress string) ([]byte, error)

// EIP712Domain EIP-712 域定义
type EIP712Domain struct {
    Name              string
    Version           string
    ChainId           int
    VerifyingContract string
}

// getActionHash 计算 Action 的哈希
func getActionHash(action Action) ([]byte, error)
```

**实现要点**:

- Domain: name="Exchange", version="1", chainId=1337（mainnet）或 1338（testnet）
- 正确编码不同类型的 Action
- 使用 Keccak256 哈希算法

**验收标准**:

- 消息哈希与官方示例一致
- 支持所有 Action 类型
- 正确处理可选字段

---

## 📋 阶段四: HTTP 客户端实现

### ✅ Task 4.1: 实现基础 HTTP 客户端 (`client.go`)

```go
type Client struct {
    baseURL        string
    exchangeURL    string
    httpClient     *http.Client
    signer         Signer
    isTestnet      bool
}

// NewClient 创建新的 Hyperliquid 客户端
func NewClient(privateKeyHex string, isTestnet bool) (*Client, error)

// doInfoRequest 执行 Info API 请求 (无需签名)
func (c *Client) doInfoRequest(ctx context.Context, req InfoRequest, result interface{}) error

// doExchangeRequest 执行 Exchange API 请求 (需要签名)
func (c *Client) doExchangeRequest(ctx context.Context, action Action, result interface{}) error
```

**实现要点**:

- Info URL: `https://api.hyperliquid.xyz/info`（mainnet）或 `https://api.hyperliquid-testnet.xyz/info`（testnet）
- Exchange URL: `https://api.hyperliquid.xyz/exchange`（mainnet）或 `https://api.hyperliquid-testnet.xyz/exchange`（testnet）
- 使用 POST 方法
- 设置合理的超时时间（30秒，交易操作可能较慢）
- 添加重试机制（最多3次，仅针对网络错误）
- 错误处理和日志记录
- **不要对失败的交易请求重试**（避免重复下单）

**验收标准**:

- 能成功发送请求到 Hyperliquid API
- 正确处理 HTTP 错误
- 支持 context 取消
- 签名请求正确

---

## 📋 阶段五: 订单管理实现

### ✅ Task 5.1: 实现下单功能 (`order.go`)

```go
// PlaceOrder 下单
func (c *Client) PlaceOrder(ctx context.Context, order Order) (*OrderResponse, error)

// PlaceOrders 批量下单
func (c *Client) PlaceOrders(ctx context.Context, orders []Order) (*OrderResponse, error)

// buildPlaceOrderAction 构造下单 Action
func buildPlaceOrderAction(orders []Order) Action
```

**实现要点**:

- 支持限价单（Limit Order）
- 支持不同的 Time-in-Force: Alo（Add Liquidity Only）、Ioc（Immediate or Cancel）、Gtc（Good Till Cancel）
- 支持只减仓订单（ReduceOnly）
- 支持客户端订单ID（Cloid）
- 价格和数量需要转换为字符串，保留适当精度

**订单参数验证**:

- 价格 > 0
- 数量 > 0
- Asset 索引有效
- Cloid 长度 <= 128

**验收标准**:

- 能成功下单
- 返回订单ID或成交信息
- 错误信息清晰
- 支持批量下单

### ✅ Task 5.2: 实现撤单功能

```go
// CancelOrder 撤销单个订单
func (c *Client) CancelOrder(ctx context.Context, asset int, oid int64) error

// CancelOrders 批量撤单
func (c *Client) CancelOrders(ctx context.Context, cancels []Cancel) error

// CancelAllOrders 撤销所有订单
func (c *Client) CancelAllOrders(ctx context.Context, asset int) error

// buildCancelAction 构造撤单 Action
func buildCancelAction(cancels []Cancel) Action
```

**实现要点**:

- 支持按订单ID撤单
- 支持按资产撤销所有订单
- 批量撤单提高效率

**验收标准**:

- 能成功撤单
- 处理订单不存在的情况
- 支持批量操作

### ✅ Task 5.3: 实现订单查询功能

```go
// GetOpenOrders 获取未完成订单
func (c *Client) GetOpenOrders(ctx context.Context) ([]OrderStatus, error)

// GetOrderStatus 获取订单状态
func (c *Client) GetOrderStatus(ctx context.Context, oid int64) (*OrderStatus, error)

// GetOrderHistory 获取历史订单
func (c *Client) GetOrderHistory(ctx context.Context) ([]OrderStatus, error)
```

**API 调用示例**:

```json
POST https://api.hyperliquid.xyz/info
{
  "type": "openOrders",
  "user": "0x..."
}
```

**验收标准**:

- 能获取未完成订单列表
- 能查询单个订单状态
- 能获取历史订单

### ✅ Task 5.4: 实现成交记录查询

```go
// GetUserFills 获取成交记录
func (c *Client) GetUserFills(ctx context.Context) ([]Fill, error)

// GetUserFillsByTime 按时间范围获取成交记录
func (c *Client) GetUserFillsByTime(ctx context.Context, startTime, endTime int64) ([]Fill, error)
```

**API 调用示例**:

```json
POST https://api.hyperliquid.xyz/info
{
  "type": "userFills",
  "user": "0x..."
}
```

**验收标准**:

- 能获取成交记录
- 支持时间过滤
- 数据格式正确

---

## 📋 阶段六: 仓位管理实现

### ✅ Task 6.1: 实现仓位查询 (`position.go`)

```go
// GetPositions 获取所有仓位
func (c *Client) GetPositions(ctx context.Context) ([]Position, error)

// GetPosition 获取指定币种的仓位
func (c *Client) GetPosition(ctx context.Context, coin string) (*Position, error)

// HasPosition 检查是否有仓位
func (c *Client) HasPosition(ctx context.Context, coin string) (bool, error)
```

**API 调用示例**:

```json
POST https://api.hyperliquid.xyz/info
{
  "type": "clearinghouseState",
  "user": "0x..."
}
```

**验收标准**:

- 能获取所有仓位
- 能查询单个币种仓位
- 正确解析仓位方向（多/空）
- 正确计算未实现盈亏

### ✅ Task 6.2: 实现杠杆调整

```go
// UpdateLeverage 调整杠杆倍数
func (c *Client) UpdateLeverage(ctx context.Context, asset int, isCross bool, leverage int) error

// buildUpdateLeverageAction 构造调整杠杆 Action
func buildUpdateLeverageAction(asset int, isCross bool, leverage int) Action
```

**实现要点**:

- 支持全仓（Cross）和逐仓（Isolated）
- 杠杆倍数范围: 1-50（具体取决于币种）
- 有仓位时可能无法调整

**验收标准**:

- 能成功调整杠杆
- 错误提示清晰
- 验证杠杆倍数范围

### ✅ Task 6.3: 实现平仓功能

```go
// ClosePosition 平仓
func (c *Client) ClosePosition(ctx context.Context, coin string) error

// ClosePositionPartial 部分平仓
func (c *Client) ClosePositionPartial(ctx context.Context, coin string, size float64) error
```

**实现要点**:

- 获取当前仓位信息
- 根据仓位方向下反向订单
- 使用市价单或限价单
- 设置 ReduceOnly 标志

**验收标准**:

- 能成功平仓
- 支持部分平仓
- 处理无仓位情况

---

## 📋 阶段七: 账户信息实现

### ✅ Task 7.1: 实现账户查询 (`account.go`)

```go
// GetAccountState 获取账户状态
func (c *Client) GetAccountState(ctx context.Context) (*AccountState, error)

// GetAccountValue 获取账户价值
func (c *Client) GetAccountValue(ctx context.Context) (float64, error)

// GetAvailableBalance 获取可用余额
func (c *Client) GetAvailableBalance(ctx context.Context) (float64, error)

// GetMarginUsage 获取保证金使用率
func (c *Client) GetMarginUsage(ctx context.Context) (float64, error)
```

**API 调用示例**:

```json
POST https://api.hyperliquid.xyz/info
{
  "type": "clearinghouseState",
  "user": "0x..."
}
```

**验收标准**:

- 能获取完整账户状态
- 正确计算可用余额
- 正确计算保证金使用率

### ✅ Task 7.2: 实现资产转账（可选）

```go
// Withdraw 提现到 L1
func (c *Client) Withdraw(ctx context.Context, amount float64, destination string) error

// buildWithdrawAction 构造提现 Action
func buildWithdrawAction(amount float64, destination string) Action
```

**实现要点**:

- 提现到以太坊 L1
- 需要支付 gas 费
- 有最小提现金额限制

**验收标准**:

- 能成功发起提现
- 验证金额和地址
- 错误处理完善

---

## 📋 阶段八: 工具函数实现

### ✅ Task 8.1: 实现币种信息查询 (`utils.go`)

```go
// GetAssetIndex 获取币种的资产索引
func (c *Client) GetAssetIndex(ctx context.Context, coin string) (int, error)

// GetAssetInfo 获取资产信息
func (c *Client) GetAssetInfo(ctx context.Context, coin string) (*AssetInfo, error)

type AssetInfo struct {
    Name          string
    SzDecimals    int     // 数量精度
    MaxLeverage   int     // 最大杠杆
    OnlyIsolated  bool    // 是否仅支持逐仓
}
```

**API 调用示例**:

```json
POST https://api.hyperliquid.xyz/info
{
  "type": "meta"
}
```

**验收标准**:

- 能获取币种索引
- 能获取币种详细信息
- 缓存币种信息（避免重复请求）

### ✅ Task 8.2: 实现价格和数量格式化

```go
// FormatPrice 格式化价格
func FormatPrice(price float64, coin string) string

// FormatSize 格式化数量
func FormatSize(size float64, coin string) string

// ParsePrice 解析价格字符串
func ParsePrice(priceStr string) (float64, error)

// ParseSize 解析数量字符串
func ParseSize(sizeStr string) (float64, error)
```

**实现要点**:

- 根据币种精度格式化
- 避免精度丢失
- 使用 decimal 库处理浮点数

**验收标准**:

- 格式化结果符合 API 要求
- 解析正确
- 处理边界情况

### ✅ Task 8.3: 实现订单辅助函数

```go
// CreateLimitOrder 创建限价单
func CreateLimitOrder(coin string, side OrderSide, price, size float64, reduceOnly bool) (*Order, error)

// CreateMarketOrder 创建市价单 (使用限价单模拟)
func CreateMarketOrder(coin string, side OrderSide, size float64) (*Order, error)

// ValidateOrder 验证订单参数
func ValidateOrder(order *Order) error
```

**验收标准**:

- 简化订单创建流程
- 参数验证完善
- 易于使用

---

## 📋 阶段九: WebSocket 实现（可选）

### ✅ Task 9.1: 实现 WebSocket 连接 (`websocket.go`)

```go
// WSClient WebSocket 客户端
type WSClient struct {
    conn      *websocket.Conn
    url       string
    isTestnet bool
    handlers  map[string]WSHandler
}

type WSHandler func(data interface{})

// NewWSClient 创建 WebSocket 客户端
func NewWSClient(isTestnet bool) *WSClient

// Connect 连接到 WebSocket
func (ws *WSClient) Connect(ctx context.Context) error

// Subscribe 订阅频道
func (ws *WSClient) Subscribe(channel string, handler WSHandler) error

// SubscribeUserEvents 订阅用户事件
func (ws *WSClient) SubscribeUserEvents(user string) error
```

**支持的订阅类型**:

- 用户事件（订单更新、成交、仓位变化）
- 订单簿更新
- 最新成交

**验收标准**:

- 能建立 WebSocket 连接
- 能订阅和接收消息
- 自动重连机制
- 错误处理完善

---

## 📋 阶段十: 接口适配

### ✅ Task 10.1: 实现交易接口 (`interface.go`)

```go
// ExchangeProvider 交易所接口
type ExchangeProvider interface {
    // 订单管理
    PlaceOrder(ctx context.Context, order Order) (*OrderResponse, error)
    CancelOrder(ctx context.Context, asset int, oid int64) error
    GetOpenOrders(ctx context.Context) ([]OrderStatus, error)
    
    // 仓位管理
    GetPositions(ctx context.Context) ([]Position, error)
    ClosePosition(ctx context.Context, coin string) error
    UpdateLeverage(ctx context.Context, asset int, isCross bool, leverage int) error
    
    // 账户信息
    GetAccountState(ctx context.Context) (*AccountState, error)
    GetAccountValue(ctx context.Context) (float64, error)
    
    // 工具方法
    GetAssetIndex(ctx context.Context, coin string) (int, error)
}

// HyperliquidProvider Hyperliquid 实现
type HyperliquidProvider struct {
    client *hyperliquid.Client
}

func NewHyperliquidProvider(privateKeyHex string, isTestnet bool) (ExchangeProvider, error)
```

**验收标准**:

- 接口定义清晰
- 易于扩展其他交易所
- 支持依赖注入

---

## 📋 阶段十一: 测试与验证

### ✅ Task 11.1: 单元测试

为每个核心函数编写单元测试:

- `TestSign`

- `TestBuildEIP712Message`

- `TestPlaceOrder`

- `TestCancelOrder`

- `TestGetPositions`

- `TestGetAccountState`

- `TestFormatPrice`

- `TestValidateOrder`

**测试策略**:

- 使用 mock 避免真实 API 调用
- 测试边界情况
- 测试错误处理

**验收标准**:

- 测试覆盖率 > 80%
- 所有测试通过
- 包含边界情况测试

### ✅ Task 11.2: 集成测试（使用测试网）

```go
func TestRealTrading(t *testing.T) {
    // 使用测试网进行真实交易测试
    client, _ := NewClient(testPrivateKey, true)
    
    // 测试下单
    // 测试撤单
    // 测试查询
}
```

**测试清单**:

- [ ] 连接测试网
- [ ] 获取账户信息
- [ ] 下限价单
- [ ] 查询订单状态
- [ ] 撤销订单
- [ ] 获取仓位
- [ ] 调整杠杆
- [ ] 平仓

**验收标准**:

- 能在测试网完成完整交易流程
- 所有功能正常工作
- 错误处理正确

### ✅ Task 11.3: 性能测试

```go
func BenchmarkPlaceOrder(b *testing.B) {
    // 性能基准测试
}
```

**验收标准**:

- 下单延迟 < 1秒
- 查询延迟 < 500ms
- 内存使用合理
- 无内存泄漏

---

## 📋 阶段十二: 文档与部署

### ✅ Task 12.1: 编写 README

包含以下内容:

- 模块功能说明
- 快速开始指南
- API 使用示例
- 配置说明
- 安全注意事项
- 常见问题

### ✅ Task 12.2: 代码注释

确保所有公开函数都有:

- 功能说明
- 参数说明
- 返回值说明
- 使用示例
- 注意事项

### ✅ Task 12.3: 安全配置

```go
type Config struct {
    PrivateKey    string        // 从环境变量读取
    IsTestnet     bool
    Timeout       time.Duration
    MaxRetries    int
    EnableWS      bool
}

// LoadConfig 从环境变量加载配置
func LoadConfig() (*Config, error)
```

**安全要点**:

- 私钥不要硬编码
- 使用环境变量或密钥管理服务
- 日志中不要输出敏感信息
- 测试网和主网严格区分

**验收标准**:

- 支持环境变量配置
- 有合理的默认值
- 配置验证完善
- 安全性高

---

## 🎯 关键技术要点

### 1. **签名安全**

- 使用 EIP-712 标准签名
- 私钥安全存储
- Nonce 使用当前时间戳毫秒
- 签名验证失败时不要重试

### 2. **订单管理**

- 下单失败不要自动重试（避免重复下单）
- 使用 Cloid 追踪订单
- 批量操作提高效率
- 注意价格和数量精度

### 3. **错误处理**

- 区分网络错误和业务错误
- 网络错误可以重试，业务错误不要重试
- 错误信息要清晰
- 记录详细日志

### 4. **性能优化**

- 复用 HTTP 连接
- 批量操作减少请求次数
- 缓存币种信息
- 使用 WebSocket 接收实时更新

### 5. **风险控制**

- 验证订单参数
- 检查账户余额
- 监控保证金使用率
- 设置止损止盈

---

## 📚 参考资源

1. **Hyperliquid API 文档**:
   - Exchange endpoint: https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/exchange-endpoint
   - Info endpoint: https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/info-endpoint
   - Signing: https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/signing

2. **以太坊相关**:
   - EIP-712: https://eips.ethereum.org/EIPS/eip-712
   - go-ethereum: https://github.com/ethereum/go-ethereum

3. **Go 技术栈**:
   - net/http 标准库
   - crypto/ecdsa
   - encoding/json
   - gorilla/websocket

---

## ✅ 验收清单

完成以下所有项即可认为任务完成:

- [ ] 所有代码文件创建完成
- [ ] 签名功能实现并验证通过
- [ ] 订单管理功能完整（下单、撤单、查询）
- [ ] 仓位管理功能完整（查询、平仓、调整杠杆）
- [ ] 账户信息查询功能完整
- [ ] 单元测试通过
- [ ] 测试网集成测试通过
- [ ] 性能测试达标
- [ ] 代码注释完整
- [ ] README 文档完成
- [ ] 安全配置完善
- [ ] 错误处理完善
- [ ] 代码符合 Go 最佳实践

---

## 🚀 快速开始示例

完成后的使用示例:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "yourproject/exchange"
    "yourproject/exchange/hyperliquid"
)

func main() {
    // 创建 Hyperliquid 客户端 (使用测试网)
    provider, err := exchange.NewHyperliquidProvider(
        "your_private_key_hex",
        true, // isTestnet
    )
    if err != nil {
        log.Fatal(err)
    }
    
    ctx := context.Background()
    
    // 获取账户信息
    accountState, err := provider.GetAccountState(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Account Value: %s
", accountState.MarginSummary.AccountValue)
    
    // 获取资产索引
    assetIndex, err := provider.GetAssetIndex(ctx, "BTC")
    if err != nil {
        log.Fatal(err)
    }
    
    // 创建限价买单
    order := hyperliquid.Order{
        Asset:      assetIndex,
        IsBuy:      true,
        LimitPx:    "50000.0",
        Sz:         "0.001",
        ReduceOnly: false,
        OrderType: hyperliquid.OrderType{
            Limit: &hyperliquid.LimitOrderType{
                Tif: "Gtc",
            },
        },
    }
    
    // 下单
    response, err := provider.PlaceOrder(ctx, order)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Order placed: %+v
", response)
    
    // 查询未完成订单
    openOrders, err := provider.GetOpenOrders(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Open orders: %d
", len(openOrders))
    
    // 查询仓位
    positions, err := provider.GetPositions(ctx)
    if err != nil {
        log.Fatal(err)
    }
    for _, pos := range positions {
        fmt.Printf("Position: %s, Size: %s, PnL: %s
",
            pos.Coin, pos.Szi, pos.UnrealizedPnl)
    }
}
```

---

## 💡 给大语言模型的提示

在实现过程中，请注意:

1. **安全第一**:
   - 签名实现必须严格按照 EIP-712 标准
   - 私钥处理要格外小心
   - 先在测试网验证，再考虑主网

2. **优先级排序**:
   - 先实现签名和认证（最关键）
   - 再实现基础订单功能（下单、撤单、查询）
   - 然后实现仓位和账户查询
   - 最后实现高级功能（WebSocket、批量操作）

3. **测试驱动**:
   - 每完成一个功能就测试
   - 使用测试网进行真实测试
   - 不要在主网测试

4. **错误处理**:
   - 交易相关的错误不要自动重试
   - 网络错误可以重试
   - 错误信息要详细，方便调试

5. **代码质量**:
   - 使用清晰的变量名和函数名
   - 添加详细注释
   - 遵循 Go 最佳实践
   - 注意并发安全

6. **参考文档**:
   - 严格按照 Hyperliquid API 文档实现
   - 有疑问时查看官方示例
   - 可以参考社区的 SDK 实现

7. **性能考虑**:
   - 批量操作减少请求次数
   - 缓存不变的数据（如币种信息）
   - 使用连接池
   - 考虑使用 WebSocket 接收实时更新

祝实现顺利! 🎉 记住:安全和正确性比速度更重要!