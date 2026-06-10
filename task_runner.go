package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var reservedCommands = map[string]bool{
	"build":   true,
	"run":     true,
	"dev":     true,
	"irun":    true,
	"clean":   true,
	"doctor":  true,
	"fmt":     true,
	"vet":     true,
	"test":    true,
	"gen":     true,
	"install": true,
	"init":    true,
	"add":     true,
	"tidy":    true,
}

// buildCobraCommand recursively builds Cobra subcommands from the YAML configuration
func buildCobraCommand(cmdName string, cmdData interface{}, depth int) *cobra.Command {
	if depth > 4 {
		pterm.Warning.Printf("[craft] Command '%s' exceeds maximum nesting depth of 4. Ignoring subcommands.\n", cmdName)
		return nil
	}

	customCmd := &cobra.Command{
		Use:                cmdName,
		Short:              fmt.Sprintf("Custom task: %s", cmdName),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeCustomCommand(cmdName, cmdData, args, 0)
		},
	}

	if subMap, ok := cmdData.(map[string]interface{}); ok {
		// Check if keys are OS names
		isOSMap := false
		for k := range subMap {
			if k == "windows" || k == "linux" || k == "darwin" || k == "macos" || k == "default" {
				isOSMap = true
				break
			}
		}

		if !isOSMap {
			// It's a subcommand group. The parent does nothing.
			customCmd.RunE = nil

			for subKey, subVal := range subMap {
				subCobra := buildCobraCommand(subKey, subVal, depth+1)
				if subCobra != nil {
					customCmd.AddCommand(subCobra)
				}
			}
		}
	}

	return customCmd
}

// injectCustomCommands registers the custom commands defined in .craft.yaml into Cobra
func injectCustomCommands(rootCmd *cobra.Command) {
	if AppConfig.Commands == nil {
		return
	}

	for key, val := range AppConfig.Commands {
		if key == "envs" {
			continue // Reserved for user macros
		}

		if reservedCommands[key] {
			pterm.Warning.Printf("[craft] '%s' is a reserved command and cannot be overridden by .craft.yaml\n", key)
			continue
		}

		cmd := buildCobraCommand(key, val, 1)
		if cmd != nil {
			cmd.GroupID = "custom"
			rootCmd.AddCommand(cmd)
		}
	}
}

// RunScriptHook executes a lifecycle hook defined in the scripts section
func RunScriptHook(name string, data interface{}) error {
	if data == nil {
		return nil
	}
	pterm.Info.Printf("[craft] Running script hook: %s\n", name)
	return executeCustomCommand(name, data, nil, 1)
}

// executeCustomCommand parses and runs a custom command block
func executeCustomCommand(name string, data interface{}, args []string, depth int) error {
	if depth > 4 {
		return fmt.Errorf("maximum command depth exceeded (circular dependency or too deep nesting in %s)", name)
	}

	switch v := data.(type) {
	case string:
		// Single command
		return runSingleTask(v, args, depth)
	case []interface{}:
		// Array of commands
		for _, step := range v {
			if stepStr, ok := step.(string); ok {
				if err := runSingleTask(stepStr, args, depth); err != nil {
					return err
				}
			}
		}
		return nil
	case map[string]interface{}:
		// OS-specific command
		osKey := runtime.GOOS
		if osKey == "darwin" {
			if _, exists := v["macos"]; exists {
				osKey = "macos"
			}
		}

		if osCmd, exists := v[osKey]; exists {
			if osCmdStr, ok := osCmd.(string); ok {
				return runSingleTask(osCmdStr, args, depth)
			}
		}

		// Fallback to "default" if exists
		if defCmd, exists := v["default"]; exists {
			if defCmdStr, ok := defCmd.(string); ok {
				return runSingleTask(defCmdStr, args, depth)
			}
		}

		pterm.Warning.Printf("No command defined for OS: %s in task '%s'\n", runtime.GOOS, name)
		return nil
	default:
		return fmt.Errorf("invalid command format for '%s'", name)
	}
}

func runSingleTask(cmdStr string, args []string, depth int) error {
	cmdStr = strings.TrimSpace(cmdStr)
	if cmdStr == "" {
		return nil
	}

	// 1. Command Composition (Reference)
	if strings.HasPrefix(cmdStr, "$") {
		refPath := strings.TrimPrefix(cmdStr, "$")
		parts := strings.Fields(refPath)
		if len(parts) > 0 {
			rootName := parts[0]
			if rootData, exists := AppConfig.Commands[rootName]; exists {
				var targetData interface{} = rootData
				for i := 1; i < len(parts); i++ {
					if subMap, ok := targetData.(map[string]interface{}); ok {
						if nextData, subExists := subMap[parts[i]]; subExists {
							targetData = nextData
						} else {
							targetData = nil
							break
						}
					} else {
						targetData = nil
						break
					}
				}

				if targetData != nil {
					return executeCustomCommand(parts[len(parts)-1], targetData, args, depth+1)
				}
			}
		}
		return fmt.Errorf("referenced command '$%s' not found", refPath)
	}

	// 2. Resolve Macros
	resolvedCmd, err := resolveMacros(cmdStr, args)
	if err != nil {
		return err
	}

	pterm.FgCyan.Printf("➜ %s\n", resolvedCmd)

	// 3. Execution
	var c *exec.Cmd
	if runtime.GOOS == "windows" {
		c = exec.Command("cmd", "/C", resolvedCmd)
	} else {
		c = exec.Command("sh", "-c", resolvedCmd)
	}

	// Toolchain Isolation: Ensure arbitrary tasks running "go" use the correct toolchain
	toolchainEnv, tcErr := GetToolchainEnv()
	if tcErr != nil {
		// Strict mode: toolchain is configured but missing — abort.
		pterm.Error.Println(tcErr.Error())
		return tcErr
	}

	env := os.Environ()
	if toolchainEnv != nil {
		env = toolchainEnv
	}

	// Inject Envs from .craft.yaml commands.envs
	if envsRaw, exists := AppConfig.Commands["envs"]; exists {
		if envMapRaw, ok := envsRaw.(map[string]interface{}); ok {
			for k, val := range envMapRaw {
				env = append(env, fmt.Sprintf("%s=%s", k, val))
			}
		}
	}

	c.Env = env
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	if err := c.Run(); err != nil {
		return fmt.Errorf("task failed: %v", err)
	}
	return nil
}

func resolveMacros(cmdStr string, args []string) (string, error) {
	// Prepare custom envs
	envs := make(map[string]string)
	if envsRaw, exists := AppConfig.Commands["envs"]; exists {
		if envMap, ok := envsRaw.(map[string]interface{}); ok {
			for k, v := range envMap {
				envs[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	argsStr := strings.Join(args, " ")
	hasArgsMacro := strings.Contains(cmdStr, "{ARGS}")

	// Standard Replacements
	replacements := map[string]string{
		"{OS}":             runtime.GOOS,
		"{ARCH}":           runtime.GOARCH,
		"{SRC_DIR}":        AppConfig.EntryPoint,
		"{OUT_DIR}":        AppConfig.OutputDir,
		"{APP_NAME}":       AppConfig.Name,
		"{WORKSPACE_ROOT}": getWorkspaceRoot(),
		"{TIMESTAMP}":      time.Now().Format("20060102_1504"),
		"{GIT_COMMIT}":     getGitCommit(),
		"{ARGS}":           argsStr,
	}

	for k, v := range envs {
		replacements["{"+k+"}"] = v
	}

	for k, v := range replacements {
		cmdStr = strings.ReplaceAll(cmdStr, k, v)
	}

	// APP_BIN_PATH Macros
	if strings.Contains(cmdStr, "{APP_BIN_PATH") {
		// We might have multiple tags. Let's do simple replacements.

		// 1. {APP_BIN_PATH:NOBUILD:linux/amd64} or {APP_BIN_PATH:linux/amd64}
		// Let's use a regex or simple string finding for {APP_BIN_PATH.*}
		for {
			startIdx := strings.Index(cmdStr, "{APP_BIN_PATH")
			if startIdx == -1 {
				break
			}
			endIdx := strings.Index(cmdStr[startIdx:], "}")
			if endIdx == -1 {
				break
			}
			endIdx += startIdx

			fullMacro := cmdStr[startIdx : endIdx+1]
			inner := strings.TrimSuffix(strings.TrimPrefix(fullMacro, "{APP_BIN_PATH"), "}")

			noBuild := false
			targetPlat := "current"

			if strings.HasPrefix(inner, ":") {
				parts := strings.Split(strings.TrimPrefix(inner, ":"), ":")
				for _, p := range parts {
					if p == "NOBUILD" {
						noBuild = true
					} else {
						targetPlat = p
					}
				}
			}

			// Force a build if required
			if !noBuild {
				pterm.FgYellow.Println("[craft] Auto-building dependency...")

				// Temporarily override target
				oldPlat := AppConfig.Platforms
				AppConfig.Platforms = []string{targetPlat}
				err := executeBuildProcess(true) // Minimal build
				AppConfig.Platforms = oldPlat

				if err != nil {
					return "", fmt.Errorf("auto-build failed: %v", err)
				}
			}

			// Get the expected binary path
			platExt := ""
			if targetPlat != "current" {
				parts := strings.Split(targetPlat, "/")
				if len(parts) == 2 {
					if !AppConfig.ExactName {
						platExt = fmt.Sprintf("-%s-%s", parts[0], parts[1])
					}
					if parts[0] == "windows" {
						platExt += ".exe"
					}
				}
			} else {
				if runtime.GOOS == "windows" {
					platExt = ".exe"
				}
			}

			binPath := filepath.Join(AppConfig.OutputDir, AppConfig.Name+platExt)
			binPath = filepath.ToSlash(binPath)

			cmdStr = strings.Replace(cmdStr, fullMacro, binPath, 1)
		}
	}

	// Auto-append args if {ARGS} wasn't used
	if !hasArgsMacro && len(args) > 0 {
		cmdStr = cmdStr + " " + argsStr
	}

	return cmdStr, nil
}

func getWorkspaceRoot() string {
	dir, _ := os.Getwd()
	return filepath.ToSlash(dir)
}

func getGitCommit() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}
