# runpost

`runpost` 是一个命令执行后处理器：

1. 实时透传子命令 `stdout/stderr`。
2. 保持与子命令一致的退出码语义。
3. 在命令结束后按策略发送通知。

## 功能

1. 输出尾部捕获（stdout/stderr）。
2. 通知策略：`always|failure|timeout|never`。
3. 模板化通知消息（Go `text/template`）。
4. 通知插件：`webhook`、`telegram`。
5. 多渠道并发发送，单渠道独立重试（指数退避，最多 3 次）。
6. `--dry-run` 预览通知内容（输出到 `stderr`），不真实发送。

## 构建

```bash
make build
```

构建产物输出到 `build/runpost`，与源码目录分离。

其他常用命令：

- `make test`
- `make clean`
- `make run`

## CI 与发布

仓库已配置 GitHub Actions：

1. `Test` 在 `ubuntu-latest`、`macos-latest`、`windows-latest` 上并行执行 `go test ./...`。
2. `Build` 产出以下二进制文件：
   - `runpost-linux-amd64`
   - `runpost-darwin-amd64`
   - `runpost-windows-amd64.exe`
3. 打 tag（`v*`）时会自动创建 Release，并上传三平台产物。

## 用法

```bash
./build/runpost [flags] -- <command> [args...]
```

也支持不写 `--`，直接把剩余参数作为子命令。

常用参数：

- `--config <path>`：配置文件路径（支持 json；yaml 为内置简化解析器）。不传时默认按文件名顺序加载 `~/.config/runpost/*.yaml`。
- `--notify-on <always|failure|timeout|never>`：覆盖配置中的通知策略。
- `--timeout <duration>`：子命令超时（例如 `30s`）。
- `--max-capture-bytes <n>`：stdout/stderr 捕获上限（尾部）。
- `--dry-run`：仅打印通知消息。

示例：

```bash
./build/runpost --notify-on failure --timeout 20s -- go test ./...
```

## 配置示例（YAML）

```yaml
notify_on: failure
capture:
  max_stdout_bytes: 65536
  max_stderr_bytes: 65536
channels:
  - type: webhook
    url: https://example.com/hook
    secret_env: WEBHOOK_SECRET
    timeout: 5s
  - type: telegram
    bot_token_env: TELEGRAM_BOT_TOKEN
    chat_id: "123456"
    timeout: 5s
```

## Telegram Bot 配置

### 1) 创建 Bot 并获取 Token

1. 在 Telegram 打开 `@BotFather`。
2. 发送 `/newbot`，按提示设置 bot 名称和用户名。
3. 创建成功后会得到一个 bot token（形如 `123456:ABC...`）。

### 2) 获取 chat_id

1. 先给你的 bot 发一条消息（私聊或群聊中 @bot）。
2. 调用：

```bash
curl "https://api.telegram.org/bot<你的TOKEN>/getUpdates"
```

3. 在返回 JSON 中找到 `message.chat.id`，这就是 `chat_id`。

### 3) 设置环境变量

```bash
export TELEGRAM_BOT_TOKEN='<你的TOKEN>'
```

Windows PowerShell:

```powershell
$env:TELEGRAM_BOT_TOKEN = '<你的TOKEN>'
```

### 4) 配置 runpost

```yaml
channels:
  - type: telegram
    bot_token_env: TELEGRAM_BOT_TOKEN
    chat_id: "123456789"
    timeout: 5s
```

建议先用 `--dry-run` 验证消息渲染，再执行真实命令发送通知。

## 配置加载规则

1. `--config <path>`：只加载该文件。
2. 未指定 `--config`：加载 `~/.config/runpost/*.yaml`，按文件名字典序依次合并。

默认值：

- `notify_on: failure`
- `capture.max_stdout_bytes: 65536`
- `capture.max_stderr_bytes: 65536`
- channel `timeout` 为空或非法时按 `5s` 处理

## 模板变量

消息模板可使用以下字段：

- `CommandLine`
- `ExitCode`
- `Duration`
- `TimedOut`
- `StartAt`
- `EndAt`
- `StdoutTail`
- `StderrTail`

说明：

- `template` 字段在 JSON 配置可用。
- 当前内置 YAML 解析器只支持部分字段，`template` 在 YAML 中不会生效。

## 兼容性说明

无配置或无可用通知渠道时，`runpost` 仍保持“只执行并透传”的行为，不改变子命令退出码语义。
