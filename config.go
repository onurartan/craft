package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
	"gopkg.in/yaml.v3"
)

const (
	ConfigFileName = ".craft.yaml"
	DefaultDistDir = "bin"
)

type ProfileConfig struct {
	OutputDir string   `yaml:"output_dir"`
	Name      string   `yaml:"name"`
	BuildAll  bool     `yaml:"build_all"`
	Platforms []string `yaml:"platforms"`
	ExactName bool     `yaml:"exact_name"`
}

type WatchConfig struct {
	// Enabled      bool     `yaml:"enabled"`
	DelayMs      int      `yaml:"delay_ms"`
	IncludeExts  []string `yaml:"include_exts"`
	ExcludeDirs  []string `yaml:"exclude_dirs"`
	ExcludeFiles []string `yaml:"exclude_files"`
}

type DevConfig struct {
	Watch WatchConfig `yaml:"watch"`
}

type Config struct {
	Name       string    `yaml:"name"`
	Version    string    `yaml:"version"`
	EntryPoint string    `yaml:"entry_point"`
	OutputDir  string    `yaml:"output_dir"`
	VersionPkg string    `yaml:"version_pkg"`
	BuildAll   bool      `yaml:"build_all"`
	Platforms  []string  `yaml:"platforms"`
	Profiles   map[string]ProfileConfig `yaml:"profiles"`
	StripDebug bool      `yaml:"strip_debug"`
	ExactName  bool      `yaml:"exact_name"`
	Trimpath   bool      `yaml:"trimpath"`
	CgoEnabled bool      `yaml:"cgo_enabled"`
	Race       bool      `yaml:"race"`
	Tags       []string  `yaml:"tags"`
	Dev        DevConfig `yaml:"dev"`
}

// Global AppConfig
var AppConfig Config

func ConfigExists() bool {
	_, err := os.Stat(ConfigFileName)
	return !os.IsNotExist(err)
}

func ConfigLoad() {
	AppConfig = Config{
		Version:    "1.0.0",
		EntryPoint: ".",
		OutputDir:  DefaultDistDir,
		StripDebug: true,
		Trimpath:   true,
		CgoEnabled: false,
		Dev: DevConfig{
			Watch: WatchConfig{
				// Enabled:     false,
				DelayMs:     500,
				IncludeExts: []string{".go", ".html", ".tpl", ".env", ".yaml", ".json"},
				ExcludeDirs: []string{".git", "vendor", "node_modules", "bin", "tmp", "assets", "testdata", ".vscode"},
			},
		},
	}

	if ConfigExists() {
		data, err := os.ReadFile(ConfigFileName)
		if err == nil {
			_ = yaml.Unmarshal(data, &AppConfig)
		}
	}
}

func ResolveVersion() {
	if strings.HasPrefix(AppConfig.Version, "in_go:") {
		ver, err := extractVersionFromAST(strings.TrimPrefix(AppConfig.Version, "in_go:"))
		if err == nil {
			AppConfig.Version = ver
		} else {
			AppConfig.Version = "unknown"
		}
	} else if strings.HasPrefix(AppConfig.Version, "file:") {
		directive := strings.TrimPrefix(AppConfig.Version, "file:")
		parts := strings.SplitN(directive, "|", 2)
		data, err := os.ReadFile(parts[0])
		if err == nil {
			if len(parts) == 1 {
				AppConfig.Version = strings.TrimSpace(string(data))
			} else {
				ver, err := extractFromStruct(data, parts[1], strings.ToLower(filepath.Ext(parts[0])))
				if err == nil {
					AppConfig.Version = ver
				} else {
					AppConfig.Version = "unknown"
				}
			}
		} else {
			AppConfig.Version = "unknown"
		}
	}
}

func extractVersionFromAST(directive string) (string, error) {
	lastDot := strings.LastIndex(directive, ".")
	if lastDot == -1 {
		return "", fmt.Errorf("invalid AST directive")
	}
	pkgPath, varName := directive[:lastDot], directive[lastDot+1:]
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedSyntax | packages.NeedFiles | packages.NeedName}, pkgPath)
	if err != nil {
		return "", err
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || (genDecl.Tok != token.CONST && genDecl.Tok != token.VAR) {
					continue
				}
				for _, spec := range genDecl.Specs {
					valSpec, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, name := range valSpec.Names {
						if name.Name == varName && i < len(valSpec.Values) {
							if basicLit, ok := valSpec.Values[i].(*ast.BasicLit); ok {
								return strings.Trim(basicLit.Value, "\"`"), nil
							}
						}
					}
				}
			}
		}
	}
	return "", fmt.Errorf("target not found")
}

func extractFromStruct(data []byte, keyPath string, ext string) (string, error) {
	var parsedData map[string]interface{}
	var err error
	if ext == ".json" {
		err = json.Unmarshal(data, &parsedData)
	} else if ext == ".yaml" || ext == ".yml" {
		err = yaml.Unmarshal(data, &parsedData)
	} else {
		return "", fmt.Errorf("unsupported format")
	}
	if err != nil {
		return "", err
	}

	keys := strings.Split(keyPath, ".")
	var current interface{} = parsedData
	for i, key := range keys {
		m, ok := current.(map[string]interface{})
		if !ok {
			if yamlMap, yamlOk := current.(map[interface{}]interface{}); yamlOk {
				m = make(map[string]interface{})
				for k, v := range yamlMap {
					m[fmt.Sprintf("%v", k)] = v
				}
			} else {
				return "", fmt.Errorf("invalid path")
			}
		}
		val, exists := m[key]
		if !exists {
			return "", fmt.Errorf("key not found")
		}
		current = val
		if i == len(keys)-1 {
			return fmt.Sprintf("%v", current), nil
		}
	}
	return "", fmt.Errorf("extraction error")
}
