.PHONY: setup backend backend-build frontend test compose-up compose-down clean

# Install backend + frontend dependencies.
setup:
	cd backend && go mod download
	cd frontend && npm install

# Run the API locally (SQLite dev mode via root .env; no Docker/Postgres needed).
backend:
	cd backend && go run ./cmd/api

# Compile the API to backend/bin/nivra-api(.exe).
backend-build:
	cd backend && go build -o bin/nivra-api ./cmd/api

# Run the Next.js dev server.
frontend:
	cd frontend && npm run dev

# Backend tests + frontend type check.
test:
	cd backend && go test ./...
	cd frontend && npm run typecheck

# Production-like stack (Postgres + Redis + API + web) via Docker.
compose-up:
	docker compose up --build

compose-down:
	docker compose down

# Remove local SQLite dev database and build artifacts.
clean:
	cd backend && rm -f nivra-dev.db nivra-dev.db-shm nivra-dev.db-wal server.log
	cd backend && rm -rf bin
