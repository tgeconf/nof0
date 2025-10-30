# Hyperliquid Market Data API Implementation

> **目标**: 使用大语言模型 + 浏览器 MCP 在 `market/` 模块下完成 Hyperliquid 市场数据的全新实现
> **实现方式**: 干净方案,不涉及迁移,从零开始构建
> **参考文档**: [Hyperliquid API Docs](https://hyperliquid.gitbook.io/hyperliquid-docs)

---

## 📋 阶段一: 项目结构搭建

### ✅ Task 1.1: 创建基础目录结构

```plaintext
market/
├── hyperliquid/
│   ├── client.go          # HTTP 客户端封装
│   ├── types.go           # 数据结构定义
│   ├── kline.go           # K线数据获取
│   ├── market_info.go     # 市场信息获取
│   ├── indicators.go      # 技术指标计算
│   ├── data.go            # 主数据聚合逻辑
│   └── client_test.go     # 单元测试
├── interface.go           # 市场数据接口定义
└── README.md              # 模块说明文档
```

**验收标准**:
- 目录结构清晰,职责分明
- 每个文件都有明确的功能定位
- 包含测试文件

---

## 📋 阶段二: 数据结构定义

### ✅ Task 2.1: 定义核心数据结构 (`types.go`)

参考定义以下类型:

```go
// Data - 市场数据主结构
type Data struct {
    Symbol            string           // 交易对符号 (如 "BTC")
    CurrentPrice      float64          // 当前价格
    PriceChange1h     float64          // 1小时价格变化百分比
    PriceChange4h     float64          // 4小时价格变化百分比
    CurrentEMA20      float64          // 当前20期EMA
    CurrentMACD       float64          // 当前MACD值
    CurrentRSI7       float64          // 当前7期RSI
    OpenInterest      *OIData          // 持仓量数据
    FundingRate       float64          // 资金费率
    IntradaySeries    *IntradayData    // 日内序列数据(3分钟)
    LongerTermContext *LongerTermData  // 长期数据(4小时)
}

// OIData - 持仓量数据
type OIData struct {
    Latest  float64  // 最新持仓量
    Average float64  // 平均持仓量
}

// IntradayData - 日内序列数据 (基于3分钟K线)
type IntradayData struct {
    MidPrices    []float64  // 收盘价序列 (最近10个)
    EMA20Values  []float64  // 20期EMA序列
    MACDValues   []float64  // MACD序列
    RSI7Values   []float64  // 7期RSI序列
    RSI14Values  []float64  // 14期RSI序列
}

// LongerTermData - 长期数据 (基于4小时K线)
type LongerTermData struct {
    EMA20         float64    // 20期EMA
    EMA50         float64    // 50期EMA
    ATR3          float64    // 3期ATR
    ATR14         float64    // 14期ATR
    CurrentVolume float64    // 当前成交量
    AverageVolume float64    // 平均成交量
    MACDValues    []float64  // MACD序列(最近10个)
    RSI14Values   []float64  // 14期RSI序列(最近10个)
}

// Kline - K线原始数据
type Kline struct {
    OpenTime  int64    // 开盘时间(毫秒时间戳)
    Open      float64  // 开盘价
    High      float64  // 最高价
    Low       float64  // 最低价
    Close     float64  // 收盘价
    Volume    float64  // 成交量
    CloseTime int64    // 收盘时间(毫秒时间戳)
}
```

**验收标准**:
- 所有字段都有清晰的注释
- 数据类型正确

### ✅ Task 2.2: 定义 API 请求/响应结构

根据 Hyperliquid API 文档定义:

```go
// API 请求结构
type InfoRequest struct {
    Type string      `json:"type"`
    Req  interface{} `json:"req,omitempty"`
}

// K线请求参数
type CandleSnapshotRequest struct {
    Coin      string `json:"coin"`
    Interval  string `json:"interval"` // "3m", "4h"
    StartTime int64  `json:"startTime"`
    EndTime   int64  `json:"endTime"`
}

// K线响应
type CandleResponse []struct {
    T int64   `json:"t"` // timestamp
    O float64 `json:"o"` // open
    H float64 `json:"h"` // high
    L float64 `json:"l"` // low
    C float64 `json:"c"` // close
    V float64 `json:"v"` // volume
}

// 市场元数据响应
type MetaAndAssetCtxsResponse []struct {
    Universe []struct {
        Name string `json:"name"` // 币种名称
    } `json:"universe"`
    AssetCtxs []struct {
        Coin         string  `json:"coin"`
        MarkPx       string  `json:"markPx"`
        MidPx        string  `json:"midPx,omitempty"`
        Funding      string  `json:"funding"`
        OpenInterest string  `json:"openInterest"`
        DayNtlVlm    string  `json:"dayNtlVlm"` // 日交易量
    } `json:"assetCtxs"`
}

// 实时价格响应
type AllMidsResponse map[string]string // {"BTC": "111317.5", ...}
```

**验收标准**:
- 结构与 Hyperliquid API 文档一致
- JSON 标签正确
- 支持所有需要的 API 端点

---

## 📋 阶段三: HTTP 客户端实现

### ✅ Task 3.1: 实现基础 HTTP 客户端 (`client.go`)

```go
type Client struct {
    baseURL    string
    httpClient *http.Client
}

// NewClient 创建新的 Hyperliquid 客户端
func NewClient() *Client

// doRequest 执行 HTTP POST 请求
func (c *Client) doRequest(ctx context.Context, req InfoRequest, result interface{}) error
```

**实现要点**:
- 基础 URL: `https://api.hyperliquid.xyz/info`
- 使用 POST 方法
- 设置合理的超时时间 (10秒)
- 添加重试机制 (最多3次)
- 错误处理和日志记录

**验收标准**:
- 能成功发送请求到 Hyperliquid API
- 正确处理 HTTP 错误
- 支持 context 取消

### ✅ Task 3.2: 实现 K线数据获取 (`kline.go`)

```go
// GetKlines 获取K线数据
func (c *Client) GetKlines(ctx context.Context, symbol string, interval string, limit int) ([]Kline, error)
```

**实现要点**:
- 支持的时间间隔: "3m", "4h"
- 自动计算 startTime 和 endTime
- 将 API 响应转换为标准 Kline 结构
- 按时间从旧到新排序

**API 调用示例**:

```json
POST https://api.hyperliquid.xyz/info
{
  "type": "candleSnapshot",
  "req": {
    "coin": "BTC",
    "interval": "3m",
    "startTime": 1234567890000,
    "endTime": 1234567900000
  }
}
```

**验收标准**:
- 能获取指定数量的 K线数据
- 数据格式正确
- 时间戳转换准确

### ✅ Task 3.3: 实现市场信息获取 (`market_info.go`)

```go
// GetCurrentPrice 获取当前价格
func (c *Client) GetCurrentPrice(ctx context.Context, symbol string) (float64, error)

// GetMarketInfo 获取市场信息(持仓量、资金费率等)
func (c *Client) GetMarketInfo(ctx context.Context, symbol string) (*MarketInfo, error)

type MarketInfo struct {
    MarkPrice    float64
    MidPrice     float64
    FundingRate  float64
    OpenInterest float64
    DayVolume    float64
}
```

**API 调用示例**:

```json
// 获取所有价格
POST https://api.hyperliquid.xyz/info
{"type": "allMids"}

// 获取市场元数据
POST https://api.hyperliquid.xyz/info
{"type": "metaAndAssetCtxs"}
```

**验收标准**:
- 能获取实时价格
- 能获取持仓量和资金费率
- 字符串到浮点数转换正确

---

## 📋 阶段四: 技术指标计算

### ✅ Task 4.1: 实现 EMA 计算 (`indicators.go`)

```go
// CalculateEMA 计算指数移动平均线
func CalculateEMA(prices []float64, period int) []float64
```

**算法**:
- 第一个值使用 SMA (简单移动平均)
- 后续值: EMA = (Close - EMA_prev) * multiplier + EMA_prev
- multiplier = 2 / (period + 1)

**验收标准**:
- 计算结果准确
- 处理边界情况 (数据不足)
- 性能优化

### ✅ Task 4.2: 实现 MACD 计算

```go
// CalculateMACD 计算 MACD
func CalculateMACD(prices []float64) (macd []float64, signal []float64, histogram []float64)
```

**算法**:
- MACD = EMA(12) - EMA(26)
- Signal = EMA(MACD, 9)
- Histogram = MACD - Signal

**验收标准**:
- 返回完整的 MACD、Signal、Histogram
- 数据长度正确

### ✅ Task 4.3: 实现 RSI 计算

```go
// CalculateRSI 计算相对强弱指标
func CalculateRSI(prices []float64, period int) []float64
```

**算法**:
- 计算价格变化
- 分别计算上涨和下跌的平均值
- RS = 平均上涨 / 平均下跌
- RSI = 100 - (100 / (1 + RS))

**验收标准**:
- 支持 7 期和 14 期 RSI
- 处理除零情况

### ✅ Task 4.4: 实现 ATR 计算

```go
// CalculateATR 计算平均真实波幅
func CalculateATR(klines []Kline, period int) []float64
```

**算法**:
- TR = max(High - Low, |High - PrevClose|, |Low - PrevClose|)
- ATR = EMA(TR, period)

**验收标准**:
- 支持 3 期和 14 期 ATR
- 正确处理第一根 K线

---

## 📋 阶段五: 主数据聚合逻辑

### ✅ Task 5.1: 实现价格变化计算 (`data.go`)

```go
// calculatePriceChange 计算价格变化百分比
func calculatePriceChange(currentPrice, previousPrice float64) float64
```

**实现要点**:
- 1小时变化: 对比 20 根 3分钟 K线前 (约1小时)
- 4小时变化: 对比 1 根 4小时 K线前
- 返回百分比 (如 2.5 表示 2.5%)

**验收标准**:
- 计算公式正确: (current - previous) / previous * 100
- 处理负数情况

### ✅ Task 5.2: 实现日内序列数据计算

```go
// calculateIntradayData 计算日内序列数据
func calculateIntradayData(klines []Kline) *IntradayData
```

**实现要点**:
- 基于 3分钟 K线
- 取最近 10 个数据点
- 计算 EMA20, MACD, RSI7, RSI14 序列
- 数据从旧到新排序

**验收标准**:
- 所有序列长度为 10
- 数据顺序正确
- 指标计算准确

### ✅ Task 5.3: 实现长期数据计算

```go
// calculateLongerTermData 计算长期数据
func calculateLongerTermData(klines []Kline) *LongerTermData
```

**实现要点**:
- 基于 4小时 K线
- 计算 EMA20, EMA50
- 计算 ATR3, ATR14
- 计算当前和平均成交量
- MACD 和 RSI14 序列 (最近10个)

**验收标准**:
- 所有指标计算正确
- 平均成交量计算合理 (如最近20根K线)

### ✅ Task 5.4: 实现主入口函数

```go
// Get 获取指定代币的完整市场数据
func Get(symbol string) (*Data, error)
```

**执行流程**:
1. 标准化 symbol (如 "BTC", "BTCUSDT" → "BTC")
2. 获取 3分钟 K线 (40根,用于指标计算)
3. 获取 4小时 K线 (60根,用于长期指标)
4. 获取当前价格
5. 获取市场信息 (持仓量、资金费率)
6. 计算所有技术指标
7. 计算价格变化
8. 组装 Data 结构

**错误处理**:
- K线获取失败: 返回错误
- 市场信息获取失败: 使用默认值 0,不中断流程
- 指标计算失败: 记录日志,使用默认值

**验收标准**:
- 能成功返回完整的 Data 结构
- 所有字段都有值
- 错误处理完善
- 执行时间 < 5秒

---

## 📋 阶段六: 接口适配

### ✅ Task 6.1: 实现市场数据接口 (`interface.go`)

```go
// MarketDataProvider 市场数据提供者接口
type MarketDataProvider interface {
    Get(symbol string) (*Data, error)
    GetCurrentPrice(symbol string) (float64, error)
}

// HyperliquidProvider Hyperliquid 实现
type HyperliquidProvider struct {
    client *hyperliquid.Client
}

func NewHyperliquidProvider() MarketDataProvider
```

**验收标准**:
- 接口定义清晰
- 易于扩展其他交易所
- 支持依赖注入

---

## 📋 阶段七: 测试与验证

### ✅ Task 7.1: 单元测试

为每个核心函数编写单元测试:
- `TestGetKlines`
- `TestGetMarketInfo`
- `TestCalculateEMA`
- `TestCalculateMACD`
- `TestCalculateRSI`
- `TestCalculateATR`
- `TestGet` (集成测试)

**验收标准**:
- 测试覆盖率 > 80%
- 所有测试通过
- 包含边界情况测试

### ✅ Task 7.2: 集成测试

```go
func TestRealDataFetch(t *testing.T) {
    // 测试真实 API 调用
    data, err := Get("BTC")
    // 验证数据完整性
}
```

**验收标准**:
- 能成功获取真实数据
- 数据格式正确
- 所有字段有值

### ✅ Task 7.3: 性能测试

```go
func BenchmarkGet(b *testing.B) {
    // 性能基准测试
}
```

**验收标准**:
- 单次调用 < 5秒
- 内存使用合理
- 无内存泄漏

---

## 📋 阶段八: 文档与部署

### ✅ Task 8.1: 编写 README

包含以下内容:
- 模块功能说明
- 使用示例
- API 文档链接
- 配置说明
- 常见问题

### ✅ Task 8.2: 代码注释

确保所有公开函数都有:
- 功能说明
- 参数说明
- 返回值说明
- 使用示例

### ✅ Task 8.3: 配置管理

```go
type Config struct {
    BaseURL        string
    Timeout        time.Duration
    MaxRetries     int
    EnableCache    bool
    CacheDuration  time.Duration
}
```

**验收标准**:
- 支持环境变量配置
- 有合理的默认值
- 配置验证

---

## 🎯 关键技术要点

### 1. API 频率限制
- Hyperliquid 有频率限制,建议:
  - 添加请求间隔控制
  - 实现请求缓存
  - 批量获取数据

### 2. 数据准确性
- K线数据可能有延迟
- 使用 `allMids` 获取最新价格
- 注意时间戳精度 (毫秒)

### 3. 错误处理
- 网络超时
- API 返回错误
- 数据格式异常
- 币种不存在

### 4. 性能优化
- 并发获取 3分钟和 4小时 K线
- 复用 HTTP 连接
- 缓存计算结果

---

## 📚 参考资源

### 1. Hyperliquid API 文档
- Info endpoint: <https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/info-endpoint>
- Exchange endpoint: <https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/exchange-endpoint>

### 2. 技术指标算法
- EMA: <https://www.investopedia.com/terms/e/ema.asp>
- MACD: <https://www.investopedia.com/terms/m/macd.asp>
- RSI: <https://www.investopedia.com/terms/r/rsi.asp>
- ATR: <https://www.investopedia.com/terms/a/atr.asp>

### 3. Go 技术栈
- net/http 标准库
- encoding/json
- context 包

---

## ✅ 验收清单

完成以下所有项即可认为任务完成:
- [ ] 所有代码文件创建完成
- [ ] 所有核心函数实现完成
- [ ] 单元测试通过
- [ ] 集成测试通过
- [ ] 性能测试达标
- [ ] 代码注释完整
- [ ] README 文档完成
- [ ] 能成功获取 BTC, ETH, SOL 等主流币种数据
- [ ] 所有技术指标计算准确
- [ ] 错误处理完善
- [ ] 代码符合 Go 最佳实践

---

## 🚀 快速开始示例

完成后的使用示例:

```go
package main

import (
    "fmt"
    "log"
    "yourproject/market"
    "yourproject/market/hyperliquid"
)

func main() {
    // 创建 Hyperliquid 提供者
    provider := market.NewHyperliquidProvider()
    
    // 获取 BTC 市场数据
    data, err := provider.Get("BTC")
    if err != nil {
        log.Fatal(err)
    }
    
    // 打印数据
    fmt.Printf("Symbol: %s
", data.Symbol)
    fmt.Printf("Current Price: %.2f
", data.CurrentPrice)
    fmt.Printf("1h Change: %.2f%%
", data.PriceChange1h)
    fmt.Printf("4h Change: %.2f%%
", data.PriceChange4h)
    fmt.Printf("EMA20: %.2f
", data.CurrentEMA20)
    fmt.Printf("MACD: %.2f
", data.CurrentMACD)
    fmt.Printf("RSI7: %.2f
", data.CurrentRSI7)
    fmt.Printf("Open Interest: %.2f
", data.OpenInterest.Latest)
    fmt.Printf("Funding Rate: %.4f%%
", data.FundingRate)
}
```

---

## 💡 给大语言模型的提示

在实现过程中,请注意:
1. **优先实现核心功能**: 先完成基础的 API 调用和数据获取,再实现技术指标计算
2. **测试驱动开发**: 每完成一个函数就编写对应的测试
3. **错误处理优先**: 确保所有可能的错误都被妥善处理
4. **代码可读性**: 使用清晰的变量名和函数名,添加必要的注释
5. **性能考虑**: 注意避免不必要的重复计算和 API 调用

祝实现顺利!