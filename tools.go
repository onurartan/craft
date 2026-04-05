// tools.go: Auxiliary operations including cleanup, validation, and diagnostics.
// Copyright (c) 2026 Onurartan/Craft Contributors.

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pterm/pterm"
)

func ExecuteCleanProcess(forceGoCache bool, cleanAllTemps bool) error {
	prepareEngine()
	pterm.DefaultSection.Println("Project Cleanup")

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Purging output directory: %s", pterm.FgCyan.Sprint(AppConfig.OutputDir)))
	if err := os.RemoveAll(AppConfig.OutputDir); err != nil {
		spinner.Fail("Failed to purge output directory.")
	} else {
		spinner.Success("Binary artifacts removed.")
	}

	if forceGoCache {
		spinner, _ = pterm.DefaultSpinner.Start("Purging global Go build cache...")
		if err := exec.Command("go", "clean", "-cache").Run(); err != nil {
			spinner.Warning("Go cache clean failed.")
		} else {
			spinner.Success("Global build cache cleared.")
		}
	}

	spinner, _ = pterm.DefaultSpinner.Start("Sweeping temporary dev artifacts...")
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
	pterm.DefaultSection.Println("System Diagnostics")

	goVersion := pterm.FgRed.Sprint("Not Found")
	if out, err := exec.Command("go", "version").Output(); err == nil {
		parts := strings.Split(string(out), " ")
		if len(parts) >= 3 {
			ver := parts[2]
			if strings.Compare(ver, "go1.21") < 0 {
				goVersion = pterm.FgYellow.Sprintf("%s (Outdated)", ver)
			} else {
				goVersion = pterm.FgGreen.Sprint(ver)
			}
		}
	}
	pterm.Printf("%-20s %s\n", "Go Toolchain:", goVersion)
	pterm.Printf("%-20s %s\n", "Host Context:", pterm.FgCyan.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))

	configState := pterm.FgRed.Sprint("Missing")
	if ConfigExists() {
		ConfigLoad()
		if AppConfig.Name == "" {
			configState = pterm.FgYellow.Sprint("Incomplete")
		} else {
			configState = pterm.FgGreen.Sprintf("Active (%s)", AppConfig.Name)
		}
	}
	pterm.Printf("%-20s %s\n", "Craft Config:", configState)

	if runtime.GOOS == "linux" {
		watchLimit := pterm.FgGreen.Sprint("Standard")
		if data, err := os.ReadFile("/proc/sys/fs/inotify/max_user_watches"); err == nil {
			limit := strings.TrimSpace(string(data))
			if limit == "8192" {
				watchLimit = pterm.FgYellow.Sprintf("%s (Increase recommended for Hot-Reload)", limit)
			} else {
				watchLimit = pterm.FgGreen.Sprint(limit)
			}
		}
		pterm.Printf("%-20s %s\n", "Inotify Limit:", watchLimit)
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

	prefixColor := pterm.FgCyan
	if isError {
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
				coloredPath := pterm.FgCyan.Sprint(pathPart)
				coloredLine := pterm.FgMagenta.Sprint(subParts[0])
				msg := strings.Join(subParts[1:], ":")
				coloredMsg := pterm.FgLightRed.Sprint(strings.TrimSpace(msg))

				pterm.Printf("%s %s:%s ➔ %s\n", prefixColor.Sprint("│"), coloredPath, coloredLine, coloredMsg)
				continue
			}
		}

		if strings.Contains(line, "--- FAIL:") || strings.Contains(line, "FAIL") {
			pterm.Printf("%s %s\n", prefixColor.Sprint("│"), pterm.FgLightRed.Sprint(line))
			continue
		}

		pterm.Printf("%s %s\n", prefixColor.Sprint("│"), pterm.FgGray.Sprint(line))
	}
	pterm.Println()
}

func ExecuteTidyProcess() error {
	spinner, _ := pterm.DefaultSpinner.Start("Synchronizing modules...")
	out, err := exec.Command("go", "mod", "tidy").CombinedOutput()

	if err != nil {
		spinner.Fail("Module synchronization failed")
		formatGoLogs(out, true)
		return err
	}

	spinner.Success("Modules synchronized")
	return nil
}

func ExecuteFmtProcess(args ...string) error {
	if len(args) == 0 {
		args = []string{"./..."}
	}
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Formatting source code (%s)...", strings.Join(args, " ")))

	cmdArgs := append([]string{"fmt"}, args...)
	out, err := exec.Command("go", cmdArgs...).CombinedOutput()

	if err != nil {
		spinner.Fail("Formatting failed due to syntax errors")
		formatGoLogs(out, true)
		return err
	}

	changedFiles := bytes.Split(bytes.TrimSpace(out), []byte("\n"))
	if len(changedFiles) > 0 && len(changedFiles[0]) > 0 {
		spinner.Success("Formatted specific files")
		formatGoLogs(out, false)
	} else {
		spinner.Success("Code is already formatted")
	}

	return nil
}

func ExecuteVetProcess(args ...string) error {
	if len(args) == 0 {
		args = []string{"./..."}
	}
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Running static analysis (%s)...", strings.Join(args, " ")))

	cmdArgs := append([]string{"vet"}, args...)
	out, err := exec.Command("go", cmdArgs...).CombinedOutput()

	if err != nil {
		spinner.Fail("Static analysis found potential issues")
		formatGoLogs(out, true)
		return err
	}

	spinner.Success("Static analysis passed")
	return nil
}

func ExecuteTestProcess(args ...string) error {
	if len(args) == 0 {
		args = []string{"./..."}
	}
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Executing test suites (%s)...", strings.Join(args, " ")))

	cmdArgs := append([]string{"test"}, args...)
	out, err := exec.Command("go", cmdArgs...).CombinedOutput()

	if err != nil {
		spinner.Fail("Test suite execution failed")
		formatGoLogs(out, true)
		return err
	}

	spinner.Success("All tests passed")
	if len(args) > 0 && strings.Contains(strings.Join(args, " "), "-v") {
		formatGoLogs(out, false)
	}

	return nil
}

func ExecuteGenerateProcess(args ...string) error {
	if len(args) == 0 {
		args = []string{"./..."}
	}
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Running code generation (%s)...", strings.Join(args, " ")))

	cmdArgs := append([]string{"generate"}, args...)
	out, err := exec.Command("go", cmdArgs...).CombinedOutput()

	if err != nil {
		spinner.Fail("Code generation failed")
		formatGoLogs(out, true)
		return err
	}

	spinner.Success("Code generation completed")
	return nil
}

func ExecuteInstallProcess() error {
	prepareEngine()
	spinner, _ := pterm.DefaultSpinner.Start("Installing binary to system...")

	out, err := exec.Command("go", "install", AppConfig.EntryPoint).CombinedOutput()

	if err != nil {
		spinner.Fail("Installation failed")
		formatGoLogs(out, true)
		return err
	}

	spinner.Success(fmt.Sprintf("Binary installed to GOBIN as '%s'", pterm.FgCyan.Sprint(AppConfig.Name)))
	return nil
}

func ExecuteCheckProcess() error {
	pterm.DefaultSection.Println("Validation Suite")
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
