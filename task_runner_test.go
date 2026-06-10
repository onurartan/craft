package main

import (
	"runtime"
	"strings"
	"testing"
)

func TestResolveMacros(t *testing.T) {
	// Setup mock config for test
	originalConfig := AppConfig
	defer func() { AppConfig = originalConfig }()

	AppConfig = Config{
		Name:       "TestApp",
		OutputDir:  "test_bin",
		EntryPoint: "./cmd",
		Commands: map[string]interface{}{
			"envs": map[string]interface{}{
				"CUSTOM_VAR": "custom_value",
			},
		},
	}

	tests := []struct {
		name     string
		cmdStr   string
		args     []string
		expected string
	}{
		{
			name:     "App Name Macro",
			cmdStr:   "echo {APP_NAME}",
			args:     nil,
			expected: "echo TestApp",
		},
		{
			name:     "Out Dir and Src Dir Macro",
			cmdStr:   "cp -r {SRC_DIR} {OUT_DIR}",
			args:     nil,
			expected: "cp -r ./cmd test_bin",
		},
		{
			name:     "Args Macro Explicit",
			cmdStr:   "run {ARGS}",
			args:     []string{"arg1", "arg2"},
			expected: "run arg1 arg2",
		},
		{
			name:     "Args Implicit Append",
			cmdStr:   "run",
			args:     []string{"arg1", "arg2"},
			expected: "run arg1 arg2", // resolveMacros appends args if {ARGS} is not used
		},
		{
			name:     "Env Macro",
			cmdStr:   "echo {CUSTOM_VAR}",
			args:     nil,
			expected: "echo custom_value",
		},
		{
			name:     "OS and Arch",
			cmdStr:   "build-{OS}-{ARCH}",
			args:     nil,
			expected: "build-" + runtime.GOOS + "-" + runtime.GOARCH,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := resolveMacros(tc.cmdStr, tc.args)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// resolveMacros may append dynamic things like APP_BIN_PATH which we might not want to mock entirely,
			// but for these simple replacements, it should match expected perfectly,
			// except for NOBUILD which requires a build, but we aren't testing NOBUILD here.

			if strings.TrimSpace(result) != tc.expected {
				t.Errorf("resolveMacros(%q) = %q; expected %q", tc.cmdStr, result, tc.expected)
			}
		})
	}
}
