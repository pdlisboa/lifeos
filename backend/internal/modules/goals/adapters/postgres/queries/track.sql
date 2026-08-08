-- name: InsertTrack :exec
INSERT INTO track (id, goal_id, user_id, version, generated_by, accepted_at, superseded_at)
VALUES ($1,$2,$3,$4,$5,$6,$7);

-- name: InsertMilestone :exec
INSERT INTO milestone (id, track_id, goal_id, user_id, ordinal, title, completion_criteria, status, completed_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9);

-- name: InsertMilestoneCompetency :exec
INSERT INTO milestone_competency (milestone_id, competency_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: UpdateMilestone :exec
UPDATE milestone SET status = $2, completed_at = $3 WHERE id = $1;

-- name: GetCurrentTrack :one
SELECT id, goal_id, user_id, version, generated_by, accepted_at, superseded_at
FROM track WHERE goal_id = $1 AND superseded_at IS NULL
ORDER BY version DESC LIMIT 1;

-- name: ListMilestonesByTrack :many
SELECT id, track_id, goal_id, user_id, ordinal, title, completion_criteria, status, completed_at
FROM milestone WHERE track_id = $1 ORDER BY ordinal;

-- name: ListMilestoneCompetenciesByMilestones :many
SELECT milestone_id, competency_id FROM milestone_competency
WHERE milestone_id = ANY($1::uuid[]);

-- name: GetMilestone :one
SELECT id, track_id, goal_id, user_id, ordinal, title, completion_criteria, status, completed_at
FROM milestone WHERE id = $1 AND user_id = $2;
