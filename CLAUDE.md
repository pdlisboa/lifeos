# LifeOS

Sistema pessoal modular do Phablo. Vários módulos de domínio (Metas, Estudo, Saúde,
Financeiro...) num único produto, com agentes de IA especializados por módulo.

**Estado atual: planejamento completo do módulo Metas, zero código escrito.**
A próxima entrega é a Fatia 0 (esqueleto + deploy).

---

## Leia isto primeiro

O raciocínio inteiro está em `docs/modulos/metas/`. **Leia antes de escrever
código** — as decisões abaixo têm justificativa lá, e refazê-las por conta própria
vai contra meses de contexto.

| Documento | O que responde |
|---|---|
| `00-negocio.md` | Problema, princípios (P1–P9), regras de negócio (RN-01–11), mecânicas antidesistência (§7), domain packs, agentes, métricas |
| `01-arquitetura.md` | Monolito modular em Go, camadas, eventos, agent gateway, infra, **roadmap fatiado (§14)** |
| `02-modelo-de-dados.md` | DDL completo pronto para migration, consultas críticas, rastreabilidade RN → constraint |
| `03-api.md` | Decisões de contrato. O contrato em si é `backend/api/openapi.yaml` |
| `04-agentes.md` | Packs, prompts, schemas de saída, evals, modos de falha |

Artefatos já escritos e validados:

```
backend/api/openapi.yaml                                  # fonte da verdade do contrato
backend/internal/modules/goals/packs/{golang,english}.yaml
backend/internal/modules/goals/adapters/agents/prompts/*.md    # 5 prompts versionados
backend/internal/modules/goals/adapters/agents/schemas/*.json  # 5 JSON Schemas
```

---

## O problema que o produto resolve

O Phablo desiste de metas por não ver evolução nem futuro. Tudo no sistema serve a
isso. Antes de aceitar qualquer feature, pergunte: **isso torna a evolução visível,
o futuro concreto, ou o próximo passo mais barato?** Se não, provavelmente não entra.

Consequência prática: a métrica que importa é sobrevivência de meta em 60 dias, não
tempo de uso do app. Um app de metas que prende o usuário dentro dele está falhando —
o trabalho acontece fora.

---

## Decisões travadas (não reabrir sem conversa)

- **Backend:** Go, monolito modular, um binário, dois entrypoints (`api`, `worker`)
- **Front:** web (React + Vite) e mobile (Expo/React Native), **UI não compartilhada**
- **Banco:** Postgres. `sqlc` para queries tipadas, **sem ORM**
- **Infra:** self-host, Docker Compose no Portainer. Modelos de LLM em cloud
- **Sem offline/sync** no mobile (v1)
- **Push:** Expo Push Notifications
- **Eventos:** in-process + outbox no Postgres. **Sem Kafka/NATS/Redis**
- **Fila de jobs:** Postgres (`SELECT ... FOR UPDATE SKIP LOCKED`), sem Redis
- **API:** REST + OpenAPI gerando servidor Go e cliente TS. Sem gRPC/GraphQL
- **Auth:** própria, simples, argon2id. Sem Keycloak/Authelia (um usuário)
- **Packs de domínio:** `golang` e `english` apenas, em YAML no repo

Racional completo em `01-arquitetura.md` §12 (ADRs).

---

## Regras de código

**Fronteira entre módulos (a mais importante):**

> Um módulo de domínio nunca importa outro módulo de domínio. Comunicação só por
> eventos. Módulos importam `platform`; `platform` nunca importa módulo de domínio.

Existe um teste de arquitetura para isso. Se ele quebrar, o problema é o import, não o teste.

**Camadas:** `domain` não importa nada do projeto nem I/O — é regra pura e testável
sem banco. `app` orquestra (transação, domínio, persistência, eventos). `adapters`
conhecem `ports` e `domain`, nunca o contrário.

**Não crie interface para tudo.** Em Go, interface nasce no consumidor e só quando
há segunda implementação plausível ou necessidade real de fake em teste.

**Contrato primeiro.** Mudou `openapi.yaml`? Regenere servidor Go e cliente TS no
mesmo commit. O objetivo é que esquecer um dos fronts vire erro de compilação.

**Migrations sempre para frente.** Sem `down`. Coluna `NOT NULL` nova entra em dois
passos (nullable → backfill → constraint).

**Nada de agente escreve direto no banco.** Agentes produzem propostas; o núcleo aplica.

---

## Armadilhas específicas deste domínio

Coisas que parecem melhorias e quebram o produto:

| Não faça | Por quê |
|---|---|
| Streak consecutiva que zera | RN-11/P6. Quebrar a streak é gatilho de abandono. A métrica é "18 dos últimos 30 dias" |
| Tratar nível `null` como `0` | `null` = não medido, `0` = não iniciado. Achatar os dois infla o progresso e mata a credibilidade do gráfico |
| Mostrar "quanto falta" como métrica principal | P2. A régua é o eu passado, nunca o ideal |
| Mudar nível de competência com uma avaliação só | RN-04. Duas concordantes, ou aceite explícito |
| Editar ou apagar evidência | RN-06. Imutável — há `RULE ... DO INSTEAD NOTHING` no banco. Correção é `supersedes_id` |
| Deixar meta ativa sem próxima ação | RN-03. A transação que fecha uma ação gera a próxima, com fallback sem LLM |
| Gerar lista de ações para escolher | P4. Escolher é atrito. Uma ação, sempre |
| Coach que cobra ou usa urgência | Em 30 dias, reconhecimento ≥ cobrança (RN-08) |
| Curador entregar link sem verificar HTTP | Link morto contamina a confiança em todo o resto do sistema |
| Meta encerrada sem debrief | RN-10. É a única fricção obrigatória, e é onde está o aprendizado |

---

## Roadmap (`01-arquitetura.md` §14)

Cada fatia funciona sozinha. **Valor antes de IA** — a ordem é deliberada.

| Fatia | Entrega | Pronto quando |
|---|---|---|
| **0** | Esqueleto: repo, compose, Postgres, migrations, `/healthz`, login | acessa por HTTPS e faz login do celular |
| 1 | Metas manuais, sem agente nenhum, web só | cadastra a meta de Go à mão e registra 3 evidências |
| 2 ⭐ | **Painel de Delta** | você olha a tela e *vê* sua evolução. Sem IA. Prova a tese do produto |
| 3 | Agent gateway + A2 (prática) + A1 (trilha), pack `golang` | abre o app e tem um exercício de Go no seu nível |
| 4 | A3 avaliador (diário `balanced` + consolidação semanal `strong`) | cola código, recebe crítica específica, nível muda sozinho |
| 5 | Mobile Expo + push | fecha a ação do dia pelo celular |
| 6 | A5 Coach, detecção de risco/estagnação, snapshot semanal | some 5 dias e o app te traz de volta sem te culpar |
| 7 | Pack `english`, evidência de áudio | valida a abstração de Domain Pack |
| 8 | A4 Curador com busca real | — |

Se a Fatia 2 não motivar o Phablo sem IA nenhuma, agente nenhum salva depois.

---

## Decisões pendentes da Fatia 0

Perguntar antes de começar:

1. **Module path** do `go.mod` — `github.com/phablo/lifeos`?
2. **Acesso** — domínio público com TLS automático via Caddy, ou rede local/Tailscale?
3. **Escopo** — stack completa (caddy + api + worker + postgres + backup) ou mínima (api + postgres)?

---

## Ambiente

- Docker + Portainer na máquina do Phablo (deploy como Stack)
- O Phablo é o único usuário. Otimize para **simplicidade de operação**, não escala
- Backup do Postgres não é opcional: o histórico *é* o produto. Perder o banco é
  perder a comparação com o eu passado, que é a única coisa que o sistema oferece

## Comunicação

Responder em **português do Brasil**, direto e conciso. O Phablo quer entender e
evoluir o sistema, não só recebê-lo pronto — explique decisões não óbvias, mas sem
palestra.
