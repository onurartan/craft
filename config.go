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

type MinifyConfig struct {
	Enabled    bool     `yaml:"enabled"`
	Extensions []string `yaml:"extensions"`
	Dirs       []string `yaml:"dirs"`
}

type ScriptsConfig struct {
	PreBuild  interface{} `yaml:"pre_build"`
	PostBuild interface{} `yaml:"post_build"`
	PreRun    interface{} `yaml:"pre_run"`
	PostRun   interface{} `yaml:"post_run"`
}

type Config struct {
	Name        string                   `yaml:"name"`
	Version     string                   `yaml:"version"`
	Toolchain   string                   `yaml:"toolchain"`
	EntryPoint  string                   `yaml:"entry_point"`
	OutputDir   string                   `yaml:"output_dir"`
	VersionPkg  string                   `yaml:"version_pkg"`
	BuildAll    bool                     `yaml:"build_all"`
	Platforms   []string                 `yaml:"platforms"`
	Scripts     ScriptsConfig            `yaml:"scripts"`
	Commands    map[string]interface{}   `yaml:"commands"`
	Profiles    map[string]ProfileConfig `yaml:"profiles"`
	AutoInstall bool                     `yaml:"auto_install"`
	StripDebug  bool                     `yaml:"strip_debug"`
	ExactName   bool                     `yaml:"exact_name"`
	Trimpath    bool                     `yaml:"trimpath"`
	CgoEnabled  bool                     `yaml:"cgo_enabled"`
	Race        bool                     `yaml:"race"`
	Tags        []string                 `yaml:"tags"`
	Dev         DevConfig                `yaml:"dev"`
	Minify      MinifyConfig             `yaml:"minify"`
}

// Global AppConfig
var AppConfig Config

func ConfigExists() bool {
	_, err := os.Stat(ConfigFileName)
	return !os.IsNotExist(err)
}

func ConfigLoad() {
	AppConfig = Config{
		Name:        "craft-app",
		Version:     "1.0.0",
		Toolchain:   "",
		EntryPoint:  ".",
		OutputDir:   DefaultDistDir,
		StripDebug:  true,
		AutoInstall: true,
		Trimpath:    true,
		CgoEnabled:  false,
		Dev: DevConfig{
			Watch: WatchConfig{
				DelayMs:      500,
				IncludeExts:  []string{"go", "html", "tpl", "env", "yaml"},
				ExcludeDirs:  []string{"bin", "tmp", "vendor", "node_modules", ".git", "assets", "testdata"},
				ExcludeFiles: []string{ConfigFileName},
			},
		},
		Minify: MinifyConfig{
			Enabled:    false,
			Extensions: []string{".html", ".css", ".js", ".json", ".svg"},
			Dirs:       []string{},
		},
	}

	if ConfigExists() {
		data, err := os.ReadFile(ConfigFileName)
		if err == nil {
			_ = yaml.Unmarshal(data, &AppConfig)
		}
	}
}

// ConfigSave marshals AppConfig back to .craft.yaml.
// WARNING: This strips all comments and reformats the file.
func ConfigSave() error {
	data, err := yaml.Marshal(&AppConfig)
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFileName, data, 0644)
}

// UpdateToolchainInConfig updates the toolchain key in .craft.yaml while preserving comments.
func UpdateToolchainInConfig(version string) error {
	data, err := os.ReadFile(ConfigFileName)
	if err != nil {
		if os.IsNotExist(err) {
			AppConfig.Toolchain = version
			return ConfigSave()
		}
		return err
	}

	var yamlRoot yaml.Node
	if err := yaml.Unmarshal(data, &yamlRoot); err != nil {
		return err
	}

	if len(yamlRoot.Content) == 0 || yamlRoot.Content[0].Kind != yaml.MappingNode {
		return ConfigSave() // Fallback if YAML is malformed
	}

	mapping := yamlRoot.Content[0]
	found := false

	for i := 0; i < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == "toolchain" {
			mapping.Content[i+1].Value = version
			found = true
			break
		}
	}

	if !found {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "toolchain"}
		valNode := &yaml.Node{Kind: yaml.ScalarNode, Value: version}
		
		newContent := make([]*yaml.Node, 0, len(mapping.Content)+2)
		inserted := false
		
		for i := 0; i < len(mapping.Content); i += 2 {
			if mapping.Content[i].Value == "entry_point" || mapping.Content[i].Value == "output_dir" {
				newContent = append(newContent, keyNode, valNode)
				inserted = true
			}
			newContent = append(newContent, mapping.Content[i], mapping.Content[i+1])
		}
		
		if !inserted {
			newContent = append(newContent, keyNode, valNode)
		}
		
		mapping.Content = newContent
	}

	out, err := yaml.Marshal(&yamlRoot)
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigFileName, out, 0644)
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
