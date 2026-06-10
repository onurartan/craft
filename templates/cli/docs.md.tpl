# {{ .ProjectName }} - CLI Application Documentation

Welcome to your powerful Command Line Interface application, bootstrapped with Craft! 

## 📦 Default Packages & Tech Stack

This template comes with the absolute gold standard for CLI development in Go:

- **`github.com/spf13/cobra`**: The leading CLI framework for Go. It is used by Kubernetes, Docker, GitHub CLI, and almost every major Go CLI project. Cobra provides simple routing for subcommands, flag parsing, and automatic `--help` generation.

## 🚀 Getting Started

### 1. Running Your CLI
You don't need to manually compile your CLI every time you want to test it. Craft can compile and run it in memory instantly:
```bash
craft run
```
If you want to pass arguments to your CLI (e.g. `--help` or `mycommand`), add `--` before the arguments:
```bash
craft run -- --help
```

### 2. Adding a New Subcommand
Open `main.go`. You can easily add nested commands to your CLI:
```go
// Add this above rootCmd.Execute()
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("{{ .ProjectName }} v1.0.0")
	},
}
rootCmd.AddCommand(versionCmd)
```
Now try running: `craft run -- version`

### 3. Adding New Packages
Use Craft's shorthand syntax to add new libraries effortlessly:
```bash
craft add gh:pterm/pterm  # Great for modern terminal animations and colors!
```
*(If you simply write `import "github.com/pterm/pterm"`, Craft's Auto-Install will download it automatically when you run `craft run`!)*

### 4. Cross-Platform Distribution
When your CLI is ready to be shared with the world, you can compile it for Windows, macOS, and Linux simultaneously with a single command:
```bash
craft build --all
```
Your compiled binaries will be sitting in the `bin/` directory, optimized and stripped of debug symbols.

## 📂 Project Structure
- `main.go`: Application entry point and root command setup.
- `.craft.yaml`: Your project's core configuration. Use the `commands` section to create custom tasks (like `craft release` or `craft test`).

Happy building! ⚡
