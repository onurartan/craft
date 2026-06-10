package main

import (
	"regexp"
	"strings"

	"github.com/pterm/pterm"
)

// PrintParsedError takes a raw Go compiler error string, parses it, and prints it elegantly.
func PrintParsedError(platform string, rawError string) {
	if flagVerbose {
		pterm.Printf("%s\n", pterm.FgDarkGray.Sprint("--- RAW GO OUTPUT ---"))
		pterm.Println(rawError)
		pterm.Printf("%s\n\n", pterm.FgDarkGray.Sprint("---------------------"))
	}

	// Print a sleek error header
	pterm.Printf("%s %s\n",
		pterm.NewStyle(pterm.FgRed, pterm.Bold).Sprint("✖ Build Failed"),
		pterm.FgDarkGray.Sprintf("[%s]", platform),
	)

	// Regex to match ".\file.go:line:col: message" or "/path/to/file.go:line: message"
	re := regexp.MustCompile(`(?m)^([a-zA-Z0-9_\-\.\/\\]+\.go):(\d+)(?::\d+)?:?\s*(.*)$`)

	matches := re.FindAllStringSubmatch(rawError, -1)
	if len(matches) == 0 {
		// If we couldn't parse the exact file/line, just print the raw error cleaned up
		cleaned := strings.TrimSpace(strings.ReplaceAll(rawError, "# github.com/onurartan/craft", ""))
		lines := strings.Split(cleaned, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				pterm.Printf("  %s %s\n", pterm.FgRed.Sprint("│"), pterm.FgLightRed.Sprint(strings.TrimSpace(line)))
			}
		}
	} else {
		// Iterate over matched errors
		for _, match := range matches {
			file := match[1]
			line := match[2]
			msg := match[3]

			pterm.Printf("  %s %s\n", pterm.FgRed.Sprint("↳"), pterm.FgLightWhite.Sprintf("%s:%s", file, line))
			pterm.Printf("    %s\n", pterm.FgLightRed.Sprint(msg))

			// Actionable hints
			if strings.Contains(msg, "missing go.sum entry") || strings.Contains(msg, "no required module provides") || strings.Contains(msg, "no required module") {
				pterm.Printf("    %s Run 'craft build' again to auto-install or 'craft tidy'.\n", pterm.FgLightYellow.Sprint("Hint:"))
			}
			if strings.Contains(msg, "undefined:") {
				pterm.Printf("    %s Did you misspell the variable or forget an import?\n", pterm.FgLightYellow.Sprint("Hint:"))
			}
			if strings.Contains(msg, "syntax error:") {
				pterm.Printf("    %s Check for missing commas, brackets, or typos around line %s.\n", pterm.FgLightYellow.Sprint("Hint:"), line)
			}
		}
	}
	pterm.Println()
}
