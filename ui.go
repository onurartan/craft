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
	pterm.Printf("%s %s\n",
		pterm.NewStyle(pterm.FgCyan, pterm.Bold).Sprint("CRAFT"),
		pterm.FgGray.Sprintf("v%s", CraftAppVersion),
	)
}

// PrintInfo displays the current build configuration in a clean table.
func (u *uiAPI) PrintInfo(targetCount int) {
	cfg := AppConfig

	data := [][]string{
		{"Project", pterm.FgCyan.Sprint(cfg.Name)},
		{"Version", pterm.FgMagenta.Sprint(cfg.Version)},
		{"Output", pterm.FgGray.Sprint(cfg.OutputDir)},
	}

	if cfg.VersionPkg != "" {
		data = append(data, []string{"Inject", pterm.FgYellow.Sprint(cfg.VersionPkg)})
	}

	tagStr := pterm.FgGray.Sprint("None")
	if len(cfg.Tags) > 0 {
		tagStr = pterm.FgCyan.Sprint(strings.Join(cfg.Tags, ", "))
	}
	data = append(data, []string{"Tags", tagStr})

	cgoStatus := pterm.FgRed.Sprint("Off")
	if cfg.CgoEnabled {
		cgoStatus = pterm.FgGreen.Sprint("On")
	}
	data = append(data, []string{"CGO", cgoStatus})
	data = append(data, []string{"Targets", fmt.Sprintf("%d Platform(s)", targetCount)})

	pterm.DefaultTable.WithData(data).WithBoxed().Render()
}

// PrintSummary renders a comprehensive final report of the build cycle.
func (u *uiAPI) PrintSummary(results []BuildResult, totalTime time.Duration) {
	pterm.Println()
	pterm.DefaultSection.Println("Build Summary")

	tableData := [][]string{
		{"PLATFORM", "STATUS", "SIZE", "TIME", "ARTIFACT"},
	}

	for _, r := range results {
		tableData = append(tableData, []string{
			pterm.FgCyan.Sprint(r.Platform),
			r.Status,
			pterm.FgMagenta.Sprint(r.Size),
			pterm.FgGray.Sprint(r.Duration.Round(time.Millisecond)),
			pterm.FgLightWhite.Sprint(r.Artifact),
		})
	}

	pterm.DefaultTable.WithHasHeader().WithBoxed().WithData(tableData).Render()

	// Error reporting for failed targets
	for _, r := range results {
		if r.ErrorMsg != "" {
			pterm.Println()
			pterm.DefaultBox.
				WithTitle(pterm.FgRed.Sprintf("Failure [%s]", r.Platform)).
				Println(pterm.FgRed.Sprint(r.ErrorMsg))
		}
	}

	pterm.Info.Printf("Total execution: %s\n\n", pterm.FgCyan.Sprint(totalTime.Round(time.Millisecond)))
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
