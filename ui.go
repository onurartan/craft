// ui.go: Visual interface and terminal reporting logic.
// Copyright (c) 2026 Onurartan/Craft Contributors.

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/pterm/pterm"
)

type uiAPI struct{}

var UI = &uiAPI{}

// PrintBanner renders the minimalist identity of the tool.
func (u *uiAPI) PrintBanner() {
	pterm.Println()
	pterm.Printf("▲ %s %s\n\n",
		pterm.NewStyle(pterm.FgCyan, pterm.Bold).Sprint("Craft"),
		pterm.FgDarkGray.Sprintf("v%s", CraftAppVersion),
	)
}

// PrintInfo displays the current build configuration cleanly.
func (u *uiAPI) PrintInfo(targetCount int) {
	cfg := AppConfig

	data := [][]string{
		{pterm.FgDarkGray.Sprint("Project"), pterm.FgLightCyan.Sprint(cfg.Name)},
		{pterm.FgDarkGray.Sprint("Version"), pterm.FgLightMagenta.Sprint(cfg.Version)},
		{pterm.FgDarkGray.Sprint("Output"), pterm.FgWhite.Sprint(cfg.OutputDir)},
	}

	if cfg.VersionPkg != "" {
		data = append(data, []string{pterm.FgDarkGray.Sprint("Inject"), pterm.FgLightYellow.Sprint(cfg.VersionPkg)})
	}

	tagStr := pterm.FgDarkGray.Sprint("-")
	if len(cfg.Tags) > 0 {
		tagStr = pterm.FgLightCyan.Sprint(strings.Join(cfg.Tags, ", "))
	}
	data = append(data, []string{pterm.FgDarkGray.Sprint("Tags"), tagStr})

	cgoStatus := pterm.FgDarkGray.Sprint("Off")
	if cfg.CgoEnabled {
		cgoStatus = pterm.FgLightGreen.Sprint("On")
	}
	data = append(data, []string{pterm.FgDarkGray.Sprint("CGO"), cgoStatus})
	data = append(data, []string{pterm.FgDarkGray.Sprint("Targets"), fmt.Sprintf("%d Platform(s)", targetCount)})

	pterm.DefaultTable.WithData(data).WithSeparator("   ").Render()
}

// PrintSummary renders a comprehensive final report of the build cycle.
func (u *uiAPI) PrintSummary(results []BuildResult, totalTime time.Duration) {
	pterm.Println()
	pterm.Printf("%s\n", pterm.NewStyle(pterm.FgCyan, pterm.Bold).Sprint("▶ Build Summary"))
	pterm.Println()

	tableData := [][]string{
		{
			pterm.FgDarkGray.Sprint("PLATFORM"),
			pterm.FgDarkGray.Sprint("STATUS"),
			pterm.FgDarkGray.Sprint("SIZE"),
			pterm.FgDarkGray.Sprint("TIME"),
			pterm.FgDarkGray.Sprint("ARTIFACT"),
		},
	}

	for _, r := range results {
		status := r.Status // usually SUCCESS or FAIL colored
		tableData = append(tableData, []string{
			pterm.FgLightCyan.Sprint(r.Platform),
			status,
			pterm.FgLightMagenta.Sprint(r.Size),
			pterm.FgDarkGray.Sprint(r.Duration.Round(time.Millisecond)),
			pterm.FgWhite.Sprint(r.Artifact),
		})
	}

	pterm.DefaultTable.WithHasHeader().WithData(tableData).WithSeparator("   ").Render()

	// Print errors gracefully if any
	for _, r := range results {
		if r.ErrorMsg != "" {
			pterm.Println()
			// Instead of a box, use our custom error parser
			PrintParsedError(r.Platform, r.ErrorMsg)
		}
	}

	pterm.Println()
	pterm.Printf("%s %s\n\n",
		pterm.FgDarkGray.Sprint("Done in"),
		pterm.FgLightCyan.Sprint(totalTime.Round(time.Millisecond)),
	)
}

// FormatSize converts bytes to a human-readable professional format.
func (u *uiAPI) FormatSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
