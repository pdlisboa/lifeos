-- name: InsertEvidence :exec
INSERT INTO evidence (id, goal_id, user_id, action_id, kind, title, body, blob_key,
    external_url, language, supersedes_id, local_on, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13);

-- name: GetEvidence :one
SELECT id, goal_id, user_id, action_id, kind, title, body, blob_key,
    external_url, language, supersedes_id, local_on, created_at
FROM evidence WHERE id = $1 AND user_id = $2;

-- name: ListEvidenceByGoalAsc :many
SELECT id, goal_id, user_id, action_id, kind, title, body, blob_key,
    external_url, language, supersedes_id, local_on, created_at
FROM evidence WHERE goal_id = $1 ORDER BY created_at ASC LIMIT $2;

-- name: ListEvidenceByGoalDesc :many
SELECT id, goal_id, user_id, action_id, kind, title, body, blob_key,
    external_url, language, supersedes_id, local_on, created_at
FROM evidence WHERE goal_id = $1 ORDER BY created_at DESC LIMIT $2;

-- name: CountEvidenceByGoal :one
SELECT count(*) FROM evidence WHERE goal_id = $1;
