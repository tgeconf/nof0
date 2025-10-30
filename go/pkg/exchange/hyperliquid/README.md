# Hyperliquid 交易接口

该包提供与 Hyperliquid 交易端点交互的基础设施, 包括:

- `Client`: 负责 HTTP 调用、签名封装与资产目录缓存。
- `Provider`: 满足 `exchange.Provider` 接口, 面向业务层提供统一的下单与账户能力。
- `auth.go`: 私钥签名器与 EIP-712 消息构建。
- `order.go` / `account.go` / `position.go`: 订单与账户相关方法的基础骨架。

## 当前状态

- ✅ 目录结构与核心类型已经搭建完成。
- ✅ Info API 请求、账户状态查询、资产索引缓存已打通。
- ✅ Exchange API 的 EIP-712 签名逻辑已接入, 下单/撤单/杠杆调节具备可用骨架。
- ⚠️ 部分高级功能 (触发单、WebSocket 等) 仍返回 `ErrFeatureUnavailable`, 需要根据业务需求继续完善。

在进一步开发时, 建议遵循 `hyperliquid-exchange-api.md` 中的分阶段任务, 逐步完善签名、订单、仓位与测试覆盖。
