.PHONY: build run test clean install dev

build:
	go build -o ctakes-tui

run: build
	./ctakes-tui

test:
	go test ./...

clean:
	rm -f ctakes-tui
	go clean -cache

install:
	go install

dev:
	go run main.go

fmt:
	go fmt ./...
	gofmt -s -w .

lint:
	golangci-lint run

vet:
	go vet ./...

deps:
	go mod download
	go mod tidy

release:
	GOOS=linux GOARCH=amd64 go build -o ctakes-tui-linux-amd64
	GOOS=darwin GOARCH=amd64 go build -o ctakes-tui-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build -o ctakes-tui-darwin-arm64
	GOOS=windows GOARCH=amd64 go build -o ctakes-tui-windows-amd64.exe