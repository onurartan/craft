// builder.go implements the core build logic for the Craft engine.
// Copyright (c) 2026 Onurartan/Craft Contributors.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pterm/pterm"
)

// BuildTarget represents a cross-compilation OS/Arch pair.
type BuildTarget struct {
	OS   string
	Arch string
}

// BuildResult encapsulates the outcome of a single compilation cycle.
type BuildResult struct {
	Platform string
	Status   string
	Duration time.Duration
	Artifact string
	Size     string
	ErrorMsg string
}

// BuilderConfig serves as a pseudo-namespace for compilation methods.
type BuilderConfig struct{}

var Builder BuilderConfig

// ResolveTargets calculates the build matrix based on AppConfig.
func (b *BuilderConfig) ResolveTargets() []BuildTarget {
	cfg := AppConfig

	if cfg.BuildAll {
		return []BuildTarget{
			{OS: string(OSLinux), Arch: string(ArchAmd64)}, {OS: string(OSLinux), Arch: string(ArchArm64)},
			{OS: string(OSWindows), Arch: string(ArchAmd64)},
			{OS: string(OSDarwin), Arch: string(ArchAmd64)}, {OS: string(OSDarwin), Arch: string(ArchArm64)},
		}
	}

	if len(cfg.Platforms) > 0 {
		customTargets := make([]BuildTarget, 0, len(cfg.Platforms))
		for _, p := range cfg.Platforms {
			if strings.ToLower(p) == "current" {
				customTargets = append(customTargets, BuildTarget{runtime.GOOS, runtime.GOARCH})
				continue
			}
			parts := strings.Split(p, "/")
			if len(parts) == 2 {
				customTargets = append(customTargets, BuildTarget{parts[0], parts[1]})
			}
		}
		return customTargets
	}

	// Default: Host architecture only
	return []BuildTarget{{runtime.GOOS, runtime.GOARCH}}
}

// GetBinaryPath constructs a sanitized destination path for the binary.
func (b *BuilderConfig) GetBinaryPath(t BuildTarget, totalTargets int) string {
	cfg := AppConfig
	fileName := cfg.Name

	// Append OS/Arch suffix unless it's a single-target exact build
	if !cfg.ExactName && totalTargets > 1 {
		fileName = fmt.Sprintf("%s-%s-%s", cfg.Name, t.OS, t.Arch)
	}

	if t.OS == string(OSWindows) && filepath.Ext(fileName) == "" {
		fileName += ".exe"
	}

	return filepath.Clean(filepath.Join(cfg.OutputDir, fileName))
}

// Compile executes the 'go build' command with injected flags and environment.
func (b *BuilderConfig) Compile(t BuildTarget, outPath string, minimal bool, suppressSpinner bool) BuildResult {
	cfg := AppConfig
	startTime := time.Now()
	label := fmt.Sprintf("%s/%s", t.OS, t.Arch)

	var spin *pterm.SpinnerPrinter
	if !minimal && !suppressSpinner {
		spin, _ = pterm.DefaultSpinner.WithSequence("⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏").
			WithRemoveWhenDone(true).
			Start(pterm.FgDarkGray.Sprintf("Compiling ") + pterm.FgLightCyan.Sprint(label) + "...")
	}

	// Command-line arguments orchestration
	cmdArgs := make([]string, 0, 10)

	if cfg.Race {
		cmdArgs = append(cmdArgs, "-race")
	}
	if cfg.Trimpath {
		cmdArgs = append(cmdArgs, "-trimpath")
	}
	if len(cfg.Tags) > 0 {
		cmdArgs = append(cmdArgs, "-tags", strings.Join(cfg.Tags, ","))
	}

	var ldflags []string
	if cfg.StripDebug {
		ldflags = append(ldflags, "-s", "-w")
	}

	if cfg.VersionPkg != "" {
		date := time.Now().Format(time.RFC3339)
		if minimal {
			date = "dev-build"
		}
		// Secure injection of variables into the binary
		ldflags = append(ldflags, fmt.Sprintf("-X %s=%s", cfg.VersionPkg, cfg.Version))
		ldflags = append(ldflags, fmt.Sprintf("-X %s_Date=%s", cfg.VersionPkg, date))
	}

	if len(ldflags) > 0 {
		cmdArgs = append(cmdArgs, "-ldflags", strings.Join(ldflags, " "))
	}

	cmdArgs = append(cmdArgs, "-o", outPath, cfg.EntryPoint)

	os.MkdirAll(filepath.Dir(outPath), 0755)

	// Triggering Go Toolchain
	cmd, tcErr := BuildGoCommand(GoCmdBuild, cmdArgs...)
	if tcErr != nil {
		if spin != nil {
			spin.Fail("Toolchain validation failed")
		}
		return BuildResult{
			Platform: label,
			Status:   pterm.FgRed.Sprint("FAIL"),
			Duration: time.Since(startTime),
			Artifact: "-",
			Size:     "-",
			ErrorMsg: tcErr.Error(),
		}
	}

	cgoVal := "0"
	if cfg.CgoEnabled {
		cgoVal = "1"
	}

	// Isolate and inject target environment variables
	env := cmd.Env
	if env == nil {
		env = os.Environ()
	}

	cmd.Env = append(env,
		"GOOS="+t.OS,
		"GOARCH="+t.Arch,
		"CGO_ENABLED="+cgoVal,
	)

	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		if spin != nil {
			spin.Fail(fmt.Sprintf("Compilation Failed: %s", label))
		}
		return BuildResult{
			Platform: label,
			Status:   pterm.FgRed.Sprint("FAIL"),
			Duration: duration,
			Artifact: "-",
			Size:     "-",
			ErrorMsg: string(output),
		}
	}

	fi, _ := os.Stat(outPath)
	fileSize := UI.FormatSize(fi.Size())

	if spin != nil {
		spin.Success(fmt.Sprintf("Compiled %s | Size: %s",
			pterm.FgCyan.Sprint(label),
			pterm.FgMagenta.Sprint(fileSize)),
		)
	}

	return BuildResult{
		Platform: label,
		Status:   pterm.FgGreen.Sprint("SUCCESS"),
		Duration: duration,
		Artifact: filepath.Base(outPath),
		Size:     fileSize,
	}
}
