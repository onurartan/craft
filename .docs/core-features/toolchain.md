# Toolchain Management

Craft completely eliminates the need to install Go globally on your system. It automatically manages, downloads, and isolates Go toolchains for you. 

This guarantees that your project always builds with the exact compiler version defined in `.craft.yaml`, regardless of what OS or machine you are on.

---

## 1. Defining Your Toolchain
To tell Craft which Go version your project needs, simply add it to your `.craft.yaml`:

```yaml
toolchain: "go1.25.0"
```
When you run `craft build` or `craft dev`, if `go1.25.0` is not found locally, Craft will automatically download and install it in the background before compiling.

---

## 2. CLI Commands (`craft toolchain`)

Craft provides a powerful set of CLI commands to manage your local environments manually.

### The Interactive Dashboard
```bash
$ craft toolchain
```
Running this command with no arguments opens a sleek, interactive terminal UI. It displays your active `CRAFT_HOME`, operating system, architecture, and the total disk space currently consumed by your cached toolchains.

### `install`
Downloads and installs a specific Go version into your isolated `~/.craft/toolchains` directory. 

```bash
$ craft toolchain install 1.25.0
```

**Advanced Usage & Flags:**
* **`--remote` (`-r`)**: Instead of typing the version manually, this flag queries the official `go.dev/dl` API and opens an interactive menu. You can scroll through all historical Go releases using your arrow keys and press Enter to install one.
  ```bash
  $ craft toolchain install --remote
  ```
* **`--no-cache`**: By default, Craft caches downloaded `.tar.gz` or `.zip` archives. If you suspect an archive is corrupted, use this flag to force a fresh download directly from the server.
  ```bash
  $ craft toolchain install 1.25.0 --no-cache
  ```

### `list`
Displays all installed Go toolchains on your machine.

```bash
$ craft toolchain list
```

**Advanced Usage & Flags:**
* **`--remote` (`-r`)**: Queries `go.dev` to list all globally available Go releases, instead of showing what's on your local disk.
* **`--select-mode`**: Opens an interactive dropdown menu to let you visually select a toolchain. When combined with `--remote`, it acts exactly like the interactive install.

### `use`
Sets the active Go toolchain for the current project. This command automatically updates your `.craft.yaml` file with the specified version.

```bash
$ craft toolchain use 1.25.0
```

### `remove`
Deletes a completely installed and extracted Go toolchain directory to free up disk space. (Note: This does not delete the downloaded `.tar.gz` archive cache).

```bash
$ craft toolchain remove 1.22.1
```

### `clean`
Craft intelligently caches downloaded toolchain archives (e.g., `go1.25.0.linux-amd64.tar.gz`) in `~/.craft/cache/archives` to speed up future reinstalls across multiple projects. Over time, this folder can grow large. The `clean` command allows you to reclaim disk space.

**Clean All Caches:**
Wipes the entire archive cache, deleting the `.tar.gz` and `.zip` files for *all* versions.
```bash
$ craft toolchain clean
```

**Clean Specific Version Cache:**
Deletes only the cached archive associated with a specific version, leaving other cached versions intact.
```bash
$ craft toolchain clean 1.25.0
```

---

## 3. How It Works (System Isolation)

To avoid polluting your global system, Craft never installs Go into standard system directories like `/usr/local/go` or `C:\Program Files\Go`.

Instead, all toolchains are sandboxed inside your user home directory:
`~/.craft/toolchains/<version>`

When Craft compiles your project, it dynamically injects the path of this isolated toolchain into the spawned process's environment variables (`GOROOT`, `PATH`). This means your host OS might have Go 1.19 installed, but Craft will seamlessly compile your project using the sandboxed Go 1.25.0 without any conflicts.

Craft also implements **Atomic Renaming** during downloads. If your internet connection drops while downloading a 100MB toolchain, the corrupted file is automatically discarded, ensuring you never end up with a broken compiler.
