# Fuclaw Requirements

本系统是基于 Go 语言和 Eino 框架重构的轻量级 AI 助手调度与执行框架，灵感与基础架构设计来源于 `nanoclaw` 项目。

---

## 为什么重构 (Why This Exists)

原项目 `nanoclaw` 虽然解决了一个庞大应用（OpenClaw）难以维护的问题，但它仍强依赖于 Node.js 生态、特定的 SDK（`claude-agent-sdk`）以及一些复杂的工具链。
`fuclaw` 旨在利用 Go 语言的强类型、高并发优势，结合 CloudWeGo 生态下的 `Eino` 框架，打造一个编译更简单、部署更轻量、模型切换更灵活的核心底座。

---

## 核心哲学 (Philosophy)

### 1. 极简与静态编译
采用 Go 语言编写，宿主机进程和容器内 Agent 进程均被编译为单一二进制文件。不再需要安装 Node.js 环境或复杂的依赖包管理（如 npm install），真正的“开箱即用”。

### 2. 模型与框架解耦
彻底移除了对 `claude-agent-sdk` 的依赖，转而使用 [Eino 框架](https://github.com/cloudwego/eino)。这使得系统不被绑定在单一的大模型提供商上，通过标准化的接口（如 OpenAI API 格式），可以无缝对接任何兼容的模型。

### 3. 环境与配置的纯粹化
抛弃了通过代码文件（如 `config.ts`）来硬编码配置的做法。所有关键配置项（模型参数、路径、轮询频率）均通过 `.env` 环境变量注入，保证了代码层面的纯净和环境的一致性。

### 4. 容器级隔离 (继承)
继承了原项目的核心安全理念：Agent 的执行环境必须是物理隔离的。每一次对话或任务处理，都会启动一个独立的 Docker 容器，Agent 只能访问被显式挂载的目录，无法触及宿主机核心数据。

---

## Fuclaw 与原项目 (Nanoclaw) 的差异点对比

| 特性 | Fuclaw (Go + Eino) | Nanoclaw (Node.js) |
|------|--------------------|--------------------|
| **核心语言** | Go (静态类型，编译型二进制) | TypeScript / Node.js |
| **Agent 框架** | `cloudwego/eino` (ADK 架构) | `@anthropic-ai/claude-agent-sdk` |
| **模型对接** | OpenAI 兼容接口 (通过环境变量配置) | 强绑定 Claude API / OAuth Token |
| **配置管理** | 纯环境变量 (`.env` 驱动) | `src/config.ts` (部分硬编码) + `.env` |
| **消息通道** | 默认提供 Console 终端通道 | 默认提供 WhatsApp / Telegram 等 |
| **任务调度** | 基于 Go `cron` 库的内置并发调度器 | Node.js 定时轮询 + SQLite 任务表 |
| **IPC 通信** | 基于文件系统的 JSON 轮询 (简单高效) | Stdio MCP Server + IPC 文件系统 |
| **扩展机制(Skill)**| 待定 (倾向于 Go 插件或独立工具注册) | 基于 Git Merge 的 `Claude Code` 技能注入 |
| **产物交付** | Docker 镜像 (`make build-agent`) + 主二进制 | 完整的源码仓库 + `npm start` |

---

## 当前阶段定位

`fuclaw` 目前是一个最小可行性架构（MVP）。它跑通了“终端输入 -> SQLite存储 -> Go编排器轮询 -> Docker容器拉起 -> Eino Agent执行(ReAct) -> IPC回传”的完整数据流。

**未包含的（非目标）:**
- 暂未接入 WhatsApp/Telegram 等真实 IM 渠道（当前使用 Console 作为验证通道）。
- 暂未实现复杂的 MCP 协议和原生技能系统（如 `/add-telegram` 代码合并机制）。
- 暂未实现容器内的无头浏览器（agent-browser）自动化。