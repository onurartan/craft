# The Story of Craft

Every tool is born out of a specific frustration. For Craft, that frustration was the repetitive, boilerplate-heavy nature of building Go applications.

For a long time, I used a crude version of Craft. It wasn't a CLI; it was just a `scripts/gocraft.go` file that I shamelessly copy-pasted from project to project. It did the heavy lifting, but carrying it around felt wrong. 

The Go toolchain is incredibly powerful, but interacting with it for complex, multi-platform releases can be exhausting. Every time I needed to compile binaries for Linux, Windows, and macOS, I found myself wrestling with `GOOS` and `GOARCH` environment variables. I tried Makefiles, but writing OS-specific bash scripts and dealing with syntax quirks quickly turned my project roots into a messy graveyard of build instructions. 

I needed something lightweight that just *worked* without draining my energy. 

So, I converted my trusty script into a standalone CLI tool and named it **Craft**. Initially, I built it purely for myself. It was my secret weapon to bypass Makefiles and instantly cross-compile my apps. But as I added more features—like a zero-config hot-reload engine for those who didn't want to mess with bloated tools like `air`—I realized something: I wasn't the only developer facing this problem.

I decided to open-source Craft. It is designed for developers who want the immense power of the Go toolchain wrapped in a frictionless, declarative, and elegant interface. 

---

## How Craft Works

Craft acts as a smart orchestrator over the native Go compiler. It doesn't replace Go; it empowers it. By defining your intentions in a single file, Craft handles the tedious environment variable injections, cross-compilation matrices, and log parsing automatically.

### The Source of Truth: `.craft.yaml`

Instead of imperative shell commands, Craft uses a declarative YAML configuration. You define *what* you want, and Craft figures out *how* to build it.

```yaml
name: "api-server"
entry_point: "."
output_dir: "bin"       

# Dynamic LDFLAGS Injection
version: "in_go:main.AppVersion"        
version_pkg: "main.Version"         

# Default Targets
build_all: false        
platforms: ["current"] 

# Hot-Reload Engine
dev:
  watch:
    enabled: true
    delay_ms: 500
    include_exts: ["go", "env"]
    exclude_dirs: ["bin", "tmp", "vendor", "node_modules"]
```

With this simple file, running `craft build` automatically injects your version via `-ldflags`, applies `-trimpath` and `-s -w` for production optimization, and drops the compiled binary right into your `bin/` folder. No bash required.

### The Masterpiece: Build Profiles

The real reason developers cling to Makefiles is to handle different deployment scenarios (e.g., `make build-local` vs `make build-release`). 

Craft eliminates this completely with **Profiles**. You can define specific workflows in your `.craft.yaml` that override your base settings dynamically.

```yaml
profiles:
  # For local NPM distribution
  npm:
    output_dir: "npm/bin"  
    build_all: true        
    exact_name: false
  
  # For production GitHub releases
  release:
    output_dir: "releases/v1"
    platforms: ["linux/amd64", "windows/amd64", "darwin/arm64"]
    strip_debug: true
```

Need to package your app for an NPM release? Just run:
```bash
$ craft build -P npm
```
Craft instantly overrides the default `bin/` directory, compiles for all operating systems concurrently, and perfectly formats the output names into `npm/bin/`. 

### Simplicity is Everything

Software engineering is already complex enough; your build tools shouldn't add to the cognitive load. Craft was built on a single philosophy: **Simplicity is everything.** Write your code, define your target, and let Craft handle the rest.