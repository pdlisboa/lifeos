-- name: InsertGoal :exec
INSERT INTO goal (id, user_id, title, archetype, pack_id, why, definition_of_done, horizon_on,
    status, scope_mode, paused_until, activated_at, last_activity_at, closed_at, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16);

-- name: UpdateGoal :execrows
UPDATE goal SET
    title = $2, why = $3, definition_of_done = $4, horizon_on = $5,
    status = $6, scope_mode = $7, paused_until = $8,
    activated_at = $9, last_activity_at = $10, closed_at = $11, updated_at = $12
WHERE id = $1;

-- name: GetGoal :one
SELECT id, user_id, title, archetype, pack_id, why, definition_of_done, horizon_on,
    status, scope_mode, paused_until, activated_at, last_activity_at, closed_at, created_at, updated_at
FROM goal WHERE id = $1 AND user_id = $2;

-- name: ListGoals :many
SELECT id, user_id, title, archetype, pack_id, why, definition_of_done, horizon_on,
    status, scope_mode, paused_until, activated_at, last_activity_at, closed_at, created_at, updated_at
FROM goal WHERE user_id = $1 ORDER BY created_at DESC;

-- name: ListGoalsByStatus :many
SELECT id, user_id, title, archetype, pack_id, why, definition_of_done, horizon_on,
    status, scope_mode, paused_until, activated_at, last_activity_at, closed_at, created_at, updated_at
FROM goal WHERE user_id = $1 AND status = ANY($2::text[]) ORDER BY created_at DESC;

-- name: CountActiveGoals :one
SELECT count(*) FROM goal
WHERE user_id = $1 AND status IN ('active','at_risk','stagnant');

-- name: LockUser :one
SELECT id FROM app_user WHERE id = $1 FOR UPDATE;
