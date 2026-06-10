# Task Runner (Custom Commands Engine)

Craft is far more than just a Go compilation tool; it includes a fully-featured, cross-platform **Task Runner** directly embedded within its engine. By defining tasks in your `.craft.yaml`, you can orchestrate your entire software development lifecycle—from code generation and frontend asset building, to database migrations and deployment—without relying on external tools like `Make`, `npm scripts`, or `Just`.

---

## 1. Defining Custom Tasks

You define your custom tasks under the `commands` block in your `.craft.yaml` file. Craft reads this block dynamically and registers each top-level key as a native CLI subcommand.

### The Anatomy of a Task

Tasks can be as simple as a single string, or as complex as nested, OS-specific arrays.

```yaml
commands:
  # 1. Simple Command
  # Easily execute a single shell instruction.
  seed: "go run ./cmd/db/seeder.go"

  # 2. Sequential Command Arrays
  # If any command fails (non-zero exit), the sequence aborts immediately.
  format:
    - "echo 'Running Go formatter...'"
    - "go fmt ./..."
    - "golangci-lint run"

  # 3. Cross-Platform OS-Specific Execution
  # Avoid messy shell 'if' statements. Craft automatically selects the correct command for the host OS.
  clean:
    windows: "del /Q /S bin\\*"
    macos: "rm -rf bin/*"
    linux: "rm -rf bin/*"
    default: "echo 'Unsupported OS'"
```

---

## 2. Command Composition (The `$` Operator)

To keep your configuration DRY (Don't Repeat Yourself), you can compose smaller, atomic commands into larger orchestrations. By prefixing a command string with `$`, Craft will dynamically resolve and inject the referenced command block.

```yaml
commands:
  # Atomic building blocks
  lint: "golangci-lint run ./..."
  test: "go test -v ./..."
  build-ui: "npm run build --prefix ./frontend"

  # Composite Pipeline
  ci-pipeline:
    - "$lint"
    - "$test"
    - "$build-ui"
    - "echo 'CI Pipeline completed successfully!'"
```
*Note: Craft resolves these references recursively, up to a safety depth of 4 to prevent infinite loops.*

---

## 3. Environment Variables (`envs`)

Instead of fighting with differing shell syntaxes to set environment variables (`export` on bash vs `set` on Windows cmd), Craft allows you to declare global environment variables that are natively injected into the spawned OS process of *every* task.

```yaml
commands:
  envs:
    DB_USER: "admin"
    DB_PASS: "supersecret"
    MIGRATE_DIR: "./db/migrations"

  migrate:
    # You can read these natively in the shell string using standard shell syntax...
    linux: "echo Migrating $DB_USER... && goose -dir $MIGRATE_DIR up"
    windows: "echo Migrating %DB_USER%... && goose.exe -dir %MIGRATE_DIR% up"
```

---

## 4. The Macro Engine

Craft provides a powerful Macro Engine. Macros are special placeholders formatted as `{MACRO_NAME}` that Craft securely resolves and interpolates into your strings *before* execution. Unlike shell variables, macros are 100% cross-platform.

### Available System Macros
- **`{OS}`**: The target operating system (`windows`, `linux`, `darwin`).
- **`{ARCH}`**: The target system architecture (`amd64`, `arm64`).
- **`{SRC_DIR}`**: The path to your main package (from `entry_point`).
- **`{OUT_DIR}`**: The output directory for artifacts (from `output_dir`).
- **`{APP_NAME}`**: The binary name of the project.
- **`{WORKSPACE_ROOT}`**: The absolute path to the directory containing `.craft.yaml`.
- **`{TIMESTAMP}`**: The current date/time in `YYYYMMDD_HHMM` format.
- **`{GIT_COMMIT}`**: The short hash of the current git HEAD (e.g., `a1b2c3d`).
- **`{ARGS}`**: Any extra arguments passed via the CLI. *(If you do not explicitly use `{ARGS}` in your string, Craft automatically appends them to the end of the command).*

### The `{APP_BIN_PATH}` Auto-Build Macro

When you want a task to execute your actual Go application, hardcoding the path (e.g., `./bin/app.exe`) is dangerous. Craft provides `{APP_BIN_PATH}` to resolve this dynamically.

> [!IMPORTANT]
> When Craft encounters the `{APP_BIN_PATH}` macro inside a command string, **it will silently and automatically trigger a compilation of your Go project** before executing the task. This guarantees you are always running the latest code!

- `{APP_BIN_PATH}`: Compiles the project and returns the executable path.
- `{APP_BIN_PATH:NOBUILD}`: Returns the path to the executable, but **skips** the compilation phase.

---

## 5. Build Lifecycle Hooks

Craft allows you to hook directly into the compilation pipeline. Using the `scripts` block, you can define tasks that execute automatically at specific stages of `craft build` or `craft run`.

These hooks natively support all the features of the `commands` block (Macros, OS-specific targets, Arrays, and `$` Composition).

```yaml
scripts:
  pre_build:
    - "swag init --parseDependency"      # Auto-generate Swagger docs
    - "npm run build --prefix ./web"     # Build React/Vue frontend
  post_build:
    - "echo 'Archiving release...'"
    - "tar -czf {APP_NAME}_{OS}_{TIMESTAMP}.tar.gz {OUT_DIR}/"
  pre_run:
    - "echo 'Starting application stack...'"
```

### The Lifecycle Execution Flow:
1. **Dependency Sync**: Craft automatically runs `go mod tidy` via the sandboxed toolchain.
2. **`pre_build`**: All your pre-build scripts execute sequentially.
3. **Compilation**: The Go compiler builds your binary.
4. **Minification**: Craft's built-in minifier compresses your frontend assets.
5. **`post_build`**: Your post-build scripts execute.
6. *(If running `craft run`)* **`pre_run`**: Scripts execute just before the binary starts.

---

## 6. Background Tasks in Watch Mode (`craft dev`)

The true power of the Task Runner shines during local development. When you use `craft dev` for hot-reloading, **Craft intelligently triggers your hooks on every single reload cycle!**

When a file change is detected:
1. The previous server instance gracefully shuts down.
2. The `pre_build` hook runs.
3. The binary is recompiled to a temporary directory.
4. The `post_build` hook runs.
5. The new server instance boots up.

### Real-World Example: TailwindCSS
Traditionally, you would need to run `npx tailwindcss --watch` in a separate terminal. With Craft, you just add it to your `pre_build` hook:

```yaml
scripts:
  pre_build:
    - "npx tailwindcss -i ./src/input.css -o ./assets/output.css"
```
Now, whenever you save an HTML file, Craft will instantly compile your Tailwind CSS and restart your Go server in one seamless motion!

---

## 7. Execution Syntax & Bypassing

You can execute your custom tasks from the terminal in two equivalent ways:

```bash
# 1. Direct Invocation
$ craft setup
$ craft migrate db --force

# 2. NPM-Style Invocation
$ craft run setup
$ craft run migrate db --force
```

### The `--` Bypass Operator

Since `craft run` is originally designed to execute your *Go application*, what happens if you want to pass an argument to your Go app that shares the exact same name as a custom task in your `.craft.yaml`? 

Craft will prioritize executing the YAML task. To bypass this and force Craft to pass the argument directly to your Go application, use the `--` operator:

```bash
# Executes the custom task named "seed" defined in .craft.yaml
$ craft run seed

# Compiles the project and passes the string "seed" directly to your Go binary
$ craft run -- seed
```

### Core Protection
Craft strictly protects its core commands. If you accidentally define a task named `build`, `dev`, `run`, `init`, or `toolchain`, Craft will issue a warning and ignore your custom definition, ensuring the core CLI remains functional.
