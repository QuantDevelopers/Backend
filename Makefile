up:
	docker compose down
	docker compose up --build -d
	docker image prune -f

ps:
	docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}"

wire:
	wire ./wire

test:
	go test -v -race ./...

cover:
	go test ./internal/application/service -coverprofile=coverage.out
	go tool cover -html=coverage.out

lint:
	test -z "$$(gofmt -l .)"
	go vet ./...

build:
	go build -o bin/quant-server ./cmd/app

log:
	git log --oneline -n 10
