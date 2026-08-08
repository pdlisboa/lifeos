.PHONY: build up down restart logs ps migrate-up migrate-status create-user test vet fmt

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

test:
	cd backend && go test ./...

vet:
	cd backend && go vet ./...

fmt:
	cd backend && gofmt -l .
