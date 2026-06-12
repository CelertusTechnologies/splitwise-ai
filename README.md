# Nivra

Nivra is a production-shaped expense sharing platform for groups, trips, families, flatmates, and friends. This repository starts the launch-ready foundation requested in the brief:

- `backend/` - Go API with Gin, PostgreSQL, Redis, JWT auth, refresh token rotation, structured logging, and Clean Architecture boundaries.
- `frontend/` - Next.js app with TypeScript, Tailwind CSS, shadcn-style primitives, mobile-first auth screens, and dashboard shell.
- `docs/` - Software architecture, product strategy, database/API/security/scalability design, and MVP roadmap.

## Quick Start

### Option A — Local dev, no Docker (fastest)

Runs the API on an embedded pure-Go SQLite file with auto-migration, so no
PostgreSQL, Redis, or Docker is required. Redis is optional and degrades
gracefully when absent. Config comes from the repo-root `.env` (already set to
`DB_DRIVER=sqlite`).

Requirements: Go 1.25+ and Node 20+.

```bash
make setup            # go mod download + npm install (first time only)

# terminal 1 — API on :8080  (reads root .env automatically)
make backend

# terminal 2 — web on :3000
make frontend
```

Then open `http://localhost:3000`. Verify the API with:

```bash
curl http://localhost:8080/health
# {"data":{"redis":"disabled","status":"ok", ...}}
```

> The first signup returns a `dev_email_verification_token` in the response (and
> forgot-password returns a `dev_reset_token`) so you can exercise the email
> verification / reset flows without an email provider in development.

### Option B — Production-like stack (Docker)

Brings up PostgreSQL, Redis, the API (on the full SQL schema), and the web app.

```bash
cp .env.example .env   # uses DB_DRIVER=postgres
docker compose up --build
```

### Services

- Frontend: `http://localhost:3000`
- Backend API: `http://localhost:8080`
- Health: `http://localhost:8080/health`
- PostgreSQL: `localhost:5432` (Docker only)
- Redis: `localhost:6379` (Docker only)

## First Slice

The initial implementation focuses on the foundation:

- Monorepo structure
- Database schema migration for users, groups, expenses, settlements, events, notifications, and audit logs
- Auth module with signup, login, logout, refresh token rotation, forgot/reset password, and email verification hooks
- Premium operational UI shell with login, signup, dashboard, activity, and settlement panels

The architecture document in [docs/architecture.md](docs/architecture.md) is the source of truth for V1 scope and V2 AI hooks.

