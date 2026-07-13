DB_URL=postgres://postgres:Shinas@localhost:5432/velocity?sslmode=disable

test:
	go test ./... -v

engine-test:
	go test ./internal/engine/... -v

matcher-test:
	go test ./internal/engine/matcher -v

migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1

migrate-reset:
	migrate -path migrations -database "$(DB_URL)" down -all
	migrate -path migrations -database "$(DB_URL)" up

sqlc:
	sqlc generate

run:
	go run ./cmd/api

fmt:
	go fmt ./...

lint:
	golangci-lint run

bench:
	go test -bench=. ./...