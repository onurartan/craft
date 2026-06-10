// init.go: Orchestrates project initialization and configuration scaffolding.
// Copyright (c) 2026 Onurartan/Craft Contributors.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pterm/pterm"
)

// ExecuteInitProcess scans the environment and generates a structured craft.yaml.
func ExecuteInitProcess(args []string) error {
	if ConfigExists() {
		pterm.Warning.Printf("Configuration already exists: %s\n", ConfigFileName)
		return nil
	}

	pterm.Printf("%s\n\n", pterm.NewStyle(pterm.FgCyan, pterm.Bold).Sprint("▶ Project Discovery & Analysis"))

	// DISCOVERY
	projectName := resolveProjectName(args)
	entryPoint := resolveEntryPoint()
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH

	// PRESENTATION
	pterm.DefaultTable.WithData([][]string{
		{pterm.FgDarkGray.Sprint("ANALYZED FIELD"), pterm.FgDarkGray.Sprint("RESOLVED VALUE"), pterm.FgDarkGray.Sprint("SOURCE")},
		{pterm.FgWhite.Sprint("Project Identity"), pterm.FgLightCyan.Sprint(projectName), pterm.FgDarkGray.Sprint("go.mod/dir")},
		{pterm.FgWhite.Sprint("Execution Entry"), pterm.FgLightCyan.Sprint(entryPoint), pterm.FgDarkGray.Sprint("filesystem")},
		{pterm.FgWhite.Sprint("Host Platform"), pterm.FgLightMagenta.Sprintf("%s/%s", hostOS, hostArch), pterm.FgDarkGray.Sprint("runtime")},
		{pterm.FgWhite.Sprint("Output Target"), pterm.FgLightGreen.Sprint(DefaultDistDir), pterm.FgDarkGray.Sprint("default")},
	}).WithHasHeader().WithSeparator("   ").Render()

	if err := GenerateDefaultConfig(ConfigFileName, projectName); err != nil {
		pterm.Error.Printf("Failed to write %s: %v\n", ConfigFileName, err)
		return err
	}

	// Create a default .version file
	versionPath := filepath.Join(filepath.Dir(ConfigFileName), ".version")
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		os.WriteFile(versionPath, []byte("0.1.0\n"), 0644)
	}

	pterm.Println()
	pterm.Success.Printf("Generated optimized configuration: %s\n", pterm.FgMagenta.Sprint(ConfigFileName))

	pterm.Info.Println("Next steps:")
	pterm.Printf("  1. Review %s to verify your entry point.\n", pterm.FgCyan.Sprint(ConfigFileName))
	pterm.Printf("  2. Run %s to start rapid development.\n", pterm.FgGreen.Sprint("craft dev"))

	return nil
}

// GenerateDefaultConfig generates and writes the default .craft.yaml file.
func GenerateDefaultConfig(path string, projectName string) error {
	entryPoint := resolveEntryPoint()
	sysGoVer := getSystemGoVersion()
	smartConfig := fmt.Sprintf(`# Craft Build Utility Configuration
# Github: https://github.com/onurartan/craft
# Documentation: %[1]s

# --- METADATA, TOOLCHAIN & PATHS ---
# Learn more: %[1]s#2-metadata-toolchain--layout
name: "%[2]s" # Name of the compiled binary
toolchain: "%[5]s" # Enforce a specific Go compiler version for this project (e.g. "1.22.1")
entry_point: "%[3]s" # Directory containing the main package
output_dir: "%[4]s" # Default destination directory for build artifacts
exact_name: true # Omit OS/Arch suffixes from the binary name

# --- VERSIONING & LDFLAGS INJECTION ---
# Learn more: %[1]s#3-versioning--dynamic-ldflags
version: "file:.version" # Application version string (Can be 'in_go:pkg.Var', 'file:VERSION', or plain string)
version_pkg: "" # Package path to inject the version via LDFLAGS (e.g., main.Version)

# --- PLATFORM TARGETS ---
# Learn more: %[1]s#4-cross-platform-targets
build_all: false # Compile for all major OS/Arch combinations concurrently
platforms:
  - "current" # Compiles only for your active host machine
  # - "linux/amd64"
  # - "linux/arm64"
  # - "windows/amd64"
  # - "darwin/amd64"
  # - "darwin/arm64"

# --- BUILD PROFILES (Workflow Overrides) ---
# Learn more: %[1]s#10-build-profiles
# Define specific build scenarios. Use 'craft build -P <profile_name>' to trigger.
# Values defined in a profile will override the global settings above.
#profiles:
#  npm:
#    output_dir: "npm/bin" # Route artifacts to a specific NPM package directory
#    build_all: true # Override local build and compile for all systems
#    exact_name: false # Ensure OS/Arch suffixes are appended for distribution
#
#  release:
#    output_dir: "releases/v1" # Isolate production release artifacts
#    platforms:
#      - "linux/amd64"
#      - "windows/amd64" # Restrict this profile to specific target architectures

# --- BUILD OPTIMIZATION ---
# Learn more: %[1]s#5-build-optimization--flags
strip_debug: true # Exclude DWARF symbols to minimize binary size
trimpath: true # Remove host absolute paths for privacy and reproducible builds
cgo_enabled: false # Enable C-language interoperability
race: false # Enable data race detector (adds runtime overhead)
tags: [] # Custom Go build tags (e.g., ["pro", "dev"])

# --- DEVELOPMENT & HOT-RELOAD ---
# Learn more: %[1]s#6-hot-reload-engine-dev-mode
dev:
  watch:
    delay_ms: 500 # Debounce delay (in milliseconds) before triggering a rebuild
    include_exts: ["go", "html", "tpl", "env", "yaml"] # File extensions to monitor
    exclude_dirs: ["bin", "tmp", "vendor", "node_modules", ".git", "assets", "testdata"] # Ignored directories
    exclude_files: [".craft.yaml"] # Specific files to ignore

# --- MAGIC FEATURES (Craft v2) ---
# Learn more: %[1]s#2-metadata-toolchain--layout
auto_install: true # Auto-resolves and installs missing modules (e.g. "go get") during build/run

# Learn more: %[1]s#7-asset-minification
minify:
  enabled: false # If true, Craft will compress assets (HTML/CSS/JS) before embedding
  dirs: ["public"] # Directories to compress. Compressed files will be saved in "public_min"
  extensions: [".html", ".css", ".js", ".json", ".svg"]

# --- BUILD LIFECYCLE HOOKS ---
# Learn more: %[1]s#8-build-lifecycle-hooks-scripts
# Execute OS-specific scripts at specific stages of the build/run cycle.
#scripts:
#  pre_build:
#    - "echo 'Running code generation...'"
#    - "swag init"
#  post_build:
#    - "echo 'Build successful, artifact ready!'"
#  pre_run:
#    - "echo 'Starting application dependencies...'"
#  post_run:
#    - "echo 'Application stopped.'"

# --- TASK RUNNER (Custom Commands) ---
# Learn more: %[1]s#9-task-runner-custom-commands
# Define custom macros and tasks here. Run them using 'craft <command>' or 'craft run <command>'.
commands:
  envs:
    # Define your custom variables here.
    # EX_VAR: "example_value"
  
  # Example: 
  # setup:
  #   - "go get ./..."
  #   - "echo Setup done on {OS}!"
`, CraftDocsURL, projectName, entryPoint, DefaultDistDir, sysGoVer)

	return os.WriteFile(path, []byte(smartConfig), 0644)
}

// resolveProjectName determines the name via args, go.mod or directory name.
func resolveProjectName(args []string) string {
	if len(args) > 0 {
		return args[0]
	}

	if modData, err := os.ReadFile("go.mod"); err == nil {
		lines := strings.Split(string(modData), "\n")
		if len(lines) > 0 && strings.HasPrefix(lines[0], "module ") {
			fullModName := strings.TrimSpace(strings.TrimPrefix(lines[0], "module "))
			parts := strings.Split(fullModName, "/")
			return parts[len(parts)-1]
		}
	}

	dir, _ := os.Getwd()
	return filepath.Base(dir)
}

// resolveEntryPoint scans for root main.go or standard cmd/ layout.
func resolveEntryPoint() string {
	if _, err := os.Stat("main.go"); err == nil {
		return "."
	}

	entries, err := os.ReadDir("cmd")
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				mainPath := filepath.Join("cmd", e.Name(), "main.go")
				if _, err := os.Stat(mainPath); err == nil {
					return filepath.ToSlash(filepath.Join("cmd", e.Name()))
				}
			}
		}
	}

	return "."
}

// getSystemGoVersion detects the local go version for the init template.
func getSystemGoVersion() string {
	cmd := exec.Command("go", "env", "GOVERSION")
	out, err := cmd.Output()
	if err != nil {
		return "1.22.1"
	}
	ver := strings.TrimSpace(string(out))
	ver = strings.TrimPrefix(ver, "go")
	if ver == "" {
		return "1.22.1"
	}
	return ver
}
