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

```
