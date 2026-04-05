package main

import (
	"os"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

// setupMockEnvironment creates an isolated temporary directory with a simulated .craft.yaml.
// It returns a cleanup function to defer in tests.
func setupMockEnvironment(t *testing.T, yamlContent string) func() {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "craft-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	if yamlContent != "" {
		err = os.WriteFile(ConfigFileName, []byte(yamlContent), 0644)
		if err != nil {
			t.Fatalf("failed to write mock config: %v", err)
		}
	}

	return func() {
		os.Chdir(originalDir)
		os.RemoveAll(tempDir)
		resetGlobals()
	}
}

// resetGlobals ensures test isolation by resetting configuration states.
func resetGlobals() {
	AppConfig = Config{}
	activeCmd = nil
	flagName = ""
	flagVersion = ""
	flagEntry = ""
	flagOut = DefaultDistDir
	flagVerPkg = ""
	flagAll = false
	flagPlats = nil
	flagStrip = true
	flagExactName = false
	flagProfile = ""
}

// TestPrepareEngine_Priority ensures that CLI flags override profiles,
// and profiles override the base YAML configuration appropriately.
func TestPrepareEngine_Priority(t *testing.T) {
	baseYAML := `
name: "base-app"
output_dir: "bin"
build_all: false
platforms: ["current"]
profiles:
  npm:
    output_dir: "npm/bin"
    build_all: true
`

	tests := []struct {
		name          string
		cliArgs       []string
		expectedName  string
		expectedOut   string
		expectedAll   bool
		expectedPlats []string
		expectedExact bool
	}{
		{
			name:          "Base configuration loading without overrides",
			cliArgs:       []string{"build"},
			expectedName:  "base-app",
			expectedOut:   "bin",
			expectedAll:   false,
			expectedPlats: []string{"current"},
		},
		{
			name:          "Profile override (npm profile)",
			cliArgs:       []string{"build", "--profile", "npm"},
			expectedName:  "base-app",
			expectedOut:   "npm/bin",
			expectedAll:   true,
			expectedPlats: nil, // build_all drops specific platforms
		},
		{
			name:          "CLI override takes highest priority over profile",
			cliArgs:       []string{"build", "--profile", "npm", "--out", "custom/dir", "--name", "cli-app"},
			expectedName:  "cli-app",
			expectedOut:   "custom/dir",
			expectedAll:   true,
			expectedPlats: nil,
		},
		{
			name:          "CLI explicit platform override disables build_all",
			cliArgs:       []string{"build", "--all", "--platform", "linux/arm64"},
			expectedName:  "base-app",
			expectedOut:   "bin",
			expectedAll:   false, // --platform overrides --all logic in prepareEngine
			expectedPlats: []string{"linux/arm64"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := setupMockEnvironment(t, baseYAML)
			defer cleanup()

			// Sandbox a dummy Cobra command to parse our simulated flags
			cmd := &cobra.Command{Use: "build"}
			setupFlags(cmd)

			// Simulate flag parsing
			if err := cmd.ParseFlags(tc.cliArgs[1:]); err != nil {
				t.Fatalf("failed to parse test flags: %v", err)
			}

			// Assign globals to simulate runtime state
			activeCmd = cmd
			if cmd.Flags().Changed("profile") {
				flagProfile, _ = cmd.Flags().GetString("profile")
			}

			// Execute the core configuration merger
			prepareEngine()

			if AppConfig.Name != tc.expectedName {
				t.Errorf("expected Name %q, got %q", tc.expectedName, AppConfig.Name)
			}
			if AppConfig.OutputDir != tc.expectedOut {
				t.Errorf("expected OutputDir %q, got %q", tc.expectedOut, AppConfig.OutputDir)
			}
			if AppConfig.BuildAll != tc.expectedAll {
				t.Errorf("expected BuildAll %v, got %v", tc.expectedAll, AppConfig.BuildAll)
			}
			if !reflect.DeepEqual(AppConfig.Platforms, tc.expectedPlats) {
				t.Errorf("expected Platforms %v, got %v", tc.expectedPlats, AppConfig.Platforms)
			}
		})
	}
}

// TestSetupFlags validates that the correct default values and flags are mounted to the command.
func TestSetupFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "dummy"}
	setupFlags(cmd)

	if !cmd.HasFlags() {
		t.Fatal("expected command to have mounted flags, found none")
	}

	defaultOut, err := cmd.Flags().GetString("out")
	if err != nil || defaultOut != DefaultDistDir {
		t.Errorf("expected default out flag to be %q, got %q", DefaultDistDir, defaultOut)
	}

	defaultStrip, err := cmd.Flags().GetBool("strip")
	if err != nil || !defaultStrip {
		t.Errorf("expected default strip flag to be true, got %v", defaultStrip)
	}
}
