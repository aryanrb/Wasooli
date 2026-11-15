-include .env
export

run:
	go run .

generate:
	sqlc generate

migrate:
	go tool tern migrate --migrations ./migrations --conn-string "$(DATABASE_URL)"

new:
	go tool tern new --migrations ./migrations $(name)

tidy:
	go mod tidy

test:
	go test ./...

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

.PHONY: run generate migrate new tidy test db-up db-down
