# Hot-Reload Engine

Stop wasting time manually stopping, compiling, and restarting your server. Craft's `dev` command provides a robust, zero-configuration hot-reload environment.

```bash
craft dev
```

## How It Works

Craft's watcher is powered by `fsnotify`, deeply integrated with a custom **Debouncer**.

### Smart Debouncing
When you save a file in a modern IDE, it often saves multiple files simultaneously or triggers rapid consecutive writes. Traditional watchers trigger a build for every write, crushing your CPU. 

Craft uses a `delay_ms` (configurable, default 500ms). It waits until the flurry of disk writes settles down before executing exactly **one** build.

### Isolated Temporary Builds
Unlike other tools that clutter your root directory with `.exe` or bin files during development, Craft compiles to a hidden OS temporary directory (e.g., `/tmp/craft-dev-xyz` or `%TEMP%`).

> [!IMPORTANT]
> Your working directory remains perfectly clean. The compiled binary is automatically executed and instantly destroyed upon exit.

### Graceful Shutdowns
Before recompiling your code, the previous running instance must be terminated. 
Craft intercepts OS signals (`SIGINT`, `SIGTERM`) and gracefully kills the sub-process hierarchy. This prevents "Address already in use" errors or zombie processes that typically plague Go web development.
