# Smart Error Parser

The default Go compiler (`go build`) often spits out dense, hard-to-read error messages. When working with large codebases, tracking down the exact file, line, and syntax error from a wall of raw compiler text slows down development.

## Elegant Terminal UI

Craft deeply integrates with `pterm` to intercept and rewrite Go compiler errors in real-time. 

When a build fails, Craft:
1. Catches the raw `stderr` from the Go compiler.
2. Uses Regex to parse the exact file path, line number, and column.
3. Formats the output into a beautiful, color-coded hierarchy.

```text
✖ Build Failed [linux/amd64]
  ↳ cmd/server/main.go:42
    undefined: fiber.New()
```

## Actionable Hints

Beyond just formatting errors, Craft's Smart Parser acts as a pair programmer. It analyzes the error message and injects **actionable hints** to help you resolve the issue faster.

For example, if you encounter a missing module error:
```text
  ↳ config/loader.go:12
    no required module provides package github.com/spf13/viper
    Hint: Run 'craft build' again to auto-install or 'go mod tidy'.
```

If you encounter an undefined variable:
```text
  ↳ services/auth.go:88
    undefined: jwtSecret
    Hint: Did you misspell the variable or forget an import?
```

> [!NOTE]
> The Error Parser works flawlessly in both standard `craft build` execution and during continuous hot-reloading in `craft dev`, keeping your terminal clean and your focus sharp.
