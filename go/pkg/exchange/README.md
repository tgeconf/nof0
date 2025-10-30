# 交易接口

该模块提供统一的交易抽象, 旨在为不同交易所实现一致的下单、撤单、仓位与账户查询接口。

当前进度:

- `interface.go`: 定义通用的 `Provider` 接口以及核心交易数据结构。
- `hyperliquid/`: Hyperliquid 交易所的初始实现, 包含 HTTP 客户端、签名器以及资产元数据缓存。

## 用法示例

```go
import (
    "context"

    "nof0-api/pkg/exchange"
    hyperliquid "nof0-api/pkg/exchange/hyperliquid"
)

func example() error {
    provider, err := hyperliquid.NewProvider("your-private-key-hex", false)
    if err != nil {
        return err
    }

    ctx := context.Background()
    accountValue, err := provider.GetAccountValue(ctx)
    if err != nil {
        return err
    }
    _ = accountValue
    return nil
}
```

> ⚠️ 当前实现仍在演进中, 平仓/高级订单等能力尚未完成, 请在集成前注意检查 `ErrFeatureUnavailable` 错误并根据需要补充功能。
