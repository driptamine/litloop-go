.PHONY: run migrate worker test integration vet build routes
run:
	go run ./cmd/litloop-go-server
migrate:
	go run ./cmd/migrate
worker:
	go run ./cmd/worker
test:
	go test ./...
vet:
	go vet ./...
build:
	go build -o bin/litloop-go-server ./cmd/litloop-go-server
	go build -o bin/worker ./cmd/worker
	go build -o bin/migrate ./cmd/migrate
routes:
	go run ./cmd/routes > docs/openapi.json
integration:
	go test ./app ./internal/worker -count=1
