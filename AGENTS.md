# Repository Guidelines

## Project Structure & Module Organization
`backend/` contains the Go service. Use `cmd/server` for the main entrypoint, `internal/{handler,service,repository,domain}` for application logic, `ent/schema` for data models, and generated Ent code under `ent/`. Database changes live in `backend/migrations/`.

`frontend/` is the Vue 3 + Vite client. Work mainly happens in `frontend/src/` under `api/`, `components/`, `views/`, `stores/`, `composables/`, `utils/`, and feature-local `__tests__/`. Deployment assets live in `deploy/`; supporting docs and static assets live in `docs/` and `assets/`.

## Build, Test, and Development Commands
- `make build`: build the backend binary and frontend production bundle.
- `make test`: run backend `go test ./...` plus frontend lint and type-check.
- `make -C backend test-unit` / `test-integration` / `test-e2e-local`: run tagged backend suites.
- `make -C backend generate`: regenerate Ent and server-generated files after schema/codegen changes.
- `pnpm --dir frontend dev`: start the local Vite dev server.
- `pnpm --dir frontend test:run` / `test:coverage`: run Vitest once or with coverage.

CI pins Go `1.26.1`, Node `20`, and frontend installs with `pnpm install --frozen-lockfile`.

## Coding Style & Naming Conventions
Format Go with `gofmt`; keep package names lowercase. Run `golangci-lint` from `backend/` before opening a PR.

Frontend code follows existing Vue 3 SFC conventions: 2-space indentation, PascalCase component files such as `UserBalanceModal.vue`, and `useXxx.ts` names for composables such as `useClipboard.ts`. Keep utility modules camelCase and follow nearby patterns.

## Testing Guidelines
Backend tests sit beside implementation files as `*_test.go`; use build tags for unit, integration, and e2e coverage. Frontend tests use Vitest + jsdom and match `src/**/*.{test,spec}.{js,ts,jsx,tsx}`. Coverage thresholds are 80% global.

If you change an interface, update affected stubs and mocks. If you edit `backend/ent/schema`, run `make -C backend generate` and commit the regenerated `backend/ent` output.

## Commit & Pull Request Guidelines
Recent history follows Conventional Commit style, often with scope: `feat(openai): ...`, `fix(oauth): ...`, `style: ...`. Keep subjects short, imperative, and focused on one change.

Before opening a PR, run backend unit/integration tests, `golangci-lint`, and frontend lint/type-check; update `pnpm-lock.yaml` whenever `frontend/package.json` changes. In the PR description, note config or migration impact, link issues, and attach screenshots for `frontend/` UI changes.

## Security & Configuration Tips
Do not commit live API keys, OAuth tokens, `.env` files, or customer data. Start from `deploy/.env.example` and `deploy/config.example.yaml`, and review deployment changes carefully because this project handles upstream credentials, quotas, and billing-related data.
