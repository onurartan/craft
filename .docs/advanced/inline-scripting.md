# Inline Scripting

Go is traditionally a compiled language. To run a Go file, you typically need to create a `go.mod`, define a `package main`, and set up a project directory. While this is fantastic for large projects, it makes writing quick, one-off scripts (like in Python or Bash) cumbersome.

Craft changes this paradigm completely: with the **Inline Scripting** feature, you can use Go just like a scripting language.

## How It Works

When you provide a `.go` file with the `--script` flag, Craft completely bypasses the standard project compilation pipeline. Behind the scenes, it automatically performs the following steps:

1. Verifies the existence of the script file.
2. Creates an isolated folder in the operating system's temporary directory (`/tmp/craft-script-*` or `%TEMP%`).
3. Copies your script file into this temporary folder.
4. Automatically runs `go mod init craft-inline` to create a temporary module.
5. Runs `go mod tidy` to automatically resolve and download all `import` dependencies found in your script.
6. Executes the script using `go run`.
7. Once execution completes, the temporary folder is completely purged (`defer os.RemoveAll`).

Because of this isolated pipeline, no compilation artifacts (no `bin/` folder, no `go.mod` file) are ever left behind in your working directory.

> [!IMPORTANT]
> If your script uses external packages (`import "github.com/pterm/pterm"`), Craft downloads them automatically. You do not need to run `go get` or `go mod init` manually.

## Usage

### Standard Execution

Write a single Go file (e.g., `script.go`) and run it directly using the `--script` flag:

```bash
craft run --script script.go
```

Craft detects the `--script` flag and automatically enters Inline Script mode.

### Passing Arguments

You can easily pass extra arguments to your script:

```bash
craft run --script script.go --verbose --count=5
```

These arguments are passed directly to the underlying `go run` command.

### Example Script

```go
package main

import (
    "fmt"
    "github.com/pterm/pterm"
)

func main() {
    pterm.Success.Println("Inline scripting works flawlessly!")
    fmt.Println("No go.mod, no project setup, just pure Go.")
}
```

When you run this file with `craft run --script script.go`, Craft will silently download the `pterm` package in the background, compile, and execute your script.

> [!TIP]
> **Use Cases:** Inline scripting is ideal for database seed scripts, quick data processing tools, or cron jobs. You get the type-safety and performance of Go without the overhead of project scaffolding.

> [!NOTE]
> If your current directory has a `.craft.yaml` with a `toolchain` defined, your inline scripts will execute using that isolated toolchain. This ensures even your scripts stay perfectly compatible with your project's target Go version.
