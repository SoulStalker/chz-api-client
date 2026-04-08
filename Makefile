.PHONY: templ build run lint

templ:
	templ generate ./views/...

build: templ
	go build -o ./bin/edo-client ./cmd/server

run: build
	./bin/edo-client --config config/example.yml

lint:
	golangci-lint run ./...
