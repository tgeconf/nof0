
> **目标**: 基于 OpenAI SDK + ZenMux 实现统一的 LLM 调用模块,支持多模型切换
> ****位置**: `go/pkg/llm/`
> ****版本**: v1.0.0 | **创建时间**: 2025-10-30

---

## 📋 项目概述

### 核心功能

- ✅ 通过 OpenAI SDK 统一调用多个 LLM 提供商 (via ZenMux)

- ✅ 支持流式和非流式响应

- ✅ 支持结构化 JSON 输出 (JSON Mode / Structured Output)

- ✅ 重试机制和错误处理

- ✅ 请求/响应日志记录

- ✅ Token 使用统计

### 技术栈

- **SDK**: OpenAI Go SDK (`github.com/openai/openai-go`)

- **网关**: ZenMux (<https://zenmux.ai>)

- **配置**: YAML 配置文件

---

## 🏗️ 模块目录结构

```plaintext
go/pkg/llm/
├── client.go                 # LLM 客户端核心
├── config.go                 # 配置定义和加载
├── types.go                  # 数据类型定义
├── provider.go               # 提供商配置
├── retry.go                  # 重试机制
├── logger.go                 # 日志记录器
├── stream.go                 # 流式响应处理
├── structured.go             # 结构化输出支持
├── examples/                 # 使用示例
│   ├── simple_chat.go        # 简单对话示例
│   ├── structured_output.go  # 结构化输出示例
│   ├── streaming.go          # 流式响应示例
└── client_test.go            # 单元测试
```

---

## ✅ 实现任务清单

### 阶段 1: 基础架构 (核心功能)

#### 任务 1.1: 项目初始化

- [ ]  **创建模块目录** `go/pkg/llm/`

- [ ]  **添加依赖到 go.mod**

  ```bash
  go get github.com/openai/openai-go
go get gopkg.in/yaml.v3
  ```

- [ ]  **创建基础文件结构** (client.go, config.go, types.go)

#### 任务 1.2: 配置管理 (`config.go`)

- [ ]  **定义配置结构体**

  ```go
  type Config struct {
    BaseURL    string            // ZenMux API 端点
    APIKey     string            // ZenMux API Key
    DefaultModel string          // 默认模型
    Timeout    time.Duration     // 请求超时
    MaxRetries int               // 最大重试次数
    LogLevel   string            // 日志级别
    Models     map[string]ModelConfig // 模型配置
}

type ModelConfig struct {
    Provider     string  // 提供商 (openai, anthropic, etc.)
    ModelName    string  // 模型名称 (gpt-5, claude-sonnet-4.5)
    Temperature  float64 // 温度参数
    MaxTokens    int     // 最大 token 数
    TopP         float64 // Top-p 采样
}
  ```

- [ ]  **实现配置加载函数**

- 从 YAML 文件加载

- 从环境变量加载 (覆盖文件配置)

- 配置验证

- [ ]  **创建默认配置** `etc/llm.yaml`

  ```yaml
  base_url: "https://zenmux.ai/api/v1"
api_key: "${ZENMUX_API_KEY}"
default_model: "openai/gpt-5"
timeout: 60s
max_retries: 3
log_level: "info"

models:
  gpt-5:
    provider: "openai"
    model_name: "openai/gpt-5"
    temperature: 0.7
    max_tokens: 4096
  
  claude-sonnet-4.5:
    provider: "anthropic"
    model_name: "anthropic/claude-sonnet-4.5"
    temperature: 0.7
    max_tokens: 4096
  
  deepseek-chat:
    provider: "deepseek"
    model_name: "deepseek/deepseek-chat-v3.1"
    temperature: 0.7
    max_tokens: 4096
  ```

#### 任务 1.3: 类型定义 (`types.go`)

- [ ]  **定义请求类型**

  ```go
  type ChatRequest struct {
    Model       string          // 模型名称
    Messages    []Message       // 对话消息
    Temperature *float64        // 温度 (可选)
    MaxTokens   *int            // 最大 token (可选)
    TopP        *float64        // Top-p (可选)
    Stream      bool            // 是否流式
    ResponseFormat *ResponseFormat // 响应格式 (JSON Mode)
}

type Message struct {
    Role    string // system, user, assistant, tool
    Content string // 消息内容
    Name    string // 工具名称 (tool role)
    ToolCallID string // 工具调用 ID
}

type ResponseFormat struct {
    Type   string      // "text" 或 "json_object"
    Schema interface{} // JSON Schema (可选)
}
  ```

- [ ]  **定义响应类型**

  ```go
  type ChatResponse struct {
    ID      string
    Model   string
    Choices []Choice
    Usage   Usage
    Created int64
}

type Choice struct {
    Index        int
    Message      Message
    FinishReason string // stop, length, tool_calls, content_filter
    ToolCalls    []ToolCall
}

type ToolCall struct {
    ID       string
    Type     string // "function"
    Function FunctionCall
}

type FunctionCall struct {
    Name      string
    Arguments string // JSON string
}

type Usage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}
  ```

- [ ]  **定义流式响应类型**

  ```go
  type StreamResponse struct {
    ID      string
    Model   string
    Choices []StreamChoice
    Created int64
}

type StreamChoice struct {
    Index        int
    Delta        Delta
    FinishReason string
}

type Delta struct {
    Role      string
    Content   string
    ToolCalls []ToolCall
}
  ```

#### 任务 1.4: 客户端核心 (`client.go`)

- [ ]  **定义 LLMClient 接口**

  ```go
  type LLMClient interface {
    // 同步调用
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    
    // 流式调用
    ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamResponse, error)
    
    // 结构化输出
    ChatStructured(ctx context.Context, req *ChatRequest, schema interface{}) (interface{}, error)
    
    // 获取配置
    GetConfig() *Config
    
    // 关闭客户端
    Close() error
}
  ```

- [ ]  **实现 Client 结构体**

  ```go
  type Client struct {
    config       *Config
    openaiClient *openai.Client
    logger       *Logger
    retryHandler *RetryHandler
}
  ```

- [ ]  **实现 NewClient 构造函数**

- 初始化 OpenAI SDK 客户端

- 配置 BaseURL 为 ZenMux 端点

- 设置 API Key

- 初始化日志和重试机制

- [ ]  **实现 Chat 方法** (同步调用)

- 构建 OpenAI SDK 请求

- 调用 API

- 解析响应

- 错误处理

- 日志记录

- [ ]  **实现 ChatStream 方法** (流式调用)

- 启用流式模式

- 返回响应 channel

- 处理流式数据

- [ ]  **实现 ChatStructured 方法** (结构化输出)

- 设置 JSON Mode

- 解析 JSON 响应到结构体

---

### 阶段 2: 高级功能

#### 任务 2.1: 重试机制 (`retry.go`)

- [ ]  **定义重试策略**

  ```go
  type RetryConfig struct {
    MaxRetries     int
    InitialBackoff time.Duration
    MaxBackoff     time.Duration
    Multiplier     float64
}
  ```

- [ ]  **实现指数退避重试**

- 可重试错误判断 (429, 500, 503)

- 指数退避算法

- 最大重试次数限制

- [ ]  **实现重试日志**

- 记录重试次数

- 记录失败原因

#### 任务 2.2: 日志记录 (`logger.go`)

- [ ]  **定义日志接口**

  ```go
  type Logger interface {
    Debug(msg string, fields ...interface{})
    Info(msg string, fields ...interface{})
    Warn(msg string, fields ...interface{})
    Error(msg string, fields ...interface{})
}
  ```

- [ ]  **实现请求日志**

- 记录请求参数 (model, messages, temperature)

- 记录请求时间

- [ ]  **实现响应日志**

- 记录响应内容

- 记录 token 使用量

- 记录响应时间

- [ ]  **实现错误日志**

- 记录错误类型

- 记录错误堆栈

#### 任务 2.3: 流式响应处理 (`stream.go`)

- [ ]  **实现 StreamReader**

  ```go
  type StreamReader struct {
    stream <-chan StreamResponse
    err    error
}

func (r *StreamReader) Next() (*StreamResponse, error)
func (r *StreamReader) Close() error
  ```

- [ ]  **实现流式数据聚合**

- 累积 delta 内容

- 处理工具调用流式响应

- [ ]  **实现流式错误处理**

- 捕获流式错误

- 优雅关闭流

#### 任务 2.4: 结构化输出 (`structured.go`)

- [ ]  **实现 JSON Schema 生成**

  ```go
  func GenerateSchema(v interface{}) (map[string]interface{}, error)
  ```

- [ ]  **实现结构化解析**

  ```go
  func ParseStructured(jsonStr string, target interface{}) error
  ```

- [ ]  **支持常见结构**

- 交易决策结构 (TradeDecision)

- 市场分析结构 (MarketAnalysis)

- 自定义结构

---

### 阶段 4: 示例和测试

#### 任务 4.1: 使用示例 (`examples/`)

- [ ]  **简单对话示例** (`simple_chat.go`)

  ```go
  func main() {
    client := llm.NewClient("etc/llm.yaml")
    
    resp, err := client.Chat(context.Background(), &llm.ChatRequest{
        Model: "gpt-5",
        Messages: []llm.Message{
            {Role: "user", Content: "What is the meaning of life?"},
        },
    })
    
    fmt.Println(resp.Choices[0].Message.Content)
}
  ```

- [ ]  **结构化输出示例** (`structured_output.go`)

  ```go
  type TradeDecision struct {
    Action     string  `json:"action"`      // BUY, SELL, HOLD
    Symbol     string  `json:"symbol"`      // BTC, ETH
    Confidence float64 `json:"confidence"`  // 0-1
    Reasoning  string  `json:"reasoning"`
}

func main() {
    client := llm.NewClient("etc/llm.yaml")
    
    var decision TradeDecision
    err := client.ChatStructured(ctx, &llm.ChatRequest{
        Model: "gpt-5",
        Messages: []llm.Message{
            {Role: "system", Content: "You are a trading assistant."},
            {Role: "user", Content: "Should I buy BTC now? Price: $68000"},
        },
    }, &decision)
    
    fmt.Printf("Action: %s, Confidence: %.2f
  ```

", decision.Action, decision.Confidence)\
}

```plaintext
- [ ] **流式响应示例** (`streaming.go`)
```go
func main() {
    client := llm.NewClient("etc/llm.yaml")
    
    stream, err := client.ChatStream(ctx, &llm.ChatRequest{
        Model: "gpt-5",
        Messages: []llm.Message{
            {Role: "user", Content: "Explain blockchain in simple terms"},
        },
        Stream: true,
    })
    
    for chunk := range stream {
        fmt.Print(chunk.Choices[0].Delta.Content)
    }
}
```

#### 任务 4.2: 单元测试 (`client_test.go`)

- [ ]  **测试客户端初始化**

- 测试配置加载

- 测试默认值

- [ ]  **测试同步调用**

- Mock OpenAI SDK 响应

- 测试正常流程

- 测试错误处理

- [ ]  **测试流式调用**

- 测试流式数据解析

- 测试流关闭

- [ ]  **测试结构化输出**

- 测试 JSON 解析

- 测试 Schema 验证

- [ ]  **测试重试机制**

- 测试可重试错误

- 测试不可重试错误

- 测试退避算法

#### 任务 4.3: 集成测试

- [ ]  **测试真实 API 调用**

- 使用测试 API Key

- 测试多个模型

- 测试不同参数组合

- [ ]  **测试性能**

- 测试响应时间

- 测试并发调用

- 测试 token 使用量

- [ ]  **测试错误场景**

- 测试网络超时

- 测试 API 限流

- 测试无效参数

---

### 阶段 5: 集成到现有系统

#### 任务 5.1: 决策引擎集成

- [ ]  **更新** `pkg/decision/engine.go`

- 替换现有 LLM 调用为新模块

- 使用结构化输出获取交易决策

  ```go
  func (e *Engine) MakeDecision(ctx *TradingContext) (*Decision, error) {
    prompt := e.buildPrompt(ctx)
    
    var decision TradeDecision
    err := e.llmClient.ChatStructured(context.Background(), &llm.ChatRequest{
        Model: e.config.Model,
        Messages: []llm.Message{
            {Role: "system", Content: e.systemPrompt},
            {Role: "user", Content: prompt},
        },
    }, &decision)
    
    return &decision, err
}
  ```

#### 任务 5.2: 配置更新

- [ ]  **更新** `etc/nof0.yaml`

- 添加 LLM 配置段

  ```yaml
  llm:
  base_url: "https://zenmux.ai/api/v1"
  api_key: "${ZENMUX_API_KEY}"
  default_model: "openai/gpt-5"
  timeout: 60s
  max_retries: 3
  ```

- [ ]  **更新环境变量文档**

- 添加 `ZENMUX_API_KEY` 说明

#### 任务 5.3: 日志集成

- [ ]  **集成到现有日志系统**

- 使用统一的日志格式

- 添加 LLM 调用追踪

#### 任务 5.4: 监控集成

- [ ]  **添加 LLM 调用指标**

- 请求数量

- 响应时间

- Token 使用量

- 错误率

- [ ]  **添加告警规则**

- API 限流告警

- 响应超时告警

- 错误率过高告警

---

## 🎯 优先级建议

### P0 (必须完成)

1. ✅ 阶段 1: 基础架构 - 实现核心调用功能

2. ✅ 任务 4.1: 使用示例 - 验证功能可用性

3. ✅ 任务 5.1: 决策引擎集成 - 接入现有系统

### P1 (高优先级)

4. ✅ 任务 2.1: 重试机制 - 提高稳定性

5. ✅ 任务 2.2: 日志记录 - 便于调试

6. ✅ 任务 4.2: 单元测试 - 保证质量

### P2 (中优先级)

7. ✅ 任务 2.3: 流式响应 - 提升用户体验

8. ✅ 任务 2.4: 结构化输出 - 简化解析

9. ✅ 任务 5.4: 监控集成 - 生产环境必备

### P3 (低优先级/可选)

10. ⚠️ 任务 3.1: 集成测试 - 可后续补充

---

## 📝 实现注意事项

### 1. OpenAI SDK 使用

```go
import "github.com/openai/openai-go"

// 初始化客户端
client := openai.NewClient(
    option.WithBaseURL("https://zenmux.ai/api/v1"),
    option.WithAPIKey(apiKey),
)

// 调用 Chat Completions
resp, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
    Model: openai.F("openai/gpt-5"),
    Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
        openai.UserMessage("Hello!"),
    }),
})
```

### 2. ZenMux 模型格式

- 格式: `provider/model-name`

- 示例:

    - `openai/gpt-5`

    - `anthropic/claude-sonnet-4.5`

    - `deepseek/deepseek-chat-v3.1`

    - `qwen/qwen3-max`

### 3. 结构化输出最佳实践

```go
// 方式 1: JSON Mode (推荐)
resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
    Model: openai.F("openai/gpt-5"),
    Messages: openai.F(messages),
    ResponseFormat: openai.F(openai.ChatCompletionNewParamsResponseFormat{
        Type: openai.F(openai.ChatCompletionNewParamsResponseFormatTypeJSONObject),
    }),
})

// 方式 2: Structured Output (需要模型支持)
// 使用 JSON Schema 定义输出格式
```

### 4. 错误处理

```go
// 可重试错误
- 429 Too Many Requests (限流)
- 500 Internal Server Error
- 503 Service Unavailable
- 网络超时

// 不可重试错误
- 400 Bad Request (参数错误)
- 401 Unauthorized (认证失败)
- 403 Forbidden (权限不足)
- 404 Not Found (模型不存在)
```

### 5. 性能优化

- 使用连接池

- 启用 HTTP/2

- 合理设置超时时间

- 实现请求去重

- 使用缓存 (对于相同请求)

### 6. 安全性

- API Key 不要硬编码,使用环境变量

- 记录日志时脱敏敏感信息

- 实现请求签名验证

- 限制并发请求数

---

## 🔗 参考资源

### 官方文档

- [ZenMux Quick Start](https://docs.zenmux.ai/quick-start)

- [OpenAI Go SDK](https://github.com/openai/openai-go)

- [OpenAI API Reference](https://platform.openai.com/docs/api-reference)

### 测试数据

- 参考 `nof1-prompt` 构建测试 Prompt

---

## ✅ 验收标准

### 功能验收

- [ ]  能够通过 OpenAI SDK 调用 ZenMux API

- [ ]  支持至少 3 个不同的模型 (GPT-5, Claude, DeepSeek)

- [ ]  能够获取结构化的交易决策输出

- [ ]  重试机制正常工作

- [ ]  日志记录完整

### 性能验收

- [ ]  单次调用响应时间 < 5s (P95)

- [ ]  支持并发 10+ 请求

- [ ]  Token 使用量统计准确

### 质量验收

- [ ]  单元测试覆盖率 > 80%

- [ ]  所有示例代码可运行

- [ ]  文档完整清晰

### 集成验收

- [ ]  能够替换现有决策引擎的 LLM 调用

- [ ]  配置文件格式统一

- [ ]  日志格式与现有系统一致

---

## 🚀 快速开始 (完成后)

```bash
# 1. 安装依赖
cd go
go mod download

# 2. 配置 API Key
export ZENMUX_API_KEY="your-api-key"

# 3. 运行示例
go run pkg/llm/examples/simple_chat.go

# 4. 运行测试
go test ./pkg/llm/... -v

# 5. 集成到决策引擎
# 更新 etc/nof0.yaml 配置
# 重启 API 服务器
```

---

## 📞 问题反馈

如果在实现过程中遇到问题:

1. 检查 ZenMux API 状态: <https://zenmux.ai/status>

2. 查看 OpenAI SDK 文档: <https://github.com/openai/openai-go>

---

**最后更新**: 2025-10-30\
**维护者**: WquGuru