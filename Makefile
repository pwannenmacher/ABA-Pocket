.PHONY: run build test docker-up docker-down migrate

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./...

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

migrate:
	psql $(DATABASE_URL) -f migrations/001_initial.sql

tidy:
	go mod tidy

fmt:
	gofmt -w .
