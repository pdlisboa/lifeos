-- +goose Up
-- 0011_evidence_eval_case.sql — captura de casos de eval para o A3 (Fatia 4)
-- Conforme docs/modulos/metas/04-agentes.md §6.1: o conjunto de eval cresce
-- do uso real — quando uma avaliação seria claramente óbvia pra você, marca
-- o caso, sem precisar do avaliador existir ainda (`make eval-export`
-- exporta pra JSON depois).
--
-- Tabela separada de evidence, não uma coluna nela: a marcação acontece
-- depois que a evidência já existe, e evidence_no_update (0005_evidence.sql)
-- bloqueia qualquer UPDATE na tabela evidence (RN-06). Aqui não há regra de
-- imutabilidade — é curadoria sua, não o registro em si, então dá pra
-- corrigir a nota ou o gabarito sem drama.

CREATE TABLE evidence_eval_case (
    evidence_id  uuid PRIMARY KEY REFERENCES evidence(id) ON DELETE CASCADE,
    note         text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- gabarito humano por competência — o nível que você daria, não uma saída
-- de agente (o A3 ainda não existe; ele é quem esse conjunto vai avaliar).
CREATE TABLE evidence_eval_case_score (
    evidence_id   uuid NOT NULL REFERENCES evidence_eval_case(evidence_id) ON DELETE CASCADE,
    competency_id uuid NOT NULL REFERENCES competency(id) ON DELETE CASCADE,
    level         integer NOT NULL CHECK (level BETWEEN 0 AND 5),
    PRIMARY KEY (evidence_id, competency_id)
);
