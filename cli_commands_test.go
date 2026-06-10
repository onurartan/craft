package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRegisterCLICommands(t *testing.T) {
	// Create a dummy root command
	mockRoot := &cobra.Command{
		Use: "mockroot",
	}

	// Call the registration function
	registerCLICommands(mockRoot)

	// Verify that groups are registered
	groups := mockRoot.Groups()
	expectedGroups := []string{"core", "go", "diag", "custom"}
	for _, expectedGrp := range expectedGroups {
		found := false
		for _, grp := range groups {
			if grp.ID == expectedGrp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected group %q to be registered, but it was not found", expectedGrp)
		}
	}

	// Verify that commands are correctly registered and assigned to groups
	expectedCommands := map[string]string{
		"init":      "core",
		"create":    "core",
		"add":       "core",
		"remove":    "core",
		"sync":      "core",
		"build":     "core",
		"run":       "core",
		"irun":      "core",
		"dev":       "core",
		"install":   "core",
		"tidy":      "go",
		"fmt":       "go",
		"vet":       "go",
		"test":      "go",
		"gen":       "go",
		"clean":     "diag",
		"check":     "diag",
		"doctor":    "diag",
		"toolchain": "core",
	}

	registeredCmds := mockRoot.Commands()
	for cmdName, expectedGroup := range expectedCommands {
		var foundCmd *cobra.Command
		for _, cmd := range registeredCmds {
			if cmd.Name() == cmdName {
				foundCmd = cmd
				break
			}
		}

		if foundCmd == nil {
			t.Errorf("Command %q was not registered", cmdName)
			continue
		}

		if foundCmd.GroupID != expectedGroup {
			t.Errorf("Command %q expected GroupID %q, got %q", cmdName, expectedGroup, foundCmd.GroupID)
		}
	}
}
