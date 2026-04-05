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
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

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

	activeCmd *cobra.Command
)

func main() {
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

	// Command registration
	initCmd := &cobra.Command{
		Use:   "init [name]",
		Short: "Smart initialization of " + ConfigFileName,
		RunE: func(cmd *cobra.Command, args []string) error {
			UI.PrintBanner()
			return ExecuteInitProcess(args)
		},
	}

	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Compile the project artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeBuildProcess(false)
		},
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Compile and execute current host binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRunProcess(args)
		},
	}

	irunCmd := &cobra.Command{
		Use:   "irun",
		Short: "Execute existing binary immediately",
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeIRunProcess(args)
		},
	}

	devCmd := &cobra.Command{
		Use:   "dev",
		Short: "Run in development mode (Hot-Reload)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeDevProcess(args)
		},
	}

	cleanCmd := &cobra.Command{
		Use:   "clean",
		Short: "Deep clean artifacts and Go cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			UI.PrintBanner()
			return ExecuteCleanProcess(flagGoCache, flagCleanAll)
		},
	}

	cleanCmd.Flags().BoolVar(&flagGoCache, "go-cache", false, "Force purge global Go build cache")
	cleanCmd.Flags().BoolVar(&flagCleanAll, "all", false, "Purge all craft-dev temporary artifacts globally")

	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Run CI/CD prep (fmt, vet, test)",
		RunE: func(cmd *cobra.Command, args []string) error {
			UI.PrintBanner()
			return ExecuteCheckProcess()
		},
	}

	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose system and configuration health",
		RunE: func(cmd *cobra.Command, args []string) error {
			UI.PrintBanner()
			return ExecuteDoctorProcess()
		},
	}

	// --- Go Toolchain Wrappers ---

	tidyCmd := &cobra.Command{
		Use:   "tidy",
		Short: "Synchronize module dependencies",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ExecuteTidyProcess()
		},
	}

	fmtCmd := &cobra.Command{
		Use:                "fmt",
		Short:              "Format Go source code",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ExecuteFmtProcess(args...)
		},
	}

	vetCmd := &cobra.Command{
		Use:                "vet",
		Short:              "Run static analysis",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ExecuteVetProcess(args...)
		},
	}

	testCmd := &cobra.Command{
		Use:                "test [packages/flags...]",
		Short:              "Execute test suites",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ExecuteTestProcess(args...)
		},
	}

	genCmd := &cobra.Command{
		Use:                "gen",
		Short:              "Run code generation",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ExecuteGenerateProcess(args...)
		},
	}

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install binary to GOBIN",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ExecuteInstallProcess()
		},
	}

	// Flag orchestration
	setupFlags(rootCmd)
	setupFlags(buildCmd)


	rootCmd.AddCommand(initCmd, buildCmd, runCmd, irunCmd, devCmd, cleanCmd, checkCmd, doctorCmd, tidyCmd, fmtCmd, vetCmd, testCmd, genCmd, installCmd)

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

	cmd.Flags().StringVarP(&flagProfile, "profile", "P", "", "Use specific build profile from craft.yaml")
}

// prepareEngine merges craft.yaml with CLI overrides.
func prepareEngine() {
	ConfigLoad()

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
				AppConfig.Platforms = nil // BuildAll varsa platform listesini sıfırla
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

	targets := Builder.ResolveTargets()
	os.MkdirAll(AppConfig.OutputDir, 0755)

	if !minimal {
		UI.PrintInfo(len(targets))
		pterm.Println()
	}

	var results []BuildResult

	for _, t := range targets {
		outPath := Builder.GetBinaryPath(t, len(targets))
		res := Builder.Compile(t, outPath, minimal)
		results = append(results, res)
	}

	if !minimal {
		UI.PrintSummary(results, time.Since(startTime))
	}
	return nil
}

func executeRunProcess(args []string) error {
	prepareEngine()
	UI.PrintBanner()
	AppConfig.Platforms = []string{"current"}
	AppConfig.BuildAll = false

	pterm.Info.Printf("Orchestrating temporary build for execution...\n")
	if err := executeBuildProcess(true); err != nil {
		return err
	}

	return executeIRunProcess(args)
}

func executeIRunProcess(args []string) error {
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

	pterm.DefaultSection.Printf("Spawning Process: %s", pterm.FgCyan.Sprint(absPath))
	pterm.Println()

	if err := RunSysCommand(absPath, args); err != nil {
		if strings.Contains(err.Error(), "not compatible") || strings.Contains(err.Error(), "exec format error") {
			pterm.Println()
			pterm.DefaultBox.WithTitle(pterm.FgRed.Sprint(" RUNTIME CONFLICT ")).
				Println("Architecture mismatch or invalid PE/ELF header detected.\n" +
					"Recommendation: Run 'craft run' to re-align with host system.")
		} else {
			pterm.Error.Printf("Execution failed: %v\n", err)
		}
		return err
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
