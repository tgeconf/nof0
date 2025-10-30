# 市场数据

该模块提供统一的市场数据接口, 当前实现包含 Hyperliquid 交易所的行情聚合。

- `hyperliquid/`: Hyperliquid 市场数据实现, 包含 HTTP 客户端、类型定义、指标计算以及数据聚合逻辑。
- `interface.go`: 市场数据提供者接口定义, 暴露 `MarketDataProvider` 以及 `NewHyperliquidProvider` 构造函数。

用法示例:

```go
provider := market.NewHyperliquidProvider()
data, err := provider.Get("BTC")
if err != nil {
    log.Fatal(err)
}
fmt.Println(data.CurrentPrice)
```
