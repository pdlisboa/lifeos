# LifeOS — Fatia 0

Esqueleto do backend: sobe, migra o banco e autentica. Sem UI ainda — a
Fatia 1 traz o front web. Ver `CLAUDE.md` e `docs/modulos/metas/` para o
raciocínio completo.

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

## Estrutura

```
backend/
├── cmd/{api,worker,migrate}/main.go
├── internal/platform/{config,db,obs,auth,idgen,httpx,migrate}/
├── db/migrations/0001_foundation.sql
└── api/openapi.yaml
docker-compose.yml   # api + postgres (escopo mínimo da Fatia 0)
```

`worker` já existe como entrypoint (mesma imagem, outro comando) mas não faz
nada além de subir e segurar conexão — a fila de jobs chega na Fatia 3.
