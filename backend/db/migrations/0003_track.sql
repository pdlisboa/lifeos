-- +goose Up
-- 0003_track.sql — track, milestone, milestone_competency
-- Conforme docs/modulos/metas/02-modelo-de-dados.md §5.

CREATE TABLE track (
    id           uuid PRIMARY KEY,
    goal_id      uuid NOT NULL REFERENCES goal(id) ON DELETE CASCADE,
    user_id      uuid NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    version      int  NOT NULL DEFAULT 1,
    generated_by text NOT NULL CHECK (generated_by IN ('agent','user')),
    accepted_at  timestamptz,
    superseded_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);

-- só uma trilha vigente por meta
CREATE UNIQUE INDEX track_one_current
    ON track (goal_id)
    WHERE superseded_at IS NULL AND accepted_at IS NOT NULL;

CREATE TABLE milestone (
    id            uuid PRIMARY KEY,
    track_id      uuid NOT NULL REFERENCES track(id) ON DELETE CASCADE,
    goal_id       uuid NOT NULL REFERENCES goal(id) ON DELETE CASCADE,
    user_id       uuid NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,

    ordinal       int  NOT NULL,
    title         text NOT NULL,
    completion_criteria text NOT NULL,
    status        text NOT NULL DEFAULT 'locked'
                  CHECK (status IN ('locked','current','completed','skipped')),
    completed_at  timestamptz,

    UNIQUE (track_id, ordinal)
);

CREATE TABLE milestone_competency (
    milestone_id  uuid NOT NULL REFERENCES milestone(id) ON DELETE CASCADE,
    competency_id uuid NOT NULL REFERENCES competency(id) ON DELETE CASCADE,
    PRIMARY KEY (milestone_id, competency_id)
);

CREATE INDEX ON milestone (goal_id, status);
