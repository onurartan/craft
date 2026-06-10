# Project Scaffolding

Craft provides a powerful scaffolding engine out of the box, allowing you to bootstrap new Go projects in seconds without writing boilerplate.

## Creating a New Project

Use the `create` command to start the interactive wizard:

```bash
craft create
```

> [!TIP]
> The interactive wizard is powered by `pterm`, providing a beautiful console UI with selection menus, spinners, and progress bars.

### Templates

Craft ships with several embedded templates designed for modern Go development:
*   **Fiber / Gin REST APIs**: Pre-configured routing, middleware, and `.craft.yaml` optimization.
*   **Cobra CLI Applications**: Standardized command-line structures for rapid CLI tool development.
*   **Blank Projects**: A minimalist starting point.

## Migrating Existing Projects

If you already have an existing Go project, Craft can adopt it intelligently:

```bash
craft init
```

This command will:
1.  Read your `go.mod` to determine the module path.
2.  Scan your directory structure to find the `main.go` entry point.
3.  Automatically generate a highly optimized `.craft.yaml` file tailored to your codebase.
