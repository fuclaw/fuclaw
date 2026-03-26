# fuclaw

[English](README.md) | 简体中文

`fuclaw` 是一个用 Go 编写的轻量级、容器优先（container-first）的 Agent 编排器。它由宿主机（Host）与容器内 Agent 两部分组成：宿主机负责消息/调度/容器生命周期管理，Agent 在隔离容器中执行，默认使用 SQLite 持久化，并提供一个本地 Console 通道用于验证闭环。

## 功能特性

- Host/Agent 分离：宿主机编排，Agent 容器内执行
- SQLite 持久化（默认：`./messages.db`）
- Console 通道：本地终端交互（无需接入外部 IM）
- 内置任务调度（cron/interval/once）与运行记录
- 基于文件系统的 IPC watcher（Agent → Host 的外发消息）
- OpenAI 兼容模型配置，全部通过环境变量（`.env`）

## 架构入口（高层）

- Host 入口：[main.go](file:///Users/wzqnls/Documents/go-ws/fuclaw/cmd/fuclaw/main.go)
- Agent 入口（容器内）：[main.go](file:///Users/wzqnls/Documents/go-ws/fuclaw/cmd/agent/main.go)
- 容器运行器：[runner.go](file:///Users/wzqnls/Documents/go-ws/fuclaw/internal/container/runner.go)
- 端到端测试：[e2e](file:///Users/wzqnls/Documents/go-ws/fuclaw/e2e)

## 前置条件

- Go 1.22+（本地开发）
- Docker（运行 agent 容器与集成测试）

## 快速开始（本地启动宿主机）

1. 创建 `.env`：

```bash
cp .env.example .env
```

2. 在 `.env` 中填写必需项：

- `FUCLAW_OPENAI_BASE_URL`
- `FUCLAW_OPENAI_MODEL`
- `FUCLAW_OPENAI_API_KEY`

3. 构建 agent 镜像并启动 host：

```bash
make build-agent
make run-host
```

## 全容器化运行（宿主机也在容器里）

构建两个镜像：

```bash
make build-agent
make build-host
```

运行宿主机容器：

```bash
make run-host-docker
```

等价 docker 命令：

```bash
docker run --rm -it \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v "$(pwd)":"$(pwd)" \
  -w "$(pwd)" \
  --env-file .env \
  fuclaw-host:latest
```

说明：

- 必须挂载 `docker.sock`：宿主程序需要通过 Docker API 拉起 agent 容器。
- bind mount 必须使用宿主机绝对路径：宿主程序通过 `docker.sock` 连接到宿主机 Docker daemon 时，agent 容器的 bind mount `Source` 路径在宿主机侧解析。
- 最省事方式是把整个项目目录按“绝对路径 -> 同路径”挂载（如上面的 `-v "$(pwd)":"$(pwd)" -w "$(pwd)"`），这样默认相对路径（`./groups`, `./data/ipc`, `./messages.db`）都能在宿主机侧正确解析并用于挂载。

## 配置

全部配置通过环境变量注入，参考 [.env.example](file:///Users/wzqnls/Documents/go-ws/fuclaw/.env.example)。

必需项：

- `FUCLAW_OPENAI_BASE_URL`
- `FUCLAW_OPENAI_MODEL`
- `FUCLAW_OPENAI_API_KEY`

常用项：

- `FUCLAW_DB_PATH`（默认：`./messages.db`）
- `FUCLAW_DOCKER_IMAGE`（默认：`fuclaw-agent:latest`）
- `FUCLAW_GROUPS_DIR`（默认：`./groups`）
- `FUCLAW_IPC_DIR`（默认：`./data/ipc`）
- `FUCLAW_POLL_INTERVAL`（默认：`2s`）
- `FUCLAW_SCHEDULER_INTERVAL`（默认：`1m`）
- `FUCLAW_IPC_INTERVAL`（默认：`1s`）

## 测试

单元测试：

```bash
go test ./...
```

集成/端到端测试（需要 Docker）：

```bash
make build-agent
go test -tags=integration ./...
```

## 在 Trae 中调试

项目提供了 VSCode 兼容的 Go 调试配置（Trae 可直接复用）：

- [launch.json](file:///Users/wzqnls/Documents/go-ws/fuclaw/.vscode/launch.json)

它会以调试模式启动 `cmd/fuclaw`，并通过 `envFile` 读取 `${workspaceFolder}/.env`。

## 文档

- [SPEC.md](file:///Users/wzqnls/Documents/go-ws/fuclaw/docs/SPEC.md)
- [REQUIREMENTS.md](file:///Users/wzqnls/Documents/go-ws/fuclaw/docs/REQUIREMENTS.md)

## License

MIT License，见 [LICENSE](file:///Users/wzqnls/Documents/go-ws/fuclaw/LICENSE)。

