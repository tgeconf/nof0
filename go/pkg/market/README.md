# 市场数据

该模块提供统一的市场数据抽象, 当前包含以下子模块:

- `provider.go`: 定义跨交易所通用的 `Provider` 接口、`Snapshot` 结构体等核心类型。
- `indicators/`: 交易所无关的技术指标实现 (EMA/MACD/RSI/ATR 等)。
- `exchanges/hyperliquid/`: Hyperliquid 适配器, 负责调用官方 API 并组装为标准 `Snapshot`。

用法示例:

```go
import (
    "context"
    "log"

    "nof0-api/pkg/market/exchanges/hyperliquid"
)

ctx := context.Background()
provider := hyperliquid.NewProvider()

snapshot, err := provider.Snapshot(ctx, "BTC")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("price=%f 1h-change=%f%%\n", snapshot.Price.Last, snapshot.Change.OneHour)
```
