/*
CRAFT - Go development and build tool.

A professional Go build tool designed to bridge the gap between simple 'go build'
and complex task runners. Craft offers a seamless, multi-platform build experience
driven by a declarative 'craft.yaml' configuration.

/*
CRAFT is a tool for managing Go builds and development workflows.

Usage:
  craft <command> [arguments]

The commands are:

  build       compile packages and dependencies
  run         compile and run Go program
  dev         start hot-reload development environment
  irun        run existing binary
  clean       remove object files and cached files
  doctor      show environment information

Toolchain wrappers:
  check       run fmt, vet and test sequentially
  fmt         run 'go fmt ./...'
  gen         run 'go generate ./...'
  install     compile and install packages and dependencies
  test        run 'go test ./...'
  tidy        run 'go mod tidy'
  vet         run 'go vet ./...'

Flags:
  All commands support CLI overrides for binary names, versions, and platforms.
  Use 'craft [command] --help' for detailed flag information.
*/

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var CraftAppVersion = "1.0.0-beta" // Default version, can be overridden via LDFLAGS

var (
	flagName      string
	flagVersion   string
	flagEntry     string
	flagOut       string
	flagVerPkg    string
	flagAll       bool
	flagPlats     []string
	flagStrip     bool
	flagExactName bool
	flagProfile   string

	flagGoCache  bool
	flagCleanAll bool

	flagNoAutoInstall bool
	flagVerbose       bool
	flagScript        bool

	activeCmd *cobra.Command

	createCmd = &cobra.Command{
		Use:     "create",
		Short:   "Interactively scaffold a new Craft project",
		GroupID: "core",
		Run: func(cmd *cobra.Command, args []string) {
			if err := ExecuteCreateProcess(); err != nil {
				pterm.Error.Printf("Failed to create project: %v\n", err)
				os.Exit(1)
			}
		},
	}
)

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		pterm.Printf("Craft Build Engine %s\n", pterm.FgCyan.Sprint(CraftAppVersion))
		os.Exit(0)
	}

	pterm.Info.Prefix = pterm.Prefix{Text: "i", Style: pterm.NewStyle(pterm.FgLightCyan, pterm.Bold)}
	pterm.Info.MessageStyle = pterm.NewStyle(pterm.FgWhite)

	pterm.Success.Prefix = pterm.Prefix{Text: "✓", Style: pterm.NewStyle(pterm.FgLightGreen)}
	pterm.Success.MessageStyle = pterm.NewStyle(pterm.FgWhite)

	pterm.Error.Prefix = pterm.Prefix{Text: "✖", Style: pterm.NewStyle(pterm.FgLightRed)}
	pterm.Error.MessageStyle = pterm.NewStyle(pterm.FgWhite)

	pterm.Warning.Prefix = pterm.Prefix{Text: "⚠", Style: pterm.NewStyle(pterm.FgLightYellow)}
	pterm.Warning.MessageStyle = pterm.NewStyle(pterm.FgWhite)

	// Override DefaultSpinner globally
	pterm.DefaultSpinner.Sequence = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	pterm.DefaultSpinner.RemoveWhenDone = true
	pterm.DefaultSpinner.MessageStyle = pterm.NewStyle(pterm.FgWhite)

	// Override Interactive Inputs globally
	pterm.DefaultInteractiveSelect.DefaultText = pterm.FgLightCyan.Sprint("?")
	pterm.DefaultInteractiveSelect.TextStyle = pterm.NewStyle(pterm.FgWhite)

	pterm.DefaultInteractiveTextInput.DefaultText = pterm.FgLightCyan.Sprint("?")
	pterm.DefaultInteractiveTextInput.TextStyle = pterm.NewStyle(pterm.FgWhite)

	// Root command: Default behavior is to initiate a full build.
	rootCmd := &cobra.Command{
		Use:   "craft",
		Short: "Professional Go Build Engine",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			activeCmd = cmd
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !ConfigExists() && !cmd.Flags().Changed("name") {
				pterm.Error.Printf("No %s found and --name flag not provided.\n", ConfigFileName)
				os.Exit(1)
			}
			return executeBuildProcess(false)
		},
	}

	// Flag orchestration
	setupFlags(rootCmd)

	registerCLICommands(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func setupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&flagName, "name", "n", "", "Binary name")
	cmd.Flags().StringVarP(&flagVersion, "version", "v", "1.0.0", "App version")
	cmd.Flags().StringVarP(&flagEntry, "entry", "e", ".", "Main package path")
	cmd.Flags().StringVarP(&flagOut, "out", "o", DefaultDistDir, "Output directory")
	cmd.Flags().StringVar(&flagVerPkg, "ver-pkg", "", "Variable path for version injection")
	cmd.Flags().BoolVar(&flagAll, "all", false, "Build for all common platforms")

	cmd.Flags().StringSliceVarP(&flagPlats, "platform", "p", []string{}, "Custom target platforms")
	cmd.Flags().BoolVar(&flagStrip, "strip", true, "Strip debug symbols")
	cmd.Flags().BoolVar(&flagExactName, "exact-name", false, "Omit OS/Arch suffixes from binary")
	cmd.Flags().BoolVar(&flagNoAutoInstall, "no-auto-install", false, "Disable magical auto-install of missing packages")

	cmd.Flags().StringVarP(&flagProfile, "profile", "P", "", "Use specific build profile from craft.yaml")

	// Make verbose persistent so it applies to all subcommands
	cmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "Enable verbose output and disable error parser")
}

// prepareEngine merges craft.yaml with CLI overrides.
func prepareEngine() {
	if flagNoAutoInstall {
		AppConfig.AutoInstall = false
	}

	if flagProfile != "" {
		if prof, exists := AppConfig.Profiles[flagProfile]; exists {
			if prof.OutputDir != "" {
				AppConfig.OutputDir = prof.OutputDir
			}
			if prof.Name != "" {
				AppConfig.Name = prof.Name
			}
			if prof.BuildAll {
				AppConfig.BuildAll = true
				AppConfig.Platforms = nil 
			}
			if len(prof.Platforms) > 0 {
				AppConfig.Platforms = prof.Platforms
			}
			if prof.ExactName {
				AppConfig.ExactName = true
			}

			pterm.Success.Printf("Loaded build profile: [%s]\n", pterm.FgCyan.Sprint(flagProfile))
		} else {
			pterm.Error.Printf("Profile '%s' not found in %s\n", flagProfile, ConfigFileName)
			os.Exit(1)
		}
	}

	if activeCmd != nil {
		if activeCmd.Flags().Changed("name") {
			AppConfig.Name = flagName
		}
		if activeCmd.Flags().Changed("version") {
			AppConfig.Version = flagVersion
		}
		if activeCmd.Flags().Changed("entry") {
			AppConfig.EntryPoint = flagEntry
		}
		if activeCmd.Flags().Changed("out") {
			AppConfig.OutputDir = flagOut
		}
		if activeCmd.Flags().Changed("ver-pkg") {
			AppConfig.VersionPkg = flagVerPkg
		}

		if activeCmd.Flags().Changed("all") {
			AppConfig.BuildAll = flagAll
			if flagAll {
				AppConfig.Platforms = nil
			}
		}
		if activeCmd.Flags().Changed("platform") {
			AppConfig.Platforms = flagPlats
			AppConfig.BuildAll = false
		}

		if activeCmd.Flags().Changed("strip") {
			AppConfig.StripDebug = flagStrip
		}
		if activeCmd.Flags().Changed("exact-name") {
			AppConfig.ExactName = flagExactName
		}
	}

	if AppConfig.Name == "" {
		pterm.Error.Printf("Project name is required. Set it in %s or use --name.\n", ConfigFileName)
		os.Exit(1)
	}

	ResolveVersion()
}

func executeBuildProcess(minimal bool) error {
	startTime := time.Now()
	prepareEngine()

	if !minimal {
		UI.PrintBanner()
	}

	// Pre-Build hook
	if err := RunScriptHook("pre_build", AppConfig.Scripts.PreBuild); err != nil {
		return fmt.Errorf("pre_build script failed: %w", err)
	}

	// Minify step
	if err := RunMinifyProcess(); err != nil {
		pterm.Error.Printf("[craft] Minification failed: %v\n", err)
	}

	targets := Builder.ResolveTargets()
	os.MkdirAll(AppConfig.OutputDir, 0755)

	if !minimal {
		UI.PrintInfo(len(targets))
		pterm.Println()
	}

	var results []BuildResult
	resultChan := make(chan BuildResult, len(targets))
	var wg sync.WaitGroup

	var multiSpin *pterm.SpinnerPrinter
	isConcurrent := len(targets) > 1 && !minimal
	if isConcurrent {
		multiSpin, _ = pterm.DefaultSpinner.Start(fmt.Sprintf("Compiling for %d platforms concurrently...", len(targets)))
	}

	for _, t := range targets {
		wg.Add(1)
		outPath := Builder.GetBinaryPath(t, len(targets))

		go func(target BuildTarget, path string) {
			defer wg.Done()
			res := Builder.Compile(target, path, minimal, isConcurrent)
			resultChan <- res
		}(t, outPath)
	}

	wg.Wait()
	close(resultChan)

	if multiSpin != nil {
		multiSpin.Success("Concurrent compilation finished")
	}

	for res := range resultChan {
		results = append(results, res)
	}

	if !minimal {
		UI.PrintSummary(results, time.Since(startTime))
	}

	// Post-Build hook
	if err := RunScriptHook("post_build", AppConfig.Scripts.PostBuild); err != nil {
		pterm.Error.Printf("[craft] post_build script failed: %v\n", err)
	}

	return nil
}

func executeRunProcess(cmd *cobra.Command, args []string) error {
	if flagScript && len(args) > 0 {
		return ExecuteInlineScript(args)
	}

	prepareEngine()

	// Intercept 'craft run <command>' if user didn't explicitly use '--' to bypass
	if len(args) > 0 && cmd.ArgsLenAtDash() != 0 {
		cmdName := args[0]
		if cmdData, exists := AppConfig.Commands[cmdName]; exists {
			pterm.Info.Printf("[craft] Intercepted 'run %s'. Executing custom task...\n", cmdName)
			return executeCustomCommand(cmdName, cmdData, args[1:], 1)
		}
	}

	UI.PrintBanner()
	AppConfig.Platforms = []string{"current"}
	AppConfig.BuildAll = false

	pterm.Info.Printf("Orchestrating temporary build for execution...\n")
	if err := executeBuildProcess(true); err != nil {
		return err
	}

	return executeIRunProcess(cmd, args)
}

func executeIRunProcess(cmd *cobra.Command, args []string) error {
	prepareEngine()

	target := BuildTarget{OS: runtime.GOOS, Arch: runtime.GOARCH}
	binPath := Builder.GetBinaryPath(target, 1)

	absPath, err := filepath.Abs(binPath)
	if err != nil {
		pterm.Error.Printf("Failed to resolve absolute path: %v\n", err)
		return err
	}

	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		pterm.Error.Printf("Binary not found: %s\n", pterm.FgLightRed.Sprint(binPath))
		pterm.Info.Println("Action Required: Run 'craft build' to generate artifacts.")
		os.Exit(1)
	}

	// Safety check for invalid/corrupted Go binaries
	if info.Size() < 500*1024 {
		pterm.Warning.Printf("Abnormal binary size detected: %s\n", UI.FormatSize(info.Size()))
		pterm.Info.Println("This often indicates a missing 'package main' or 'func main()'.")
		return fmt.Errorf("invalid executable format")
	}

	pterm.Printf("%s\n\n", pterm.NewStyle(pterm.FgCyan, pterm.Bold).Sprint("▶ Running Project..."))

	// Pre-Run hook
	if err := RunScriptHook("pre_run", AppConfig.Scripts.PreRun); err != nil {
		return fmt.Errorf("pre_run script failed: %w", err)
	}

	if err := RunSysCommand(absPath, args); err != nil {
		if strings.Contains(err.Error(), "not compatible") || strings.Contains(err.Error(), "exec format error") {
			pterm.Println()
			pterm.Printf("%s %s\n", pterm.FgRed.Sprint("✖"), pterm.FgLightRed.Sprint("Architecture mismatch or invalid PE/ELF header detected."))
			pterm.Printf("  %s %s\n", pterm.FgLightYellow.Sprint("Hint:"), pterm.FgWhite.Sprint("Run 'craft run' to re-align with host system."))
		} else {
			pterm.Printf("%s %s\n", pterm.FgRed.Sprint("✖"), pterm.FgLightRed.Sprintf("Execution failed: %v", err))
		}
		return err
	}

	// Post-Run hook
	if err := RunScriptHook("post_run", AppConfig.Scripts.PostRun); err != nil {
		pterm.Error.Printf("[craft] post_run script failed: %v\n", err)
	}

	return nil
}

func executeDevProcess(args []string) error {
	prepareEngine()
	UI.PrintBanner()

	AppConfig.Platforms = []string{"current"}
	AppConfig.BuildAll = false

	return StartDevMode(args)
}

// RunSysCommand handles low-level process execution and stream piping.
func RunSysCommand(binPath string, args []string) error {
	cmd := exec.Command(binPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
