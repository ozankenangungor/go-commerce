FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/user-service ./cmd/user-service

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /out/user-service /app/user-service
COPY --from=builder /src/internal/user/db/migrations /app/internal/user/db/migrations

EXPOSE 50051

ENTRYPOINT ["/app/user-service"]
