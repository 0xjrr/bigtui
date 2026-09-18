# bigtui

A keyboard-first terminal workspace for querying BigQuery across projects, built with Go and Charm.

## Run

```sh
go run ./cmd/bigtui
```

On startup, bigtui checks for Google Application Default Credentials. If none are available, it automatically starts the interactive `gcloud auth application-default login` flow before opening the TUI. The `gcloud` CLI must be installed and available on `PATH`.

The default screen starts with no connected projects so the empty state is visible. Press `a` to add a project, `tab` to switch focus, `ctrl+r` to run a query, and `?` to open the keymap.

For UI testing without relying on BigQuery resources, start with the fixture catalog:

```sh
go run ./cmd/bigtui --mock
```

The fixture catalog contains sandbox projects with datasets, tables, and views. It is opt-in and is never used as the default project list.

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
