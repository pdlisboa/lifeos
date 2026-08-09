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

-- name: ListEvidenceByGoalAndCompetencyAsc :many
SELECT e.id, e.goal_id, e.user_id, e.action_id, e.kind, e.title, e.body, e.blob_key,
    e.external_url, e.language, e.supersedes_id, e.local_on, e.created_at
FROM evidence e
JOIN evidence_competency ec ON ec.evidence_id = e.id
WHERE e.goal_id = $1 AND ec.competency_id = $2
ORDER BY e.created_at ASC LIMIT $3;

-- name: ListEvidenceByGoalAndCompetencyDesc :many
SELECT e.id, e.goal_id, e.user_id, e.action_id, e.kind, e.title, e.body, e.blob_key,
    e.external_url, e.language, e.supersedes_id, e.local_on, e.created_at
FROM evidence e
JOIN evidence_competency ec ON ec.evidence_id = e.id
WHERE e.goal_id = $1 AND ec.competency_id = $2
ORDER BY e.created_at DESC LIMIT $3;

-- name: CountEvidenceByGoal :one
SELECT count(*) FROM evidence WHERE goal_id = $1;

-- name: InsertEvidenceCompetency :exec
INSERT INTO evidence_competency (evidence_id, competency_id) VALUES ($1, $2);

-- name: ListCompetencyIDsForEvidences :many
-- traz os pares (evidence_id, competency_id) pras evidências pedidas de uma
-- vez só — evita N+1 ao montar o museu (§7.4).
SELECT evidence_id, competency_id FROM evidence_competency
WHERE evidence_id = ANY(sqlc.arg(evidence_ids)::uuid[]);
