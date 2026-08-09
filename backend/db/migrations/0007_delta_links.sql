-- +goose Up
-- 0007_delta_links.sql — liga evidência a nível (Fatia 2, dá profundidade ao
-- Painel de Delta sem esperar o avaliador da Fatia 4).
--
-- competency_level_event.evidence_id: qual evidência sustentou a mudança de
-- nível, quando você mesmo define o nível a partir de algo que registrou
-- (05-ux.md §5.3 — "link para a evidência que sustentou a mudança"). Nullable
-- porque nem todo evento de nível nasce de uma evidência específica (ex.:
-- sondagem inicial, decaimento).
ALTER TABLE competency_level_event
    ADD COLUMN evidence_id uuid REFERENCES evidence(id) ON DELETE SET NULL;

CREATE INDEX ON competency_level_event (evidence_id);
