# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Purpose

This is a system design learning repository. The owner is actively learning — prioritize explaining **why** things work, not just providing working code. When implementing something, explain the core concept, the tradeoffs, and what would break or change at scale.

## Plan

Build 11 isolated components from scratch in Go, then take the URL shortener logic into a separate repo for a production-grade Dockerized cluster.

### Part 1: Isolated Components
```
[ ] 00-tcp/
[ ] 01-http-server/
[ ] 02-websocket/
[ ] 03-rate-limiter/
[ ] 04-consistent-hashing/
[ ] 05-load-balancer/        ← can use consistent hashing from 04
[ ] 06-distributed-cache/    ← can use consistent hashing from 04
[ ] 07-message-queue/
[ ] 08-log-aggregator/
[ ] 09-task-scheduler/
[ ] 10-url-shortener/
```

### Part 2: Grand Finale (separate repo)
Take the URL shortener logic and build a production cluster with Docker Compose:
- Stage 1: App + PostgreSQL
- Stage 2: + Redis cache with cache-aside pattern
- Stage 3: 3 app instances + Nginx load balancer
- Stage 4 (stretch): Prometheus + Grafana observability

## Repository Structure

Each component is fully isolated — its own directory, its own `go.mod`, runnable and testable independently.

```
SYS-Design/
├── CLAUDE.md
├── 00-tcp/
│   ├── go.mod
│   └── main.go
├── 01-http-server/
│   ├── go.mod
│   └── main.go
└── ...
```

## Go Commands

Each component is its own module. Always `cd` into the component directory first.

```bash
cd 00-tcp
go run main.go          # run
go test ./...           # test all packages in component
go test -run TestName   # run a single test
go build ./...          # build
```

## Approach for Each Component

1. Build it from scratch — no importing libraries that do the thing being learned (e.g. no `gorilla/websocket` for the WebSocket component)
2. Get it working with a simple demo (`main.go` that exercises the core logic)
3. Add tests for the interesting edge cases
4. Leave comments explaining the protocol/algorithm decisions, not just what the code does
