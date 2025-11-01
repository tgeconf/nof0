# 配置整合优化提案

当前统一配置体系已经接入 LLM、Executor、Manager，并新增 Exchange / Market provider 配置。但实现与最初“Trader 绑定外部依赖、Executor 仅负责执行管线”的理念仍存在差距，需要进一步梳理。

## 核心理念回顾

1. **Trader 绑定模型与 Prompt**：不同 Trader 可选择不同 LLM 模型、Prompt，实现策略差异；Executor 不直接感知模型细节。
2. **Trader 绑定外部依赖**：包括交易所 Provider、市场数据 Provider、风险参数等，便于独立扩展。 
3. **Executor、Manager 负责编排**：Executor 只处理调度与校验；Manager 主导多 Trader 协同与资源划分。
4. **单一配置源**：各类配置通过 `etc/` 统一落地，环境变量覆盖敏感信息；ServiceContext 在启动时一次性加载。

## 现状与不一致点

| 模块/配置 | 当前行为 | 与目标的差距或风险 |
|-----------|-----------|---------------------|
| `executor.Config` (`etc/executor.yaml`) | 要求 `model_alias`、`prompt_template` | 模型应由 Trader 决定；Executor 默认 prompt 也应由 Trader 注入，避免硬编码。 |
| `manager.Config` | Trader 同时包含 `exchange`（旧）、`exchange_provider`（新），默认 `market_provider` 为空 | 存在双轨配置，易错；未强制指定 market provider 时只依赖默认值，可能为 nil。 |
| `exchange.Config` vs `manager.Config.Exchanges` | 统一 Provider 已在 `pkg/exchange` 内实现，但 manager 仍维护凭证副本 | 配置重复，易产生不一致。应让 Manager 引用 `exchange.Config` 的 Provider ID；凭证统一放在 `etc/exchange.yaml`。 |
| `market.Config` | 默认 provider 可选编配 | 若未配置 default 且 Trader 未指定 `market_provider`，后续 Manager 实装时需处理 nil provider 情况。 |
| ServiceContext | 暴露 `ExchangeProviders`、`MarketProviders`、`ManagerTrader*` 映射 | 需要后续 Manager runtime 消费这些映射，确保实际运行时使用统一 Provider。 |

## 优化 / 修复方向

### Executor 相关
- 将模型选择（`model_alias`）从 `executor.Config` 移除，迁移到 Trader 层或调用上下文。
- Executor 接口改成接受外部注入的「已选模型 + Prompt」信息；`etc/executor.yaml` 只保留纯粹的执行参数（超时、并发控制、风险默认值等）。

### Trader / Manager 相关
- 完全弃用 `TraderConfig.Exchange` 字段，改用 `exchange_provider`；在文档与示例中体现。
- 强制要求 `market_provider` 或提供清晰默认策略，避免隐式 nil。
- Manager 在构造 Trader 实例时，从 `ServiceContext.ManagerTraderExchange/Market` 获取 Provider；更新设计文档说明注入流程。
- 将 `Manager.Config.Exchanges`（凭证）替换为引用 `exchange.Config` 中 Provider ID，避免凭证重复配置。

### Exchange / Market 配置
- 确立统一目录结构：`etc/exchange.yaml` / `etc/market.yaml` 作为唯一凭证/数据源配置；Manager YAML 只引用 Provider ID。
- 在文档中列出 Provider 支持的字段、环境变量示例，以及如何引用。
- 增加配置加载测试：`config.MustLoad` 冒烟测试应验证 Exchange/Market section 成功解析。

### 文档 & 工具
- 更新 `etc/design.md` 或主文档，明确 Trader → 模型/Prompt → Provider 链接关系。
- 在 README 或开发文档中增加配置使用示例，强调环境变量注入和 Provider ID 约束。
- 评估是否需要 CLI/脚本来验证配置（例如检查 Trader 引用的 Provider 是否存在）。

## TODO 列表

- [ ] Executor：重构配置结构，去掉 `model_alias`，改为从 Manager/Trader 注入模型信息。
- [ ] Manager：删掉 `TraderConfig.Exchange`，强制 `exchange_provider` / `market_provider`，更新 YAML 示例与验证逻辑。
- [ ] Manager：在运行期使用 `ServiceContext.ManagerTraderExchange/Market` 注入 provider，并处理默认/缺省场景。
- [ ] Exchange：将凭证定义迁移到 `etc/exchange.yaml`，移除 Manager YAML 中的重复凭证字段（或生成自 `exchange.Config`）。
- [ ] Market：为无默认 provider 的情况增加显式报错或 fallback 机制，并更新文档。
- [ ] 文档：同步更新 `etc/design.md` / README，解释新的配置流转路径与必要的环境变量。
- [ ] 测试：为 `config.MustLoad` 增加集成测试，验证 Exchange/Market sections 加载成功，并校验 Trader 引用合法性。
