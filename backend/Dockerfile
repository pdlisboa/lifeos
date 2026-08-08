# api, worker e migrate saem da mesma imagem — só o comando muda
# (01-arquitetura.md §10.1).

FROM golang:1.25-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -o /out/worker ./cmd/worker && \
    CGO_ENABLED=0 go build -o /out/migrate ./cmd/migrate

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=build /out/api /out/worker /out/migrate ./
COPY db/migrations ./db/migrations

ENV MIGRATIONS_DIR=/app/db/migrations
EXPOSE 8080

USER nobody
ENTRYPOINT ["/app/api"]
