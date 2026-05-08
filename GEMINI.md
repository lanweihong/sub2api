# Sub2API

## Project Overview
Sub2API is an AI API gateway platform designed to distribute and manage API quotas from AI product subscriptions. It handles multi-account management, precise token-level billing, API key distribution, smart routing (with sticky sessions), rate limiting, and includes an integrated self-service payment system. 

## Tech Stack
- **Backend:** Go 1.26+, Gin web framework, Ent ORM
- **Frontend:** Vue 3.4+, TypeScript, Vite 5+, TailwindCSS, Pinia (State Management), Vue Router
- **Databases/Cache:** PostgreSQL (15+), Redis (7+)
- **Infrastructure:** Docker, Docker Compose, Make

## Building and Running

### Prerequisites
- Go 1.26+
- Node.js 18+ and pnpm
- PostgreSQL 15+
- Redis 7+

### Commands
The project uses a `Makefile` at the root for orchestration.

- **Full Build (Frontend + Backend):** `make build`
- **Backend Build:** `make build-backend` 
- **Frontend Build:** `make build-frontend` 
- **Frontend Dev Server:** `cd frontend && pnpm run dev`
- **Full Test:** `make test`
- **Testing Backend:** `make test-backend`
- **Testing Frontend:** `make test-frontend` (runs ESLint, typecheck, and vitest for critical components)
- **Secret Scan:** `make secret-scan`

*(Note: When building the backend manually without make, use the `-tags embed` flag if you want to embed the frontend dist folder into the backend binary.)*

## Development Conventions
- **Frontend:** Built with Vue 3 (Composition API) and TypeScript. Uses TailwindCSS for styling and ESLint + Vue-tsc for code quality and type checking. Ensure tests are updated when modifying critical views or components.
- **Backend:** Structured around Go modules. Employs `ent` for database schema management and `Gin` for the HTTP API layer.
- **Docker:** There are multiple `docker-compose` configurations provided in the `deploy/` directory for local development, production, and standalone deployments.
- **Security:** Security is a key concern (e.g., token billing, payment processing). Any changes to API gateways or token distribution logic must be carefully reviewed and tested.
