.PHONY: setup dev dev-backend dev-frontend fmt vet test test-race build compose-config compose-build compose-up compose-down compose-logs compose-tunnel

setup:
	cd server && go mod download
	cd client && npm ci

# Run with `make -j2 dev` so both long-running processes start together.
dev: dev-backend dev-frontend

dev-backend:
	cd server && go run ./cmd/api

dev-frontend:
	cd client && npm run dev

fmt:
	cd server && gofmt -w .

vet:
	cd server && go vet ./...

test:
	cd server && go test ./...
	cd client && npm test

test-race:
	cd server && go test -race ./...

build:
	cd server && go build -buildvcs=false -o api ./cmd/api
	cd client && npm run build

compose-config:
	docker compose config

compose-build:
	docker compose build

compose-up:
	docker compose up -d --build

compose-down:
	docker compose down

compose-logs:
	docker compose logs -f

compose-tunnel:
	docker compose --profile tunnel up -d --build
