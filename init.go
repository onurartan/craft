// init.go: Orchestrates project initialization and configuration scaffolding.
// Copyright (c) 2026 Onurartan/Craft Contributors.

package main

import (
	"fmt"
	"os"
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

	pterm.DefaultSection.Println("Project Discovery & Analysis")
	
	// DISCOVERY
	projectName := resolveProjectName(args)
	entryPoint := resolveEntryPoint()
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH

	// PRESENTATION
	pterm.DefaultTable.WithData([][]string{
		{"Analyzed Field", "Resolved Value", "Source"},
		{"Project Identity", pterm.FgCyan.Sprint(projectName), "go.mod/dir"},
		{"Execution Entry", pterm.FgCyan.Sprint(entryPoint), "filesystem"},
		{"Host Platform", pterm.FgMagenta.Sprintf("%s/%s", hostOS, hostArch), "runtime"},
		{"Output Target", pterm.FgGreen.Sprint(DefaultDistDir), "default"},
	}).WithHasHeader().WithBoxed().Render()

	smartConfig := fmt.Sprintf(`# Craft Build Utility Configuration
# Github: https://github.com/onurartan/craft
# Documentation: https://craft.trymagic.xyz

# --- METADATA & PATHS ---
name: "%s" # Name of the compiled binary
entry_point: "%s" # Directory containing the main package
output_dir: "%s" # Default destination directory for build artifacts
exact_name: true # Omit OS/Arch suffixes from the binary name

# --- VERSIONING & LDFLAGS INJECTION ---
version: "0.1.0" # Application version string (Can be 'in_go:pkg.Var' or 'file:VERSION')
version_pkg: "" # Package path to inject the version via LDFLAGS (e.g., main.Version)

# --- PLATFORM TARGETS ---
build_all: false # Compile for all major OS/Arch combinations concurrently
platforms:
  - "current" # Compiles only for your active host machine
  # - "linux/amd64"
  # - "linux/arm64"
  # - "windows/amd64"
  # - "darwin/amd64"
  # - "darwin/arm64"

# --- BUILD PROFILES (Workflow Overrides) ---
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
strip_debug: true # Exclude DWARF symbols to minimize binary size
trimpath: true # Remove host absolute paths for privacy and reproducible builds
cgo_enabled: false # Enable C-language interoperability
race: false # Enable data race detector (adds runtime overhead)
tags: [] # Custom Go build tags (e.g., ["pro", "dev"])

# --- DEVELOPMENT & HOT-RELOAD ---
dev:
  watch:
    delay_ms: 500 # Debounce delay (in milliseconds) before triggering a rebuild
    include_exts: ["go", "html", "tpl", "env", "yaml"] # File extensions to monitor
    exclude_dirs: ["bin", "tmp", "vendor", "node_modules", ".git", "assets", "testdata"] # Ignored directories
    exclude_files: [".craft.yaml"] # Specific files to ignore
`, projectName, entryPoint, DefaultDistDir)

	if err := os.WriteFile(ConfigFileName, []byte(smartConfig), 0644); err != nil {
		pterm.Error.Printf("Failed to write %s: %v\n", ConfigFileName, err)
		return err
	}

	pterm.Println()
	pterm.Success.Printf("Generated optimized configuration: %s\n", pterm.FgMagenta.Sprint(ConfigFileName))
	
	pterm.Info.Println("Next steps:")
	pterm.Printf("  1. Review %s to verify your entry point.\n", pterm.FgCyan.Sprint(ConfigFileName))
	pterm.Printf("  2. Run %s to start rapid development.\n", pterm.FgGreen.Sprint("craft dev"))

	return nil
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