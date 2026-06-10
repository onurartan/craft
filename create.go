package main

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/pterm/pterm"
)

//go:embed templates/*/*
var templateFiles embed.FS

type TemplateData struct {
	ProjectName string
}

func ExecuteCreateProcess() error {
	pterm.Println()
	pterm.Printf("▲ %s\n\n", pterm.NewStyle(pterm.FgCyan, pterm.Bold).Sprint("Craft Scaffolding Wizard"))

	// 1. Select Project Type
	projectType, _ := pterm.DefaultInteractiveSelect.
		WithOptions([]string{"REST API Server", "CLI Tool (Cobra)"}).
		WithDefaultText(pterm.FgLightGreen.Sprint("?") + pterm.FgDarkGray.Sprint(" What type of project do you want to create?")).
		Show()

	pterm.Println() // Add spacing

	var framework string
	templateDir := ""

	if projectType == "REST API Server" {
		framework, _ = pterm.DefaultInteractiveSelect.
			WithOptions([]string{"Fiber (Fastest)", "Gin (Most Popular)"}).
			WithDefaultText(pterm.FgLightGreen.Sprint("?") + pterm.FgDarkGray.Sprint(" Which web framework do you prefer?")).
			Show()

		pterm.Println() // Add spacing

		if framework == "Fiber (Fastest)" {
			templateDir = "templates/api_fiber"
		} else {
			templateDir = "templates/api_gin"
		}
	} else {
		templateDir = "templates/cli"
	}

	// 2. Select Project Name
	projectName, _ := pterm.DefaultInteractiveTextInput.
		WithDefaultText(pterm.FgLightGreen.Sprint("?") + pterm.FgDarkGray.Sprint(" What is the project/module name? ") + pterm.FgWhite.Sprint("(e.g. my-awesome-api)")).
		Show()

	if projectName == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	pterm.Println() // Add spacing

	// 3. Select Output Directory
	outDir, _ := pterm.DefaultInteractiveTextInput.
		WithDefaultText(pterm.FgLightGreen.Sprint("?") + pterm.FgDarkGray.Sprint(" Where should we create it? ") + pterm.FgWhite.Sprintf("(default: ./%s)", projectName)).
		Show()

	if outDir == "" {
		outDir = projectName
	}

	// Ensure outDir is an absolute path
	absOutDir, err := filepath.Abs(outDir)
	if err != nil {
		return err
	}

	// 4. Create Directory
	if err := os.MkdirAll(absOutDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %v", err)
	}

	// 5. Generate Files
	pterm.Println()
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Scaffolding %s in %s...", pterm.FgCyan.Sprint(projectName), outDir))
	time.Sleep(500 * time.Millisecond) // Artificial delay for premium feel

	entries, err := templateFiles.ReadDir(templateDir)
	if err != nil {
		spinner.Fail("Failed to read templates")
		return err
	}

	data := TemplateData{
		ProjectName: projectName,
	}

	for _, entry := range entries {
		content, err := templateFiles.ReadFile(templateDir + "/" + entry.Name())
		if err != nil {
			continue
		}

		// Parse as template
		t, err := template.New(entry.Name()).Parse(string(content))
		if err != nil {
			return err
		}

		var parsed bytes.Buffer
		if err := t.Execute(&parsed, data); err != nil {
			return err
		}

		outName := strings.TrimSuffix(entry.Name(), ".tpl")
		outFile := filepath.Join(absOutDir, outName)

		if err := os.WriteFile(outFile, parsed.Bytes(), 0644); err != nil {
			return err
		}
	}

	// 6. Initialize Go Module
	spinner.UpdateText("Initializing go.mod...")
	time.Sleep(500 * time.Millisecond) // Artificial delay for premium feel

	modCmd, err := BuildGoCommand("mod", "init", projectName)
	if err != nil {
		spinner.Warning(err.Error())
	} else {
		modCmd.Dir = absOutDir
		if err := modCmd.Run(); err != nil {
			spinner.Warning("Failed to initialize go module automatically. Please run 'go mod init' manually.")
		} else {
			// Run go mod tidy to fetch dependencies
			spinner.UpdateText("Fetching dependencies (go mod tidy)...")
			tidyCmd, tidyErr := BuildGoCommand("mod", "tidy")
			if tidyErr == nil {
				tidyCmd.Dir = absOutDir
				tidyCmd.Run()
			}
		}
	}

	// Generate craft.yaml
	GenerateDefaultConfig(filepath.Join(absOutDir, ConfigFileName), projectName)

	spinner.Success("Project successfully scaffolded!")

	pterm.Println()
	pterm.Info.Println("Next steps:")
	pterm.Println("  cd " + outDir)
	pterm.Println("  craft dev")
	pterm.Println()

	return nil
}
