# AGENTS.md

## Build conventions

- Always cross-compile from WSL for Windows: `GOOS=windows GOARCH=amd64 go build`
- Output test builds to `build/` — never to the repository root.
- Run `make build-verify` to confirm the project compiles without errors.
  Confirm the output is `Build OK` with no errors.
