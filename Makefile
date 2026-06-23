.PHONY: test harness harness-mock harness-eino fmt vet

test:
	go test ./...

harness:
	go run ./cmd/harness

harness-mock:
	go run ./cmd/harness -planner mock

harness-eino:
	go run ./cmd/harness -planner eino

fmt:
	gofmt -w .

vet:
	go vet ./...
