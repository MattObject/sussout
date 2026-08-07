package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/matt/sussout/internal/llm"
	"gopkg.in/yaml.v3"
)

type Preset struct {
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
	APIKey  string `yaml:"api_key,omitempty"`
}

type File struct {
	DefaultPreset string            `yaml:"default_preset"`
	Presets       map[string]Preset `yaml:"presets"`
}

type Config struct {
	DatabaseURL string
	LLM         llm.ClientConfig
}

var defaultPresets = map[string]Preset{
	"lmstudio": {BaseURL: "http://localhost:1234/v1"},
	"ollama":   {BaseURL: "http://localhost:11434/v1"},
}

func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".sussout.yaml"
	}
	return filepath.Join(home, ".sussout.yaml")
}

func LoadFile() (*File, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			f := &File{
				DefaultPreset: "lmstudio",
				Presets:       defaultPresets,
			}
			_ = SaveFile(f)
			return f, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if f.Presets == nil {
		f.Presets = defaultPresets
	}
	if f.DefaultPreset == "" {
		f.DefaultPreset = "lmstudio"
	}
	return &f, nil
}

func SaveFile(f *File) error {
	data, err := yaml.Marshal(f)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath()), 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	return os.WriteFile(configPath(), data, 0600)
}

func LoadPreset(filename *File, name string) (Preset, bool) {
	if name != "" {
		p, ok := filename.Presets[name]
		return p, ok
	}
	p, ok := filename.Presets[filename.DefaultPreset]
	return p, ok
}

func Load(presetName string) *Config {
	f, err := LoadFile()
	if err != nil {
		f = &File{DefaultPreset: "lmstudio", Presets: defaultPresets}
	}

	preset, ok := LoadPreset(f, presetName)
	if !ok {
		preset = Preset{BaseURL: "http://localhost:1234/v1"}
	}

	baseURL := preset.BaseURL
	if v := os.Getenv("LLM_STUDIO_URL"); v != "" {
		baseURL = v
	}
	model := preset.Model
	if v := os.Getenv("LLM_MODEL"); v != "" {
		model = v
	}
	apiKey := preset.APIKey
	if v := os.Getenv("LLM_API_KEY"); v != "" {
		apiKey = v
	}

	return &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		LLM: llm.ClientConfig{
			BaseURL: baseURL,
			Model:   model,
			APIKey:  apiKey,
		},
	}
}

func ConfigPath() string {
	return configPath()
}

func SaveCurrentPreset(baseURL, model, apiKey string) error {
	f, err := LoadFile()
	if err != nil {
		f = &File{DefaultPreset: "lmstudio", Presets: defaultPresets}
	}
	preset := f.Presets[f.DefaultPreset]
	preset.BaseURL = baseURL
	preset.Model = model
	if apiKey != "" {
		preset.APIKey = apiKey
	}
	f.Presets[f.DefaultPreset] = preset
	return SaveFile(f)
}
