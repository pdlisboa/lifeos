-- name: InsertNextAction :exec
INSERT INTO next_action (id, goal_id, user_id, milestone_id, competency_id, title, detail,
    practice_format, estimated_min, minimal_variant, difficulty_hint, generated_by,
    origin_kind, status, skip_reason, resolved_at, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17);

-- name: ResolveNextAction :exec
UPDATE next_action SET status = $2, skip_reason = $3, resolved_at = $4
WHERE id = $1;

-- name: GetPendingAction :one
SELECT id, goal_id, user_id, milestone_id, competency_id, title, detail,
    practice_format, estimated_min, minimal_variant, difficulty_hint, generated_by,
    origin_kind, status, skip_reason, resolved_at, created_at
FROM next_action WHERE goal_id = $1 AND status = 'pending';

-- name: GetAction :one
SELECT id, goal_id, user_id, milestone_id, competency_id, title, detail,
    practice_format, estimated_min, minimal_variant, difficulty_hint, generated_by,
    origin_kind, status, skip_reason, resolved_at, created_at
FROM next_action WHERE id = $1 AND user_id = $2;
