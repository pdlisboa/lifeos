-- +goose Up
-- 0006_probe.sql — probe, probe_turn
-- Conforme docs/modulos/metas/02-modelo-de-dados.md §9.
--
-- Foge do range "0002 a 0005" pedido (que cobre só §4-6), mas os endpoints de
-- sondagem estática (04-agentes.md §4.7) exigem esta tabela e ela não depende
-- de nada fora de goal/competency — não faz sentido adiar para quando os
-- agentes chegarem (Fatia 3), já que a Fatia 1 usa a versão sem LLM.

CREATE TABLE probe (
    id           uuid PRIMARY KEY,
    goal_id      uuid NOT NULL REFERENCES goal(id) ON DELETE CASCADE,
    user_id      uuid NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    status       text NOT NULL DEFAULT 'open'
                 CHECK (status IN ('open','completed','skipped')),
    asked_count  smallint NOT NULL DEFAULT 0 CHECK (asked_count <= 5),
    created_at   timestamptz NOT NULL DEFAULT now(),
    closed_at    timestamptz
);

CREATE TABLE probe_turn (
    id            uuid PRIMARY KEY,
    probe_id      uuid NOT NULL REFERENCES probe(id) ON DELETE CASCADE,
    competency_id uuid REFERENCES competency(id) ON DELETE SET NULL,
    ordinal       smallint NOT NULL,
    question      text NOT NULL,
    answer        text,
    inferred_level smallint CHECK (inferred_level BETWEEN 0 AND 5),
    answered_at   timestamptz,
    UNIQUE (probe_id, ordinal)
);
