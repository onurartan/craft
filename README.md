# Craft

A declarative, minimalist build utility and orchestrator for Go projects. 

Craft bridges the gap between basic `go build` commands and complex release pipelines. It harnesses the raw power of the native Go toolchain and wraps it in a seamless, cross-platform build experience with intelligent hot-reloading and dynamic version injection.

## Why Craft?

The Go toolchain is incredibly powerful, but orchestrating multi-platform releases, managing LDFLAGS for versioning, and setting up hot-reloading often requires juggling environment variables and writing custom shell scripts. Craft simplifies this entire workflow:

* **Declarative Configuration:** Define your build targets, flags, and optimization rules in a single, readable `.craft.yaml` file.
* **Frictionless Cross-Compilation:** Compile for multiple OS/Arch targets concurrently without manually exporting `GOOS` or `GOARCH`.
* **Build Profiles:** Define specific workflows (e.g., `npm`, `release`, `local`) to override configurations dynamically based on your target environment.
* **Zero-Config Hot Reload:** Built-in `dev` engine that intelligently watches your files and isolates temporary builds for rapid development.

## Installation

Since Craft is built with Go, you can install it globally using the native toolchain. Ensure your `~/go/bin` (or `%USERPROFILE%\go\bin` on Windows) is added to your system's `PATH`.

```bash
go install github.com/onurartan/craft@latest
````

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

Craft relies on a strictly typed YAML configuration. Below is a standard configuration demonstrating dynamic version extraction and compiler optimizations.

```yaml
name: "app"
entry_point: "."
output_dir: "bin"       
exact_name: false        

# Dynamically extract version from a Go file and inject it via LDFLAGS
version: "in_go:main.AppVersion"        
version_pkg: "main.Version"         

build_all: false        
platforms: ["current"] 

# Compiler optimizations for production
strip_debug: true       
trimpath: true          
cgo_enabled: false      
```

## Build Profiles (Workflow Routing)

Managing different deployment targets usually means writing multiple scripts or memorizing long command-line flags. Craft simplifies this with **Profiles**. A profile allows you to override your base configuration for specific scenarios.

Add a `profiles` block to your `.craft.yaml`:

```yaml
profiles:
  npm:
    output_dir: "npm/bin"  
    build_all: true        
    exact_name: false
  
  release:
    output_dir: "releases/v1"
    platforms: ["linux/amd64", "windows/amd64"]
    strip_debug: true
```

Trigger a specific profile using the `-P` or `--profile` flag. Craft will automatically route the artifacts to the designated directories:

```bash
$ craft build -P release
```

## Command Reference

Craft acts as a unified interface for your daily Go workflows.

### Core Engine

  * `craft build` - Orchestrates the compilation process based on `.craft.yaml`.
  * `craft dev` - Starts the hot-reload development engine. It compiles to a secure temporary directory and refreshes on file changes.
  * `craft run` - Bypasses cross-compilation to build and execute the binary explicitly for your current host architecture.
  * `craft clean` - Deep cleans build artifacts (`bin/`) and sweeps any zombie temporary files left by the dev engine.
  * `craft doctor` - Prints a diagnostic report of your Go toolchain, host context, and OS-specific limits.

### Toolchain Wrappers

Craft intercepts raw Go toolchain outputs and formats them into a clean, human-readable tree structure. You can pass standard Go flags to these commands.

  * `craft check` - The CI/CD pre-flight sequence. Runs `fmt`, `vet`, and `test` sequentially. Halts on any failure.
  * `craft fmt` - Runs `go fmt`
  * `craft vet` - Runs `go vet`
  * `craft test` - Runs `go test`
  * `craft tidy` - Runs `go mod tidy`

## Built for Production

From local development to automated CI/CD pipelines, Craft is engineered to fail gracefully, log intelligently, and build predictably. By providing a single, strictly typed source of truth, Craft ensures that your Go toolchain works for you, not against you.

## License

MIT License. See [LICENSE](https://www.google.com/search?q=LICENSE) for more information.
