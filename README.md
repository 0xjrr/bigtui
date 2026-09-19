# bigtui

A keyboard-first terminal workspace for querying BigQuery across projects, built with Go and Charm.

## Run

```sh
go run ./cmd/bigtui
```

On startup, bigtui checks for Google Application Default Credentials. If none are available, it automatically starts the interactive `gcloud auth application-default login` flow before opening the TUI. The `gcloud` CLI must be installed and available on `PATH`.

The default screen loads the Google Cloud projects available to the authenticated user and lists them in a fixed-size, scrolling pane. Datasets load when a project is expanded, tables and views load when a dataset is expanded, and table metadata plus five-row previews load only when a resource is inspected. This keeps startup responsive for accounts with many projects and tables. If the account has no accessible projects or datasets, the empty state is shown. Press `tab` to switch focus, `ctrl+r` to run a query, and `?` to open the keymap.

For UI testing without relying on BigQuery resources, start with the fixture catalog:

```sh
go run ./cmd/bigtui --mock
```

The fixture catalog contains sandbox projects with datasets. It is opt-in and is never used as the default project list.

To create real BigQuery demo data in an accessible project, run the standalone seed utility:

```sh
go run ./cmd/bigtui-seed --project YOUR_PROJECT_ID
```

This creates or replaces the `bigtui_demo` dataset, `customers` and `orders` tables, and the `customer_order_totals` view. It is safe to rerun for that demo dataset, but it replaces those three named resources.

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
