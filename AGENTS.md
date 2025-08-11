# AGENTS.md

## Build, Lint & Test Commands
- **Build:** `go build ./...`
- **Test:** `go test ./...`
- **Format code:** `go fmt ./...`

## Code Style & Conventions
- Imports are ordered: standard library, third-party, then project packages.
- Types are named in CamelCase; variables and functions in camelCase.
- Errors must be handled and logged (using `slog` for structured logging).
- Keep functions short and focused; prefer single responsibility.
- For testing, isolate components and use table-driven tests when possible.
- Always use `any` instead of `interface{}`.
- Use conventional commits and do not use more than 2 contexts words i.e `feat(firmwares.updates): [...]`.
- When following a plan, only implement 1 step at a time. Prepare a summary of what has been done and present it. This will enable better reviewing.
- Always commit your changes UNLESS on prodution branch. If it is a correction to the last commit, make a fixup commit.
