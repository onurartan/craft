# Craft

A declarative, minimalist build utility, orchestrator, and **Task Runner** for Go projects. 

📚 **Official Documentation:** [craft.trymagic.xyz](https://craft.trymagic.xyz)

Craft bridges the gap between basic `go build` commands and complex release pipelines. It harnesses the raw power of the native Go toolchain and wraps it in a seamless, cross-platform experience featuring **Toolchain Isolation**, **Intelligent Hot-Reloading**, and an advanced **Macro Engine**.

## Why Craft?

The Go toolchain is incredibly powerful, but orchestrating multi-platform releases, managing environments across teams, and setting up complex scripts often requires juggling Makefiles, bash scripts, and environment variables. Craft simplifies this entire workflow:

* **Zero-Install Toolchains:** Craft automatically downloads, caches, and completely isolates the exact Go version your project needs (e.g., `go1.25.0`) in `~/.craft/toolchains`. No more "It works on my machine" conflicts.
* **Built-in Task Runner:** Toss out your `Makefile` or `package.json`. Define sequential, OS-specific tasks with advanced Macro interpolation (`{OS}`, `{APP_BIN_PATH}`) directly in `.craft.yaml`.
* **Smart Package Management:** Use `craft add gh:gofiber/fiber/v2` to seamlessly download packages using intelligent registry shortcuts, completely synced with your isolated environment.
* **Zero-Config Hot Reload:** Built-in `dev` engine intelligently watches your files, triggers your custom hooks, and restarts your server in milliseconds.
* **Frictionless Cross-Compilation:** Compile for multiple OS/Arch targets concurrently without manually exporting `GOOS` or `GOARCH`.

## Installation

Since Craft is built with Go, you can install it globally using the native toolchain. Ensure your `~/go/bin` (or `%USERPROFILE%\go\bin` on Windows) is added to your system's `PATH`.

```bash
go install github.com/onurartan/craft@latest
```

## Quick Start

Navigate to the root of any existing Go project and run the initialization command. Craft will safely scan your `go.mod` and generate a base configuration.

```bash
craft init
```

To compile your project based on the generated configuration:

```bash
craft build
```

## Configuration (`.craft.yaml`)

Craft relies on a strictly typed YAML configuration. Below is a standard configuration demonstrating dynamic version extraction, isolated toolchains, and the powerful Task Runner.

```yaml
name: "app"
version: 0.1.0
toolchain: "go1.25.0"   # Craft will auto-download and isolate this version!
entry_point: "."
output_dir: "bin"       

# Build Profiles
build_all: false        
platforms: ["current"] 

# The Task Runner Engine
commands:
  envs:
    DB_USER: "admin"
  
  # 1. OS-Specific commands
  clean:
    windows: "del /Q /S bin\\*"
    default: "rm -rf bin/*"
    
  # 2. Macros & Composition
  migrate: "{APP_BIN_PATH:NOBUILD} migrate --user {DB_USER}"
  
  setup:
    - "$clean"
    - "$migrate"
    - "echo 'Project {APP_NAME} is ready on {OS}!'"
```

## Build Profiles (Workflow Routing)

Managing different deployment targets usually means writing multiple scripts. Craft simplifies this with **Profiles**. A profile allows you to override your base configuration for specific scenarios.

Add a `profiles` block to your `.craft.yaml`:

```yaml
profiles:
  release:
    output_dir: "releases/v1"
    platforms: ["linux/amd64", "windows/amd64"]
    strip_debug: true
```

Trigger a specific profile using the `-P` or `--profile` flag:

```bash
$ craft build -P release
```

## Advanced Features

### 1. Toolchain Dashboard
Craft provides an interactive terminal UI to manage your downloaded Go compilers.
```bash
$ craft toolchain
$ craft toolchain install 1.25.0 --remote
```

### 2. Inline Scripting
Run single `.go` files like python scripts, fully isolated and instantly cached.
```bash
$ craft run script.go --script
```

### 3. Error Parser & Semantic Search
Craft features a revolutionary error parser that translates cryptic compiler panics into human-readable advice. It also includes an offline Semantic Search engine for its documentation!

## Command Reference

Craft acts as a unified interface for your daily Go workflows.

### Core Engine
  * `craft build` - Orchestrates the compilation process based on `.craft.yaml`.
  * `craft dev` - Starts the hot-reload development engine. 
  * `craft run` - Compiles and executes the binary for your current host architecture.
  * `craft clean` - Deep cleans build artifacts (`bin/`) and global go caches.
  * `craft doctor` - Prints a diagnostic report of your Go toolchain and system health.

### Toolchain & Package Wrappers
  * `craft toolchain` - Manage, download, and isolate Go compiler versions.
  * `craft add <pkg>` - Smart package downloader with GitHub (`gh:`) aliases.
  * `craft sync` - Forcefully download missing modules and verify cryptographic checksums.
  * `craft check` - The CI/CD pre-flight sequence. Runs `fmt`, `vet`, and `test` sequentially. 

## Built for Production

From local development to automated CI/CD pipelines, Craft is engineered to fail gracefully, log intelligently, and build predictably. By providing a single, strictly typed source of truth, Craft ensures that your Go toolchain works for you, not against you.

## License

MIT License. See [LICENSE](https://github.com/onurartan/craft/blob/main/LICENSE) for more information.
