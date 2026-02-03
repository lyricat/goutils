# Repository Guidelines

## Project Structure & Module Organization

- `go.mod`: single Go module (`github.com/lyricat/goutils`) targeting Go 1.22.
- Package directories: `ai/`, `httphelper/`, `qdrant/`, `social/`, `storage/`, `model/`, plus smaller utilities like `uuid/`, `structs/`, `crypto/`, `convert/`.
- Code generation: `gen/gen.go` drives model/store generation (see `model/store/`).
- Examples/scratch: `main/` contains runnable snippets. Treat it as local-only; don’t rely on committed binaries.
- Tests live next to code as `*_test.go` (e.g., `ai/`, `bayesian/`, `httphelper/middleware/`).

## Build, Test, and Development Commands

- `go test ./...`: run all unit tests.
- `go test -cover ./...`: quick coverage check.
- `go vet ./...`: static analysis.
- `go fmt ./...`: format packages (gofmt via `go fmt`).
- `go mod tidy && go mod verify`: keep module graph clean.
- `go run gen/gen.go`: regenerate generated store/repository code after changing models.
- `go build -o ./main/main ./main/main.go`: build the local example (avoid committing build outputs).

## Coding Style & Naming Conventions

- Use standard Go formatting (`gofmt`); imports grouped by `gofmt`/`go fmt`.
- Exported identifiers use `PascalCase`; unexported use `camelCase`.
- Prefer wrapping errors: `fmt.Errorf("context: %w", err)`; keep error strings lowercase.
- Keep packages focused; avoid cross-package cycles (higher-level packages depend on lower-level utilities).

## Testing Guidelines

- Use Go’s standard `testing` package with table-driven tests (`t.Run`).
- Prefer `httptest` and mocks/fakes over real network calls (many packages integrate external APIs).

## Commit & Pull Request Guidelines

- Follow the repo’s common convention: `feat(scope): ...`, `fix(scope): ...`, `refactor(scope): ...` (imperative mood).
- Keep commits small and scoped; include a short test plan in PRs (commands run, key cases).
- Link issues/PRs when applicable (e.g., `(#9)`), and call out any API changes that require README updates.

## Security & Configuration Tips

- Never commit secrets (API keys, OAuth tokens). Use environment variables and local `.env` (ignored by `.gitignore`).
- Keep example code in `main/` free of real credentials; prefer reading from env at runtime.
