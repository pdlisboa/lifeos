-- name: InsertProposal :exec
INSERT INTO proposal (id, user_id, goal_id, kind, payload, rationale, agent_call_id, status, expires_at, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10);

-- name: GetProposal :one
SELECT id, user_id, goal_id, kind, payload, rationale, agent_call_id, status,
    resolved_at, reject_reason, expires_at, created_at
FROM proposal WHERE id = $1 AND user_id = $2;

-- name: ListProposalsByUserAndStatus :many
SELECT id, user_id, goal_id, kind, payload, rationale, agent_call_id, status,
    resolved_at, reject_reason, expires_at, created_at
FROM proposal WHERE user_id = $1 AND status = $2 ORDER BY created_at DESC;

-- name: ListProposalsByUserStatusAndGoal :many
SELECT id, user_id, goal_id, kind, payload, rationale, agent_call_id, status,
    resolved_at, reject_reason, expires_at, created_at
FROM proposal WHERE user_id = $1 AND status = $2 AND goal_id = $3 ORDER BY created_at DESC;

-- name: ResolveProposal :exec
UPDATE proposal SET status = $2, resolved_at = $3, reject_reason = $4 WHERE id = $1;
