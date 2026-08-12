-- +goose Up
-- 0008_outbox_jobs.sql — plataforma de eventos e jobs (Fatia 3,
-- 02-modelo-de-dados.md §13 / 01-arquitetura.md §5, §7).
--
-- outbox: publicado na mesma transação do escritor (evento e estado nunca
-- divergem). processed_event: idempotência dos consumidores do barramento
-- in-process — entrega é pelo menos uma vez, esta tabela garante "no
-- máximo um efeito" por consumidor. job: fila própria em Postgres,
-- SELECT ... FOR UPDATE SKIP LOCKED, sem Redis e sem River (ADR A3).

CREATE TABLE outbox (
    id            bigserial PRIMARY KEY,
    aggregate     text NOT NULL,
    aggregate_id  uuid NOT NULL,
    event_type    text NOT NULL,
    payload       jsonb NOT NULL,
    occurred_at   timestamptz NOT NULL DEFAULT now(),
    published_at  timestamptz
);

CREATE INDEX outbox_unpublished
    ON outbox (id) WHERE published_at IS NULL;

CREATE TABLE processed_event (
    consumer     text NOT NULL,
    event_id     bigint NOT NULL,
    processed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (consumer, event_id)
);

CREATE TABLE job (
    id            bigserial PRIMARY KEY,
    kind          text NOT NULL,
    payload       jsonb NOT NULL,
    unique_key    text,
    run_at        timestamptz NOT NULL DEFAULT now(),
    attempts      smallint NOT NULL DEFAULT 0,
    max_attempts  smallint NOT NULL DEFAULT 5,
    locked_at     timestamptz,
    locked_by     text,
    status        text NOT NULL DEFAULT 'queued'
                  CHECK (status IN ('queued','running','done','failed','dead')),
    last_error    text,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX ON job (kind, unique_key)
    WHERE unique_key IS NOT NULL AND status IN ('queued','running');

CREATE INDEX ON job (status, run_at) WHERE status = 'queued';
