# Smart Package Management

Go's built-in module system (`go mod`) is incredibly robust, but managing dependencies via raw `go get`, manually editing `go.mod`, and remembering to run `go mod tidy` can interrupt your development flow. 

Craft solves this by providing a unified, high-level **Package Management API** directly within the CLI, taking inspiration from modern package managers like `npm`, `yarn`, and Python's `uv`.

## Adding Packages

Instead of using `go get`, use `craft add`. This command acts as a smart wrapper that not only downloads the package but ensures your module tree is instantly synchronized.

```bash
# Add a single package
craft add github.com/gofiber/fiber/v2

# Or use the built-in Registry Aliases (e.g., gh: for GitHub)
craft add gh:gofiber/fiber/v2

# Add multiple packages simultaneously
craft add gh:joho/godotenv x:crypto
```

### Registry Aliases (Shortcuts)

To save you from repeatedly typing out long domain names, Craft provides built-in aliases that automatically expand before passing the package to the Go toolchain. You can use these with `add`, `remove`, and `install` commands.

* **`gh:`** expands to `github.com/`
* **`gl:`** expands to `gitlab.com/`
* **`bb:`** expands to `bitbucket.org/`
* **`x:`**  expands to `golang.org/x/`
* **`in:`** expands to `gopkg.in/`

For example, `craft add in:yaml.v3` is identical to typing `go get gopkg.in/yaml.v3`.

> [!TIP]
> **Auto-Install Magic:** If you have `auto_install: true` in your `.craft.yaml`, you often don't even need to use `craft add`. Just import the new package in your `.go` file and hit save. Craft's watcher will automatically detect the missing dependency, download it, and compile without breaking your flow!

## Removing Packages

Removing a package in standard Go usually requires deleting the `import` statement and running `go mod tidy`. With Craft, you can explicitly uninstall a package from your project using `craft remove`.

```bash
craft remove github.com/joho/godotenv
```

This command automatically:
1. Safely removes the dependency from your `go.mod`.
2. Cleans up the `go.sum` file.
3. Ensures your dependency tree is perfectly healthy after the removal.

## Syncing Dependencies

When cloning a new repository or after pulling major changes from Git, your local module cache might be out of sync with the `go.mod` file. 

```bash
craft sync
```

This command works similarly to `uv sync`. It forcefully scans your project, downloads all missing modules, trims orphaned dependencies, and verifies the cryptographic checksums in your `go.sum` to ensure a pristine and secure build environment.

---

## Tidy & Verify

If you prefer the granular control of native Go commands but still want the benefits of Craft's isolated toolchains, you can use the direct wrapper commands.

### `craft tidy`
Acts exactly like `go mod tidy` but uses your sandboxed `toolchain` to prevent version mismatch errors across different developer machines.

```bash
craft tidy
```

### Module Security
Because Craft uses isolated toolchains, running `craft build` automatically verifies the cryptographic signatures of your downloaded packages against the global checksum database. If a package has been tampered with or its checksum differs from `go.sum`, Craft will refuse to build, protecting your pipeline from supply chain attacks.

---

## 4. Why Not Just Use `go get`?

You absolutely can still use `go get` by prefixing it with craft: `craft run "go get ..."`. However, using `craft add` and `craft sync` ensures that you are utilizing the isolated compiler environment defined in `.craft.yaml`, preventing the classic issue where Developer A uses Go 1.19 and Developer B uses Go 1.25, creating conflicting `go.mod` files.
