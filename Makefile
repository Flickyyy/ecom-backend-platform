.PHONY: build run test lint integration-test docker-up docker-down migrate

APP_NAME := api
BUILD_DIR := ./bin

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/api

run: build
	$(BUILD_DIR)/$(APP_NAME)

test:
	go test -v -race -count=1 ./internal/service/...

integration-test:
	go test -v -tags integration ./internal/repository/...

lint:
	golangci-lint run ./...

vet:
	go vet ./...

docker-up:
	docker-compose up --build -d

docker-down:
	docker-compose down -v

migrate-up:
	@echo "Migrations are applied automatically via docker-entrypoint-initdb.d"

clean:
	rm -rf $(BUILD_DIR)

.DEFAULT_GOAL := build
