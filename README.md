# go-commerce

Production-grade Go e-commerce microservices monorepo.

This project simulates a scalable backend system handling order processing, inventory management, and user notifications using event-driven architecture.

## Architecture (planned services)
- API Gateway
- User Service
- Product Service
- Inventory Service
- Order Service
- Payment Service
- Notification Service

## Tech Stack
- Go (monorepo module + workspace)
- gRPC + Protocol Buffers (Buf)
- PostgreSQL (database-per-service)
- Redis
- RabbitMQ
- Kafka + Zookeeper
- Jaeger / OpenTelemetry local ingest ports
- REST at gateway boundaries (planned)

## Local Development

**Prerequisites:** Ensure you have `Docker`, `Go`, `Make`, `Buf`, and `golangci-lint` installed.

```bash
cp .env.example .env
make compose-up
make compose-ps
make buf-lint
make lint
make test
```

## Protobuf Contracts

- Protobuf source files live under `api/proto/...` (for example `api/proto/common/v1` and `api/proto/users/v1`).
- Generated Go files are emitted to `api/gen/go/...`.
- Lint and generate contracts with:

```bash
make buf-lint
make buf-generate
```

- Replace `<YOUR_GH_USERNAME>` in module/go_package paths when cloning this template into your own repository namespace.

## Project Layout

```text
.
├── api
│   ├── gen
│   │   └── go
│   └── proto
├── cmd
├── deployments
├── internal
├── pkg
└── .github
```

- `cmd/`: service entrypoints (to be added)
- `internal/`: private service internals
- `pkg/`: shared libraries
- `api/proto/`: protobuf source files
- `api/gen/`: generated protobuf output (do not hand-edit)
- `deployments/`: local infra and deployment assets
- `.github/`: repository templates and metadata

## Engineering Notes

- This bootstrap intentionally avoids service implementations; they will land in subsequent PRs.
- Prefer dependency injection via interfaces once services are introduced.
- Avoid global state and propagate `context.Context` through request paths.
- Conventional commits are required for repository history.
