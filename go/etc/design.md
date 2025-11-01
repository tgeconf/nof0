# etc 配置设计

## 背景与目标
- 当前仓库中仅 `internal/config.Config` 和 `pkg/llm` 拥有可运行的配置加载逻辑，`pkg/executor`、`pkg/manager` 仍停留在设计阶段，导致整体缺乏统一的配置体系。
- 参考 go-zero 推荐实践，需要建立「单一可信配置源（Single Source of Truth）」：所有运行时参数以 YAML 为主、配合环境变量覆盖，并通过 `conf.MustLoad` / `conf.With` 等能力在启动时一次性注入。
- 本文档用于约束 `etc/` 目录下配置文件的结构、加载流程与验证规则，同时给出后续落地的 TODO 列表，指导各模块在实现阶段保持一致的约定。

## 全局配置体系
### 目录规划
- `etc/nof0.yaml`：HTTP 服务主配置（内含 `rest.RestConf`、数据库、缓存、数据目录等）。
- `etc/llm.yaml`：LLM Provider 配置（保留 `pkg/llm` 已实现的加载流程）。
- `etc/executor.yaml`：执行器运行参数（风险阈值、模型选择、策略开关等，待新增）。
- `etc/manager.yaml`：虚拟交易员与资源编排配置（Trader 列表、交易所账号、监控配置等，待新增）。
- `etc/prompts/`：Prompt 模板仓库（按模块划分子目录，如 `executor/`、`manager/`），与配置文件同目录发布以便统一变更。
- `etc/exchange.yaml`：交易所 Provider 统一配置（访问凭证、Host、重试与限流策略，后续需要与 `pkg/exchange` 实现对齐）。
- `etc/market.yaml`：行情数据 Provider 配置（数据源、刷新频率、缓存 TTL、Mock 选项等，需与 `pkg/market` 约定加载）。
- `etc/traders/*.yaml`（可选）：当 Trader 配置需要拆分时使用，主配置通过路径引用。
- `etc/secrets.example`：列出需要通过环境变量注入的敏感键名，避免直接写入 YAML。

### 配置加载流程
1. 入口 `main` 通过 `config.MustLoad(*configFile)` 解析 `etc/nof0.yaml`，并启用 `conf.UseEnv()` 支持环境变量覆盖。
2. `config.MustLoad` 除返回 `Config` 外，还要负责：
   - 根据主配置中的路径字段加载模块级配置（如 `cfg.LLM.File`, `cfg.Manager.File`, `cfg.Executor.File`）。
   - 调用各模块的 `LoadConfig + Validate`，保证返回值已经过结构化校验。
   - 将解析后的结果合并进 `config.Config`（例如 `cfg.LLM` 保留 `*llm.Config` 指针）。
3. `svc.NewServiceContext` 只接收结构化后的 `config.Config`，避免在业务层重复解析文件。
4. 对需要热更新的配置（如 Trader 列表、LLM 模型权重）预留 `config.Watcher` 钩子，后续可结合 go-zero 的 `conf.Watch` 实现。

### 环境覆盖与默认值
- 所有结构体字段需补充 `json` tag 的 `default` / `optional` 标记，确保 go-zero 在缺省值时行为一致。
- Secret 信息（API Key、密钥、Webhook）一律写成 `${ENV_VAR}`，并在 `config.MustLoad` 中调用 `os.ExpandEnv`。
- 对需要时间/时长类型的字段，使用字符串表示（如 `3m`, `1h`），在模块层统一解析为 `time.Duration`。
- 对于列表或映射，优先使用 YAML 原生类型，减少自定义解析逻辑。

## 模块级配置要求
### 服务入口（internal/config）
- 新增 `func MustLoad(path string) *Config`，内部调用 `conf.MustLoad` 并处理模块级文件加载、验证与错误包装。
- `Config` 结构需要扩展出新的嵌套字段：
  ```go
  type Config struct {
      rest.RestConf
      DataPath string          `json:",default=../../mcp/data"`
      Postgres PostgresConf    `json:",optional"`
      Redis    redis.RedisConf `json:",optional"`
      TTL      CacheTTL        `json:",optional"`

      LLM      LLMSection      `json:",optional"`
      Executor ExecutorSection `json:",optional"`
      Manager  ManagerSection  `json:",optional"`
  }
  ```
  其中 `LLMSection`、`ExecutorSection`、`ManagerSection` 至少包含 `File`（配置文件路径）与 `Config`（已解析结构）两个字段。
- 提供 `Validate()` / `Sanitize()` 方法，检查必填字段、路径存在性以及分配比例的边界条件。

### LLM 模块
- 保持 `pkg/llm` 现有的 `LoadConfig`、`Validate` 实现，但需要新增 `FromConfig(cfg *llm.Config)` 形式的构造器，减少重复 I/O。
- 支持通过主配置传入模型别名覆盖（如 `default_model_override`），并允许在运行期间热切换时复用相同数据结构。
- 日志等级、重试次数等字段继续支持环境变量覆盖，保证与 `llm.yaml` 中的默认值一致。

### Executor 模块
- 按照 `pkg/executor/design.md` 中的 `ExecutorConfig` 结构实现配置文件解析，字段包括杠杆、最小信心度、风险报酬比等。
- 配置文件需支持：
  - LLM 模型别名及 Prompt 模板路径。
  - 决策节流参数（最小决策间隔、最大并发调用数）。
  - 与 Manager 交互的安全策略（允许的 Trader ID 列表、签名密钥等）。
- 提供 `LoadConfig(path string) (*Config, error)` 与 `Validate()`，并在 `config.MustLoad` 中调用。
- 预留 `Overrides` 字段用于针对特定 Trader 或市场的定制（例如 BTC/ETH 专属杠杆）。

### Manager 模块
- 依据设计文档实现 `pkg/manager/config.go`，完整支持 `ManagerConfig`、`TraderConfig`、`ExchangeConfig`、`MonitoringConfig` 等结构。
- 在加载阶段校验：
  - Trader `allocation_pct` 总和 ≤ 100%，并允许保留 `reserve_equity_pct`。
  - Prompt 模板、热路径文件存在。
  - Trader 通过 `exchange_provider` / `market_provider` 字段引用统一的 Provider ID，校验需确保对应 ID 在 `exchange.Config` / `market.Config` 中存在。
  - Exchange 配置中敏感信息必须由环境变量提供。
- 对 Trader 列表支持按文件拆分：当 `TraderConfig` 设置 `config_file` 时，允许从外部文件读取并合并。
- 为后续热加载预留 `Watcher` 接口（比如 `type WatchCallback func(*Config)`），与主配置保持一致。

### Exchange 模块
- 现有 `pkg/exchange` 实现尚未提供统一的 YAML/JSON 配置加载逻辑，需要补齐：
  - 设计 `exchange.Config` 结构（建议包含 `type`, `base_url`, `api_key`, `api_secret`, `passphrase`, `timeout`, `max_retries`, `rate_limit` 等字段）。
  - 支持环境变量覆盖敏感信息。
  - 在 `internal/config` 中新增 `ExchangeSection`，可加载 `etc/exchange.yaml` 并在 `svc.ServiceContext` 中注册 Provider 工厂。
- 文档需要明确不同交易所的差异化配置字段，以及与 `pkg/manager` 引用的 `ExchangeConfig` 之间的映射关系，避免重复定义。
- 当前 `pkg/manager.Config.TraderConfig` 已通过 `exchange_provider` 映射到 `exchange.Config` 的 Provider ID，应继续剥离冗余的旧字段（如 `exchange`）。

### Market 模块
- `pkg/market` 当前缺乏统一的配置入口，需要定义 `market.Config` 用于描述：
  - 数据源（如 Hyperliquid、Backtest Mock）选择与认证信息。
  - 刷新频率、批量大小、缓存 TTL。
  - 指标计算开关或参数（EMA 长度、MACD 参数等）。
- 在 `etc/market.yaml` 中给出默认模板，`internal/config` 需要增加 `MarketSection` 并在 `svc.ServiceContext` 中注入 Market Provider 初始化逻辑。
- 为测试/回放场景预留 `mode` 字段（如 `live` / `replay` / `mock`），并约定额外文件路径或数据源配置。
- 现阶段 Market Provider 多通过代码常量初始化，缺少配置驱动能力，需在实现中补齐对上述字段的消费路径。

### Exchange / Market / 其他外部依赖
- 在 Manager 配置中集中维护交易所凭证，避免散落在代码或环境变量中。
- 可选地为 `pkg/market` 定义 `MarketSection`，管理数据源刷新频率、缓存策略等。
- 为测试环境提供 `mock` 类型配置，加载后自动切换到模拟实现。

### 安全与合规
- 所有输出日志中禁止直接打印完整密钥，必要时只显示前后缀。
- 配置加载失败必须返回明确错误（包含文件名、字段路径），并阻止服务启动。
- 建议新增 `etc/secrets.example`，列出需要提前导出的环境变量，配合 `.env` 或 CI/CD Secret 管理。

### 测试与验证
- 每个模块的配置包至少包含一个 table-driven 单元测试，覆盖：
  - 正常加载流程。
  - 缺失必填字段、非法枚举、超出范围的失败路径。
  - 环境变量覆盖与 `${VAR}` 展开。
- 在集成测试中增加「配置冒烟检查」：调用 `config.MustLoad` 并验证关键字段非空，作为 CI 的快速失败点。

## TODO 列表
- [ ] `internal/config`: 实现 `MustLoad`，封装 go-zero `conf.MustLoad` 与环境变量加载逻辑。
- [ ] `internal/config`: 扩展 `Config` 结构体，引入 `LLMSection`、`ExecutorSection`、`ManagerSection` 及校验方法。
- [ ] `etc/nof0.yaml`: 新增模块配置入口（路径或内嵌结构），并同步更新示例默认值。
- [ ] `etc/manager.yaml` / `etc/executor.yaml`: 草拟初版模板，确保字段与设计文档一一对应。
- [ ] `pkg/manager/config.go`: 实现配置结构、加载、验证、拆分 Trader 子配置的逻辑与单元测试。
- [ ] `pkg/executor/config.go`: 完成执行器配置解析、验证、覆盖优先级等实现与测试。
- [ ] `pkg/llm`: 补充从主配置注入的构造逻辑，并校验环境变量覆盖顺序。
- [ ] `svc.NewServiceContext`: 按新的配置结构初始化 LLM 客户端、Manager、Executor，确保依赖注入一致。
- [ ] `docs`: 在 README 或开发文档中补充配置使用说明及环境变量清单。
- [ ] `pkg/exchange`: 设计统一的配置结构/加载函数，并在 `internal/config` 中增加 `ExchangeSection` 与 ServiceContext 注入逻辑。
- [ ] `pkg/market`: 定义配置模型与加载流程，提供默认模板 `etc/market.yaml` 并在 ServiceContext 中注入 Provider。
- [ ] `etc/exchange.yaml` / `etc/market.yaml`: 提供示例配置（含必填字段、环境变量占位），确保与代码实现一致。
