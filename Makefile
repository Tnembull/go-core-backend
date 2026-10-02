.PHONY: all build test test-coverage lint run docker-build docker-run clean e2e

APP_NAME = go-core-backend
BIN_DIR = bin

all: test build

build:
	@echo "==> Building binary..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BIN_DIR)/server ./cmd/server

test:
	@echo "==> Running unit tests..."
	go test -v -race ./...

test-coverage:
	@echo "==> Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

run: build
	@echo "==> Starting server..."
	./$(BIN_DIR)/server

docker-build:
	@echo "==> Building Docker image..."
	docker build -t $(APP_NAME):latest .

docker-run:
	@echo "==> Running Docker container..."
	docker run -p 8080:8080 --rm $(APP_NAME):latest

e2e:
	@echo "==> Running Newman E2E tests..."
	./tests/run_e2e_newman.sh

clean:
	@echo "==> Cleaning build artifacts..."
	rm -rf $(BIN_DIR) coverage.out
