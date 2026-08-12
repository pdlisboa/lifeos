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

-- name: UpdatePendingActionContent :execrows
-- "Upgrade no lugar" do fallback determinístico pro resultado do A2
-- (04-agentes.md §5): mesmo ID, mesmo created_at, só o conteúdo muda.
-- :execrows deixa o chamador saber se a ação ainda estava pending — se não
-- estava (já foi concluída/pulada enquanto o job rodava), 0 linhas afetadas
-- e o resultado do agente é descartado, sem sobrescrever o que a pessoa já
-- fez (03-api.md, RN-03).
UPDATE next_action SET
    title = $2, detail = $3, practice_format = $4, estimated_min = $5,
    minimal_variant = $6, milestone_id = $7, competency_id = $8, generated_by = $9
WHERE id = $1 AND status = 'pending';

-- name: ListRecentActionsByGoal :many
-- Alimenta RecentActions/RecentTitles do prompt de A2 — só ações já
-- resolvidas (pending nunca entra aqui, é sempre a que está sendo gerada).
SELECT title, status, skip_reason FROM next_action
WHERE goal_id = $1 AND status != 'pending'
ORDER BY resolved_at DESC LIMIT $2;
