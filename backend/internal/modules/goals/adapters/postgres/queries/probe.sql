-- name: InsertProbe :exec
INSERT INTO probe (id, goal_id, user_id, status, asked_count, closed_at)
VALUES ($1,$2,$3,$4,$5,$6);

-- name: UpdateProbe :exec
UPDATE probe SET status = $2, asked_count = $3, closed_at = $4 WHERE id = $1;

-- name: InsertProbeTurn :exec
INSERT INTO probe_turn (id, probe_id, competency_id, ordinal, question, answer, inferred_level, answered_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8);

-- name: UpdateProbeTurnAnswer :exec
UPDATE probe_turn SET answer = $2, answered_at = $3 WHERE id = $1;

-- name: GetProbeByGoal :one
SELECT id, goal_id, user_id, status, asked_count, closed_at
FROM probe WHERE goal_id = $1;

-- name: ListProbeTurnsByProbe :many
SELECT id, probe_id, competency_id, ordinal, question, answer, inferred_level, answered_at
FROM probe_turn WHERE probe_id = $1 ORDER BY ordinal;
