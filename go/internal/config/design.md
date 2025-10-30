
> **实现目标**: 基于 go-zero 框架最佳实践，从零开始实现 [nof1.ai](http://nof1.ai) 系统的配置管理模块
> ****参考标准**: go-zero 配置管理规范 + [nof1.ai](http://nof1.ai) 业务需求
> ****实现原则**: 类型安全、验证完整、环境友好、可扩展

---

## 📋 模块概述

### 核心职责

- ✅ 加载和解析 YAML 配置文件

- ✅ 多交易员配置管理

- ✅ 多交易平台支持（币安、Hyperliquid）

- ✅ 多 AI 模型支持（Qwen、DeepSeek、自定义 API）

- ✅ 配置验证与默认值处理

- ✅ 环境变量支持

- ✅ 敏感信息保护

### 技术栈

- **框架**: go-zero/core/conf

- **配置格式**: YAML

- **验证**: 结构体标签 + 自定义验证逻辑

- **环境变量**: 支持 `${ENV_VAR:default}` 语法

---

## 🏗️ 阶段一：项目结构搭建

### ✅ Task 1.1: 创建目录结构

**优先级**: P0 🔴\
**预计时间**: 10 分钟

```bash
internal/config/
├── config.go           # 主配置结构定义
├── trader.go          # 交易员配置
├── leverage.go        # 杠杆配置
├── risk.go            # 风险控制配置
├── validator.go       # 配置验证逻辑
├── loader.go          # 配置加载器
└── config_test.go     # 单元测试
```

**验收标准**:

- [ ]  目录结构创建完成

- [ ]  每个文件包含基础 package 声明

---

### ✅ Task 1.2: 初始化 Go Module 依赖

**优先级**: P0 🔴\
**预计时间**: 5 分钟

```bash
go get github.com/zeromicro/go-zero/core/conf
go get gopkg.in/yaml.v3
```

**验收标准**:

- [ ]  go.mod 包含 go-zero 依赖

- [ ]  依赖版本锁定在 go.sum

---

## 🔧 阶段二：核心数据结构定义

### ✅ Task 2.1: 定义 TraderConfig 结构体

**优先级**: P0 🔴\
**预计时间**: 30 分钟\
**文件**: `internal/config/trader.go`

```go
package config

import "time"

// TraderConfig 交易员配置
type TraderConfig struct {
    // 基础信息
    ID   string `json:"id" yaml:"id"`                     // 交易员唯一标识
    Name string `json:"name" yaml:"name"`                 // 交易员名称
    
    // AI 模型配置
    AIModel string `json:"ai_model" yaml:"ai_model"`      // qwen/deepseek/custom
    
    // Qwen 配置
    QwenKey string `json:"qwen_key,optional" yaml:"qwen_key,omitempty"`
    
    // DeepSeek 配置
    DeepSeekKey string `json:"deepseek_key,optional" yaml:"deepseek_key,omitempty"`
    
    // 自定义 API 配置
    CustomAPIURL    string `json:"custom_api_url,optional" yaml:"custom_api_url,omitempty"`
    CustomAPIKey    string `json:"custom_api_key,optional" yaml:"custom_api_key,omitempty"`
    CustomModelName string `json:"custom_model_name,optional" yaml:"custom_model_name,omitempty"`
    
    // 交易平台配置
    Exchange string `json:"exchange" yaml:"exchange"`      // binance/hyperliquid
    
    // 币安配置
    BinanceAPIKey    string `json:"binance_api_key,optional" yaml:"binance_api_key,omitempty"`
    BinanceSecretKey string `json:"binance_secret_key,optional" yaml:"binance_secret_key,omitempty"`
    
    // Hyperliquid 配置
    HyperliquidPrivateKey string `json:"hyperliquid_private_key,optional" yaml:"hyperliquid_private_key,omitempty"`
    HyperliquidTestnet    bool   `json:"hyperliquid_testnet,optional" yaml:"hyperliquid_testnet,omitempty"`
    
    // 交易参数
    InitialBalance       float64 `json:"initial_balance" yaml:"initial_balance"`                          // 初始余额
    ScanIntervalMinutes  int     `json:"scan_interval_minutes,default=3" yaml:"scan_interval_minutes"`   // 扫描间隔(分钟)
}

// GetScanInterval 获取扫描间隔时长
func (tc *TraderConfig) GetScanInterval() time.Duration {
    if tc.ScanIntervalMinutes <= 0 {
        return 3 * time.Minute // 默认 3 分钟
    }
    return time.Duration(tc.ScanIntervalMinutes) * time.Minute
}

// GetAIProvider 获取 AI 提供商标识
func (tc *TraderConfig) GetAIProvider() string {
    return tc.AIModel
}

// GetExchangeName 获取交易所名称
func (tc *TraderConfig) GetExchangeName() string {
    return tc.Exchange
}
```

**验收标准**:

- [ ]  所有字段定义完整，包含 json 和 yaml 标签

- [ ]  使用 go-zero 标签语法（`optional`, `default`）

- [ ]  实现辅助方法（GetScanInterval 等）

- [ ]  代码注释清晰

---

### ✅ Task 2.2: 定义 LeverageConfig 结构体

**优先级**: P0 🔴\
**预计时间**: 15 分钟\
**文件**: `internal/config/leverage.go`

```go
package config

// LeverageConfig 杠杆配置
type LeverageConfig struct {
    BTCETHLeverage  int `json:"btc_eth_leverage,default=5" yaml:"btc_eth_leverage"`    // BTC/ETH 杠杆
    AltcoinLeverage int `json:"altcoin_leverage,default=5" yaml:"altcoin_leverage"`    // 山寨币杠杆
}

// GetBTCETHLeverage 获取 BTC/ETH 杠杆倍数
func (lc *LeverageConfig) GetBTCETHLeverage() int {
    if lc.BTCETHLeverage <= 0 {
        return 5 // 默认 5 倍
    }
    return lc.BTCETHLeverage
}

// GetAltcoinLeverage 获取山寨币杠杆倍数
func (lc *LeverageConfig) GetAltcoinLeverage() int {
    if lc.AltcoinLeverage <= 0 {
        return 5 // 默认 5 倍
    }
    return lc.AltcoinLeverage
}

// IsHighLeverage 判断是否为高杠杆（>5x）
func (lc *LeverageConfig) IsHighLeverage() bool {
    return lc.GetBTCETHLeverage() > 5 || lc.GetAltcoinLeverage() > 5
}
```

**验收标准**:

- [ ]  字段定义完整，包含默认值标签

- [ ]  实现 Getter 方法

- [ ]  实现高杠杆判断方法

- [ ]  代码注释清晰

---

### ✅ Task 2.3: 定义 RiskConfig 结构体

**优先级**: P1 🟡\
**预计时间**: 15 分钟\
**文件**: `internal/config/risk.go`

```go
package config

import "time"

// RiskConfig 风险控制配置
type RiskConfig struct {
    MaxDailyLoss        float64 `json:"max_daily_loss,optional" yaml:"max_daily_loss,omitempty"`              // 每日最大亏损
    MaxDrawdown         float64 `json:"max_drawdown,optional" yaml:"max_drawdown,omitempty"`                  // 最大回撤
    StopTradingMinutes  int     `json:"stop_trading_minutes,default=0" yaml:"stop_trading_minutes"`           // 触发风控后停止交易时长(分钟)
}

// IsEnabled 判断风控是否启用
func (rc *RiskConfig) IsEnabled() bool {
    return rc.MaxDailyLoss > 0 || rc.MaxDrawdown > 0
}

// GetStopTradingDuration 获取停止交易时长
func (rc *RiskConfig) GetStopTradingDuration() time.Duration {
    if rc.StopTradingMinutes <= 0 {
        return 0
    }
    return time.Duration(rc.StopTradingMinutes) * time.Minute
}

// HasDailyLossLimit 是否设置了每日亏损限制
func (rc *RiskConfig) HasDailyLossLimit() bool {
    return rc.MaxDailyLoss > 0
}

// HasDrawdownLimit 是否设置了回撤限制
func (rc *RiskConfig) HasDrawdownLimit() bool {
    return rc.MaxDrawdown > 0
}
```

**验收标准**:

- [ ]  字段定义完整

- [ ]  实现风控判断方法

- [ ]  代码注释清晰

---

### ✅ Task 2.4: 定义主配置 Config 结构体

**优先级**: P0 🔴\
**预计时间**: 20 分钟\
**文件**: `internal/config/config.go`

```go
package config

// Config 系统总配置
type Config struct {
    // 交易员配置列表
    Traders []TraderConfig `json:"traders" yaml:"traders"`
    
    // 币种池配置
    UseDefaultCoins bool   `json:"use_default_coins,default=true" yaml:"use_default_coins"`
    CoinPoolAPIURL  string `json:"coin_pool_api_url,optional" yaml:"coin_pool_api_url,omitempty"`
    OITopAPIURL     string `json:"oi_top_api_url,optional" yaml:"oi_top_api_url,omitempty"`
    
    // API 服务器配置
    APIServerPort int `json:"api_server_port,default=8080" yaml:"api_server_port"`
    
    // 风险控制配置
    Risk RiskConfig `json:"risk,optional" yaml:"risk,omitempty"`
    
    // 杠杆配置
    Leverage LeverageConfig `json:"leverage,optional" yaml:"leverage,omitempty"`
}

// GetAPIServerAddress 获取 API 服务器地址
func (c *Config) GetAPIServerAddress() string {
    if c.APIServerPort <= 0 {
        c.APIServerPort = 8080
    }
    return fmt.Sprintf(":%d", c.APIServerPort)
}

// GetTraderByID 根据 ID 获取交易员配置
func (c *Config) GetTraderByID(id string) *TraderConfig {
    for i := range c.Traders {
        if c.Traders[i].ID == id {
            return &c.Traders[i]
        }
    }
    return nil
}

// GetTraderCount 获取交易员数量
func (c *Config) GetTraderCount() int {
    return len(c.Traders)
}

// IsUsingCoinPool 是否使用币种池 API
func (c *Config) IsUsingCoinPool() bool {
    return c.CoinPoolAPIURL != ""
}

// IsUsingOITop 是否使用 OI Top API
func (c *Config) IsUsingOITop() bool {
    return c.OITopAPIURL != ""
}
```

**验收标准**:

- [ ]  字段定义完整

- [ ]  实现辅助查询方法

- [ ]  代码注释清晰

---

## 🔍 阶段三：配置验证逻辑

### ✅ Task 3.1: 实现 TraderConfig 验证

**优先级**: P0 🔴\
**预计时间**: 45 分钟\
**文件**: `internal/config/validator.go`

```go
package config

import (
    "fmt"
    "strings"
)

// ValidateTrader 验证交易员配置
func ValidateTrader(tc *TraderConfig) error {
    // 1. 基础信息验证
    if strings.TrimSpace(tc.ID) == "" {
        return fmt.Errorf("交易员 ID 不能为空")
    }
    if strings.TrimSpace(tc.Name) == "" {
        return fmt.Errorf("交易员 [%s] 名称不能为空", tc.ID)
    }
    
    // 2. AI 模型验证
    if err := validateAIModel(tc); err != nil {
        return fmt.Errorf("交易员 [%s] AI 模型配置错误: %w", tc.ID, err)
    }
    
    // 3. 交易平台验证
    if err := validateExchange(tc); err != nil {
        return fmt.Errorf("交易员 [%s] 交易平台配置错误: %w", tc.ID, err)
    }
    
    // 4. 初始余额验证
    if tc.InitialBalance <= 0 {
        return fmt.Errorf("交易员 [%s] 初始余额必须大于 0", tc.ID)
    }
    
    // 5. 扫描间隔验证
    if tc.ScanIntervalMinutes < 0 {
        return fmt.Errorf("交易员 [%s] 扫描间隔不能为负数", tc.ID)
    }
    
    return nil
}

// validateAIModel 验证 AI 模型配置
func validateAIModel(tc *TraderConfig) error {
    aiModel := strings.ToLower(strings.TrimSpace(tc.AIModel))
    
    switch aiModel {
    case "qwen":
        if strings.TrimSpace(tc.QwenKey) == "" {
            return fmt.Errorf("使用 Qwen 模型时必须提供 qwen_key")
        }
    case "deepseek":
        if strings.TrimSpace(tc.DeepSeekKey) == "" {
            return fmt.Errorf("使用 DeepSeek 模型时必须提供 deepseek_key")
        }
    case "custom":
        if strings.TrimSpace(tc.CustomAPIURL) == "" {
            return fmt.Errorf("使用自定义 API 时必须提供 custom_api_url")
        }
        if strings.TrimSpace(tc.CustomAPIKey) == "" {
            return fmt.Errorf("使用自定义 API 时必须提供 custom_api_key")
        }
        if strings.TrimSpace(tc.CustomModelName) == "" {
            return fmt.Errorf("使用自定义 API 时必须提供 custom_model_name")
        }
    default:
        return fmt.Errorf("不支持的 AI 模型: %s (支持: qwen, deepseek, custom)", tc.AIModel)
    }
    
    return nil
}

// validateExchange 验证交易平台配置
func validateExchange(tc *TraderConfig) error {
    exchange := strings.ToLower(strings.TrimSpace(tc.Exchange))
    
    switch exchange {
    case "binance":
        if strings.TrimSpace(tc.BinanceAPIKey) == "" {
            return fmt.Errorf("使用币安交易所时必须提供 binance_api_key")
        }
        if strings.TrimSpace(tc.BinanceSecretKey) == "" {
            return fmt.Errorf("使用币安交易所时必须提供 binance_secret_key")
        }
    case "hyperliquid":
        if strings.TrimSpace(tc.HyperliquidPrivateKey) == "" {
            return fmt.Errorf("使用 Hyperliquid 交易所时必须提供 hyperliquid_private_key")
        }
    default:
        return fmt.Errorf("不支持的交易平台: %s (支持: binance, hyperliquid)", tc.Exchange)
    }
    
    return nil
}
```

**验收标准**:

- [ ]  实现完整的交易员配置验证

- [ ]  验证 AI 模型配置完整性

- [ ]  验证交易平台配置完整性

- [ ]  错误信息清晰明确

- [ ]  包含边界条件检查

---

### ✅ Task 3.2: 实现 Config 总体验证

**优先级**: P0 🔴\
**预计时间**: 30 分钟\
**文件**: `internal/config/validator.go`

```go
// Validate 验证总配置
func (c *Config) Validate() error {
    // 1. 交易员列表验证
    if len(c.Traders) == 0 {
        return fmt.Errorf("至少需要配置一个交易员")
    }
    
    // 2. 交易员 ID 唯一性验证
    traderIDs := make(map[string]bool)
    for i, trader := range c.Traders {
        if traderIDs[trader.ID] {
            return fmt.Errorf("交易员 ID 重复: %s", trader.ID)
        }
        traderIDs[trader.ID] = true
        
        // 3. 验证每个交易员配置
        if err := ValidateTrader(&c.Traders[i]); err != nil {
            return err
        }
    }
    
    // 4. 币种池配置验证
    if err := c.validateCoinPool(); err != nil {
        return fmt.Errorf("币种池配置错误: %w", err)
    }
    
    // 5. 杠杆配置验证
    if err := c.validateLeverage(); err != nil {
        return fmt.Errorf("杠杆配置错误: %w", err)
    }
    
    // 6. 风险控制配置验证
    if err := c.validateRisk(); err != nil {
        return fmt.Errorf("风险控制配置错误: %w", err)
    }
    
    // 7. API 服务器端口验证
    if c.APIServerPort < 0 || c.APIServerPort > 65535 {
        return fmt.Errorf("API 服务器端口无效: %d (有效范围: 0-65535)", c.APIServerPort)
    }
    
    return nil
}

// validateCoinPool 验证币种池配置
func (c *Config) validateCoinPool() error {
    // 如果不使用默认币种且没有配置 API，自动启用默认币种
    if !c.UseDefaultCoins && c.CoinPoolAPIURL == "" {
        c.UseDefaultCoins = true
    }
    
    return nil
}

// validateLeverage 验证杠杆配置
func (c *Config) validateLeverage() error {
    btcEthLev := c.Leverage.GetBTCETHLeverage()
    altcoinLev := c.Leverage.GetAltcoinLeverage()
    
    // 杠杆范围验证
    if btcEthLev < 1 || btcEthLev > 125 {
        return fmt.Errorf("BTC/ETH 杠杆倍数无效: %d (有效范围: 1-125)", btcEthLev)
    }
    if altcoinLev < 1 || altcoinLev > 125 {
        return fmt.Errorf("山寨币杠杆倍数无效: %d (有效范围: 1-125)", altcoinLev)
    }
    
    // 高杠杆警告
    if c.Leverage.IsHighLeverage() {
        fmt.Printf("⚠️  警告: 检测到高杠杆配置 (BTC/ETH: %dx, 山寨币: %dx)
", btcEthLev, altcoinLev)
        fmt.Println("   币安子账户杠杆限制为最大 5 倍，超过限制可能导致交易失败")
    }
    
    return nil
}

// validateRisk 验证风险控制配置
func (c *Config) validateRisk() error {
    if c.Risk.MaxDailyLoss < 0 {
        return fmt.Errorf("每日最大亏损不能为负数: %.2f", c.Risk.MaxDailyLoss)
    }
    if c.Risk.MaxDrawdown < 0 {
        return fmt.Errorf("最大回撤不能为负数: %.2f", c.Risk.MaxDrawdown)
    }
    if c.Risk.StopTradingMinutes < 0 {
        return fmt.Errorf("停止交易时长不能为负数: %d", c.Risk.StopTradingMinutes)
    }
    
    return nil
}
```

**验收标准**:

- [ ]  实现总配置验证逻辑

- [ ]  验证交易员 ID 唯一性

- [ ]  验证币种池配置逻辑

- [ ]  验证杠杆配置范围

- [ ]  验证风险控制参数

- [ ]  高杠杆警告输出

---

## 📥 阶段四：配置加载器实现

### ✅ Task 4.1: 实现基于 go-zero 的配置加载

**优先级**: P0 🔴\
**预计时间**: 40 分钟\
**文件**: `internal/config/loader.go`

```go
package config

import (
    "fmt"
    "os"
    
    "github.com/zeromicro/go-zero/core/conf"
)

// Load 加载配置文件（使用 go-zero conf 包）
func Load(filename string) (*Config, error) {
    // 1. 检查文件是否存在
    if _, err := os.Stat(filename); os.IsNotExist(err) {
        return nil, fmt.Errorf("配置文件不存在: %s", filename)
    }
    
    // 2. 使用 go-zero 加载配置
    var c Config
    if err := conf.Load(filename, &c); err != nil {
        return nil, fmt.Errorf("加载配置文件失败: %w", err)
    }
    
    // 3. 应用默认值（go-zero 标签已处理大部分默认值）
    applyDefaults(&c)
    
    // 4. 验证配置
    if err := c.Validate(); err != nil {
        return nil, fmt.Errorf("配置验证失败: %w", err)
    }
    
    return &c, nil
}

// MustLoad 加载配置文件，失败则 panic（适用于启动阶段）
func MustLoad(filename string) *Config {
    c, err := Load(filename)
    if err != nil {
        panic(fmt.Sprintf("配置加载失败: %v", err))
    }
    return c
}

// applyDefaults 应用默认值（补充 go-zero 标签未覆盖的场景）
func applyDefaults(c *Config) {
    // 币种池配置默认值
    if !c.UseDefaultCoins && c.CoinPoolAPIURL == "" {
        c.UseDefaultCoins = true
    }
    
    // API 服务器端口默认值
    if c.APIServerPort == 0 {
        c.APIServerPort = 8080
    }
    
    // 杠杆配置默认值
    if c.Leverage.BTCETHLeverage == 0 {
        c.Leverage.BTCETHLeverage = 5
    }
    if c.Leverage.AltcoinLeverage == 0 {
        c.Leverage.AltcoinLeverage = 5
    }
    
    // 交易员扫描间隔默认值
    for i := range c.Traders {
        if c.Traders[i].ScanIntervalMinutes == 0 {
            c.Traders[i].ScanIntervalMinutes = 3
        }
    }
}
```

**验收标准**:

- [ ]  使用 go-zero conf.Load 加载配置

- [ ]  实现 Load 和 MustLoad 两个版本

- [ ]  应用默认值逻辑

- [ ]  集成配置验证

- [ ]  错误信息清晰

---

### ✅ Task 4.2: 支持环境变量覆盖

**优先级**: P1 🟡\
**预计时间**: 30 分钟\
**文件**: `internal/config/loader.go`

```go
// LoadWithEnv 加载配置并支持环境变量覆盖
func LoadWithEnv(filename string) (*Config, error) {
    c, err := Load(filename)
    if err != nil {
        return nil, err
    }
    
    // 应用环境变量覆盖
    applyEnvOverrides(c)
    
    // 重新验证（环境变量可能改变配置）
    if err := c.Validate(); err != nil {
        return nil, fmt.Errorf("环境变量覆盖后配置验证失败: %w", err)
    }
    
    return c, nil
}

// applyEnvOverrides 应用环境变量覆盖
func applyEnvOverrides(c *Config) {
    // API 服务器端口
    if port := os.Getenv("NOF1_API_PORT"); port != "" {
        if p, err := strconv.Atoi(port); err == nil {
            c.APIServerPort = p
        }
    }
    
    // 币种池 API
    if url := os.Getenv("NOF1_COIN_POOL_API"); url != "" {
        c.CoinPoolAPIURL = url
    }
    
    // OI Top API
    if url := os.Getenv("NOF1_OI_TOP_API"); url != "" {
        c.OITopAPIURL = url
    }
    
    // 遍历交易员，应用环境变量覆盖
    for i := range c.Traders {
        applyTraderEnvOverrides(&c.Traders[i])
    }
}

// applyTraderEnvOverrides 应用交易员环境变量覆盖
func applyTraderEnvOverrides(tc *TraderConfig) {
    prefix := fmt.Sprintf("NOF1_TRADER_%s_", strings.ToUpper(tc.ID))
    
    // 币安 API Key
    if key := os.Getenv(prefix + "BINANCE_API_KEY"); key != "" {
        tc.BinanceAPIKey = key
    }
    if secret := os.Getenv(prefix + "BINANCE_SECRET_KEY"); secret != "" {
        tc.BinanceSecretKey = secret
    }
    
    // Hyperliquid Private Key
    if key := os.Getenv(prefix + "HYPERLIQUID_PRIVATE_KEY"); key != "" {
        tc.HyperliquidPrivateKey = key
    }
    
    // AI 模型 Key
    if key := os.Getenv(prefix + "QWEN_KEY"); key != "" {
        tc.QwenKey = key
    }
    if key := os.Getenv(prefix + "DEEPSEEK_KEY"); key != "" {
        tc.DeepSeekKey = key
    }
    if key := os.Getenv(prefix + "CUSTOM_API_KEY"); key != "" {
        tc.CustomAPIKey = key
    }
}
```

**验收标准**:

- [ ]  实现环境变量覆盖逻辑

- [ ]  支持敏感信息通过环境变量传递

- [ ]  环境变量命名规范清晰

- [ ]  覆盖后重新验证配置

---

## 📝 阶段五：配置文件模板

### ✅ Task 5.1: 创建配置文件示例

**优先级**: P1 🟡\
**预计时间**: 20 分钟\
**文件**: `etc/config.example.yaml`

```yaml
# nof1.ai 配置文件示例

# 交易员配置列表
traders:
  - id: trader_001
    name: "Alpha Trader"
    
    # AI 模型配置 (qwen/deepseek/custom)
    ai_model: qwen
    qwen_key: ${QWEN_API_KEY}  # 支持环境变量
    
    # 交易平台配置 (binance/hyperliquid)
    exchange: binance
    binance_api_key: ${BINANCE_API_KEY}
    binance_secret_key: ${BINANCE_SECRET_KEY}
    
    # 交易参数
    initial_balance: 10000.0
    scan_interval_minutes: 3

  - id: trader_002
    name: "Beta Trader"
    
    # 使用 DeepSeek 模型
    ai_model: deepseek
    deepseek_key: ${DEEPSEEK_API_KEY}
    
    # 使用 Hyperliquid 交易所
    exchange: hyperliquid
    hyperliquid_private_key: ${HYPERLIQUID_PRIVATE_KEY}
    hyperliquid_testnet: false
    
    initial_balance: 5000.0
    scan_interval_minutes: 5

# 币种池配置
use_default_coins: true
coin_pool_api_url: ""
oi_top_api_url: ""

# API 服务器配置
api_server_port: 8080

# 杠杆配置
leverage:
  btc_eth_leverage: 5
  altcoin_leverage: 5

# 风险控制配置
risk:
  max_daily_loss: 1000.0      # 每日最大亏损 (USD)
  max_drawdown: 2000.0         # 最大回撤 (USD)
  stop_trading_minutes: 60     # 触发风控后停止交易时长 (分钟)
```

**验收标准**:

- [ ]  配置文件格式正确

- [ ]  包含所有必要字段

- [ ]  包含注释说明

- [ ]  展示环境变量用法

---

### ✅ Task 5.2: 创建多环境配置示例

**优先级**: P2 🟢\
**预计时间**: 15 分钟

创建以下配置文件：

- `etc/config-dev.yaml` - 开发环境

- `etc/config-test.yaml` - 测试环境

- `etc/config-prod.yaml` - 生产环境

**验收标准**:

- [ ]  三个环境配置文件创建完成

- [ ]  每个环境配置适配对应场景

- [ ]  生产环境使用环境变量保护敏感信息

---

## 🧪 阶段六：单元测试

### ✅ Task 6.1: 测试配置加载

**优先级**: P0 🔴\
**预计时间**: 45 分钟\
**文件**: `internal/config/loader_test.go`

```go
package config

import (
    "os"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
    t.Run("加载有效配置", func(t *testing.T) {
        // 创建临时配置文件
        content := `
traders:
  - id: test_trader
    name: Test Trader
    ai_model: qwen
    qwen_key: test_key
    exchange: binance
    binance_api_key: test_api_key
    binance_secret_key: test_secret_key
    initial_balance: 10000.0
`
        tmpfile := createTempConfigFile(t, content)
        defer os.Remove(tmpfile)
        
        // 加载配置
        cfg, err := Load(tmpfile)
        require.NoError(t, err)
        require.NotNil(t, cfg)
        
        // 验证配置
        assert.Equal(t, 1, len(cfg.Traders))
        assert.Equal(t, "test_trader", cfg.Traders[0].ID)
        assert.Equal(t, 8080, cfg.APIServerPort) // 默认值
    })
    
    t.Run("文件不存在", func(t *testing.T) {
        _, err := Load("nonexistent.yaml")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "配置文件不存在")
    })
    
    t.Run("配置验证失败", func(t *testing.T) {
        content := `
traders:
  - id: ""
    name: Invalid Trader
`
        tmpfile := createTempConfigFile(t, content)
        defer os.Remove(tmpfile)
        
        _, err := Load(tmpfile)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "配置验证失败")
    })
}

func createTempConfigFile(t *testing.T, content string) string {
    tmpfile, err := os.CreateTemp("", "config-*.yaml")
    require.NoError(t, err)
    
    _, err = tmpfile.Write([]byte(content))
    require.NoError(t, err)
    
    err = tmpfile.Close()
    require.NoError(t, err)
    
    return tmpfile.Name()
}
```

**验收标准**:

- [ ]  测试正常加载场景

- [ ]  测试文件不存在场景

- [ ]  测试配置验证失败场景

- [ ]  测试默认值应用

- [ ]  测试覆盖率 > 80%

---

### ✅ Task 6.2: 测试配置验证

**优先级**: P0 🔴\
**预计时间**: 60 分钟\
**文件**: `internal/config/validator_test.go`

```go
package config

import (
    "testing"
    
    "github.com/stretchr/testify/assert"
)

func TestValidateTrader(t *testing.T) {
    t.Run("有效的交易员配置", func(t *testing.T) {
        tc := &TraderConfig{
            ID:               "trader_001",
            Name:             "Test Trader",
            AIModel:          "qwen",
            QwenKey:          "test_key",
            Exchange:         "binance",
            BinanceAPIKey:    "api_key",
            BinanceSecretKey: "secret_key",
            InitialBalance:   10000.0,
        }
        
        err := ValidateTrader(tc)
        assert.NoError(t, err)
    })
    
    t.Run("ID 为空", func(t *testing.T) {
        tc := &TraderConfig{
            ID:   "",
            Name: "Test Trader",
        }
        
        err := ValidateTrader(tc)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "ID 不能为空")
    })
    
    t.Run("AI 模型配置缺失", func(t *testing.T) {
        tc := &TraderConfig{
            ID:      "trader_001",
            Name:    "Test Trader",
            AIModel: "qwen",
            // QwenKey 缺失
        }
        
        err := ValidateTrader(tc)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "qwen_key")
    })
    
    t.Run("交易平台配置缺失", func(t *testing.T) {
        tc := &TraderConfig{
            ID:      "trader_001",
            Name:    "Test Trader",
            AIModel: "qwen",
            QwenKey: "test_key",
            Exchange: "binance",
            // BinanceAPIKey 缺失
        }
        
        err := ValidateTrader(tc)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "binance_api_key")
    })
    
    t.Run("初始余额无效", func(t *testing.T) {
        tc := &TraderConfig{
            ID:               "trader_001",
            Name:             "Test Trader",
            AIModel:          "qwen",
            QwenKey:          "test_key",
            Exchange:         "binance",
            BinanceAPIKey:    "api_key",
            BinanceSecretKey: "secret_key",
            InitialBalance:   -100.0, // 负数
        }
        
        err := ValidateTrader(tc)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "初始余额")
    })
}

func TestConfigValidate(t *testing.T) {
    t.Run("有效配置", func(t *testing.T) {
        cfg := &Config{
            Traders: []TraderConfig{
                {
                    ID:               "trader_001",
                    Name:             "Test Trader",
                    AIModel:          "qwen",
                    QwenKey:          "test_key",
                    Exchange:         "binance",
                    BinanceAPIKey:    "api_key",
                    BinanceSecretKey: "secret_key",
                    InitialBalance:   10000.0,
                },
            },
        }
        
        err := cfg.Validate()
        assert.NoError(t, err)
    })
    
    t.Run("交易员列表为空", func(t *testing.T) {
        cfg := &Config{
            Traders: []TraderConfig{},
        }
        
        err := cfg.Validate()
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "至少需要配置一个交易员")
    })
    
    t.Run("交易员 ID 重复", func(t *testing.T) {
        cfg := &Config{
            Traders: []TraderConfig{
                {
                    ID:   "trader_001",
                    Name: "Trader 1",
                },
                {
                    ID:   "trader_001", // 重复 ID
                    Name: "Trader 2",
                },
            },
        }
        
        err := cfg.Validate()
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "ID 重复")
    })
}
```

**验收标准**:

- [ ]  测试所有验证规则

- [ ]  测试边界条件

- [ ]  测试错误信息准确性

- [ ]  测试覆盖率 > 85%

---

### ✅ Task 6.3: 测试环境变量覆盖

**优先级**: P1 🟡\
**预计时间**: 30 分钟\
**文件**: `internal/config/loader_test.go`

```go
func TestLoadWithEnv(t *testing.T) {
    t.Run("环境变量覆盖 API 端口", func(t *testing.T) {
        content := `
traders:
  - id: test_trader
    name: Test Trader
    ai_model: qwen
    qwen_key: test_key
    exchange: binance
    binance_api_key: test_api_key
    binance_secret_key: test_secret_key
    initial_balance: 10000.0
api_server_port: 8080
`
        tmpfile := createTempConfigFile(t, content)
        defer os.Remove(tmpfile)
        
        // 设置环境变量
        os.Setenv("NOF1_API_PORT", "9090")
        defer os.Unsetenv("NOF1_API_PORT")
        
        // 加载配置
        cfg, err := LoadWithEnv(tmpfile)
        require.NoError(t, err)
        
        // 验证环境变量覆盖生效
        assert.Equal(t, 9090, cfg.APIServerPort)
    })
    
    t.Run("环境变量覆盖交易员密钥", func(t *testing.T) {
        content := `
traders:
  - id: test_trader
    name: Test Trader
    ai_model: qwen
    qwen_key: original_key
    exchange: binance
    binance_api_key: original_api_key
    binance_secret_key: original_secret_key
    initial_balance: 10000.0
`
        tmpfile := createTempConfigFile(t, content)
        defer os.Remove(tmpfile)
        
        // 设置环境变量
        os.Setenv("NOF1_TRADER_TEST_TRADER_QWEN_KEY", "env_key")
        defer os.Unsetenv("NOF1_TRADER_TEST_TRADER_QWEN_KEY")
        
        // 加载配置
        cfg, err := LoadWithEnv(tmpfile)
        require.NoError(t, err)
        
        // 验证环境变量覆盖生效
        assert.Equal(t, "env_key", cfg.Traders[0].QwenKey)
    })
}
```

**验收标准**:

- [ ]  测试环境变量覆盖功能

- [ ]  测试敏感信息覆盖

- [ ]  测试环境变量优先级

- [ ]  测试覆盖率 > 75%

---

## 📚 阶段七：文档和示例

### ✅ Task 7.1: 编写 README 文档

**优先级**: P1 🟡\
**预计时间**: 30 分钟\
**文件**: `internal/config/README.md`

内容包括：

- 模块概述

- 配置文件格式说明

- 环境变量使用指南

- 配置验证规则

- 使用示例代码

- 常见问题解答

**验收标准**:

- [ ]  文档结构清晰

- [ ]  包含完整示例

- [ ]  包含最佳实践建议

---

### ✅ Task 7.2: 编写使用示例

**优先级**: P2 🟢\
**预计时间**: 20 分钟\
**文件**: `examples/config_usage/main.go`

```go
package main

import (
    "flag"
    "fmt"
    "log"
    
    "your-project/internal/config"
)

var configFile = flag.String("f", "etc/config.yaml", "配置文件路径")

func main() {
    flag.Parse()
    
    // 方式 1: 基础加载
    cfg, err := config.Load(*configFile)
    if err != nil {
        log.Fatalf("加载配置失败: %v", err)
    }
    
    // 方式 2: 支持环境变量覆盖
    // cfg, err := config.LoadWithEnv(*configFile)
    
    // 方式 3: 启动时加载（失败则 panic）
    // cfg := config.MustLoad(*configFile)
    
    // 使用配置
    fmt.Printf("配置加载成功!
")
    fmt.Printf("交易员数量: %d
", cfg.GetTraderCount())
    fmt.Printf("API 服务器地址: %s
", cfg.GetAPIServerAddress())
    
    // 遍历交易员
    for _, trader := range cfg.Traders {
        fmt.Printf("
交易员: %s (%s)
", trader.Name, trader.ID)
        fmt.Printf("  AI 模型: %s
", trader.GetAIProvider())
        fmt.Printf("  交易所: %s
", trader.GetExchangeName())
        fmt.Printf("  扫描间隔: %v
", trader.GetScanInterval())
    }
}
```

**验收标准**:

- [ ]  示例代码可运行

- [ ]  展示多种加载方式

- [ ]  展示配置使用方法

---

## 🚀 阶段八：集成和优化

### ✅ Task 8.1: 集成到主程序

**优先级**: P0 🔴\
**预计时间**: 30 分钟

修改 `cmd/server/main.go`，集成配置管理：

```go
package main

import (
    "flag"
    "fmt"
    "log"
    
    "your-project/internal/config"
)

var configFile = flag.String("f", "etc/config.yaml", "配置文件路径")

func main() {
    flag.Parse()
    
    // 加载配置
    cfg := config.MustLoad(*configFile)
    
    fmt.Printf("✅ 配置加载成功
")
    fmt.Printf("📊 交易员数量: %d
", cfg.GetTraderCount())
    fmt.Printf("🌐 API 服务器: %s
", cfg.GetAPIServerAddress())
    
    // 启动系统...
}
```

**验收标准**:

- [ ]  主程序成功加载配置

- [ ]  启动日志清晰

- [ ]  配置错误时优雅退出

---

### ✅ Task 8.2: 性能优化

**优先级**: P2 🟢\
**预计时间**: 20 分钟

优化点：

1. 配置对象缓存（避免重复加载）

2. 验证逻辑优化（减少重复检查）

3. 字符串处理优化（减少内存分配）

**验收标准**:

- [ ]  配置加载时间 < 100ms

- [ ]  内存占用合理

- [ ]  无明显性能瓶颈

---

### ✅ Task 8.3: 日志集成

**优先级**: P1 🟡\
**预计时间**: 15 分钟

使用 go-zero 日志系统记录配置加载过程：

```go
import "github.com/zeromicro/go-zero/core/logx"

func Load(filename string) (*Config, error) {
    logx.Infof("开始加载配置文件: %s", filename)
    
    // ... 加载逻辑
    
    logx.Infof("配置加载成功, 交易员数量: %d", len(c.Traders))
    return c, nil
}
```

**验收标准**:

- [ ]  关键步骤有日志记录

- [ ]  日志级别合理

- [ ]  错误日志包含上下文信息

---

## ✅ 验收清单

### 功能完整性

- [ ]  支持 YAML 配置文件加载

- [ ]  支持多交易员配置

- [ ]  支持多交易平台（币安、Hyperliquid）

- [ ]  支持多 AI 模型（Qwen、DeepSeek、自定义）

- [ ]  支持杠杆配置

- [ ]  支持风险控制配置

- [ ]  支持环境变量覆盖

- [ ]  配置验证完整

### 代码质量

- [ ]  符合 go-zero 最佳实践

- [ ]  代码注释清晰

- [ ]  错误处理完善

- [ ]  单元测试覆盖率 > 80%

- [ ]  无 lint 警告

### 文档完整性

- [ ]  README 文档完整

- [ ]  配置文件示例完整

- [ ]  使用示例代码可运行

- [ ]  API 文档清晰

### 性能指标

- [ ]  配置加载时间 < 100ms

- [ ]  内存占用合理

- [ ]  无性能瓶颈

---

## 📊 进度追踪

| 阶段 | 任务数 | 完成数 | 进度 |
| --- | --- | --- | --- |
| 阶段一：项目结构 | 2 | 0 | 0% |
| 阶段二：数据结构 | 4 | 0 | 0% |
| 阶段三：配置验证 | 2 | 0 | 0% |
| 阶段四：配置加载 | 2 | 0 | 0% |
| 阶段五：配置模板 | 2 | 0 | 0% |
| 阶段六：单元测试 | 3 | 0 | 0% |
| 阶段七：文档示例 | 2 | 0 | 0% |
| 阶段八：集成优化 | 3 | 0 | 0% |
| **总计** | **20** | **0** | **0%** |

---

## 🎯 实现建议

### 开发顺序

1. **先实现核心**: 数据结构 → 验证逻辑 → 加载器

2. **再完善功能**: 环境变量 → 默认值 → 错误处理

3. **最后优化**: 测试 → 文档 → 性能优化

### 注意事项

1. **严格遵循 go-zero 规范**: 使用 conf.Load，遵循标签语法

2. **敏感信息保护**: 密钥通过环境变量传递，不写入配置文件

3. **验证优先**: 配置加载后立即验证，及早发现问题

4. **错误信息友好**: 提供清晰的错误提示，帮助用户快速定位问题

5. **测试覆盖完整**: 覆盖正常场景和异常场景

### 参考资源

- [go-zero 配置管理文档](https://go-zero.dev/docs/tutorials/go-zero/configuration/config)

- [YAML 语法规范](https://yaml.org/spec/1.2/spec.html)

- [Go 结构体标签最佳实践](https://go.dev/wiki/Well-known-struct-tags)

---

## 🔄 迭代计划

### v1.0 (基础版本)

- ✅ 基础配置加载

- ✅ 配置验证

- ✅ 默认值处理

### v1.1 (增强版本)

- ✅ 环境变量支持

- ✅ 多环境配置

- ✅ 配置热更新（可选）

### v2.0 (高级版本)

- ✅ 配置中心集成（Nacos/Etcd）

- ✅ 配置加密

- ✅ 配置审计日志

---

**最后更新**: 2025-01-30\
**负责人**: LLM 辅助实现\
**状态**: 待开始 🟡