-- name: InsertCompetency :exec
INSERT INTO competency (id, goal_id, user_id, pack_key, label, weight,
    current_level, confidence, baseline_level, last_evidence_at, retired_at, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12);

-- name: ListCompetenciesByGoal :many
SELECT id, goal_id, user_id, pack_key, label, weight,
    current_level, confidence, baseline_level, last_evidence_at, retired_at, created_at
FROM competency WHERE goal_id = $1 ORDER BY weight DESC, pack_key;

-- name: CountCompetenciesByGoal :one
SELECT count(*) FROM competency WHERE goal_id = $1 AND retired_at IS NULL;

-- name: GetCompetency :one
SELECT id, goal_id, user_id, pack_key, label, weight,
    current_level, confidence, baseline_level, last_evidence_at, retired_at, created_at
FROM competency WHERE id = $1 AND user_id = $2;

-- name: UpdateCompetencyState :exec
UPDATE competency SET current_level = $2, confidence = $3, baseline_level = $4, last_evidence_at = $5
WHERE id = $1;

-- name: InsertLevelEvent :exec
INSERT INTO competency_level_event
    (id, competency_id, user_id, from_level, to_level, confidence, source, assessment_id, rationale, occurred_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10);

-- name: ListLevelEvents :many
SELECT id, competency_id, user_id, from_level, to_level, confidence, source, assessment_id, rationale, occurred_at
FROM competency_level_event WHERE competency_id = $1 ORDER BY occurred_at;

-- name: ListLevelEventsForGoal :many
SELECT e.id, e.competency_id, e.user_id, e.from_level, e.to_level, e.confidence, e.source, e.assessment_id, e.rationale, e.occurred_at
FROM competency_level_event e
JOIN competency c ON c.id = e.competency_id
WHERE c.goal_id = $1
ORDER BY e.occurred_at;
