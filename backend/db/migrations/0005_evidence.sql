-- +goose Up
-- 0005_evidence.sql — evidence, evidence_competency, imutabilidade (RN-06)
-- Conforme docs/modulos/metas/02-modelo-de-dados.md §6.3.

CREATE TABLE evidence (
    id            uuid PRIMARY KEY,
    goal_id       uuid NOT NULL REFERENCES goal(id) ON DELETE CASCADE,
    user_id       uuid NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    action_id     uuid REFERENCES next_action(id) ON DELETE SET NULL,

    kind          text NOT NULL CHECK (kind IN
                    ('code_snippet','repo_link','commit_diff','exercise_solution',
                     'written_text','audio_recording','listening_log','shadowing_clip','file')),
    title         text,
    body          text,
    blob_key      text,
    external_url  text,
    language      text,

    supersedes_id uuid REFERENCES evidence(id) ON DELETE SET NULL,
    local_on      date NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT evidence_has_content
        CHECK (body IS NOT NULL OR blob_key IS NOT NULL OR external_url IS NOT NULL)
);

CREATE INDEX ON evidence (goal_id, created_at DESC);
CREATE INDEX ON evidence (user_id, local_on DESC);

-- preenchida pela avaliação (Fatia 4+); a tabela já nasce aqui porque faz
-- parte do modelo de evidência e não exige a tabela assessment para existir.
CREATE TABLE evidence_competency (
    evidence_id   uuid NOT NULL REFERENCES evidence(id) ON DELETE CASCADE,
    competency_id uuid NOT NULL REFERENCES competency(id) ON DELETE CASCADE,
    PRIMARY KEY (evidence_id, competency_id)
);
CREATE INDEX ON evidence_competency (competency_id);

-- Imutabilidade garantida no banco, não na boa vontade (RN-06). Correção é
-- uma nova linha com supersedes_id — nunca um UPDATE na evidência antiga.
CREATE RULE evidence_no_update AS ON UPDATE TO evidence DO INSTEAD NOTHING;
CREATE RULE evidence_no_delete AS ON DELETE TO evidence DO INSTEAD NOTHING;
