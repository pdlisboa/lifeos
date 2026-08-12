-- +goose Up
-- 0010_proposals.sql — propostas de agente, nunca fato direto (RN-07,
-- 02-modelo-de-dados.md §8).

CREATE TABLE proposal (
    id            uuid PRIMARY KEY,
    user_id       uuid NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    goal_id       uuid REFERENCES goal(id) ON DELETE CASCADE,

    kind          text NOT NULL CHECK (kind IN
                    ('track','milestone_revision','level_change','scope_reduction',
                     'pack_draft','competency_set')),
    payload       jsonb NOT NULL,              -- o que será aplicado, já validado por schema
    rationale     text NOT NULL,               -- por que o agente propõe isso
    agent_call_id uuid REFERENCES agent_call(id) ON DELETE SET NULL,

    status        text NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','accepted','rejected','expired')),
    resolved_at   timestamptz,
    reject_reason text,

    expires_at    timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON proposal (user_id, status) WHERE status = 'pending';
CREATE INDEX ON proposal (goal_id, created_at DESC);
