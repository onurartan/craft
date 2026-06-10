package main

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func registerCLICommands(rootCmd *cobra.Command) {
	// Define command groups
	rootCmd.AddGroup(&cobra.Group{ID: "core", Title: "Core Commands"})
	rootCmd.AddGroup(&cobra.Group{ID: "go", Title: "Go Toolchain Wrappers"})
	rootCmd.AddGroup(&cobra.Group{ID: "diag", Title: "Diagnostics & Tools"})
	rootCmd.AddGroup(&cobra.Group{ID: "custom", Title: "Custom Task Macros"})

	initCmd := &cobra.Command{
		Use:     "init [name]",
		Short:   "Smart initialization of " + ConfigFileName,
		GroupID: "core",
		RunE: func(cmd *cobra.Command, args []string) error {
			UI.PrintBanner()
			return ExecuteInitProcess(args)
		},
	}

	addCmd := &cobra.Command{
		Use:     "add [packages...]",
		Short:   "Add Go packages and update dependencies",
		GroupID: "core",
		RunE: func(cmd *cobra.Command, args []string) error {
			UI.PrintBanner()
			return ExecuteAddProcess(args)
		},
	}

	removeCmd := &cobra.Command{
		Use:     "remove [packages...]",
		Short:   "Remove Go packages and clean dependencies",
		GroupID: "core",
		RunE: func(cmd *cobra.Command, args []string) error {
			UI.PrintBanner()
			return ExecuteRemoveProcess(args)
		},
	}

	syncCmd := &cobra.Command{
		Use:     "sync",
		Short:   "Synchronize, download and verify project dependencies",
		GroupID: "core",
		RunE: func(cmd *cobra.Command, args []string) error {
			UI.PrintBanner()
			return ExecuteSyncProcess()
		},
	}

	buildCmd := &cobra.Command{
		Use:     "build",
		Short:   "Compile the project artifacts",
		GroupID: "core",
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeBuildProcess(false)
		},
	}

	runCmd := &cobra.Command{
		Use:     "run",
		Short:   "Compile and execute current host binary",
		GroupID: "core",
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRunProcess(cmd, args)
		},
	}

	irunCmd := &cobra.Command{
		Use:     "irun",
		Short:   "Execute existing binary immediately",
		GroupID: "core",
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeIRunProcess(cmd, args)
		},
	}

	devCmd := &cobra.Command{
		Use:     "dev",
		Short:   "Run in development mode (Hot-Reload)",
		GroupID: "core",
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeDevProcess(args)
		},
	}

	cleanCmd := &cobra.Command{
		Use:     "clean",
		Short:   "Deep clean artifacts and Go cache",
		GroupID: "diag",
		RunE: func(cmd *cobra.Command, args []string) error {
			UI.PrintBanner()
			return ExecuteCleanProcess(flagGoCache, flagCleanAll)
		},
	}

	cleanCmd.Flags().BoolVar(&flagGoCache, "go-cache", false, "Force purge global Go build cache")
	cleanCmd.Flags().BoolVar(&flagCleanAll, "all", false, "Purge all craft-dev temporary artifacts globally")

	checkCmd := &cobra.Command{
		Use:     "check",
		Short:   "Run CI/CD prep (fmt, vet, test)",
		GroupID: "diag",
		RunE: func(cmd *cobra.Command, args []string) error {
			UI.PrintBanner()
			return ExecuteCheckProcess()
		},
	}

	doctorCmd := &cobra.Command{
		Use:     "doctor",
		Short:   "Diagnose system and configuration health",
		GroupID: "diag",
		RunE: func(cmd *cobra.Command, args []string) error {
			UI.PrintBanner()
			return ExecuteDoctorProcess()
		},
	}

	tidyCmd := &cobra.Command{
		Use:     "tidy",
		Short:   "Synchronize module dependencies",
		GroupID: "go",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ExecuteTidyProcess()
		},
	}

	fmtCmd := &cobra.Command{
		Use:                "fmt",
		Short:              "Format Go source code",
		GroupID:            "go",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ExecuteFmtProcess(args...)
		},
	}

	vetCmd := &cobra.Command{
		Use:                "vet",
		Short:              "Run static analysis",
		GroupID:            "go",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ExecuteVetProcess(args...)
		},
	}

	testCmd := &cobra.Command{
		Use:                "test [packages/flags...]",
		Short:              "Execute test suites",
		GroupID:            "go",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ExecuteTestProcess(args...)
		},
	}

	genCmd := &cobra.Command{
		Use:                "gen",
		Short:              "Run code generation",
		GroupID:            "go",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ExecuteGenerateProcess(args...)
		},
	}

	installCmd := &cobra.Command{
		Use:     "install [package]",
		Aliases: []string{"i"},
		Short:   "Install binary to GOBIN",
		GroupID: "core",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ExecuteInstallProcess(args)
		},
	}

	// --- Toolchain Management ---
	toolchainCmd := &cobra.Command{
		Use:     "toolchain",
		Short:   "Manage isolated Go toolchains",
		GroupID: "core",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return ShowToolchainDashboard()
			}
			return cmd.Help()
		},
	}

	var installRemote bool
	var noCache bool
	toolchainInstallCmd := &cobra.Command{
		Use:   "install [version]",
		Short: "Download and install a specific Go version (e.g. 1.22.1)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if installRemote {
				return ListRemoteToolchains(true)
			}
			if len(args) == 0 {
				return fmt.Errorf("requires a version argument or --remote flag")
			}
			return InstallToolchain(args[0], noCache)
		},
	}
	toolchainInstallCmd.Flags().BoolVarP(&installRemote, "remote", "r", false, "Interactive menu to install from go.dev")
	toolchainInstallCmd.Flags().BoolVar(&noCache, "no-cache", false, "Disable cache and force download from go.dev")

	var remoteList bool
	var selectMode bool
	toolchainListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all installed Go toolchains",
		RunE: func(cmd *cobra.Command, args []string) error {
			if selectMode {
				if remoteList {
					return ListRemoteToolchains(true)
				}
				return ListLocalToolchainsInteractive()
			}

			if remoteList {
				return ListRemoteToolchains(false)
			}
			return ListToolchains()
		},
	}
	toolchainListCmd.Flags().BoolVarP(&remoteList, "remote", "r", false, "List available remote versions from go.dev")
	toolchainListCmd.Flags().BoolVar(&selectMode, "select-mode", false, "Interactive menu to select a toolchain")

	toolchainUseCmd := &cobra.Command{
		Use:   "use <version>",
		Short: "Set the active Go toolchain for the current project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return UseToolchain(args[0])
		},
	}

	toolchainRemoveCmd := &cobra.Command{
		Use:   "remove <version>",
		Short: "Remove a specific Go toolchain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RemoveToolchain(args[0])
		},
	}

	toolchainCleanCmd := &cobra.Command{
		Use:   "clean [version]",
		Short: "Clear the toolchain archive cache to free up space",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			version := ""
			if len(args) > 0 {
				version = args[0]
			}
			return CleanToolchainCache(version)
		},
	}

	toolchainCmd.AddCommand(toolchainInstallCmd, toolchainListCmd, toolchainUseCmd, toolchainRemoveCmd, toolchainCleanCmd)

	// Flag orchestration
	setupFlags(buildCmd)
	runCmd.Flags().BoolVar(&flagScript, "script", false, "Execute the target file as an isolated script")

	ConfigLoad()
	injectCustomCommands(rootCmd)

	versionCmd := &cobra.Command{
		Use:     "version",
		Short:   "Print the version number of Craft",
		GroupID: "core",
		Run: func(cmd *cobra.Command, args []string) {
			pterm.Printf("Craft Build Engine %s\n", pterm.FgCyan.Sprint(CraftAppVersion))
		},
	}

	rootCmd.AddCommand(initCmd, createCmd, addCmd, removeCmd, syncCmd, buildCmd, runCmd, irunCmd, devCmd, cleanCmd, checkCmd, doctorCmd, tidyCmd, fmtCmd, vetCmd, testCmd, genCmd, installCmd, toolchainCmd, versionCmd)
}
