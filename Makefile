.PHONY: run build test docker-up docker-down migrate sync-assets

run:
	go run .

build:
	go build -o bin/server .

test:
	go test ./...

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

migrate:
	psql $(DATABASE_URL) -f migrations/001_initial.sql

sync-assets:
	npm ci --ignore-scripts && cp node_modules/htmx.org/dist/htmx.min.js web/static/js/htmx.min.js

tidy:
	go mod tidy

fmt:
	gofmt -w .
