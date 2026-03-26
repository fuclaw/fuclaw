# Fuclaw Specification

本规范描述了 Fuclaw 框架的架构设计、数据流、配置管理以及各组件的职责。它基于 Go 语言和 Eino 框架实现。

---

## 目录
1. [架构图](#架构图)
2. [项目结构](#项目结构)
3. [配置管理](#配置管理)
4. [核心组件](#核心组件)
5. [数据流与通信](#数据流与通信)

---

## 架构图

```
┌──────────────────────────────────────────────────────────────────────┐
│                        HOST (macOS / Linux)                          │
│                     (Go 二进制主进程: cmd/fuclaw)                      │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────────┐                  ┌────────────────────┐        │
│  │ Channel: Console │─────────────────▶│   SQLite Database  │        │
│  │ (可扩展其他通道)   │◀────────────────│   (messages.db)    │        │
│  └──────────────────┘  persist/send    └─────────┬──────────┘        │
│                                                  │                   │
│         ┌────────────────────────────────────────┘                   │
│         │                                                            │
│         ▼                                                            │
│  ┌──────────────────┐    ┌──────────────────┐    ┌───────────────┐   │
│  │  Message Loop    │    │  Scheduler Loop  │    │  IPC Watcher  │   │
│  │  (polls SQLite)  │    │  (cron based)    │    │  (file poll)  │   │
│  └────────┬─────────┘    └────────┬─────────┘    └───────────────┘   │
│           │                       │                                  │
│           └───────────┬───────────┘                                  │
│                       │ Docker API 创建临时容器                         │
│                       ▼                                              │
├──────────────────────────────────────────────────────────────────────┤
│                     CONTAINER (Alpine Linux VM)                      │
├──────────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │                    AGENT RUNNER (Go 二进制: cmd/agent)         │    │
│  │                                                              │    │
│  │  环境变量: 继承自宿主机透传的 FUCLAW_OPENAI_*                     │    │
│  │  挂载卷 (Volume Mounts):                                      │    │
│  │    • groups/{folder}/ → /workspace/group                     │    │
│  │                                                              │    │
│  │  Agent 引擎: Eino ADK (ChatModelAgent)                        │    │
│  │  工具集:                                                       │    │
│  │    • send_message (写入 /workspace/ipc/messages)              │    │
│  │                                                              │    │
│  │  输出模式: 标准输出 (stdout) 使用 FUCLAW_OUTPUT 标记包裹         │    │
│  └──────────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 项目结构

```
fuclaw/
├── .env.example                   # 环境变量配置模板
├── messages.db                    # 自动生成: SQLite 数据库文件（默认）
├── Makefile                       # 构建脚本 (build-agent, run-host)
├── go.mod / go.sum                # 依赖管理
│
├── cmd/
│   ├── fuclaw/
│   │   └── main.go                # 宿主机主程序入口
│   └── agent/
│       └── main.go                # 容器内 Agent 程序入口 (基于 Eino)
│
├── internal/
│   ├── config/                    # 配置加载模块 (.env 解析)
│   │   └── config.go
│   ├── channel/                   # 通道实现
│   │   └── console.go             # 控制台交互通道
│   ├── container/                 # 容器运行器
│   │   └── runner.go              # 基于 Docker API 的容器启停与流处理
│   ├── db/                        # 数据库层
│   │   └── db.go                  # SQLite 增删改查
│   ├── orchestrator/              # 核心编排逻辑
│   │   ├── orchestrator.go        # 消息轮询与容器调度
│   │   ├── scheduler.go           # 定时任务执行 (cron)
│   │   └── ipc.go                 # 文件系统 IPC 监听
│   └── types/                     # 共享类型与接口定义
│       └── types.go
│
├── build/
│   └── package/
│       └── Dockerfile             # Agent 镜像构建脚本 (多阶段构建)
│
├── groups/                        # 自动生成: 各群组的隔离挂载目录
└── data/ipc/                      # 自动生成: 宿主机与容器的 IPC 通信目录
```

---

## 配置管理

系统配置完全由环境变量驱动，不使用命令行参数。启动时会自动尝试加载项目根目录下的 `.env` 文件。

**核心环境变量前缀**: `FUCLAW_`

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `FUCLAW_OPENAI_BASE_URL` | 大模型 API 地址 (必须) | 无 |
| `FUCLAW_OPENAI_MODEL` | 使用的模型名称 (必须) | 无 |
| `FUCLAW_OPENAI_API_KEY` | 鉴权 Key (必须) | 无 |
| `FUCLAW_DOCKER_IMAGE` | Agent 容器使用的镜像名 | `fuclaw-agent:latest` |
| `FUCLAW_DB_PATH` | SQLite 数据库文件路径 | `./messages.db` |
| `FUCLAW_GROUPS_DIR` | 容器挂载的组目录基路径 | `./groups` |
| `FUCLAW_IPC_DIR` | IPC 文件存放目录 | `./data/ipc` |
| `FUCLAW_POLL_INTERVAL` | 消息轮询间隔 | `2s` |
| `FUCLAW_SCHEDULER_INTERVAL`| 定时任务扫描间隔 | `1m` |

---

## 核心组件

### 1. 编排器 (Orchestrator)
宿主机的核心大脑，负责协调整个系统的运转。
- **Message Loop**: 根据 `POLL_INTERVAL` 定期从 SQLite 抓取增量消息，按会话（JID）打包，然后触发 `Container Runner`。
- **Scheduler**: 利用 `github.com/robfig/cron/v3` 定期扫描 `scheduled_tasks` 表，到期的任务会触发容器执行，并记录日志。
- **IPC Watcher**: 定期扫描 `data/ipc/{group_folder}/messages` 目录，读取 Agent 工具产生的 JSON 文件，并将其路由给对应的 Channel 发送。

### 2. 容器运行器 (Container Runner)
使用 `moby/moby` 客户端与宿主机的 Docker 守护进程通信。
- 每次收到任务时，创建一个临时容器（`AutoRemove: true`）。
- 挂载该群组的专属目录到 `/workspace/group`。
- 透传大模型相关的环境变量 (`FUCLAW_OPENAI_*`)。
- 通过容器的 `stdin` 写入 JSON 格式的上下文输入 (`types.ContainerInput`)。
- 监听容器的 `stdout`，解析被 `---FUCLAW_OUTPUT_START---` 和 `---FUCLAW_OUTPUT_END---` 包裹的 JSON 输出 (`types.ContainerOutput`)。

### 3. Eino Agent (容器内部)
运行在容器内部的独立二进制程序，负责具体的业务逻辑推理。
- 基于 `cloudwego/eino` 和 `eino-ext` 构建。
- 使用 `ChatModelAgent` 范式进行 ReAct 循环推理。
- 提供内置 Tool：`send_message`。当模型决定调用此工具时，会向 `/workspace/ipc/messages` 目录写入请求文件，由宿主机 IPC 捕获处理。

---

## 数据流与通信

### 宿主机 -> 容器 (Input)
容器启动时，宿主机通过标准输入（stdin）向其写入序列化后的 JSON 字符串：
```json
{
  "prompt": "[Jan 02 3:04 PM] Console User: 你好",
  "groupFolder": "console",
  "chatJid": "console_user",
  "isMain": true,
  "assistantName": "Fuclaw"
}
```

### 容器 -> 宿主机 (Output)
Agent 的推理结果（如流式输出内容、最终状态）通过标准输出（stdout）打印。为了避免与其它日志混淆，输出必须严格包裹在 Marker 中：
```text
---FUCLAW_OUTPUT_START---
{"status":"success","result":"你好！我是 Fuclaw，有什么可以帮你的吗？"}
---FUCLAW_OUTPUT_END---
```

### 容器 -> 宿主机 (IPC 文件)
当 Agent 需要异步或主动向外部通道发消息（而非仅回复当前 Prompt）时，通过 `send_message` Tool 执行文件写入：
1. Agent 写入 `/workspace/ipc/messages/msg-123456.json` (内容为 `{ "text": "异步通知内容" }`)。
2. 由于宿主机的 `data/ipc` 与容器内的相关路径可能尚未完全映射一致（当前需保证挂载路径对齐），宿主机轮询检测到该文件后。
3. 宿主机解析 JSON，调用对应 Channel 的 `SendMessage` 方法。
4. 宿主机删除该 IPC 文件。
