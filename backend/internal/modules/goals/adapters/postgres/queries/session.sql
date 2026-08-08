-- name: InsertSession :exec
INSERT INTO session (id, goal_id, user_id, action_id, started_at, duration_min, note, energy, local_on, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10);

-- name: ListSessionsByGoal :many
SELECT id, goal_id, user_id, action_id, started_at, duration_min, note, energy, local_on, created_at
FROM session WHERE goal_id = $1 ORDER BY started_at DESC LIMIT $2;

-- name: SumSessionMinutes :one
SELECT coalesce(sum(duration_min), 0)::int AS total FROM session WHERE goal_id = $1;

-- name: ActiveLocalDates :many
SELECT s.local_on FROM session s WHERE s.goal_id = $1 AND s.local_on > current_date - $2::int
UNION
SELECT e.local_on FROM evidence e WHERE e.goal_id = $1 AND e.local_on > current_date - $2::int;
