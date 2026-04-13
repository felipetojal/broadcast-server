build:
	@go build -o bin/broadcast-server

run: build
	@./bin/broadcast-server
