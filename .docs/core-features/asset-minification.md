# Built-in Asset Minification

Modern web applications require minified HTML, CSS, JS, and JSON assets to improve load times and save bandwidth. 

Traditionally, Go developers either ship raw, unoptimized templates or are forced to introduce heavy dependencies like `Node.js`, `Webpack`, or `esbuild` into their pure Go projects just to minify some CSS.

**Craft eliminates this pain.** It includes a blazing-fast, natively compiled asset minifier built directly into the Go build engine.

## Configuration

Enable the minifier in your `.craft.yaml`:

```yaml
minify:
  enabled: true
  extensions: [".html", ".css", ".js", ".json", ".svg", ".tpl"]
  dirs: ["web/templates", "public/assets"]
```

## How It Works

During `craft build`, the minifier process intercepts your designated `dirs`. 
For every directory listed (e.g., `public`), Craft:
1. Creates a mirrored output directory with the `_min` suffix (e.g., `public_min`).
2. Recursively scans the directory for files matching your `extensions`.
3. Compresses the matched files (stripping whitespace, optimizing SVGs, minifying JS/CSS logic) and places them in the `_min` directory.
4. Raw files (like `.png` or `.woff2`) that don't match the extensions are simply copied over so directory integrity is preserved.

### Usage in Go

Once your files are minified into `public_min`, you simply point your Go file server or `//go:embed` directive to the optimized directory:

```go
//go:embed public_min/*
var content embed.FS
```

> [!TIP]
> The Craft Minifier uses the industry-leading `tdewolff/minify` engine under the hood, ensuring perfectly safe and highly compressed frontend assets without ever installing Node.js.
