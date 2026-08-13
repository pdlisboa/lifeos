-- name: UpsertEvalCase :exec
-- Marcar de novo só atualiza a nota — não é imutável como evidence (RN-06
-- é sobre a evidência em si, não sobre a curadoria de eval em cima dela).
INSERT INTO evidence_eval_case (evidence_id, note, created_at, updated_at)
VALUES ($1, $2, $3, $3)
ON CONFLICT (evidence_id) DO UPDATE SET note = EXCLUDED.note, updated_at = EXCLUDED.updated_at;

-- name: ReplaceEvalCaseScores :exec
DELETE FROM evidence_eval_case_score WHERE evidence_id = $1;

-- name: InsertEvalCaseScore :exec
INSERT INTO evidence_eval_case_score (evidence_id, competency_id, level) VALUES ($1, $2, $3);

-- name: GetEvalCase :one
SELECT evidence_id, note, created_at, updated_at FROM evidence_eval_case WHERE evidence_id = $1;

-- name: ListEvalCasesByEvidences :many
SELECT evidence_id, note, created_at, updated_at FROM evidence_eval_case
WHERE evidence_id = ANY(sqlc.arg(evidence_ids)::uuid[]);

-- name: ListEvalCaseScores :many
SELECT evidence_id, competency_id, level FROM evidence_eval_case_score WHERE evidence_id = $1;

-- name: ListEvalCaseScoresForEvidences :many
SELECT evidence_id, competency_id, level FROM evidence_eval_case_score
WHERE evidence_id = ANY(sqlc.arg(evidence_ids)::uuid[]);

-- name: DeleteEvalCase :exec
DELETE FROM evidence_eval_case WHERE evidence_id = $1;

-- name: ListEvalCasesForExport :many
-- Todos os casos marcados, mais antigo primeiro — é a ordem que numera os
-- arquivos exportados (04-agentes.md §6.1).
SELECT ec.evidence_id, ec.note, ec.created_at,
    e.goal_id, e.user_id, e.action_id, e.kind, e.title, e.body,
    e.external_url, e.language
FROM evidence_eval_case ec
JOIN evidence e ON e.id = ec.evidence_id
ORDER BY ec.created_at ASC;
