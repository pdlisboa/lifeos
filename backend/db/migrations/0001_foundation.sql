-- +goose Up
-- 0001_foundation.sql — app_user, user_session, device
-- Exatamente como em docs/modulos/metas/02-modelo-de-dados.md §3.
-- Sem seção de rollback: migration é sempre para frente (§16 do modelo de dados).

CREATE TABLE app_user (
    id            uuid PRIMARY KEY,
    email         text NOT NULL UNIQUE,
    password_hash text NOT NULL,               -- argon2id
    timezone      text NOT NULL DEFAULT 'America/Sao_Paulo',
    quiet_hours   int4range,                   -- ex: [22,7) — janela sem push (RN-08)
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE user_session (
    id          uuid PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    token_hash  text NOT NULL UNIQUE,          -- guarda o hash, nunca o token
    kind        text NOT NULL CHECK (kind IN ('web','mobile')),
    user_agent  text,
    expires_at  timestamptz NOT NULL,
    revoked_at  timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ON user_session (user_id) WHERE revoked_at IS NULL;

CREATE TABLE device (
    id            uuid PRIMARY KEY,
    user_id       uuid NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    expo_token    text NOT NULL UNIQUE,        -- §8.3 da arquitetura
    platform      text NOT NULL CHECK (platform IN ('android','ios','web')),
    last_seen_at  timestamptz NOT NULL DEFAULT now(),
    disabled_at   timestamptz                  -- token rejeitado pelo Expo
);
