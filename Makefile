APP_NAME=supply-chain-api

.PHONY: run
run:
	go run ./cmd/api

.PHONY: test
test:
	go test ./...

.PHONY: docker-build
docker-build:
	docker build -t $(APP_NAME):local .

.PHONY: docker-run
docker-run:
	docker run --rm -p 8080:8080 --env-file .env.example $(APP_NAME):local
