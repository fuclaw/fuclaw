# fuclaw

English | [简体中文](README-cn.md)

A lightweight container-first agent orchestrator written in Go. `fuclaw` runs a host process (also containerizable) that spawns isolated agent containers per invocation, with SQLite persistence and a minimal console channel.

## Features

- Host/agent split: host orchestrates, agent runs in an isolated container
- SQLite persistence (default: `./messages.db`)
- Console channel for local interaction (no external IM integration required)
- Built-in scheduler loop (cron/interval/once) with run history
- File-based IPC watcher for agent-to-host outbound messages
- OpenAI-compatible model configuration via environment variables (`.env`)

## Architecture (high-level)

- Host binary: [cmd/fuclaw/main.go](file:///Users/wzqnls/Documents/go-ws/fuclaw/cmd/fuclaw/main.go)
- Agent binary (container entrypoint): [cmd/agent/main.go](file:///Users/wzqnls/Documents/go-ws/fuclaw/cmd/agent/main.go)
- Container runner: [runner.go](file:///Users/wzqnls/Documents/go-ws/fuclaw/internal/container/runner.go)
- E2E tests: [e2e](file:///Users/wzqnls/Documents/go-ws/fuclaw/e2e)

## Prerequisites

- Go 1.22+ (for local development)
- Docker (for agent containers and integration tests)

## Quick Start (Local Host)

1. Create `.env`:

```bash
cp .env.example .env
```

2. Fill required variables in `.env`:

- `FUCLAW_OPENAI_BASE_URL`
- `FUCLAW_OPENAI_MODEL`
- `FUCLAW_OPENAI_API_KEY`

3. Build agent image and run host:

```bash
make build-agent
make run-host
```

## Fully Containerized Run (Host in Docker)

Build both images:

```bash
make build-agent
make build-host
```

Run host container:

```bash
make run-host-docker
```

Equivalent docker command:

```bash
docker run --rm -it \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v "$(pwd)":"$(pwd)" \
  -w "$(pwd)" \
  --env-file .env \
  fuclaw-host:latest
```

Notes:

- `docker.sock` must be mounted because the host uses Docker API to spawn agent containers.
- Bind mounts must use absolute host paths. The simplest approach is mapping the whole project directory to the same absolute path inside the host container (as above), so relative defaults (`./groups`, `./data/ipc`, `./messages.db`) resolve correctly on the host.

## Configuration

All configuration is via environment variables (see [.env.example](file:///Users/wzqnls/Documents/go-ws/fuclaw/.env.example)).

Required:

- `FUCLAW_OPENAI_BASE_URL`
- `FUCLAW_OPENAI_MODEL`
- `FUCLAW_OPENAI_API_KEY`

Common:

- `FUCLAW_DB_PATH` (default: `./messages.db`)
- `FUCLAW_DOCKER_IMAGE` (default: `fuclaw-agent:latest`)
- `FUCLAW_GROUPS_DIR` (default: `./groups`)
- `FUCLAW_IPC_DIR` (default: `./data/ipc`)
- `FUCLAW_POLL_INTERVAL` (default: `2s`)
- `FUCLAW_SCHEDULER_INTERVAL` (default: `1m`)
- `FUCLAW_IPC_INTERVAL` (default: `1s`)

## Testing

Unit tests:

```bash
go test ./...
```

Integration/E2E tests (Docker required):

```bash
make build-agent
go test -tags=integration ./...
```

## Debugging in Trae

This repository includes a VSCode-compatible debug configuration that Trae can reuse:

- [.vscode/launch.json](file:///Users/wzqnls/Documents/go-ws/fuclaw/.vscode/launch.json)

It launches `cmd/fuclaw` with `envFile` pointing to `${workspaceFolder}/.env`.

## Docs

- [SPEC.md](file:///Users/wzqnls/Documents/go-ws/fuclaw/docs/SPEC.md)
- [REQUIREMENTS.md](file:///Users/wzqnls/Documents/go-ws/fuclaw/docs/REQUIREMENTS.md)

## License

MIT License. See [LICENSE](file:///Users/wzqnls/Documents/go-ws/fuclaw/LICENSE).
