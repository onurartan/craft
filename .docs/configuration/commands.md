---
deprecated: true
---

> [!WARNING]
> **Deprecated Document**
> The contents of this document have been vastly expanded and migrated to a unified guide. 
> Please refer to the [Task Runner Guide](#core-features/task-runner) for the most up-to-date and comprehensive information.

# Commands & Task Runner Configuration

The `commands` block in `.craft.yaml` is one of Craft's most powerful features. It essentially acts as a built-in cross-platform task runner (like `Make`, `npm scripts`, or `Just`), deeply integrated into your build lifecycle.

It supports sequential arrays, OS-specific targets, and even dynamic command referencing!

---

## 1. Multi-Command Execution (Sequential)

You can define any arbitrary sequence of shell scripts within the `commands` block by providing a list (array) of commands. These will be executed sequentially.

```yaml
commands:
  lint:
    - "echo 'Running golangci-lint...'"
    - "golangci-lint run ./..."
```

**Usage:**

```bash
$ craft run lint
```

> [!TIP]
> If any step in the sequence fails (returns a non-zero exit code), the task runner immediately aborts the rest of the sequence.

---

## 2. OS-Specific Commands

Often, commands differ depending on whether the developer is using Windows or a UNIX-based system. Craft natively supports OS-specific dictionaries within your tasks.

If Craft encounters an object instead of an array, it automatically resolves the command based on your current operating system (`windows`, `linux`, `macos`, or `default`).

```yaml
commands:
  open-api:
    windows: "swag.exe init --parseDependency --parseInternal"
    macos: "swag init --parseDependency --parseInternal"
    linux: "swag init --parseDependency --parseInternal"
    default: "swag init"

  clean:
    windows: "del /Q /S bin\\*"
    default: "rm -rf bin/*"
```

---

## 3. Command Referencing & Composition

You can dramatically reduce duplication in your `.craft.yaml` by composing smaller commands into larger ones. By prefixing a command string with `$`, Craft will dynamically resolve and call the referenced command block.

```yaml
commands:
  # Atomic commands
  lint: "golangci-lint run ./..."
  test: "go test -v ./..."
  build-ui: "npm run build"

  # Composite command using the '$' operator
  ci-pipeline:
    - "$lint"
    - "$test"
    - "$build-ui"
    - "echo 'CI pipeline completed successfully!'"
```

This acts exactly like invoking `craft run lint`, `craft run test`, etc., but keeps your configuration DRY (Don't Repeat Yourself).

---

## 4. Build Lifecycle Hooks (Pre & Post Build)

Craft allows you to hook directly into the build pipeline using the dedicated `scripts` block in your `.craft.yaml`. The supported lifecycle hooks are `pre_build`, `post_build`, `pre_run`, and `post_run`. These hooks natively support all the macro and OS-specific features supported by the `commands` block.

```yaml
scripts:
  pre_build:
    - "$open-api"
    - "echo 'Code generation completed.'"
  post_build:
    - "echo 'Build completed successfully!'"
  pre_run:
    - "echo 'Starting dev server...'"
```

### Execution Flow:

1. **Dependency Resolution**: Craft installs `toolchain` and runs `go mod tidy`.
2. **`pre_build` Hook**: Your pre-build scripts run (e.g., code generation, proto compilation).
3. **Compilation**: Go compiler compiles your binary.
4. **Minification**: Craft minifies your HTML/JS/CSS assets.
5. **`post_build` Hook**: Your post-build scripts run (e.g., zipping, uploading).

---

## 5. Macros & Variable Interpolation

Commands defined in `.craft.yaml` have automatic access to an incredibly powerful built-in Macro engine. Instead of relying on unreliable bash environment variables, Craft injects variables directly into your strings using the `{MACRO}` syntax before execution.

### Built-in Macros

- `{OS}` - The target operating system (e.g., `linux`, `windows`).
- `{ARCH}` - The target architecture (e.g., `amd64`, `arm64`).
- `{SRC_DIR}` - The entry point directory defined in your config.
- `{OUT_DIR}` - The destination directory defined in your config.
- `{APP_NAME}` - The base name of your application.
- `{WORKSPACE_ROOT}` - The root directory of your project.
- `{TIMESTAMP}` - Current time formatted as `YYYYMMDD_HHMM`.
- `{GIT_COMMIT}` - The current Git commit hash.
- `{ARGS}` - Any extra arguments passed via CLI.

### Dynamic Binary Path (`{APP_BIN_PATH}`)

Craft knows exactly where your compiled binary will be. You can use the `{APP_BIN_PATH}` macro to refer to the generated executable, preventing hardcoded paths.

Modifiers:

- `{APP_BIN_PATH:NOBUILD}`: Returns the path but tells Craft **not** to trigger an automatic compile before running the task.
- `{APP_BIN_PATH:linux/amd64}`: Gets the binary path for a specific platform.

### Custom Variables (`envs`)

You can define your own custom variables under `commands.envs`. Craft handles these in two incredibly powerful ways:

1. **As Macros:** You can use them directly in your commands using `{KEY}`.
2. **As Shell Environment Variables:** Craft natively injects them into the spawned OS process. This means your shell scripts or Node/Go apps can read them using `$KEY` (Linux/Mac) or `%KEY%` (Windows).

```yaml
commands:
  envs:
    DB_USER: "admin"
    DB_HOST: "localhost:5432"

  migrate:
    db: "echo Migrating DB on {OS} for {DB_USER} at {DB_HOST}"
```

## 6. Background Tasks in Watch Mode

If you are using `craft dev`, the task runner intelligently triggers your hooks on every hot-reload cycle!

1. When a file changes, `pre_build` runs.
2. The binary recompiles.
3. `post_build` runs.
4. The server restarts.

This is perfect for TailwindCSS compilation during development:

```yaml
scripts:
  pre_build:
    - "npx tailwindcss -i ./src/input.css -o ./assets/output.css"
```

---

## 7. Running Commands & Execution Flow

Once defined, there are two equivalent ways to execute custom tasks:

1. **Direct Invocation:**

```bash
$ craft lint
$ craft migrate db --force
```

2. **Run Syntax (`npm`-style):**

```bash
$ craft run lint
$ craft run migrate db --force
```

### Potential Collisions & The `--` Bypass

Since `craft run` is originally designed to compile and execute your _Go binary_, running `craft run <arg>` will first check if `<arg>` is a defined task in `.craft.yaml`. If it is, the task executes. If it isn't, the Go project compiles and `<arg>` is passed to the Go application.

If you specifically need to pass an argument to your Go application that perfectly matches the name of a custom task, use the double-dash `--` bypass operator to instruct Craft to ignore task matching:

```bash
# Executes the custom YAML task
$ craft run setup

# Compiles the project and passes "setup" to the Go binary
$ craft run -- setup
```

### Cross-Platform Shell Execution

Under the hood, Craft routes commands through the native shell of the host OS (`cmd.exe /C` on Windows, `/bin/sh -c` on Unix/macOS). This means native operators like `&&`, `||`, and standard output piping `>` are fully supported out-of-the-box in your `.craft.yaml` strings.
