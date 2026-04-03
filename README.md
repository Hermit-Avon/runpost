# runpost

`runpost` 是一个命令执行后处理器：

1. 实时透传子命令 `stdout/stderr`。
2. 保持与子命令一致的退出码语义。
3. 在命令结束后按策略发送通知（V2）。

## V2 功能

1. 输出尾部捕获（stdout/stderr）。
2. 通知策略：`always|failure|timeout|never`。
3. 模板化通知消息。
4. 通知插件：`webhook`、`telegram`。
5. 多渠道并发发送，单渠道独立重试（指数退避，最多 3 次）。
6. `--dry-run` 预览通知内容，不真实发送。

## 用法

```bash
runpost [flags] -- <command> [args...]
```

常用参数：

- `--config <path>`：配置文件路径（支持 json；yaml 为内置简化解析器）。不传时默认按文件名顺序加载 `~/.config/runpost/*.yaml`。
- `--notify-on <always|failure|timeout|never>`：覆盖配置中的通知策略。
- `--timeout <duration>`：子命令超时（例如 `30s`）。
- `--max-capture-bytes <n>`：stdout/stderr 捕获上限（尾部）。
- `--dry-run`：仅打印通知消息。

示例：

```bash
runpost --notify-on failure --timeout 20s -- go test ./...
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

## 兼容性说明

无配置或无可用通知渠道时，`runpost` 仍保持“只执行并透传”的行为，不改变子命令退出码语义。
