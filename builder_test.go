package main

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// resetBuilderGlobals ensures a clean state for AppConfig before each test.
func resetBuilderGlobals() {
	AppConfig = Config{}
}

func TestResolveTargets(t *testing.T) {
	tests := []struct {
		name            string
		setupConfig     func()
		expectedTargets []BuildTarget
	}{
		{
			name: "BuildAll true returns major matrix",
			setupConfig: func() {
				AppConfig.BuildAll = true
			},
			expectedTargets: []BuildTarget{
				{"linux", "amd64"}, {"linux", "arm64"},
				{"windows", "amd64"},
				{"darwin", "amd64"}, {"darwin", "arm64"},
			},
		},
		{
			name: "Specific platforms",
			setupConfig: func() {
				AppConfig.Platforms = []string{"linux/amd64", "windows/arm64"}
			},
			expectedTargets: []BuildTarget{
				{"linux", "amd64"},
				{"windows", "arm64"},
			},
		},
		{
			name: "Current platform keyword",
			setupConfig: func() {
				AppConfig.Platforms = []string{"current"}
			},
			expectedTargets: []BuildTarget{
				{runtime.GOOS, runtime.GOARCH},
			},
		},
		{
			name: "Fallback to host when nothing specified",
			setupConfig: func() {
				AppConfig.BuildAll = false
				AppConfig.Platforms = []string{}
			},
			expectedTargets: []BuildTarget{
				{runtime.GOOS, runtime.GOARCH},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetBuilderGlobals()
			tc.setupConfig()

			got := Builder.ResolveTargets()
			if !reflect.DeepEqual(got, tc.expectedTargets) {
				t.Errorf("expected %v, got %v", tc.expectedTargets, got)
			}
		})
	}
}

func TestGetBinaryPath(t *testing.T) {
	tests := []struct {
		name         string
		target       BuildTarget
		totalTargets int
		setupConfig  func()
		expectedPath string
	}{
		{
			name:         "Single target standard build",
			target:       BuildTarget{"linux", "amd64"},
			totalTargets: 1,
			setupConfig: func() {
				AppConfig.Name = "app"
				AppConfig.OutputDir = "bin"
				AppConfig.ExactName = false
			},
			expectedPath: filepath.Clean("bin/app"),
		},
		{
			name:         "Multi target appends OS/Arch suffix",
			target:       BuildTarget{"linux", "amd64"},
			totalTargets: 2,
			setupConfig: func() {
				AppConfig.Name = "api"
				AppConfig.OutputDir = "dist"
				AppConfig.ExactName = false
			},
			expectedPath: filepath.Clean("dist/api-linux-amd64"),
		},
		{
			name:         "Windows target automatically appends .exe",
			target:       BuildTarget{"windows", "amd64"},
			totalTargets: 1,
			setupConfig: func() {
				AppConfig.Name = "craft"
				AppConfig.OutputDir = "bin"
				AppConfig.ExactName = false
			},
			expectedPath: filepath.Clean("bin/craft.exe"),
		},
		{
			name:         "Multi target Windows appends suffix and .exe",
			target:       BuildTarget{"windows", "amd64"},
			totalTargets: 3,
			setupConfig: func() {
				AppConfig.Name = "craft"
				AppConfig.OutputDir = "bin"
				AppConfig.ExactName = false
			},
			expectedPath: filepath.Clean("bin/craft-windows-amd64.exe"),
		},
		{
			name:         "ExactName true bypasses suffix even on multi-target",
			target:       BuildTarget{"darwin", "arm64"},
			totalTargets: 5,
			setupConfig: func() {
				AppConfig.Name = "server"
				AppConfig.OutputDir = "build"
				AppConfig.ExactName = true
			},
			expectedPath: filepath.Clean("build/server"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetBuilderGlobals()
			tc.setupConfig()

			got := Builder.GetBinaryPath(tc.target, tc.totalTargets)
			if got != tc.expectedPath {
				t.Errorf("expected %q, got %q", tc.expectedPath, got)
			}
		})
	}
}

// TestCompile performs a real end-to-end compilation test using a temporary workspace.
func TestCompile(t *testing.T) {
	// Create a secure, isolated temporary directory for the test.
	tempDir, err := os.MkdirTemp("", "craft-builder-test-*")
	if err != nil {
		t.Fatalf("failed to create temp workspace: %v", err)
	}
	defer os.RemoveAll(tempDir) // Ensure cleanup after test

	modContent := []byte("module testapp\n\ngo 1.21\n")
	modFile := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(modFile, modContent, 0644); err != nil {
		t.Fatalf("failed to write dummy go.mod: %v", err)
	}

	mainGoContent := []byte(`
package main
import "fmt"
func main() {
	fmt.Println("Craft Test Builder")
}
`)
	mainFile := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(mainFile, mainGoContent, 0644); err != nil {
		t.Fatalf("failed to write dummy main.go: %v", err)
	}

	resetBuilderGlobals()
	AppConfig.Name = "testapp"

	AppConfig.EntryPoint = mainFile
	AppConfig.OutputDir = filepath.Join(tempDir, "bin")
	AppConfig.StripDebug = true

	target := BuildTarget{OS: runtime.GOOS, Arch: runtime.GOARCH}
	outPath := Builder.GetBinaryPath(target, 1)

	// 5. Execute the actual compiler
	result := Builder.Compile(target, outPath, true) // minimal = true

	if !strings.Contains(result.Status, "SUCCESS") {
		t.Fatalf("expected compilation to succeed, got status: %s, error: %s", result.Status, result.ErrorMsg)
	}

	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Fatalf("expected binary to be created at %q, but it was not found", outPath)
	}
}
