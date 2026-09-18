# bigtui

A keyboard-first terminal workspace for querying BigQuery across projects, built with Go and Charm.

## Run

```sh
gcloud auth application-default login
go run ./cmd/bigtui
```

The first screen uses two example projects. Press `a` to add a project, `tab` to switch focus, `ctrl+enter` to run a query, and `?` to open the keymap.

## Architecture

- `internal/app`: Bubble Tea model and keyboard-driven workspace.
- `internal/bigquery`: BigQuery transport boundary, backed by Application Default Credentials.
- `internal/completion`: provider chain for future LSP and AI completion adapters.
- `internal/project`: project/session state.

Completion providers implement `completion.Provider`, so an LSP client or AI agent can be attached without coupling it to the TUI or BigQuery transport.

## Development

```sh
go test ./...
go vet ./...
```
