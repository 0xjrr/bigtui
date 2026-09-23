# bigtui

A keyboard-first terminal workspace for querying BigQuery across projects, built with Go and Charm.

## Install

The latest production release is [v1.0.0](https://github.com/0xjrr/bigtui/releases/latest). Release binaries are standalone; Go is not required to run them.

### Homebrew

```sh
brew tap 0xjrr/bigtui
brew install --cask bigtui
```

### Scoop (Windows)

```powershell
scoop bucket add bigtui https://github.com/0xjrr/scoop-bigtui
scoop install bigtui
```

### WinGet (Windows)

```powershell
winget install --id 0xjrr.bigtui
```

### Debian or Ubuntu

Download the `.deb` package for your architecture from the [release page](https://github.com/0xjrr/bigtui/releases/latest), then install it with:

```sh
sudo apt install ./bigtui_1.0.0_linux_amd64.deb
```

For ARM64 systems, use the `linux_arm64` package instead.

### Fedora, RHEL, or compatible systems

Download the `.rpm` package for your architecture from the [release page](https://github.com/0xjrr/bigtui/releases/latest), then install it with:

```sh
sudo dnf install ./bigtui_1.0.0_linux_amd64.rpm
```

For repository-based updates, add the bigtui RPM repository:

```sh
sudo tee /etc/yum.repos.d/bigtui.repo >/dev/null <<'EOF'
[bigtui]
name=bigtui
baseurl=https://ricardoribeiro.dev/bigtui/rpm
enabled=1
gpgcheck=0
EOF
sudo dnf install bigtui
```

### APT repository

```sh
curl -fsSL https://ricardoribeiro.dev/bigtui/apt/bigtui-archive-keyring.asc \
	| sudo tee /etc/apt/keyrings/bigtui.asc >/dev/null
echo "deb [signed-by=/etc/apt/keyrings/bigtui.asc] https://ricardoribeiro.dev/bigtui/apt stable main" \
	| sudo tee /etc/apt/sources.list.d/bigtui.list >/dev/null
sudo apt update
sudo apt install bigtui
```

### Release archives

Linux and macOS `.tar.gz` archives are available for AMD64 and ARM64. Extract the archive and place the binary on your `PATH`:

```sh
tar -xzf bigtui_1.0.0_linux_amd64.tar.gz
sudo install -m 0755 bigtui /usr/local/bin/bigtui
```

Use the `darwin_amd64` archive for Intel Macs and `darwin_arm64` for Apple Silicon Macs.

### Go

Users with Go installed can install the command directly from the tagged module:

```sh
go install github.com/0xjrr/bigtui/cmd/bigtui@v1.0.0
```

Every release includes checksums in `checksums.txt`. Verify downloaded files with `sha256sum` before installing them.

Regardless of installation method, bigtui requires Google Application Default Credentials and BigQuery access. The `gcloud` CLI is needed only when credentials have not already been configured.

## Run

```sh
go run ./cmd/bigtui
```

On startup, bigtui checks for Google Application Default Credentials. If none are available, it automatically starts the interactive `gcloud auth application-default login` flow before opening the TUI. The `gcloud` CLI must be installed and available on `PATH`.

The default screen loads the Google Cloud projects available to the authenticated user and lists them in a fixed-size, scrolling pane. Datasets load when a project is expanded, tables and views load when a dataset is expanded, and table metadata plus five-row previews load only when a resource is inspected. Hidden datasets are excluded by default; focus the projects pane and press `ctrl+h` to show or hide them. This keeps startup responsive for accounts with many projects and tables. If the account has no accessible projects or datasets, the empty state is shown. Press `tab` to switch focus, `ctrl+r` to run a query, and `?` to open the keymap. The query editor suggests projects, datasets, tables, views, columns, and SQL keywords automatically as you type; use up/down to navigate, `enter` to accept, and `esc` to dismiss.

For UI testing without relying on BigQuery resources, start with the fixture catalog:

```sh
go run ./cmd/bigtui --mock
```

The fixture catalog contains sandbox projects with datasets. It is opt-in and is never used as the default project list.

To create real BigQuery demo data in an accessible project during development, run the standalone seed utility:

```sh
go run ./cmd/bigtui-seed --project YOUR_PROJECT_ID
```

This creates or replaces the `bigtui_demo` dataset, `customers` and `orders` tables, and the `customer_order_totals` view. It is safe to rerun for that demo dataset, but it replaces those three named resources.

The seed utility is a development tool and is not included in production release packages.

## Architecture

- `internal/app`: Bubble Tea model and keyboard-driven workspace.
- `internal/bigquery`: BigQuery transport boundary, backed by Application Default Credentials.
- `internal/completion`: provider chain for LSP and AI completion adapters.
- `internal/project`: project/session state.

Completion providers implement `completion.Provider`, so an LSP client (such as [bqls](https://github.com/kitagry/bqls)) or AI agent can be attached without coupling it to the TUI or BigQuery transport. By default, the query editor suggests completions automatically as you type using three built-in providers: `completion.KeywordProvider`, `completion.FunctionProvider`, and `completion.CatalogProvider`. The keyword and official BigQuery function mappings live in `internal/completion/catalog.go`; function descriptions are sourced from the [BigQuery Standard SQL functions reference](https://docs.cloud.google.com/bigquery/docs/reference/standard-sql/functions-all). Dataset discovery is requested only when completion enters a `FROM`/`JOIN` resource context, and table/view discovery is requested only after a dataset is named. These requests reuse the Projects pane cache and honor its hidden-dataset setting. Providers are synchronous and local once data is loaded, so no subprocess or extra catalog request is needed. Wiring in an LSP-backed provider later is possible by implementing `completion.Provider` and adding it to the providers merged in `internal/app`.

## Development

```sh
go test ./...
go vet ./...
```

## Release

Releases are built by GoReleaser through GitHub Actions. To publish a new version:

```sh
go test ./...
go vet ./...
git tag -a v1.0.1 -m "Release v1.0.1"
git push origin v1.0.1
```

Pushing a `v*` tag creates the GitHub Release and publishes standalone Linux/macOS archives, Linux `.deb` and `.rpm` packages, checksums, and the Homebrew Cask. The release configuration is in [.goreleaser.yaml](.goreleaser.yaml) and the workflow is in [.github/workflows/release.yml](.github/workflows/release.yml).

## License

bigtui is licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE)
and [NOTICE](NOTICE) for the project terms and attribution. Third-party dependency
licenses and notices are collected in [THIRD_PARTY_NOTICES](THIRD_PARTY_NOTICES).
