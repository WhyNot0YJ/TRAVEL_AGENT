.PHONY: test harness harness-mock harness-eino server fmt vet

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
