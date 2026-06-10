package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pterm/pterm"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/json"
	"github.com/tdewolff/minify/v2/svg"
)

// RunMinifyProcess scans the configured directories and outputs minified copies
func RunMinifyProcess() error {
	if !AppConfig.Minify.Enabled || len(AppConfig.Minify.Dirs) == 0 {
		return nil
	}

	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("text/html", html.Minify)
	m.AddFunc("application/javascript", js.Minify)
	m.AddFunc("image/svg+xml", svg.Minify)
	m.AddFuncRegexp(regexp.MustCompile("application/json"), json.Minify)

	validExts := make(map[string]bool)
	for _, ext := range AppConfig.Minify.Extensions {
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		validExts[strings.ToLower(ext)] = true
	}

	totalMinified := 0
	var totalSavedBytes int64 = 0

	for _, dir := range AppConfig.Minify.Dirs {
		// e.g. "public" -> "public_min"
		minDir := dir + "_min"

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			pterm.Warning.Printf("[craft] Minify: directory '%s' not found, skipping.\n", dir)
			continue
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Generate mirror path in minDir
			relPath, _ := filepath.Rel(dir, path)
			targetPath := filepath.Join(minDir, relPath)

			if info.IsDir() {
				return os.MkdirAll(targetPath, 0755)
			}

			ext := strings.ToLower(filepath.Ext(path))
			if validExts[ext] {
				// Minify file
				origSize := info.Size()
				err := minifyFile(m, path, targetPath, ext)
				if err != nil {
					pterm.Error.Printf("[craft] Failed to minify %s: %v\n", path, err)
					// Copy fallback
					copyFile(path, targetPath)
				} else {
					fi, _ := os.Stat(targetPath)
					totalSavedBytes += (origSize - fi.Size())
					totalMinified++
				}
			} else {
				// Just copy non-minifiable files (images, fonts, etc.)
				copyFile(path, targetPath)
			}
			return nil
		})

		if err != nil {
			pterm.Error.Printf("[craft] Minify Error on directory %s: %v\n", dir, err)
		}
	}

	if totalMinified > 0 {
		pterm.Success.Printf("[craft] Minified %d files | Saved: %s\n", totalMinified, UI.FormatSize(totalSavedBytes))
	}

	return nil
}

func minifyFile(m *minify.M, src, dst, ext string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	mimeType := ""
	switch ext {
	case ".css":
		mimeType = "text/css"
	case ".html", ".htm", ".tpl":
		mimeType = "text/html"
	case ".js":
		mimeType = "application/javascript"
	case ".svg":
		mimeType = "image/svg+xml"
	case ".json":
		mimeType = "application/json"
	}

	if mimeType != "" {
		return m.Minify(mimeType, out, in)
	}

	return fmt.Errorf("unsupported mime type")
}

func copyFile(src, dst string) error {
	in, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, in, 0644)
}
