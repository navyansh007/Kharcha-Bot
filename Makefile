-include .env
export

.PHONY: dev build test test-integration fuzz lint migrate sqlc docker db-up db-down

dev:
	go run ./cmd/bot

build:
	CGO_ENABLED=0 go build -trimpath -o bin/bot ./cmd/bot

test:
	go test -race ./...

test-integration:
	go test -race -tags=integration ./...

fuzz:
	go test -run=^$$ -fuzz=FuzzParse -fuzztime=30s ./internal/parser

lint:
	go vet ./...
	golangci-lint run

migrate:
	go run ./cmd/migrate

sqlc:
	sqlc generate

docker:
	docker build -f deploy/Dockerfile -t kharcha-bot .

db-up:
	docker compose up -d

db-down:
	docker compose down
