OAPI_VERSION := v2.8.0
SPEC := api/openapi/marketplace.yaml
CONFIG := internal/api/generated/oapi-codegen.yaml

.PHONY: generate tidy build run test fmt verify compose-up compose-down demo db

generate:
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_VERSION) -config $(CONFIG) $(SPEC)

tidy:
	go mod tidy

build: generate tidy
	go build -o bin/marketplace ./cmd/api

run: generate
	go run ./cmd/api

test: generate
	go test ./...

fmt:
	gofmt -w cmd internal

verify: generate tidy
	test -z "$$(gofmt -l cmd internal)"
	go test ./...

compose-up:
	docker compose up --build -d

compose-down:
	docker compose down

demo:
	bash scripts/demo.sh

db:
	docker compose exec -T postgres psql -U marketplace -d marketplace -f /scripts/defense.sql
