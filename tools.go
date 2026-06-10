// tools.go: Auxiliary operations including cleanup, validation, and diagnostics.
// Copyright (c) 2026 Onurartan/Craft Contributors.

package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pterm/pterm"
)

// --- Internal Helpers ---

// BuildGoCommand creates an exec.Cmd for the 'go' tool, automatically
// injecting the isolated toolchain environment if configured.
func BuildGoCommand(cmdName GoCommand, args ...string) (*exec.Cmd, error) {
	fullArgs := append([]string{string(cmdName)}, args...)
	cmd := exec.Command("go", fullArgs...)
	env, err := GetToolchainEnv()
	if err != nil {
		pterm.Error.Println(err.Error())
		return nil, err
	}
	if env != nil {
		cmd.Env = env
	}
	return cmd, nil
}

var isTestEnv bool

func RunGoCommand(spinnerMsg, successMsg, failMsg string, cmdName GoCommand, args ...string) ([]byte, error) {
	var spinner *pterm.SpinnerPrinter
	if !isTestEnv {
		spinner, _ = pterm.DefaultSpinner.Start(spinnerMsg)
		time.Sleep(300 * time.Millisecond)
	}

	cmd, err := BuildGoCommand(cmdName, args...)
	if err != nil {
		if spinner != nil {
			spinner.Fail("Toolchain validation failed")
		}
		return nil, err
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		if spinner != nil {
			spinner.Fail(failMsg)
		}
		formatGoLogs(out, true)
		return out, err
	}

	if spinner != nil {
		spinner.Success(successMsg)
	}
	return out, nil
}

func ResolvePackageAlias(pkg string) string {
	if strings.HasPrefix(pkg, "gh:") {
		return strings.Replace(pkg, "gh:", "github.com/", 1)
	} else if strings.HasPrefix(pkg, "gl:") {
		return strings.Replace(pkg, "gl:", "gitlab.com/", 1)
	} else if strings.HasPrefix(pkg, "bb:") {
		return strings.Replace(pkg, "bb:", "bitbucket.org/", 1)
	} else if strings.HasPrefix(pkg, "x:") {
		return strings.Replace(pkg, "x:", "golang.org/x/", 1)
	} else if strings.HasPrefix(pkg, "in:") {
		return strings.Replace(pkg, "in:", "gopkg.in/", 1)
	}
	return pkg
}

func ExecuteCleanProcess(forceGoCache bool, cleanAllTemps bool) error {
	prepareEngine()
	pterm.Printf("%s\n\n", pterm.NewStyle(pterm.FgCyan, pterm.Bold).Sprint("▶ Project Cleanup"))

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Purging output directory: %s", pterm.FgCyan.Sprint(AppConfig.OutputDir)))
	time.Sleep(300 * time.Millisecond)
	if err := os.RemoveAll(AppConfig.OutputDir); err != nil {
		spinner.Fail("Failed to purge output directory.")
	} else {
		spinner.Success("Binary artifacts removed.")
	}

	if forceGoCache {
		spinner, _ = pterm.DefaultSpinner.Start("Purging global Go build cache...")
		time.Sleep(300 * time.Millisecond)
		cmd, err := BuildGoCommand(GoCmdClean, "-cache")
		if err != nil {
			spinner.Fail("Toolchain error")
			return err
		}
		if err := cmd.Run(); err != nil {
			spinner.Warning("Go cache clean failed.")
		} else {
			spinner.Success("Global build cache cleared.")
		}
	}

	spinner, _ = pterm.DefaultSpinner.Start("Purging downloaded toolchain archives...")
	time.Sleep(300 * time.Millisecond)
	if err := os.RemoveAll(GetToolchainCacheDir()); err != nil {
		spinner.Warning("Could not purge toolchain archives.")
	} else {
		spinner.Success("Toolchain archives cleared.")
	}

	spinner, _ = pterm.DefaultSpinner.Start("Sweeping temporary dev artifacts...")
	time.Sleep(300 * time.Millisecond)
	tmpDir := os.TempDir()
	files, err := os.ReadDir(tmpDir)
	zombieCount := 0

	prefix := "craft-dev-" + AppConfig.Name + "-"
	if cleanAllTemps {
		prefix = "craft-dev-"
	}

	if err == nil {
		for _, f := range files {
			if strings.HasPrefix(f.Name(), prefix) {
				_ = os.Remove(filepath.Join(tmpDir, f.Name()))
				zombieCount++
			}
		}
	}

	targetDesc := "for this project"
	if cleanAllTemps {
		targetDesc = "globally"
	}
	spinner.Success(fmt.Sprintf("Removed %s temporary artifacts %s.", pterm.FgMagenta.Sprint(zombieCount), targetDesc))

	pterm.Println()
	pterm.Success.Println("Cleanup sequence completed.")
	return nil
}

func ExecuteDoctorProcess() error {
	pterm.Printf("%s\n\n", pterm.NewStyle(pterm.FgCyan, pterm.Bold).Sprint("▶ System Diagnostics"))

	goVersion := pterm.FgLightRed.Sprint("Not Found")
	cmd, err := BuildGoCommand(GoCmdVersion)
	if err != nil {
		pterm.Error.Println("Failed to validate toolchain")
		return err
	}

	if out, err := cmd.Output(); err == nil {
		parts := strings.Split(string(out), " ")
		if len(parts) >= 3 {
			ver := parts[2]
			if strings.Compare(ver, "go1.21") < 0 {
				goVersion = pterm.FgLightYellow.Sprintf("%s (Outdated)", ver)
			} else {
				goVersion = pterm.FgLightGreen.Sprint(ver)
			}
		}
	}
	pterm.Printf("%s %s\n", pterm.FgDarkGray.Sprint(fmt.Sprintf("%-20s", "Go Toolchain:")), goVersion)

	pterm.Printf("%s %s\n", pterm.FgDarkGray.Sprint(fmt.Sprintf("%-20s", "Host Context:")), pterm.FgLightCyan.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))

	configState := pterm.FgLightRed.Sprint("Missing")
	if ConfigExists() {
		ConfigLoad()
		if AppConfig.Name == "" {
			configState = pterm.FgLightYellow.Sprint("Incomplete")
		} else {
			configState = pterm.FgLightGreen.Sprintf("Active (%s)", AppConfig.Name)
		}
	}
	pterm.Printf("%s %s\n", pterm.FgDarkGray.Sprint(fmt.Sprintf("%-20s", "Craft Config:")), configState)

	if runtime.GOOS == "linux" {
		watchLimit := pterm.FgLightGreen.Sprint("Standard")
		if data, err := os.ReadFile("/proc/sys/fs/inotify/max_user_watches"); err == nil {
			limit := strings.TrimSpace(string(data))
			if limit == "8192" {
				watchLimit = pterm.FgLightYellow.Sprintf("%s (Increase recommended for Hot-Reload)", limit)
			} else {
				watchLimit = pterm.FgLightGreen.Sprint(limit)
			}
		}
		pterm.Printf("%s %s\n", pterm.FgDarkGray.Sprint(fmt.Sprintf("%-20s", "Inotify Limit:")), watchLimit)
	}

	pterm.Println()
	return nil
}

// GO TOOLCHAIN WRAPPERS

func formatGoLogs(raw []byte, isError bool) {
	if len(raw) == 0 {
		return
	}

	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	pterm.Println()

	// prefixColor := pterm.FgCyan
	prefixColor := pterm.FgCyan
	if isError {
		// prefixColor = pterm.FgRed
		prefixColor = pterm.FgRed
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "exit status") || strings.HasPrefix(line, "#") {
			continue
		}

		if idx := strings.Index(line, ".go:"); idx != -1 {
			pathPart := line[:idx+3]
			restPart := line[idx+4:]

			subParts := strings.SplitN(restPart, ":", 3)
			if len(subParts) >= 2 {
				coloredPath := pterm.FgLightWhite.Sprint(pathPart)
				coloredLine := pterm.FgDarkGray.Sprint(subParts[0])
				msg := strings.Join(subParts[1:], ":")
				coloredMsg := pterm.FgLightRed.Sprint(strings.TrimSpace(msg))

				pterm.Printf("  %s %s:%s ➔ %s\n", prefixColor.Sprint("│"), coloredPath, coloredLine, coloredMsg)
				continue
			}
		}

		if strings.Contains(line, "--- FAIL:") || strings.Contains(line, "FAIL") {
			pterm.Printf("  %s %s\n", prefixColor.Sprint("│"), pterm.FgLightRed.Sprint(line))
			continue
		}

		pterm.Printf("  %s %s\n", prefixColor.Sprint("│"), pterm.FgGray.Sprint(line))
	}
	pterm.Println()
}

func ExecuteTidyProcess() error {
	_, err := RunGoCommand("Synchronizing modules...", "Modules synchronized", "Module synchronization failed", GoCmdMod, "tidy")
	return err
}

func ExecuteFmtProcess(args ...string) error {
	if len(args) == 0 {
		args = []string{"./..."}
	}

	out, err := RunGoCommand(
		fmt.Sprintf("Formatting source code (%s)...", strings.Join(args, " ")),
		"Code formatting complete",
		"Formatting failed due to syntax errors",
		GoCmdFmt,
		args...,
	)

	if err != nil {
		return err
	}

	changedFiles := bytes.Split(bytes.TrimSpace(out), []byte("\n"))
	if len(changedFiles) > 0 && len(changedFiles[0]) > 0 {
		formatGoLogs(out, false)
	}

	return nil
}

func ExecuteVetProcess(args ...string) error {
	if len(args) == 0 {
		args = []string{"./..."}
	}

	_, err := RunGoCommand(
		fmt.Sprintf("Running static analysis (%s)...", strings.Join(args, " ")),
		"Static analysis passed",
		"Static analysis found potential issues",
		GoCmdVet,
		args...,
	)
	return err
}

func ExecuteTestProcess(args ...string) error {
	if len(args) == 0 {
		args = []string{"./..."}
	}

	out, err := RunGoCommand(
		fmt.Sprintf("Executing test suites (%s)...", strings.Join(args, " ")),
		"All tests passed",
		"Test suite execution failed",
		GoCmdTest,
		args...,
	)

	if err != nil {
		return err
	}

	if len(out) > 0 {
		formatGoLogs(out, false)
	}

	return nil
}

func ExecuteGenerateProcess(args ...string) error {
	if len(args) == 0 {
		args = []string{"./..."}
	}
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Running code generation (%s)...", strings.Join(args, " ")))
	time.Sleep(300 * time.Millisecond)

	cmd, err := BuildGoCommand(GoCmdGenerate, args...)
	if err != nil {
		spinner.Fail("Toolchain error")
		return err
	}
	out, err := cmd.CombinedOutput()

	if err != nil {
		spinner.Fail("Code generation failed")
		formatGoLogs(out, true)
		return err
	}

	spinner.Success("Code generation completed")
	return nil
}

func ExecuteInstallProcess(args []string) error {
	prepareEngine()

	if len(args) == 0 {
		_, err := RunGoCommand(pterm.FgGray.Sprint("Installing local binary to system..."), fmt.Sprintf("Binary installed to GOBIN as '%s'", pterm.FgWhite.Sprint(AppConfig.Name)), "Installation failed", GoCmdInstall, AppConfig.EntryPoint)
		return err
	}

	var resolvedPackages []string
	for _, pkg := range args {
		pkg = ResolvePackageAlias(pkg)
		if !strings.Contains(pkg, "@") && !strings.HasPrefix(pkg, ".") {
			pkg = pkg + "@latest"
		}
		resolvedPackages = append(resolvedPackages, pkg)
	}

	_, err := RunGoCommand(pterm.FgGray.Sprintf("Installing %s globally...", pterm.FgWhite.Sprint(strings.Join(resolvedPackages, ", "))), "Packages installed globally", "Failed to install packages", GoCmdInstall, resolvedPackages...)
	return err
}

func ExecuteAddProcess(packages []string) error {
	if len(packages) == 0 {
		return fmt.Errorf("no packages provided")
	}

	var resolvedPackages []string
	for _, pkg := range packages {
		resolvedPackages = append(resolvedPackages, ResolvePackageAlias(pkg))
	}

	_, err := RunGoCommand(pterm.FgGray.Sprintf("Adding packages (%s)...", pterm.FgWhite.Sprint(strings.Join(resolvedPackages, " "))), "Packages downloaded successfully", "Failed to add packages", GoCmdGet, resolvedPackages...)
	if err != nil {
		return err
	}

	return ExecuteTidyProcess()
}

func ExecuteCheckProcess() error {
	pterm.Printf("%s\n\n", pterm.NewStyle(pterm.FgCyan, pterm.Bold).Sprint("▶ Validation Suite"))
	hasError := false

	if err := ExecuteFmtProcess(); err != nil {
		hasError = true
	}
	if err := ExecuteVetProcess(); err != nil {
		hasError = true
	}
	if err := ExecuteTestProcess(); err != nil {
		hasError = true
	}

	pterm.Println()
	if hasError {
		pterm.Error.Println("Validation failed. Resolve the issues above before proceeding.")
		os.Exit(1)
	}

	pterm.Success.Println("Project is healthy and ready for production.")
	return nil
}

func ExecuteRemoveProcess(packages []string) error {
	if len(packages) == 0 {
		return fmt.Errorf("no packages provided")
	}

	var resolvedPackages []string
	for _, pkg := range packages {
		pkg = ResolvePackageAlias(pkg)
		if !strings.Contains(pkg, "@") {
			pkg = pkg + "@none"
		} else {
			parts := strings.Split(pkg, "@")
			pkg = parts[0] + "@none"
		}
		resolvedPackages = append(resolvedPackages, pkg)
	}

	displayNames := make([]string, len(packages))
	for i, p := range packages {
		displayNames[i] = strings.Split(p, "@")[0]
	}

	_, err := RunGoCommand(fmt.Sprintf("Removing packages (%s)...", strings.Join(displayNames, " ")), "Packages removed successfully", "Failed to remove packages", GoCmdGet, resolvedPackages...)
	if err != nil {
		return err
	}

	return ExecuteTidyProcess()
}

func getGoModDeps() map[string]string {
	cmd, err := BuildGoCommand(GoCmdList, "-m", "all")
	if err != nil {
		return nil
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	deps := make(map[string]string)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			deps[parts[0]] = parts[1]
		}
	}
	return deps
}

func ExecuteSyncProcess() error {
	spinner, _ := pterm.DefaultSpinner.Start("Synchronizing and downloading dependencies...")

	// Track dependencies before sync
	depsBefore := getGoModDeps()

	// Step 1: Tidy
	spinner.UpdateText("Resolving go.mod dependencies (Tidy)...")
	cmd, err := BuildGoCommand(GoCmdMod, "tidy")
	if err != nil {
		spinner.Fail("Toolchain error")
		return err
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		spinner.Fail("Failed to synchronize go.mod")
		formatGoLogs(out, true)
		return err
	}

	// Step 2: Download
	spinner.UpdateText("Downloading missing modules...")
	cmdDl, err := BuildGoCommand(GoCmdMod, "download")
	if err != nil {
		spinner.Fail("Toolchain error")
		return err
	}
	out, err = cmdDl.CombinedOutput()
	if err != nil {
		spinner.Fail("Failed to download dependencies")
		formatGoLogs(out, true)
		return err
	}

	// Step 3: Verify
	spinner.UpdateText("Verifying package checksums...")
	cmdVf, err := BuildGoCommand(GoCmdMod, "verify")
	if err != nil {
		spinner.Fail("Toolchain error")
		return err
	}
	out, err = cmdVf.CombinedOutput()
	if err != nil {
		spinner.Fail("Dependency verification failed")
		formatGoLogs(out, true)
		return err
	}

	// Track dependencies after sync
	depsAfter := getGoModDeps()

	spinner.Success("Project is fully synchronized")

	// Calculate and display differences
	if depsBefore != nil && depsAfter != nil {
		var added, removed, updated []string

		for mod, vAfter := range depsAfter {
			// Skip the main project module itself (usually has no version or is tied to path)
			if mod == AppConfig.Name || vAfter == "" {
				continue
			}
			vBefore, exists := depsBefore[mod]
			if !exists {
				added = append(added, fmt.Sprintf("%s %s", mod, vAfter))
			} else if vBefore != vAfter {
				updated = append(updated, fmt.Sprintf("%s (%s -> %s)", mod, vBefore, vAfter))
			}
		}

		for mod, vBefore := range depsBefore {
			if mod == AppConfig.Name || vBefore == "" {
				continue
			}
			if _, exists := depsAfter[mod]; !exists {
				removed = append(removed, fmt.Sprintf("%s %s", mod, vBefore))
			}
		}

		// Print summary in uv/npm style
		if len(added) > 0 {
			pterm.Println()
			pterm.FgGreen.Printf("+ Added %d packages:\n", len(added))
			for _, pkg := range added {
				pterm.FgDarkGray.Printf("  %s\n", pkg)
			}
		}
		if len(removed) > 0 {
			if len(added) == 0 {
				pterm.Println()
			}
			pterm.FgRed.Printf("- Removed %d packages:\n", len(removed))
			for _, pkg := range removed {
				pterm.FgDarkGray.Printf("  %s\n", pkg)
			}
		}
		if len(updated) > 0 {
			if len(added) == 0 && len(removed) == 0 {
				pterm.Println()
			}
			pterm.FgCyan.Printf("~ Updated %d packages:\n", len(updated))
			for _, pkg := range updated {
				pterm.FgDarkGray.Printf("  %s\n", pkg)
			}
		}
		if len(added) == 0 && len(removed) == 0 && len(updated) == 0 {
			pterm.FgDarkGray.Println("  No dependency changes detected.")
		}
	}

	return nil
}

func ExecuteInlineScript(args []string) error {
	scriptFile := args[0]
	absScript, err := filepath.Abs(scriptFile)
	if err != nil {
		return err
	}

	pterm.Printf("%s\n\n", pterm.NewStyle(pterm.FgCyan, pterm.Bold).Sprint("▶ Running Inline Script"))

	if _, err := os.Stat(absScript); os.IsNotExist(err) {
		pterm.Error.Printf("Script file not found: %s\n", scriptFile)
		return err
	}

	input, err := os.ReadFile(absScript)
	if err != nil {
		pterm.Error.Println("Failed to read script")
		return err
	}

	// 1. Generate CacheID based on Normalized Absolute Path
	hasher := sha256.New()
	normalizedPath := strings.ToLower(filepath.Clean(absScript))
	hasher.Write([]byte(normalizedPath))
	cacheID := hex.EncodeToString(hasher.Sum(nil))

	// 2. Generate Content Hash based on file contents
	contentHasher := sha256.New()
	contentHasher.Write(input)
	contentHash := hex.EncodeToString(contentHasher.Sum(nil))

	homeDir, _ := os.UserHomeDir()
	craftCacheDir := filepath.Join(homeDir, string(CraftHomeDir), string(CraftScriptCache))
	scriptDir := filepath.Join(craftCacheDir, cacheID)
	hashFile := filepath.Join(scriptDir, string(CraftContentHash))
	targetScript := filepath.Join(scriptDir, filepath.Base(scriptFile))

	os.MkdirAll(scriptDir, 0755)

	// Touch directory to update modification time for GC
	os.Chtimes(scriptDir, time.Now(), time.Now())

	// Run passive GC asynchronously, skipping the active cacheID to prevent race conditions
	go cleanOldScriptCaches(craftCacheDir, cacheID)

	contentChanged := true
	if existingHash, err := os.ReadFile(hashFile); err == nil {
		if string(existingHash) == contentHash {
			contentChanged = false
		}
	}

	if contentChanged {
		pterm.Info.Println("Initializing secure sandbox environment...")

		if err := os.WriteFile(targetScript, input, 0644); err != nil {
			pterm.Error.Printf("Failed to prepare script: %v\n", err)
			return err
		}

		if _, err := os.Stat(filepath.Join(scriptDir, "go.mod")); os.IsNotExist(err) {
			modCmd, _ := BuildGoCommand(GoCmdMod, "init", "craft-inline")
			modCmd.Dir = scriptDir
			modCmd.Run()
		}

		var spinner *pterm.SpinnerPrinter
		if !isTestEnv {
			spinner, _ = pterm.DefaultSpinner.Start("Analyzing script dependencies...")
		}

		tidyCmd, _ := BuildGoCommand(GoCmdMod, "tidy", "-v")
		tidyCmd.Dir = scriptDir
		stderr, err := tidyCmd.StderrPipe()
		if err != nil {
			if spinner != nil {
				spinner.Fail("Failed to initialize dependency resolution")
			}
			return err
		}

		if err := tidyCmd.Start(); err != nil {
			if spinner != nil {
				spinner.Fail("Failed to start dependency resolution")
			}
			return err
		}

		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if spinner != nil {
				if strings.HasPrefix(line, "go: downloading ") {
					pkg := strings.TrimPrefix(line, "go: downloading ")
					spinner.UpdateText(fmt.Sprintf("Downloading %s", pkg))
				} else if strings.HasPrefix(line, "go: finding module for package ") {
					pkg := strings.TrimPrefix(line, "go: finding module for package ")
					spinner.UpdateText(fmt.Sprintf("Resolving %s", pkg))
				}
			}
			if strings.HasPrefix(line, "go: added ") {
				pkg := strings.TrimPrefix(line, "go: added ")
				if !isTestEnv {
					pterm.Success.Printf("  Added %s\n", pkg)
				}
			}
		}

		if err := tidyCmd.Wait(); err != nil {
			if spinner != nil {
				spinner.Fail("Dependency resolution failed! Please check your imports.")
			}
			return err
		}

		os.WriteFile(hashFile, []byte(contentHash), 0644)
		if spinner != nil {
			spinner.Success("Environment ready and dependencies cached")
		}
		pterm.Println()
	} else {
		pterm.Success.Printf("Using cached execution environment...\n\n")
	}

	runCmd, err := BuildGoCommand(GoCmdRun, append([]string{filepath.Base(scriptFile)}, args[1:]...)...)
	if err != nil {
		return err
	}
	runCmd.Dir = scriptDir
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr
	runCmd.Stdin = os.Stdin

	return runCmd.Run()
}

func cleanOldScriptCaches(cacheDir string, activeCacheID string) {
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return
	}
	now := time.Now()
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == activeCacheID {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		// If older than 24 hours
		if now.Sub(info.ModTime()) > 24*time.Hour {
			os.RemoveAll(filepath.Join(cacheDir, entry.Name()))
		}
	}
}
