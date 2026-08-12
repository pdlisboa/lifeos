-- +goose Up
-- 0009_agents.sql — custo e rastreabilidade dos agentes (Fatia 3,
-- 02-modelo-de-dados.md §11 / 01-arquitetura.md §6).

CREATE TABLE agent_call (
    id             uuid PRIMARY KEY,
    user_id        uuid NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,

    task           text NOT NULL,              -- 'assess','plan_track','generate_action',
                                                -- 'curate','coach','probe'
    tier           text NOT NULL CHECK (tier IN ('fast','balanced','strong')),
    model          text NOT NULL,              -- modelo concreto resolvido
    prompt_version text NOT NULL,              -- 'assessor.v3'

    input_tokens   int,
    output_tokens  int,
    cost_usd       numeric(10,6),
    latency_ms     int,

    status         text NOT NULL CHECK (status IN ('ok','invalid_output','provider_error',
                                                    'budget_blocked','timeout')),
    error          text,
    cache_hit      boolean NOT NULL DEFAULT false,
    input_hash     text,                       -- cache por hash + prompt_version

    -- Desvio deliberado de 02-modelo-de-dados.md §11 (que não tem esta
    -- coluna): sem guardar a saída, um cache hit não teria o que devolver e
    -- a promessa de 01-arquitetura.md §6.1 ("reavaliar a mesma evidência não
    -- custa de novo") não se sustentaria. NULL quando status != 'ok'.
    output         jsonb,

    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON agent_call (user_id, created_at DESC);
CREATE INDEX ON agent_call (input_hash, prompt_version) WHERE status = 'ok';
CREATE INDEX ON agent_call (task, prompt_version, created_at DESC);

-- teto mensal (01-arquitetura.md §6.3)
CREATE TABLE agent_budget (
    user_id      uuid NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    period_on    date NOT NULL,                -- primeiro dia do mês
    limit_usd    numeric(10,2) NOT NULL,
    spent_usd    numeric(10,6) NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, period_on)
);
