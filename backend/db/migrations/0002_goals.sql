-- +goose Up
-- 0002_goals.sql — goal, competency, competency_level_event, trigger de RN-02
-- Conforme docs/modulos/metas/02-modelo-de-dados.md §4.
-- Sem seção de rollback: migration é sempre para frente (§16 do modelo de dados).

CREATE TABLE goal (
    id                uuid PRIMARY KEY,
    user_id           uuid NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,

    title             text NOT NULL,
    archetype         text NOT NULL CHECK (archetype IN ('skill','habit','project','metric')),
    pack_id           text NOT NULL,
    why               text,
    definition_of_done text,
    horizon_on        date,

    status            text NOT NULL DEFAULT 'draft'
                      CHECK (status IN ('draft','active','at_risk','stagnant',
                                        'paused','completed','abandoned')),
    scope_mode        text NOT NULL DEFAULT 'normal'
                      CHECK (scope_mode IN ('normal','minimal')),
    paused_until      date,

    activated_at      timestamptz,
    last_activity_at  timestamptz,
    closed_at         timestamptz,

    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT goal_activation_requires_dod
        CHECK (status = 'draft' OR definition_of_done IS NOT NULL),
    CONSTRAINT goal_closed_has_timestamp
        CHECK (status NOT IN ('completed','abandoned') OR closed_at IS NOT NULL)
);

CREATE INDEX ON goal (user_id, status);
CREATE INDEX ON goal (user_id, last_activity_at) WHERE status IN ('active','at_risk');

-- RN-02: máximo 3 metas ativas. DEFERRABLE porque "pausar A e ativar B" passa
-- por um instante com 4 linhas em status ativo dentro da mesma transação.
-- A validação amigável (com a lista de quais pausar) mora no caso de uso;
-- isto aqui é a rede de segurança que nenhum caminho esquecido consegue furar.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION enforce_max_active_goals() RETURNS trigger AS $$
BEGIN
    IF (SELECT count(*) FROM goal
        WHERE user_id = NEW.user_id
          AND status IN ('active','at_risk','stagnant')) > 3 THEN
        RAISE EXCEPTION 'RN-02: máximo de 3 metas ativas';
    END IF;
    RETURN NULL;
END $$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE CONSTRAINT TRIGGER goal_max_active
    AFTER INSERT OR UPDATE ON goal
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION enforce_max_active_goals();

CREATE TABLE competency (
    id             uuid PRIMARY KEY,
    goal_id        uuid NOT NULL REFERENCES goal(id) ON DELETE CASCADE,
    user_id        uuid NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,

    pack_key       text NOT NULL,
    label          text NOT NULL,
    weight         numeric(3,2) NOT NULL DEFAULT 0.10 CHECK (weight > 0 AND weight <= 1),

    -- cache derivado de competency_level_event (02-modelo-de-dados.md §1)
    current_level  smallint CHECK (current_level BETWEEN 0 AND 5),  -- NULL = desconhecido
    confidence     text NOT NULL DEFAULT 'unknown'
                   CHECK (confidence IN ('unknown','low','medium','high')),
    baseline_level smallint CHECK (baseline_level BETWEEN 0 AND 5),
    last_evidence_at timestamptz,

    retired_at     timestamptz,
    retired_reason text CHECK (retired_reason IN
                     ('removed_from_pack','user_dropped','merged')),
    merged_into_id uuid REFERENCES competency(id) ON DELETE SET NULL,

    created_at     timestamptz NOT NULL DEFAULT now(),

    UNIQUE (goal_id, pack_key)
);

CREATE INDEX ON competency (goal_id);
CREATE INDEX ON competency (user_id, last_evidence_at);

CREATE TABLE competency_level_event (
    id              uuid PRIMARY KEY,
    competency_id   uuid NOT NULL REFERENCES competency(id) ON DELETE CASCADE,
    user_id         uuid NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,

    from_level      smallint CHECK (from_level BETWEEN 0 AND 5),
    to_level        smallint NOT NULL CHECK (to_level BETWEEN 0 AND 5),
    confidence      text NOT NULL CHECK (confidence IN ('low','medium','high')),

    source          text NOT NULL CHECK (source IN
                      ('probe','self','assessment','weekly_consolidation','decay','revalidation')),
    -- assessment_id fica sem FK até a Fatia 4 criar a tabela assessment
    -- (02-modelo-de-dados.md §16: dependência circular resolvida via ALTER TABLE futuro).
    assessment_id   uuid,
    rationale       text NOT NULL,
    occurred_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON competency_level_event (competency_id, occurred_at DESC);
CREATE INDEX ON competency_level_event (user_id, occurred_at DESC);
