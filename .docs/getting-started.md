# Getting Started with Craft

Craft brings a modern, fast, and unified developer experience to the Go ecosystem. Whether you are bootstrapping a new project or migrating an existing codebase, Craft makes the process seamless.

## Installation

Since Craft is built entirely in Go, it compiles to a single binary. You can install it directly via the Go toolchain. Ensure your `~/go/bin` (or `%USERPROFILE%\go\bin` on Windows) is available in your system's `PATH`.

```bash
go install github.com/onurartan/craft@latest
```

Verify the installation by running the diagnostic tool:

```bash
craft doctor
```

---

## 1. Starting a New Project (The Easy Way)

If you are starting a fresh project, use the **Scaffolding Wizard**. Craft includes an interactive CLI that templates out complete, production-ready project structures (like REST API Servers with Fiber/Gin or CLI tools).

```bash
craft create
```

You will be prompted to:
1. Select your project type.
2. Enter your module name.
3. Choose an output directory.

Craft will automatically generate the directory, create the necessary boilerplate files, initialize the `go.mod`, fetch dependencies, and generate a highly optimized `.craft.yaml` configuration.

---

## 2. Adopting Craft in an Existing Project

If you already have a Go project running with raw `go build` commands or other scripts, you can easily integrate Craft.

Navigate to your project root and run:

```bash
craft init
```

Craft will:
- Discover your entry point (e.g., `main.go` or `cmd/api/main.go`).
- Determine your project name from your `go.mod`.
- Generate a strictly-typed `.craft.yaml` file tailored to your project.

You can now immediately start building:

```bash
craft build
```

---

## Daily Developer Workflow

Craft is designed to be your daily driver. Here are the core commands you'll use regularly.

### ⚡ Rapid Development (Hot-Reload)
Stop manually stopping and restarting your server. Craft features a zero-config, ultra-fast development engine.

```bash
craft dev
```
Craft will monitor your source files (`.go`, `.yaml`, `.html`, etc.), securely compile the app to a hidden temporary directory, and automatically restart your server on every save.

### 📦 Building for Production
When it's time to compile your artifact, simply run:

```bash
craft build
```
This reads your `.craft.yaml`, strips debug symbols, optimizes the binary size, and places it in your configured `output_dir` (default: `bin/`).

### 🧽 Cleaning Up
To perform a deep clean of all generated artifacts and temporary files left behind by `craft dev`:

```bash
craft clean
```
Append `--all` to comprehensively purge all craft cache globally across your system.

### 🛡️ Pre-Flight Checks
Before committing your code, run the built-in continuous integration helper. It runs `fmt`, `vet`, and `test` sequentially and halts immediately if any step fails.

```bash
craft check
```

---

> [!TIP]
> Ready to unlock Craft's full potential? Check out the [Configuration Guide](configuration.md) to learn how to inject versions, target multiple platforms, and define custom Build Profiles!
