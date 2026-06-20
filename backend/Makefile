run:
	go run ./cmd/server

build:
	go build -o bin/pulsechat ./cmd/server

test:
	go test ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

sqlc-postgres:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate -f sqlc.postgres.yaml

sqlc-mysql:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate -f sqlc.mysql.yaml

sqlc:
	make sqlc-postgres
	make sqlc-mysql

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-version:
	go run ./cmd/migrate version
