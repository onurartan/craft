package main

import (
	"os"
	"reflect"
	"testing"
)

// setupConfigWorkspace sets up a temporary directory for configuration tests.
func setupConfigWorkspace(t *testing.T) func() {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "craft-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	return func() {
		os.Chdir(origDir)
		os.RemoveAll(tempDir)
		AppConfig = Config{} // Reset global configuration
	}
}

// TestConfigLoad verifies default assignments and YAML unmarshaling logic.
func TestConfigLoad(t *testing.T) {
	t.Run("Defaults loaded when no config file exists", func(t *testing.T) {
		cleanup := setupConfigWorkspace(t)
		defer cleanup()

		if ConfigExists() {
			t.Fatal("expected ConfigExists to be false in an empty directory")
		}

		ConfigLoad()

		if AppConfig.OutputDir != DefaultDistDir {
			t.Errorf("expected default OutputDir %q, got %q", DefaultDistDir, AppConfig.OutputDir)
		}
		if AppConfig.Dev.Watch.DelayMs != 500 {
			t.Errorf("expected default DelayMs 500, got %d", AppConfig.Dev.Watch.DelayMs)
		}
		if !AppConfig.StripDebug {
			t.Errorf("expected default StripDebug to be true")
		}
	})

	t.Run("YAML overrides defaults seamlessly", func(t *testing.T) {
		cleanup := setupConfigWorkspace(t)
		defer cleanup()

		yamlContent := `
name: "custom-api"
output_dir: "build/releases"
strip_debug: false
dev:
  watch:
    delay_ms: 150
`
		if err := os.WriteFile(ConfigFileName, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("failed to write mock yaml: %v", err)
		}

		if !ConfigExists() {
			t.Fatal("expected ConfigExists to be true after writing yaml file")
		}

		ConfigLoad()

		if AppConfig.Name != "custom-api" {
			t.Errorf("expected Name 'custom-api', got %q", AppConfig.Name)
		}
		if AppConfig.OutputDir != "build/releases" {
			t.Errorf("expected OutputDir 'build/releases', got %q", AppConfig.OutputDir)
		}
		if AppConfig.StripDebug {
			t.Errorf("expected StripDebug to be overridden to false")
		}
		if AppConfig.Dev.Watch.DelayMs != 150 {
			t.Errorf("expected DelayMs to be overridden to 150, got %d", AppConfig.Dev.Watch.DelayMs)
		}
	})
}

// TestExtractFromStruct validates the dynamic file parsing engine for JSON and YAML.
func TestExtractFromStruct(t *testing.T) {
	tests := []struct {
		name          string
		rawData       []byte
		keyPath       string
		fileExt       string
		expectedValue string
		expectError   bool
	}{
		{
			name:          "Extract flat JSON key",
			rawData:       []byte(`{"version": "2.1.0", "name": "app"}`),
			keyPath:       "version",
			fileExt:       ".json",
			expectedValue: "2.1.0",
			expectError:   false,
		},
		{
			name:          "Extract nested JSON key",
			rawData:       []byte(`{"metadata": {"release": {"version": "3.0.0"}}}`),
			keyPath:       "metadata.release.version",
			fileExt:       ".json",
			expectedValue: "3.0.0",
			expectError:   false,
		},
		{
			name:          "Extract nested YAML key",
			rawData:       []byte("app:\n  info:\n    build: 'v1.5.0-beta'"),
			keyPath:       "app.info.build",
			fileExt:       ".yaml",
			expectedValue: "v1.5.0-beta",
			expectError:   false,
		},
		{
			name:          "Fail on missing key",
			rawData:       []byte(`{"version": "1.0"}`),
			keyPath:       "metadata.version",
			fileExt:       ".json",
			expectedValue: "",
			expectError:   true,
		},
		{
			name:          "Fail on invalid JSON format",
			rawData:       []byte(`{version: "1.0"`), // missing quotes and bracket
			keyPath:       "version",
			fileExt:       ".json",
			expectedValue: "",
			expectError:   true,
		},
		{
			name:          "Fail on unsupported extension",
			rawData:       []byte(`version=1.0`),
			keyPath:       "version",
			fileExt:       ".toml", // Unsupported format in our engine
			expectedValue: "",
			expectError:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val, err := extractFromStruct(tc.rawData, tc.keyPath, tc.fileExt)

			if tc.expectError {
				if err == nil {
					t.Fatalf("expected an error but got none (value: %s)", val)
				}
			} else {
				if err != nil {
					t.Fatalf("did not expect error, got: %v", err)
				}
				if !reflect.DeepEqual(val, tc.expectedValue) {
					t.Errorf("expected %q, got %q", tc.expectedValue, val)
				}
			}
		})
	}
}