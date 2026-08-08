-- +goose Up
-- 0004_actions.sql — next_action, session
-- Conforme docs/modulos/metas/02-modelo-de-dados.md §6.1-6.2.

CREATE TABLE next_action (
    id             uuid PRIMARY KEY,
    goal_id        uuid NOT NULL REFERENCES goal(id) ON DELETE CASCADE,
    user_id        uuid NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    milestone_id   uuid REFERENCES milestone(id) ON DELETE SET NULL,
    competency_id  uuid REFERENCES competency(id) ON DELETE SET NULL,

    title          text NOT NULL,
    detail         text,
    practice_format text,
    estimated_min  smallint NOT NULL DEFAULT 20 CHECK (estimated_min BETWEEN 5 AND 30),
    minimal_variant text,

    difficulty_hint text CHECK (difficulty_hint IN ('easier','same','harder')),
    generated_by   text NOT NULL CHECK (generated_by IN ('agent','user','fallback')),
    origin_kind    text CHECK (origin_kind IN ('practice','revalidation','recovery')),

    status         text NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending','completed','skipped')),
    skip_reason    text CHECK (skip_reason IN
                     ('too_hard','too_easy','no_time','not_relevant','other')),
    resolved_at    timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT action_resolved_has_timestamp
        CHECK (status = 'pending' OR resolved_at IS NOT NULL)
);

-- RN-03: no máximo uma ação pendente por meta. "Pelo menos uma" é garantido
-- pelo caso de uso, que sempre gera a próxima na mesma transação que fecha a atual.
CREATE UNIQUE INDEX next_action_one_pending
    ON next_action (goal_id) WHERE status = 'pending';

CREATE INDEX ON next_action (user_id, status) WHERE status = 'pending';

CREATE TABLE session (
    id           uuid PRIMARY KEY,
    goal_id      uuid NOT NULL REFERENCES goal(id) ON DELETE CASCADE,
    user_id      uuid NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    action_id    uuid REFERENCES next_action(id) ON DELETE SET NULL,

    started_at   timestamptz NOT NULL,
    duration_min smallint NOT NULL CHECK (duration_min > 0 AND duration_min <= 600),
    note         text,
    energy       smallint CHECK (energy BETWEEN 1 AND 5),

    -- dia LOCAL do usuário, gravado pela aplicação a partir de started_at + timezone
    -- (nunca calculado no navegador — 02-modelo-de-dados.md §6.2, CLAUDE.md armadilhas).
    local_on     date NOT NULL,

    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON session (user_id, local_on DESC);
CREATE INDEX ON session (goal_id, started_at DESC);
