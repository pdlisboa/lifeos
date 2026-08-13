# api, worker, migrate, seed e evalexport saem da mesma imagem — só o
# comando muda (01-arquitetura.md §10.1). seed é dev-only (guard
# ENV=development no próprio main.go), mas sai da mesma imagem pra não
# duplicar build. evalexport é a captura de casos de eval do A3
# (04-agentes.md §6.1) — lê o banco, não chama agente nenhum.

FROM golang:1.25-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -o /out/worker ./cmd/worker && \
    CGO_ENABLED=0 go build -o /out/migrate ./cmd/migrate && \
    CGO_ENABLED=0 go build -o /out/seed ./cmd/seed && \
    CGO_ENABLED=0 go build -o /out/evalexport ./cmd/evalexport

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=build /out/api /out/worker /out/migrate /out/seed /out/evalexport ./
COPY db/migrations ./db/migrations

ENV MIGRATIONS_DIR=/app/db/migrations
EXPOSE 8080

USER nobody
ENTRYPOINT ["/app/api"]
