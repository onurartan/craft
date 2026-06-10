# Configuration Guide (`.craft.yaml`)

Craft derives its power from a strict, declarative configuration file: `.craft.yaml`. This single source of truth consolidates your entire project configuration, providing a deterministic and reproducible build experience across every machine and OS.

---

## 1. The Complete Anatomy
Here is the ultimate, complete example of a production-ready `.craft.yaml`, covering every single object and field supported by the Craft Engine:

```yaml
name: "my-app"
toolchain: "1.22.1"
version: "in_go:cmd/main.AppVersion"
version_pkg: "github.com/myorg/myapp/cmd.Version"
entry_point: "cmd/api"
output_dir: "dist"
exact_name: false
auto_install: true

platforms:
  - "linux/amd64"
  - "linux/arm64"
  - "windows/amd64"
  - "darwin/arm64"
build_all: false

strip_debug: true
trimpath: true
cgo_enabled: false
race: false
tags: ["production", "jsoniter"]

minify:
  enabled: true
  extensions: [".html", ".css", ".js", ".json", ".svg"]
  dirs: ["web/templates", "assets"]

dev:
  watch:
    delay_ms: 500
    include_exts: ["go", "html", "yaml", "env"]
    exclude_dirs: ["bin", "dist", "vendor", ".git", "node_modules"]
    exclude_files: [".craft.yaml"]

commands:
  lint:
    - "echo 'Running golangci-lint...'"
    - "golangci-lint run ./..."

scripts:
  pre_build:
    - "echo 'Running Pre-Build Macro...'"
    - "craft gen"
  post_build:
    - "echo 'Build Finished!'"

profiles:
  release:
    output_dir: "releases/v1"
    strip_debug: true
    build_all: true
  local:
    output_dir: "bin"
    platforms: ["current"]
    race: true
```

---

## 2. Metadata, Toolchain & Layout

```yaml
name: "my-app"          
toolchain: "1.22.1"
entry_point: "cmd/server"        
output_dir: "bin"       
exact_name: false       
auto_install: true
```

* **`name`**: The base name of your compiled binary.
* **`toolchain`**: The exact Go compiler version to use (e.g., `go1.22.1`). Craft will download and use this specific version automatically.
* **`entry_point`**: The path containing your `main` package (e.g., `"."` or `"cmd/server"`).
* **`output_dir`**: Destination folder for compiled artifacts.
* **`exact_name`**: By default, cross-compiled binaries get suffixes (e.g., `my-app-linux-amd64`). Set to `true` to disable suffixes (ideal for Docker containers).
* **`auto_install`**: When `true`, Craft will automatically run `go get` for missing dependencies and `go mod tidy` before building, acting as a smart package manager.

---

## 3. Versioning & Dynamic LDFLAGS

Automates version extraction and injection without `-ldflags` nightmares.

```yaml
version: "in_go:pkg/version.AppVersion"   
version_pkg: "main.BuildVersion"        
```

### Extractor (`version`)
1. **Static String**: `"1.0.5"`
2. **File Parsers (`file:`)**: Read from JSON/YAML or raw files.
   * `"file:package.json|version"` (Reads the `version` key from JSON)
   * `"file:VERSION.txt"` (Reads raw text)
3. **AST Parser (`in_go:`)**: Reads Go source code dynamically without compilation.
   * `"in_go:pkg/config.AppVersion"` (Finds `const AppVersion` in `pkg/config`).

### Injector (`version_pkg`)
Craft injects the extracted version into this variable path (e.g., `main.BuildVersion`) at compile time.

---

## 4. Cross-Platform Targets

```yaml
build_all: false 
platforms:
  - "linux/amd64"
  - "windows/amd64"
```

* **`platforms`**: Array of `GOOS/GOARCH` targets. Use `"current"` to build for the host OS.
* **`build_all`**: If `true`, overrides `platforms` and compiles concurrently for Windows, Linux, and macOS (AMD64 & ARM64).

---

## 5. Build Optimization & Flags

```yaml
strip_debug: true      
trimpath: true         
cgo_enabled: false     
race: false            
tags: ["pro"]   
```

* **`strip_debug`**: Removes DWARF symbols (`-s -w`), massively reducing binary size.
* **`trimpath`**: Removes host absolute paths from the binary for reproducible, secure builds.
* **`cgo_enabled`**: Enable C-language interop. (Warning: Limits cross-compilation capability without explicit toolchains like `zig cc`).
* **`race`**: Enables data race detection.
* **`tags`**: Applies Go build tags.

---

## 6. HTML/JS/CSS Minifier (Asset Optimization)

Craft includes a blazing-fast built-in asset minifier that runs automatically during the build phase.

```yaml
minify:
  enabled: true
  extensions: [".html", ".css", ".js", ".json", ".svg"]
  dirs: ["web/templates", "assets"]
```

* **`enabled`**: Activates the minifier engine.
* **`dirs`**: Arrays of directories to scan (recursive).
* **`extensions`**: The file types to minify. Removes whitespace, comments, and optimizes frontend assets without needing Node.js, Webpack, or external tools!

---

## 7. Hot-Reload Watcher (`craft dev`)

```yaml
dev:
  watch:
    delay_ms: 500
    include_exts: ["go", "html"]
    exclude_dirs: ["bin", "vendor", "node_modules", ".git"]
```

* **`delay_ms`**: A highly intelligent debouncer. Waits for rapid sequential saves (e.g., IDE "Save All") to settle before triggering a single, optimized rebuild.
* **`include_exts`**: Extensions to monitor for changes.
* **`exclude_dirs`**: Ignored directories. Crucial for saving CPU and memory.

---

## 8. Custom Commands & Macros (Task Runner)

Craft acts as an integrated Task Runner, allowing you to define custom terminal commands.

```yaml
commands:
  lint:
    - "echo 'Linting...'"
    - "craft check"
  deploy:
    - "echo 'Deploying to staging...'"
```

* **Custom Tasks**: Define any sequence of shell commands. You can trigger them via `craft lint` or `craft deploy`.
* **Execution**: Commands are executed sequentially. If any command fails, the task is aborted.

---

## 9. Build Lifecycle Scripts

Craft allows you to execute commands automatically at specific stages of the application lifecycle.

```yaml
scripts:
  pre_build:
    - "npm run build:ui"
  post_build:
    - "echo 'Build completed'"
```

* **`pre_build`**: Runs right before compilation (e.g. asset generation, swag init).
* **`post_build`**: Runs after compilation finishes (e.g. copying files, zipping).
* **`pre_run`**: Runs before `craft run` executes the binary.
* **`post_run`**: Runs after `craft run` exits.

---

## 10. Build Profiles (Workflow Routing)

Dynamically override your configuration for specific environments via CLI (e.g., `craft build -P release`).

```yaml
profiles:
  release:
    output_dir: "releases/v1"
    strip_debug: true
    build_all: true
  docker:
    exact_name: true
    platforms: ["linux/amd64"]
```

---

## 11. What's Next?

With your `.craft.yaml` properly configured, you have unlocked a professional, deterministic build pipeline. Your codebase is now ready to scale from a single binary to a cross-platform matrix with automated dependency tracking and asset minification out of the box.

*   To learn more about how Craft executes custom shell scripts during the build lifecycle, read the [Task Runner Guide](task-runner.md).
*   For a deep dive into the flags that override these YAML configurations on the fly, visit the [CLI Reference](cli-reference.md).
