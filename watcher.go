// watcher.go: Implementation of the 'craft dev' command logic.
// This module orchestrates a recursive filesystem watcher with an event
// debouncing strategy to manage the hot-reload development cycle.
//
// Workflow:
// 1. Recursive discovery of directories (excluding blacklisted paths).
// 2. Monitoring of specific extensions (.go, .yaml, .html, etc.).
// 3. Debouncing rapid events (500ms) to prevent build spamming.
// 4. Sub-process lifecycle: Terminate -> Clean Old Artifact -> Rebuild -> Start.
// 5. Graceful shutdown: Resource cleanup on SIGINT/SIGTERM.
//
// Copyright (c) 2026 Onurartan/Craft Contributors.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pterm/pterm"
)

// StartDevMode initiates the hot-reload engine and process orchestrator.
func StartDevMode(args []string) error {
	cfg := AppConfig

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	watchCount := 0
	excludeCount := 0

	// Recursive directory discovery with error resilience and exclusion logic.
	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			pterm.FgYellow.Printf("[craft] warning: skipping '%s' (reason: %v)\n", path, err)
			if info != nil && info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			name := info.Name()
			for _, exclude := range cfg.Dev.Watch.ExcludeDirs {
				if name == exclude {
					excludeCount++
					return filepath.SkipDir
				}
			}
			watchCount++
			return watcher.Add(path)
		}
		return nil
	})

	pterm.Println()
	pterm.FgCyan.Printf("[craft] watching %d directories\n", watchCount)
	if excludeCount > 0 || len(cfg.Dev.Watch.ExcludeFiles) > 0 {
		pterm.FgDarkGray.Printf("[craft] excluded %d directories and %d files\n", excludeCount, len(cfg.Dev.Watch.ExcludeFiles))
	}
	pterm.FgDarkGray.Printf("[craft] extensions: %v\n", cfg.Dev.Watch.IncludeExts)
	pterm.Println()

	var currentProcess *exec.Cmd
	var lastTempFile string

	// Capture OS signals for graceful shutdown and resource cleanup.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	buildChan := make(chan bool, 1)
	buildChan <- true

	// Event processor goroutine with debounce logic to handle rapid IDE saves.
	go func() {
		var timer *time.Timer
		delay := time.Duration(cfg.Dev.Watch.DelayMs) * time.Millisecond

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Skip metadata-only changes (Chmod) to avoid redundant triggers.
				if event.Op == fsnotify.Chmod {
					continue
				}

				baseName := filepath.Base(event.Name)

				// Filename-based exclusion check.
				isExcludedFile := false
				for _, excFile := range cfg.Dev.Watch.ExcludeFiles {
					if baseName == excFile || strings.HasSuffix(baseName, excFile) {
						isExcludedFile = true
						break
					}
				}
				if isExcludedFile {
					continue
				}

				// Extension-based inclusion check.
				ext := filepath.Ext(event.Name)
				isWatchedExt := false
				for _, incExt := range cfg.Dev.Watch.IncludeExts {
					cleanExt := incExt
					if !strings.HasPrefix(cleanExt, ".") {
						cleanExt = "." + cleanExt
					}
					if ext == cleanExt {
						isWatchedExt = true
						break
					}
				}

				if isWatchedExt {
					if timer != nil {
						timer.Stop()
					}
					timer = time.AfterFunc(delay, func() {
						cleanName := strings.Replace(event.Name, "\\", "/", -1)
						pterm.FgCyan.Printf("\n[craft] file changed: %s\n", cleanName)
						buildChan <- true
					})
				}
			case <-watcher.Errors:
			}
		}
	}()

	// Orchestration loop: Manages the build -> run -> kill lifecycle.
	for {
		select {
		case <-buildChan:
			// Terminate previous instance and wait for port release.
			if currentProcess != nil && currentProcess.Process != nil {
				_ = currentProcess.Process.Kill()
				_ = currentProcess.Wait()
			}

			// Atomic cleanup of the previous binary.
			if lastTempFile != "" {
				_ = os.Remove(lastTempFile)
			}

			pterm.FgYellow.Println("building...")

			tmpFilePath := getTempFilePath()
			lastTempFile = tmpFilePath

			res := Builder.Compile(BuildTarget{OS: runtime.GOOS, Arch: runtime.GOARCH}, tmpFilePath, true)

			if res.ErrorMsg != "" {
				pterm.FgRed.Println("build failed!")
				fmt.Println(res.ErrorMsg)
				continue
			}

			pterm.FgMagenta.Printf("building success in %s\n", res.Duration.Round(time.Millisecond))
			pterm.FgGreen.Println("running...")

			currentProcess = exec.Command(tmpFilePath, args...)
			currentProcess.Stdout = os.Stdout
			currentProcess.Stderr = os.Stderr
			currentProcess.Stdin = os.Stdin

			if err := currentProcess.Start(); err != nil {
				pterm.FgRed.Printf("failed to start process: %v\n", err)
			}

		case <-sigChan:
			pterm.Println()
			pterm.FgYellow.Println("[craft] shutting down gracefully...")

			if currentProcess != nil && currentProcess.Process != nil {
				_ = currentProcess.Process.Kill()
				_ = currentProcess.Wait()
			}

			if lastTempFile != "" {
				_ = os.Remove(lastTempFile)
			}

			pterm.FgGreen.Println("[craft] cleanup complete. bye!")
			return nil
		}
	}
}

func getTempFilePath() string {
	tmpDir := os.TempDir()
	binName := fmt.Sprintf("craft-dev-%s-%d", AppConfig.Name, time.Now().UnixMilli())
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	return filepath.Join(tmpDir, binName)
}
