# Command Line Interface (CLI) Reference

Craft uses `cobra` to provide a robust, self-documenting CLI. This reference covers all available commands and their flags in deep detail.

## Core Build & Execution Commands

### `craft build`
Compiles your project based on the declarative rules in `.craft.yaml` and any CLI flag overrides.

**Flags:**
* `-n, --name <string>`: Override the binary name defined in YAML.
* `-v, --version <string>`: Override the injected application version (default "1.0.0").
* `-e, --entry <string>`: Override the main package path (default ".").
* `-o, --out <string>`: Override the output directory (default "bin").
* `--ver-pkg <string>`: Explicitly set the variable path for `LDFLAGS` version injection (e.g. `main.AppVersion`).
* `--all`: Cross-compile for all common platforms concurrently. Ignores YAML `platforms`.
* `-p, --platform <strings>`: Specify custom target platforms (e.g. `linux/arm64`).
* `--strip`: Strip DWARF debug symbols to minimize binary size (default true).
* `--exact-name`: Omit OS/Arch suffixes from the final binary name (e.g., outputs `app` instead of `app-linux-amd64`).
* `-P, --profile <string>`: Execute a specific build profile defined in your `craft.yaml` (e.g., `release`, `npm`).
* `--no-auto-install`: Disables Craft's magical auto-installation of missing Go dependencies during build.

### `craft run [--script] [file.go]`
Compiles the application specifically for your current host machine (`current` platform) into a temporary path, and executes it immediately.
If you provide the `--script` flag followed by a `.go` file (e.g., `craft run --script script.go`), Craft bypasses standard compilation and treats the file as an **Inline Script**, running it securely in isolation.

### `craft irun`
"Immediate Run". Skips the compilation phase entirely and attempts to locate and execute the previously compiled binary for your host architecture from the `output_dir`. Fails if the binary is missing or corrupted.

### `craft dev`
Starts the Hot-Reload watcher. Craft monitors `.go`, `.html`, `.env` and other configured extensions, compiling to a secure temporary directory and automatically restarting your server upon saves.

## Package & Project Management

### `craft create`
Starts an interactive terminal wizard (powered by `pterm`). It allows you to bootstrap entirely new projects using embedded templates like Fiber/Gin REST APIs, Cobra CLIs, or Blank boilerplates.

### `craft init [name]`
Adopts an existing Go codebase into Craft. It scans your `go.mod` and directory structure to automatically generate a tailored `.craft.yaml`.

### `craft add [packages...]`
A smart wrapper around `go get`. Adds Go packages and seamlessly updates dependencies. Supports Registry Aliases (e.g., `gh:` for `github.com/`, `x:` for `golang.org/x/`).

### `craft remove [packages...]`
Removes specific Go packages from your project and cleans up your `go.mod` and `go.sum`. Registry Aliases are fully supported here as well (e.g., `craft remove gh:pterm/pterm`).

### `craft sync`
Synchronizes, downloads, and verifies all project dependencies (similar to `uv sync`). Ensures your module cache exactly matches your `go.mod`.

## Diagnostics & Maintenance

### `craft clean`
Removes the `output_dir` (e.g., `bin/`) and purges any local artifacts.

**Flags:**
* `--go-cache`: Forces a deep purge of the global Go build cache (`go clean -cache`).
* `--all`: Purges all `craft-dev-*` temporary compilation directories globally across your operating system.

### `craft check`
Your local CI/CD pre-flight check. It sequentially runs `go fmt`, `go vet`, and `go test`. If any step fails, it halts immediately, preventing you from committing broken code.

### `craft doctor`
Validates your environment health. It checks the Go toolchain version, system paths, and ensures the `craft` binary permissions are correctly set.

## Go Toolchain Wrappers

Craft passes these commands directly to the underlying Go compiler with optimized defaults.

* `craft tidy`: Runs `go mod tidy` to synchronize module dependencies.
* `craft fmt`: Runs `go fmt ./...` across the entire project.
* `craft vet`: Runs `go vet ./...` for static analysis.
* `craft test [args...]`: Runs `go test ./...` and accepts standard Go test flags.
* `craft gen`: Runs `go generate ./...` for code generation tasks.
* `craft install`: Compiles and installs the binary to your global `GOBIN` (`~/go/bin`).

## Toolchain Management

Craft acts as a built-in Go version manager, securely downloading and isolating exact Go compiler versions in `~/.craft/toolchains`.

* `craft toolchain install <version>`: Downloads and extracts a specific Go compiler version. Use `--no-cache` to bypass cached archives.
* `craft toolchain use <version>`: Configures the current `.craft.yaml` to strictly use a specific Go compiler.
* `craft toolchain list`: Lists all downloaded Go toolchains and highlights the active one.
  * `--remote`: Browse a scrollable list of available releases from go.dev.
  * `--select-mode`: Open an interactive TUI to select and install a release.
* `craft toolchain remove <version>`: Deletes a specific toolchain from your machine.
* `craft toolchain clean`: Purges downloaded `.tar.gz` and `.zip` archives from `~/.craft/cache/archives` to free up disk space.
