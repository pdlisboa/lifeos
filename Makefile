.PHONY: build up down restart logs ps migrate-up migrate-status create-user seed test vet fmt sqlc-generate sqlc-diff

build:
	docker compose build

up:
	docker compose up -d --build

down:
	docker compose down

restart:
	docker compose restart api

logs:
	docker compose logs -f api

ps:
	docker compose ps

migrate-up:
	docker compose run --rm --entrypoint /app/migrate api up

migrate-status:
	docker compose run --rm --entrypoint /app/migrate api status

# uso: make create-user EMAIL=voce@exemplo.com PASSWORD=senha-forte
create-user:
	docker compose run --rm --entrypoint /app/migrate api create-user $(EMAIL) $(PASSWORD)

# popula uma meta de Go com ~10 semanas de histórico (sessões, evidências,
# eventos de nível) — só roda com ENV=development (guard no próprio
# cmd/seed). Rode `make create-user` antes se ainda não houver usuário.
# uso opcional: make seed SEED_EMAIL=dev@lifeos.local (senão usa o usuário mais antigo)
seed:
	docker compose run --rm -e ENV=development -e SEED_EMAIL=$(SEED_EMAIL) --entrypoint /app/seed api

test:
	cd backend && go test ./...

vet:
	cd backend && go vet ./...

fmt:
	cd backend && gofmt -l .

# sqlc via Docker — não precisa instalar o binário localmente. Roda depois
# de mudar qualquer .sql em internal/modules/goals/adapters/postgres/queries/
# ou qualquer migration (ver docs/modulos/metas/fatia-1-implementacao.md §2).
sqlc-generate:
	cd backend && docker run --rm -v "$$(pwd)":/src -w /src sqlc/sqlc:latest generate

# confere se o código gerado está sincronizado com queries/schema, sem
# sobrescrever nada — bom pra CI ou pra checar antes de commitar.
sqlc-diff:
	cd backend && docker run --rm -v "$$(pwd)":/src -w /src sqlc/sqlc:latest diff
