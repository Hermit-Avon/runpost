# runpost 设计文档（DESIGN）

## 1. 设计目标

`runpost` 的定位是“命令执行后处理器”：

1. 可靠执行任意子命令，保持其 `stdout/stderr` 与退出码语义。
2. 在命令结束后做统一后处理（摘要、通知、持久化、告警策略）。
3. 支持可插拔通知渠道（Email/Telegram/企业微信/Webhook 等）。
4. 支持结合 LLM 做失败原因提炼、输出压缩与告警文本生成。

## 2. 范围与非目标

### 2.1 范围（V1-V3）

1. `V1`（当前）：命令执行 + 输出透传 + 退出码透传。
2. `V2`：输出捕获、通知插件、失败时通知策略、模板化消息。
3. `V3`：LLM 摘要、智能降噪、分级告警。

### 2.2 非目标

1. 不做任务调度器（不替代 cron / Airflow）。
2. 不做分布式执行器（不负责远程机器执行）。
3. 不做全量日志平台（仅保留与通知相关的摘要和片段）。

## 3. 总体架构

采用“单进程主程序 + 可选 LLM sidecar”的分层架构。

```text
CLI(cmd/runpost)
  -> Orchestrator(流程编排)
    -> Executor(命令执行)
    -> Capture(输出采集/截断)
    -> Policy(通知判定)
    -> Formatter(模板渲染)
    -> Notifier(多渠道发送)
    -> Store(可选落盘)
    -> LLMClient(可选：摘要/分类)
```

### 3.1 模块职责切分

1. `cmd/runpost`
- 参数解析、配置加载、启动流程。

2. `internal/orchestrator`
- 串联执行链路，保证阶段顺序和错误处理一致。

3. `internal/executor`
- 使用 `os/exec` 启动子命令。
- 输出实时透传到终端。
- 返回标准化执行结果。

4. `internal/capture`
- 采集 `stdout/stderr`（环形缓冲，避免 OOM）。
- 支持最大字节数、行数截断。

5. `internal/policy`
- 决定是否通知（总是通知 / 仅失败通知 / 超时通知）。

6. `internal/formatter`
- 把执行结果渲染为渠道可消费消息（text/markdown/json）。

7. `internal/notifier`
- 渠道适配器层：Email、Telegram、WeCom、Webhook。

8. `internal/llm`
- 生成摘要、失败原因分类、建议动作。
- 失败降级：LLM 不可用时回落到规则摘要。

9. `internal/store`（可选）
- 将执行记录落盘（jsonl/sqlite）用于审计与回放。

## 4. 语言与技术选型

### 4.1 语言切分

1. Go（核心必须）
- CLI、流程编排、执行引擎、通知插件框架、配置解析。
- 原因：单二进制部署简单，适合系统工具。

2. Python（可选增强）
- 仅用于复杂 LLM 工作流（如多模型路由、长文本压缩、实验性 prompt 链）。
- 通过 HTTP sidecar 或 stdin/stdout JSON 协议与 Go 交互。

### 4.2 结合策略

1. 默认路径：Go 直接调用 OpenAI 兼容 HTTP API（依赖最少）。
2. 增强路径：启用 Python `llm-worker`，Go 只关心统一接口。
3. 回退策略：LLM 超时/失败时使用规则摘要，不影响通知主链路。

## 5. 核心执行链路

1. 解析参数与配置。
2. 构造 `ExecutionContext`（trace_id、开始时间、命令信息）。
3. `Executor` 启动命令并实时透传输出。
4. `Capture` 同步采集输出尾部片段。
5. 命令退出后生成 `CommandResult`。
6. `Policy` 判断是否需要通知。
7. 若启用 LLM：调用 `LLMClient` 生成摘要与分类。
8. `Formatter` 渲染消息体。
9. `Notifier` 并发发送到多个渠道。
10. 主进程以子命令退出码退出。

## 6. 接口设计

### 6.1 CLI 接口

```bash
runpost [flags] -- <command> [args...]
```

建议 flags：

1. `--config <path>`：配置文件路径。
2. `--notify-on <always|failure|timeout|never>`：通知策略。
3. `--timeout <duration>`：命令超时。
4. `--max-capture-bytes <n>`：采集上限。
5. `--llm <on|off|auto>`：是否启用 LLM 摘要。
6. `--dry-run`：仅打印将发送的消息，不真正发送。

### 6.2 配置文件接口（YAML）

```yaml
version: 1
notify_on: failure
capture:
  max_stdout_bytes: 65536
  max_stderr_bytes: 65536
  tail_lines: 200
llm:
  enabled: true
  provider: openai
  base_url: https://api.openai.com/v1
  model: gpt-5-mini
  api_key_env: OPENAI_API_KEY
  timeout: 8s
channels:
  - type: telegram
    bot_token_env: TELEGRAM_BOT_TOKEN
    chat_id: "123456"
  - type: webhook
    url: https://example.com/hook
    secret_env: WEBHOOK_SECRET
```

### 6.3 内部领域模型

```go
type CommandResult struct {
    Command         []string
    StartAt         time.Time
    EndAt           time.Time
    Duration        time.Duration
    ExitCode        int
    TimedOut        bool
    StdoutTail      string
    StderrTail      string
    Err             string
}

type Summary struct {
    Title           string
    ShortReason     string
    Severity        string // info/warn/error
    SuggestedAction string
}
```

### 6.4 关键抽象接口（Go）

```go
type Executor interface {
    Run(ctx context.Context, cmd []string, opt ExecOption) (CommandResult, error)
}

type Summarizer interface {
    Summarize(ctx context.Context, result CommandResult) (Summary, error)
}

type Notifier interface {
    Name() string
    Send(ctx context.Context, msg Message) error
}
```

### 6.5 LLM Worker（可选）接口

`POST /v1/summarize`

请求：

```json
{
  "command": ["go", "test", "./..."],
  "exit_code": 1,
  "stdout_tail": "...",
  "stderr_tail": "...",
  "duration_ms": 12345
}
```

响应：

```json
{
  "title": "Go tests failed",
  "short_reason": "package xyz has compile error",
  "severity": "error",
  "suggested_action": "run go test ./pkg/xyz -run TestA"
}
```

## 7. 通知系统设计

### 7.1 统一消息模型

```go
type Message struct {
    Title       string
    Body        string
    Level       string
    Tags        map[string]string
    OccurredAt  time.Time
}
```

### 7.2 渠道适配器

1. `telegramNotifier`
- 输出 Markdown 文本。

2. `emailNotifier`
- 输出 HTML + 纯文本双格式。

3. `webhookNotifier`
- 输出 JSON，可配置签名头。

4. `wecomNotifier`
- 按企业微信机器人协议封装。

### 7.3 重试与容错

1. 每渠道独立重试（指数退避，最多 3 次）。
2. 单渠道失败不影响其他渠道发送。
3. 所有渠道失败时只记录日志，不覆盖子命令退出码。

## 8. LLM 集成设计

### 8.1 使用场景

1. 失败输出摘要：把长日志压缩为 3-5 行可读结论。
2. 错误分类：编译错误、网络错误、权限错误、测试失败。
3. 建议动作：给出下一条排查命令。

### 8.2 Prompt 输入边界

1. 只传尾部片段和关键元数据，避免敏感信息泄露。
2. 对 token 进行预算控制（例如 4k 输入上限）。
3. 脱敏规则先执行（token/password/key）。

### 8.3 安全与稳定性

1. `timeout` + `circuit breaker`，防止通知链路被 LLM 卡死。
2. LLM 输出必须经过 schema 校验（JSON decode + 字段白名单）。
3. 不信任 LLM 的命令建议，不自动执行。

### 8.4 成本控制

1. 仅在失败或超时时触发 LLM。
2. 同类错误命中缓存时跳过 LLM。
3. 提供 `--llm off` 全局开关。

## 9. 目录结构建议

```text
cmd/runpost/main.go
internal/orchestrator/
internal/executor/
internal/capture/
internal/policy/
internal/formatter/
internal/notifier/
internal/llm/
internal/config/
internal/store/
```

## 10. 里程碑拆分

1. `M1`：重构为模块化结构，补齐配置与策略层。
2. `M2`：实现 webhook + telegram 通知，补充集成测试。
3. `M3`：接入 LLM 摘要（Go 直连 API），实现失败降级。
4. `M4`：可选 Python worker，支持多模型路由与 A/B prompt。

## 11. 测试策略

1. 单元测试
- `executor` 退出码映射、超时行为。
- `policy` 条件组合。
- `formatter` 模板渲染正确性。

2. 集成测试
- 用假命令验证通知触发条件。
- 用 mock server 验证 webhook/LLM 接口兼容。

3. 端到端测试
- 真实执行 `success/fail/timeout` 三类命令，校验最终退出码不变。

## 12. 向后兼容原则

1. 无通知配置时，行为必须与当前版本一致（只执行并透传）。
2. 通知/LLM 模块异常不能改变子命令退出码语义。
3. 新增配置项均提供默认值，避免升级后启动失败。
