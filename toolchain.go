package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"atomicgo.dev/cursor"
	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
	"github.com/pterm/pterm"
)

var (
	goDevDLURL  = "https://go.dev/dl/"
	goDevAPIURL = "https://go.dev/dl/?mode=json&include=all"
)

// GetCraftHome returns the ~/.craft directory
func GetCraftHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return filepath.Join(home, string(CraftHomeDir))
}

// DownloadTracker wraps io.Writer to update a pterm.Progressbar
type DownloadTracker struct {
	io.Writer
	Total    int64
	Current  int64
	Progress *pterm.ProgressbarPrinter
}

func (dt *DownloadTracker) Write(p []byte) (int, error) {
	n, err := dt.Writer.Write(p)
	dt.Current += int64(n)
	if dt.Progress != nil {
		dt.Progress.Add(n)
	}
	return n, err
}

// GetToolchainDir returns ~/.craft/toolchains
func GetToolchainDir() string {
	return filepath.Join(GetCraftHome(), string(CraftToolchainsDir))
}

// GetToolchainCacheDir returns ~/.craft/cache/archives
func GetToolchainCacheDir() string {
	return filepath.Join(GetCraftHome(), string(CraftCacheDir), "archives")
}

// Download and install a specific Go version
func InstallToolchain(version string, noCache bool) error {
	version = strings.TrimPrefix(version, "go") // e.g. "1.22.1"
	goVer := "go" + version

	osName := runtime.GOOS
	archName := runtime.GOARCH

	ext := ".tar.gz"
	if osName == string(OSWindows) {
		ext = ".zip"
	}

	fileName := fmt.Sprintf("%s.%s-%s%s", goVer, osName, archName, ext)
	downloadURL := fmt.Sprintf("%s%s", goDevDLURL, fileName)

	targetDir := filepath.Join(GetToolchainDir(), goVer)
	if _, err := os.Stat(targetDir); err == nil {
		pterm.Success.Printf("Toolchain %s is already installed. Use 'craft toolchain remove' to reset.\n", goVer)
		return nil
	}

	pterm.Printf("%s\n\n", pterm.NewStyle(pterm.FgCyan, pterm.Bold).Sprintf("▶ Installing Toolchain: %s", goVer))

	if err := os.MkdirAll(GetToolchainDir(), 0755); err != nil {
		return fmt.Errorf("failed to create toolchain directory: %v", err)
	}

	var archivePath string
	var usingCache bool

	if !noCache {
		if err := os.MkdirAll(GetToolchainCacheDir(), 0755); err != nil {
			return fmt.Errorf("failed to create cache directory: %v", err)
		}
		archivePath = filepath.Join(GetToolchainCacheDir(), fileName)
		if info, err := os.Stat(archivePath); err == nil && info.Size() > 10*1024*1024 {
			usingCache = true
		}
	} else {
		// No cache mode, use temp dir
		archivePath = filepath.Join(os.TempDir(), "craft-dl-"+fileName)
	}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Acquiring %s...", fileName))

	if !usingCache {
		spinner.Stop() // Stop spinner to show progress bar properly
		spinner.UpdateText(fmt.Sprintf("Downloading %s...", fileName))
		resp, err := http.Get(downloadURL)
		if err != nil {
			pterm.Error.Println("Failed to download archive")
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			pterm.Error.Printf("Version %s not found for %s/%s\n", goVer, osName, archName)
			return fmt.Errorf("download failed: %d", resp.StatusCode)
		}

		tmpArchivePath := archivePath + ".tmp"
		outFile, err := os.Create(tmpArchivePath)
		if err != nil {
			pterm.Error.Println("Failed to create temporary archive file")
			return err
		}

		if resp.ContentLength > 0 {
			pb, _ := pterm.DefaultProgressbar.
				WithTotal(int(resp.ContentLength)).
				WithTitle(fmt.Sprintf("Downloading %s", fileName)).
				Start()

			tracker := &DownloadTracker{
				Writer:   outFile,
				Total:    resp.ContentLength,
				Progress: pb,
			}

			_, err = io.Copy(tracker, resp.Body)
			pb.Stop()
		} else {
			// Chunked response, no Content-Length
			dlSpinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Downloading %s (Unknown Size)...", fileName))
			_, err = io.Copy(outFile, resp.Body)
			dlSpinner.Success(fmt.Sprintf("Downloaded %s", fileName))
		}

		outFile.Close()
		if err != nil {
			pterm.Error.Println("Failed to download archive")
			os.Remove(tmpArchivePath)
			return err
		}
		
		// Atomic rename to ensure cache integrity
		if err := os.Rename(tmpArchivePath, archivePath); err != nil {
			pterm.Error.Println("Failed to finalize archive download")
			return err
		}
	} else {
		pterm.Success.Printf("Using cached archive for %s...\n", fileName)
	}

	spinner.UpdateText("Extracting toolchain...")

	// Extract
	var extractErr error
	if ext == ".zip" {
		extractErr = unzip(archivePath, GetToolchainDir())
	} else {
		extractErr = untar(archivePath, GetToolchainDir())
	}

	if extractErr != nil {
		spinner.Fail("Extraction failed. Archive might be corrupted.")
		// Corrupted cache defense
		if usingCache {
			pterm.Info.Println("Purging corrupted cached archive. Run install again.")
		}
		os.Remove(archivePath) // delete the bad file
		return extractErr
	}

	// Clean up the downloaded archive to save disk space
	os.Remove(archivePath)

	extractedGoDir := filepath.Join(GetToolchainDir(), "go")
	if err := os.Rename(extractedGoDir, targetDir); err != nil {
		spinner.Fail("Failed to finalize toolchain directory")
		return err
	}

	spinner.Success(fmt.Sprintf("Toolchain %s installed successfully to %s", goVer, targetDir))
	return nil
}

func RemoveToolchain(version string) error {
	version = strings.TrimPrefix(version, "go")
	goVer := "go" + version

	targetDir := filepath.Join(GetToolchainDir(), goVer)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		pterm.Error.Printf("Toolchain %s is not installed.\n", goVer)
		return err
	}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Removing toolchain %s...", goVer))
	err := os.RemoveAll(targetDir)
	if err != nil {
		spinner.Fail("Failed to remove toolchain directory")
		return err
	}

	// Unset if active
	if AppConfig.Toolchain == goVer {
		UpdateToolchainInConfig("")
		pterm.Info.Println("Active toolchain has been reset to system default.")
	}

	spinner.Success(fmt.Sprintf("Toolchain %s removed successfully.", goVer))
	return nil
}

func CleanToolchainCache(version string) error {
	cacheDir := GetToolchainCacheDir()

	if version == "" {
		spinner, _ := pterm.DefaultSpinner.Start("Clearing ALL toolchain archives...")
		if err := os.RemoveAll(cacheDir); err != nil {
			spinner.Fail("Failed to clean toolchain cache")
			return err
		}
		spinner.Success("All toolchain archives cleared.")
		return nil
	}

	// Clean specific version
	version = strings.TrimPrefix(version, "go")
	goVer := "go" + version
	osName := runtime.GOOS
	archName := runtime.GOARCH

	ext := ".tar.gz"
	if osName == string(OSWindows) {
		ext = ".zip"
	}
	fileName := fmt.Sprintf("%s.%s-%s%s", goVer, osName, archName, ext)
	archivePath := filepath.Join(cacheDir, fileName)

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Clearing archive cache for %s...", goVer))
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		spinner.Warning(fmt.Sprintf("No archive cache found for %s", goVer))
		return nil
	}

	if err := os.Remove(archivePath); err != nil {
		spinner.Fail("Failed to delete specific cache")
		return err
	}

	spinner.Success(fmt.Sprintf("Cache for %s cleared.", goVer))
	return nil
}

// ShowToolchainDashboard displays a minimal, sleek dashboard summarizing the active Go environment and commands.
func ShowToolchainDashboard() error {
	pterm.Println()
	pterm.Printf("%s\n\n", pterm.NewStyle(pterm.FgCyan, pterm.Bold).Sprint("▶ Craft Toolchain Manager"))

	// Detect Project Rule
	projectRule := pterm.FgYellow.Sprint("None (Using System Default)")
	if AppConfig.Toolchain != "" {
		projectRule = pterm.FgGreen.Sprint(AppConfig.Toolchain) + pterm.FgDarkGray.Sprint(" (Pinned in craft.yaml)")
	}

	// Detect Resolved Engine
	resolvedEngine := pterm.FgRed.Sprint("Not Found")
	cmd, err := BuildGoCommand(GoCmdVersion)
	if err == nil {
		if out, err := cmd.Output(); err == nil {
			resolvedEngine = pterm.FgCyan.Sprint(strings.TrimSpace(string(out)))
		}
	}

	// Environment Section
	pterm.Printf("  %s\n", pterm.NewStyle(pterm.FgWhite, pterm.Bold).Sprint("Environment"))
	pterm.Printf("  %s %s\n", pterm.FgDarkGray.Sprint("├─ Project Rule :"), projectRule)
	pterm.Printf("  %s %s\n\n", pterm.FgDarkGray.Sprint("└─ Active Engine:"), resolvedEngine)

	// Quick Commands Section
	pterm.Printf("  %s\n", pterm.NewStyle(pterm.FgWhite, pterm.Bold).Sprint("Quick Commands"))

	cmdFmt := func(isLast bool, cmd, desc string) {
		prefix := pterm.FgDarkGray.Sprint("├─")
		if isLast {
			prefix = pterm.FgDarkGray.Sprint("└─")
		}
		pterm.Printf("  %s %s %s\n", prefix, pterm.FgGreen.Sprint(fmt.Sprintf("%-33s", cmd)), pterm.FgDarkGray.Sprint(desc))
	}

	cmdFmt(false, "craft toolchain install <version>", "Download a specific Go version")
	cmdFmt(false, "craft toolchain install --remote", "Interactive menu to install a remote version")
	cmdFmt(false, "craft toolchain use <version>", "Pin this project to a Go version")
	cmdFmt(false, "craft toolchain list", "Show local Go versions")
	cmdFmt(false, "craft toolchain list --remote", "List available remote versions")
	cmdFmt(false, "craft toolchain list --select-mode", "Interactive menu to switch local versions")
	cmdFmt(false, "craft toolchain clean", "Free up disk space from ALL caches")
	cmdFmt(true, "craft toolchain clean <version>", "Clean up cache for a specific version")
	pterm.Println()

	return nil
}

func ListToolchains() error {
	dir := GetToolchainDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			pterm.Info.Println("No toolchains installed yet.")
			return nil
		}
		return err
	}

	active := "system"
	if AppConfig.Toolchain != "" {
		active = "go" + strings.TrimPrefix(AppConfig.Toolchain, "go")
	}

	pterm.Printf("%s\n\n", pterm.NewStyle(pterm.FgCyan, pterm.Bold).Sprint("▶ Installed Toolchains"))

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "go") {
			if entry.Name() == active {
				pterm.Printf("  %s %s %s\n", pterm.FgGreen.Sprint("★"), pterm.FgLightWhite.Sprint(entry.Name()), pterm.FgDarkGray.Sprint("(active)"))
			} else {
				pterm.Printf("  %s %s\n", pterm.FgDarkGray.Sprint("-"), pterm.FgWhite.Sprint(entry.Name()))
			}
		}
	}
	return nil
}

func ListLocalToolchainsInteractive() error {
	dir := GetToolchainDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			pterm.Info.Println("No toolchains installed yet.")
			return nil
		}
		return err
	}

	var localVers []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "go") {
			localVers = append(localVers, entry.Name())
		}
	}

	if len(localVers) == 0 {
		pterm.Info.Println("No toolchains installed yet.")
		return nil
	}

	// Hide cursor
	cursor.Hide()
	defer cursor.Show()

	pageSize := 15
	total := len(localVers)

	area, _ := pterm.DefaultArea.Start()
	defer area.Stop()

	selectedIndex := 0
	windowStart := 0
	var chosenVersion string

	renderSelect := func() {
		var buf strings.Builder
		buf.WriteString(pterm.FgLightCyan.Sprint("? "));buf.WriteString(pterm.FgDarkGray.Sprint("Which local toolchain would you like to use?\n\n"))

		end := windowStart + pageSize
		if end > total {
			end = total
		}

		for i := windowStart; i < end; i++ {
			ver := localVers[i]
			padLen := 40 - len(ver)
			if padLen < 0 { padLen = 0 }
			paddedText := ver + strings.Repeat(" ", padLen)

			if i == selectedIndex {
				style := pterm.NewStyle(pterm.BgCyan, pterm.FgWhite, pterm.Bold)
				fmt.Fprintf(&buf, "%s\033[0m\033[K\n", style.Sprintf(" ❯ %s ", paddedText))
			} else {
				fmt.Fprintf(&buf, "%s\033[0m\033[K\n", pterm.FgWhite.Sprintf("   %s ", paddedText))
			}
		}

		for i := (end - windowStart); i < pageSize; i++ {
			buf.WriteString("\033[0m\033[K\n")
		}

		buf.WriteString(pterm.FgDarkGray.Sprintf("\n-- Version %d / %d --\033[0m\033[K\n\n", selectedIndex+1, total))
		buf.WriteString(pterm.FgDarkGray.Sprint("↑↓ Navigate    Enter Use Selected    Q Quit\033[0m\033[K\n"))
		area.Update(buf.String())
	}

	renderSelect()

	keyboard.Listen(func(key keys.Key) (stop bool, err error) {
		switch key.Code {
		case keys.Up:
			if selectedIndex > 0 {
				selectedIndex--
				if selectedIndex < windowStart {
					windowStart--
				}
				renderSelect()
			}
		case keys.Down:
			if selectedIndex < total-1 {
				selectedIndex++
				if selectedIndex >= windowStart+pageSize {
					windowStart++
				}
				renderSelect()
			}
		case keys.Enter:
			chosenVersion = localVers[selectedIndex]
			return true, nil
		case keys.Esc, keys.CtrlC:
			return true, nil
		case keys.RuneKey:
			if key.String() == "q" {
				return true, nil
			}
		}
		return false, nil
	})

	area.Stop()
	pterm.Println()

	if chosenVersion != "" {
		return UseToolchain(chosenVersion)
	}

	return nil
}

type GoDevRelease struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

func FetchRemoteToolchains() ([]GoDevRelease, error) {
	cachePath := filepath.Join(GetCraftHome(), string(CraftCacheDir), RemoteVersionsCacheFile)

	// Check if cache exists and is less than TTL
	if stat, err := os.Stat(cachePath); err == nil {
		if time.Since(stat.ModTime()) < RemoteVersionsCacheTTL {
			data, err := os.ReadFile(cachePath)
			if err == nil {
				var releases []GoDevRelease
				if err := json.Unmarshal(data, &releases); err == nil && len(releases) > 0 {
					return releases, nil
				}
			}
		}
	}

	resp, err := http.Get(goDevAPIURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status: %d", resp.StatusCode)
	}

	var releases []GoDevRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	// Save to cache
	os.MkdirAll(filepath.Dir(cachePath), 0755)
	if data, err := json.Marshal(&releases); err == nil {
		os.WriteFile(cachePath, data, 0644)
	}

	return releases, nil
}

func ListRemoteToolchains(selectMode bool) error {
	spinner, _ := pterm.DefaultSpinner.Start("Fetching remote toolchains from go.dev...")

	releases, err := FetchRemoteToolchains()
	if err != nil {
		spinner.Fail("Failed to fetch toolchains: " + err.Error())
		return err
	}

	spinner.Success("Remote toolchains synchronized")
	pterm.Println()

	// Hide the terminal cursor to prevent the blinking '|' artifact while scrolling
	cursor.Hide()
	defer cursor.Show()

	pageSize := 15
	total := len(releases)

	area, _ := pterm.DefaultArea.Start()
	defer area.Stop()

	if !selectMode {
		// --- SCROLL MODE (List) ---
		windowStart := 0

		renderList := func() {
			var buf strings.Builder
			buf.WriteString(pterm.FgCyan.Sprint("Remote Toolchains (go.dev)\n\n"))

			end := windowStart + pageSize
			if end > total {
				end = total
			}

			for i := windowStart; i < end; i++ {
				rel := releases[i]
				if rel.Stable {
					buf.WriteString(pterm.FgLightGreen.Sprint("  ✓ "));buf.WriteString(pterm.FgWhite.Sprint(rel.Version));buf.WriteString("\033[0m\033[K\n")
				} else {
					buf.WriteString(pterm.FgDarkGray.Sprint("  - "));buf.WriteString(pterm.FgGray.Sprint(rel.Version));buf.WriteString(pterm.FgDarkGray.Sprint(" (unstable)"));buf.WriteString("\033[0m\033[K\n")
				}
			}

			// Add empty lines to keep height consistent
			for i := (end - windowStart); i < pageSize; i++ {
				buf.WriteString("\033[0m\033[K\n")
			}

			buf.WriteString(pterm.FgDarkGray.Sprintf("\n--- %d-%d / %d ---\033[0m\033[K\n\n", windowStart+1, end, total))
			buf.WriteString(pterm.FgDarkGray.Sprint("↑↓ Scroll   Q Quit\033[0m\033[K\n\n"))
			buf.WriteString(pterm.FgCyan.Sprint(" i "));buf.WriteString(pterm.FgWhite.Sprint(" Use 'craft toolchain install <version>' to download.\033[0m\033[K\n"))

			area.Update(buf.String())
		}

		renderList()

		keyboard.Listen(func(key keys.Key) (stop bool, err error) {
			switch key.Code {
			case keys.Up:
				if windowStart > 0 {
					windowStart--
					renderList()
				}
			case keys.Down:
				if windowStart < total-pageSize {
					windowStart++
					renderList()
				}
			case keys.Esc, keys.CtrlC:
				return true, nil
			case keys.RuneKey:
				if key.String() == "q" {
					return true, nil
				}
			}
			return false, nil
		})
		return nil
	}

	// --- SELECT MODE (Interactive Scroll) ---
	selectedIndex := 0
	windowStart := 0
	var chosenVersion string

	renderSelect := func() {
		var buf strings.Builder
		buf.WriteString(pterm.FgLightCyan.Sprint("? "));buf.WriteString(pterm.FgDarkGray.Sprint("Which toolchain would you like to install?\n\n"))

		end := windowStart + pageSize
		if end > total {
			end = total
		}

		for i := windowStart; i < end; i++ {
			rel := releases[i]

			text := rel.Version
			if !rel.Stable {
				text += " (unstable)"
			}

			// Manually pad the text to 40 characters for a uniform block width
			padLen := 40 - len(text)
			if padLen < 0 {
				padLen = 0
			}
			paddedText := text + strings.Repeat(" ", padLen)

			if i == selectedIndex {
				// Selected item: Full width Cyan Background, White Text
				style := pterm.NewStyle(pterm.BgCyan, pterm.FgDarkGray, pterm.Bold)
				buf.WriteString(style.Sprintf(" ❯ %s ", paddedText));buf.WriteString("\033[0m\033[K\n")
			} else {
				// Unselected item
				if rel.Stable {
					buf.WriteString(pterm.FgWhite.Sprintf("   %s ", paddedText));buf.WriteString("\033[0m\033[K\n")
				} else {
					buf.WriteString(pterm.FgDarkGray.Sprintf("   %s ", paddedText));buf.WriteString("\033[0m\033[K\n")
				}
			}
		}

		// Empty padding to keep height consistent
		for i := (end - windowStart); i < pageSize; i++ {
			buf.WriteString("\033[0m\033[K\n")
		}

		buf.WriteString(pterm.FgDarkGray.Sprintf("\n-- Version %d / %d --\033[0m\033[K\n\n", selectedIndex+1, total))
		buf.WriteString(pterm.FgDarkGray.Sprint("↑↓ Navigate    Enter Install Selected    Q Quit\033[0m\033[K\n"))
		area.Update(buf.String())
	}

	renderSelect()

	keyboard.Listen(func(key keys.Key) (stop bool, err error) {
		switch key.Code {
		case keys.Up:
			if selectedIndex > 0 {
				selectedIndex--
				if selectedIndex < windowStart {
					windowStart--
				}
				renderSelect()
			}
		case keys.Down:
			if selectedIndex < total-1 {
				selectedIndex++
				if selectedIndex >= windowStart+pageSize {
					windowStart++
				}
				renderSelect()
			}
		case keys.Enter:
			chosenVersion = releases[selectedIndex].Version
			return true, nil
		case keys.Esc, keys.CtrlC:
			return true, nil
		case keys.RuneKey:
			if key.String() == "q" {
				return true, nil
			}
		}
		return false, nil
	})

	area.Stop()

	if chosenVersion != "" {
		pterm.Println()
		pterm.Success.Printf("Selected toolchain: %s\n", chosenVersion)
		return InstallToolchain(chosenVersion, false)
	}

	return nil
}

func CheckSystemGoVersion(goVer string) (bool, error) {
	cmd := exec.Command("go", "version")
	out, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	// go version output format: "go version go1.22.1 windows/amd64"
	return strings.Contains(string(out), " "+goVer+" "), nil
}

func UseToolchain(version string) error {
	version = strings.TrimPrefix(version, "go")
	goVer := "go" + version

	targetDir := filepath.Join(GetToolchainDir(), goVer)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		// Not in .craft/toolchains. Check global system Go.
		isSys, _ := CheckSystemGoVersion(goVer)
		if isSys {
			pterm.Printf("  %s %s\n", pterm.FgLightGreen.Sprint("✓"), pterm.FgGray.Sprintf("System Go matches %s. Will use global toolchain.", pterm.FgWhite.Sprint(goVer)))
		} else {
			// Not on system and not in craft. Auto-install it!
			pterm.Warning.Printf("Toolchain %s not found locally. Auto-downloading...\n", goVer)
			err := InstallToolchain(goVer, false)
			if err != nil {
				return err
			}
		}
	}

	err := UpdateToolchainInConfig(goVer)
	if err != nil {
		pterm.Error.Println("Failed to save config:", err)
		return err
	}

	pterm.Printf("  %s %s\n", pterm.FgLightGreen.Sprint("✓"), pterm.FgGray.Sprintf("Configured to use toolchain %s for this project.", pterm.FgWhite.Sprint(goVer)))
	return nil
}

// GetToolchainEnv returns the GOROOT and PATH variables to inject into exec.Cmd
func GetToolchainEnv() ([]string, error) {
	if AppConfig.Toolchain == "" {
		return nil, nil
	}

	goVer := "go" + strings.TrimPrefix(AppConfig.Toolchain, "go")
	targetDir := filepath.Join(GetToolchainDir(), goVer)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		// 1. Not in craft toolchains. Check system.
		isSys, _ := CheckSystemGoVersion(goVer)
		if isSys {
			// It matches the global system Go. Do not override GOROOT/PATH.
			return nil, nil
		}

		// 2. Not in system either. Auto-install!
		pterm.Warning.Printf("Toolchain %s is required but missing. Auto-downloading...\n", goVer)
		err := InstallToolchain(goVer, false)
		if err != nil {
			return nil, fmt.Errorf("auto-install failed: %v", err)
		}

		// Verify it was actually installed
		if _, err := os.Stat(targetDir); os.IsNotExist(err) {
			return nil, fmt.Errorf("toolchain %s missing after auto-install", goVer)
		}
	}

	gorootEnv := "GOROOT=" + targetDir

	pathEnv := os.Getenv("PATH")
	binDir := filepath.Join(targetDir, "bin")
	pathSeparator := string(os.PathListSeparator)

	newPath := "PATH=" + binDir + pathSeparator + pathEnv

	// Copy existing environment and replace/add GOROOT and PATH
	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GOROOT=") || strings.HasPrefix(e, "PATH=") {
			continue
		}
		env = append(env, e)
	}

	env = append(env, gorootEnv, newPath)
	return env, nil
}

// unzip is a simple zip extractor
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue // ZipSlip prevention
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}
		io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
	}
	return nil
}

// untar is a simple tar.gz extractor
func untar(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dest, header.Name)
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue // TarSlip prevention
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	return nil
}
