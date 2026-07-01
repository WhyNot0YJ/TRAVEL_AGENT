.PHONY: test harness harness-mock harness-eino server fmt vet docker-up docker-build docker-down docker-logs docker-clean

test:
	go test ./...

harness:
	go run ./cmd/harness

harness-mock:
	go run ./cmd/harness -planner mock

harness-eino:
	go run ./cmd/harness -planner eino

server:
	go run ./cmd/server

fmt:
	gofmt -w .

vet:
	go vet ./...

docker-up:
	docker compose up

docker-build:
	docker compose up --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-clean:
	docker compose down -v
