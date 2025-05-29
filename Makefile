.PHONY: build run test clean install

BINARY_NAME=redis-clone
CLI_BINARY_NAME=redis-cli

build:
	go build -o bin/$(BINARY_NAME) cmd/server/main.go
	go build -o bin/$(CLI_BINARY_NAME) cmd/cli/main.go

run:
	go run cmd/server/main.go

run-cli:
	go run cmd/cli/main.go localhost:6379

test:
	go test -v ./...

test-race:
	go test -race -v ./...

benchmark:
	go test -bench=. -benchmem ./...

clean:
	rm -rf bin/
	rm -f dump.rdb appendonly.aof

install:
	go install cmd/server/main.go
	go install cmd/cli/main.go

lint:
	golangci-lint run

format:
	go fmt ./...

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY_NAME)-linux-amd64 cmd/server/main.go
	GOOS=darwin GOARCH=amd64 go build -o bin/$(BINARY_NAME)-darwin-amd64 cmd/server/main.go
	GOOS=windows GOARCH=amd64 go build -o bin/$(BINARY_NAME)-windows-amd64.exe cmd/server/main.go

