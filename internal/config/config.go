package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	CTakes    CTakesConfig     `json:"ctakes"`
	Pipelines []PipelineConfig `json:"pipelines"`
}

type CTakesConfig struct {
	JavaBin     string `json:"javaBin"`
	HeapSize    string `json:"heapSize"`
	WrapperJar  string `json:"wrapperJar"`
	Resources   string `json:"resources"`
	TimeoutSecs int    `json:"timeoutSecs"`
}

type PipelineConfig struct {
	Name       string   `json:"name"`
	Components []string `json:"components"`
	Default    bool     `json:"default"`
}

func LoadConfig(path string) (*Config, error) {
	if path == "" {
		path = getDefaultConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func getDefaultConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".ctakes-tui", "config.json")
}

func DefaultConfig() *Config {
	return &Config{
		CTakes: CTakesConfig{
			JavaBin:     "java",
			HeapSize:    "2G",
			WrapperJar:  "./java/ctakes-wrapper.jar",
			Resources:   "./resources",
			TimeoutSecs: 30,
		},
		Pipelines: []PipelineConfig{
			{
				Name: "default",
				Components: []string{
					"sentenceDetector",
					"tokenizer",
					"posTagger",
					"chunker",
					"dictionaryLookup",
					"assertion",
				},
				Default: true,
			},
			{
				Name: "minimal",
				Components: []string{
					"sentenceDetector",
					"tokenizer",
					"dictionaryLookup",
				},
			},
		},
	}
}
