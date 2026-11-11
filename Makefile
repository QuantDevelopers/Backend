up:
	docker compose down
	docker compose up --build -d
	docker image prune -f

up-monitoring:
	docker compose --profile monitoring up --build -d

down:
	docker compose down

seed:
	docker compose run --rm seed

logs:
	docker compose logs -f backend

ps:
	docker compose ps

wire:
	wire ./wire

test:
	go test -v -race ./...

test-integration:
	TEST_DATABASE_URL=$${TEST_DATABASE_URL:-postgresql://postgres:postgres@localhost:5432/quant_db?sslmode=disable} go test -v -race ./...

cover:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

lint:
	test -z "$$(gofmt -l .)"
	go vet ./...

build:
	go build -o bin/quant-server ./cmd/app
	go build -o bin/quant-benchmark ./cmd/benchmark

log:
	git log --oneline -n 10
