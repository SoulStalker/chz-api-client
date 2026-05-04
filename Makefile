.PHONY: templ build run explore lint

templ:
	go run github.com/a-h/templ/cmd/templ@v0.3.865 generate ./views/...

build: templ
	go build -o ./bin/edo-client ./cmd/server

run: build
	./bin/edo-client --config config/prod.yml

explore:
	go run ./cmd/crpt-explore/ --config config/prod.yml

lint:
	golangci-lint run ./...
