# LifeOS

Backend das Fatias 0 e 1: sobe, migra o banco, autentica, e tem o módulo
Metas completo (metas manuais, sondagem, evidência, delta) sem nenhum
agente ainda. Sem UI — o front web é o próximo passo. Ver `CLAUDE.md` e
`docs/modulos/metas/` para o raciocínio completo, e
`docs/modulos/metas/fatia-1-implementacao.md` para o estado real da Fatia 1
(divergências, lacunas conhecidas).

## Subir localmente

```bash
cp .env.example .env
# edite POSTGRES_PASSWORD no .env

make up             # builda e sobe api + postgres
make migrate-up      # aplica db/migrations/0001_foundation.sql
make create-user EMAIL=voce@exemplo.com PASSWORD=uma-senha-forte
```

Verifique:

```bash
curl http://localhost:8080/api/v1/healthz
# {"status":"ok","db":true,"queue":true}

curl -i -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"voce@exemplo.com","password":"uma-senha-forte","client":"mobile"}'
# 200 com { user, token, expiresAt }

curl http://localhost:8080/api/v1/me \
  -H "Authorization: Bearer <token recebido acima>"
```

## Acessar pelo celular

Esta stack (`docker-compose.yml`) é o escopo mínimo: só `api` + `postgres`,
acessível na rede local pela porta `API_PORT` (padrão `8080`). Para acessar
de fora de casa com HTTPS de verdade (o critério de pronto da Fatia 0), falta
colocar um **Caddy** na frente, apontando um domínio para esta máquina:

```yaml
# adicionar ao docker-compose.yml quando o domínio estiver pronto
caddy:
  image: caddy:2-alpine
  restart: unless-stopped
  ports: ["80:80", "443:443"]
  volumes:
    - ./Caddyfile:/etc/caddy/Caddyfile
    - caddydata:/data
```

```
# Caddyfile
seu-dominio.com {
    reverse_proxy api:8080
}
```

Isso fica para o momento em que o domínio estiver apontado — a decisão de
arquitetura (Caddy + TLS automático) já está tomada, só falta o DNS.

## Comandos do dia a dia

| Comando | O que faz |
|---|---|
| `make up` | builda e sobe (ou atualiza) a stack |
| `make down` | derruba a stack |
| `make logs` | segue os logs da api |
| `make migrate-up` | aplica migrations pendentes |
| `make migrate-status` | lista migrations aplicadas/pendentes |
| `make create-user EMAIL=... PASSWORD=...` | cria o usuário (você) |
| `make test` / `make vet` | roda testes e vet do módulo Go |
| `make sqlc-generate` | regenera o código de acesso a banco a partir das queries |
| `make sqlc-diff` | confere se o gerado está em dia, sem sobrescrever nada |

## Banco de dados: sqlc, não ORM

Toda query SQL do módulo Metas vive em arquivos `.sql` de verdade em
`backend/internal/modules/goals/adapters/postgres/queries/`. O
[sqlc](https://sqlc.dev) lê essas queries **e o schema real** (as
migrations em `backend/db/migrations/`) e gera Go tipado — struct por
tabela, uma função por query, com os tipos batendo com as colunas de
verdade. Não escrevemos `Scan()` na mão em lugar nenhum do projeto.

**Por quê, comparado a escrever `pgx` puro (SQL + `row.Scan(...)` na mão):**
a correspondência entre a ordem das colunas do `SELECT` e a ordem dos campos
no `Scan()` deixa de ser algo que você mantém sincronizado de cabeça. Se
você adiciona uma coluna e esquece de atualizar um `Scan()` manual em outro
lugar, o código compila normal e os dados desalinham silenciosamente em
runtime — sqlc torna esse tipo de bug impossível, porque gera as duas
pontas a partir da mesma fonte. Não é ORM: você continua escrevendo SQL de
verdade, sqlc só tipa o resultado. Motivação completa em
`docs/modulos/metas/fatia-1-implementacao.md` §2.

### Como adicionar uma query nova

1. Escreva a query em `internal/modules/goals/adapters/postgres/queries/<agregado>.sql`,
   com a anotação de nome logo acima (`-- name: MinhaQuery :one` / `:many` /
   `:exec` / `:execrows` — o sufixo diz o shape do retorno que sqlc gera).
2. Rode `make sqlc-generate` (usa Docker, não precisa instalar o `sqlc`
   localmente). Isso regenera **só**
   `internal/modules/goals/adapters/postgres/sqlcgen/` — nunca edite esses
   arquivos à mão, o cabeçalho `DO NOT EDIT` é literal.
3. Em `internal/modules/goals/adapters/postgres/<agregado>.go`, escreva (ou
   ajuste) a função wrapper que chama `sqlcgen.New(q).MinhaQuery(ctx, ...)`
   e converte entre os tipos gerados (`*int16`, `pgtype.*` quando não há
   override) e os tipos do domínio (`*int`, `time.Time`, os enums do
   pacote `domain`). É essa camada fina que mantém `app/` sem saber que
   sqlc existe — as assinaturas exportadas de `adapters/postgres` não
   mudam por causa disso.
4. `mapeamentos de tipo` (uuid→string, timestamptz/date→time.Time,
   numeric→float64) já estão configurados em `backend/sqlc.yaml`. Um tipo
   novo (ex: uma coluna `jsonb`) provavelmente precisa de uma entrada de
   `overrides` ali — sqlc exige **duas** entradas por tipo (uma para coluna
   `NOT NULL`, outra com `nullable: true` para a versão ponteiro), senão a
   coluna nullable cai no tipo `pgtype.*` bruto por padrão.

`make sqlc-diff` é o jeito de checar, antes de commitar, que ninguém mexeu
numa query e esqueceu de rodar `generate` — compara sem sobrescrever.

## Estrutura

```
backend/
├── cmd/{api,worker,migrate}/main.go
├── internal/platform/{config,db,obs,auth,idgen,httpx}/
├── internal/modules/goals/            # módulo Metas (Fatia 1)
│   ├── domain/                        # regras puras, testadas sem banco
│   ├── packs/                         # domain packs (golang.yaml, english.yaml) + loader
│   ├── app/                           # casos de uso e transações
│   └── adapters/
│       ├── postgres/                  # wrappers finos sobre o sqlc
│       │   ├── queries/*.sql          # fonte de verdade das queries
│       │   └── sqlcgen/               # gerado — não editar (DO NOT EDIT)
│       └── http/                      # handlers chi + DTOs
├── db/migrations/0001-0006*.sql
├── sqlc.yaml
└── api/openapi.yaml
docker-compose.yml   # api + postgres (escopo mínimo da Fatia 0)
```

`worker` já existe como entrypoint (mesma imagem, outro comando) mas não faz
nada além de subir e segurar conexão — a fila de jobs chega na Fatia 3.
