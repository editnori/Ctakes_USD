package dictionary

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BuildService provides a unified interface for building dictionaries
type BuildService struct {
	progressCallback func(stage, message string, progress float64)
	onComplete       func(error)
}

// NewBuildService creates a new build service
func NewBuildService(progressCallback func(stage, message string, progress float64), onComplete func(error)) *BuildService {
	return &BuildService{
		progressCallback: progressCallback,
		onComplete:       onComplete,
	}
}

// BuildDictionaryAsync builds a dictionary asynchronously
func (bs *BuildService) BuildDictionaryAsync(config *Config, umlsPath, outputPath string) {
	go func() {
		err := bs.buildDictionary(config, umlsPath, outputPath)
		if bs.onComplete != nil {
			bs.onComplete(err)
		}
	}()
}

// buildDictionary performs the actual dictionary building
func (bs *BuildService) buildDictionary(config *Config, umlsPath, outputPath string) error {
	buildStart := time.Now()

	// Create output directory
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Save configuration
	configPath := filepath.Join(outputPath, "config.json")
	if err := SaveConfig(config, configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Generate dictionary descriptor XML
	descriptorPath := filepath.Join(outputPath, "dictionary.xml")
	if err := bs.generateBSVDescriptor(config, descriptorPath); err != nil {
		return fmt.Errorf("failed to generate descriptor: %w", err)
	}

	// Generate pipeline file
	pipelinePath := filepath.Join(outputPath, "pipeline.piper")
	if err := bs.generatePipelineFile(config, pipelinePath); err != nil {
		return fmt.Errorf("failed to generate pipeline: %w", err)
	}

	// Build BSV dictionary with progress reporting
	stats, err := BuildBSVDictionaryWithProgress(config, umlsPath, outputPath, bs.progressCallback)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Update statistics and save final config
	stats.BuildTime = time.Since(buildStart)
	config.Statistics = stats
	if err := SaveConfig(config, configPath); err != nil {
		return fmt.Errorf("failed to save final config: %w", err)
	}

	return nil
}

// generateBSVDescriptor generates a cTAKES descriptor pointing to a BSV dictionary
func (bs *BuildService) generateBSVDescriptor(config *Config, path string) error {
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<lookupSpecification>
  <dictionaries>
    <dictionary>
      <name>%s</name>
      <implementationName>org.apache.ctakes.dictionary.lookup2.dictionary.BsvRareWordDictionary</implementationName>
      <properties>
        <property key="bsvPath" value="dictionaries/%s/terms.bsv"/>
        <property key="caseSensitive" value="%t"/>
      </properties>
    </dictionary>
  </dictionaries>
  <dictionaryConceptPairs>
    <dictionaryConceptPair>
      <name>%sPair</name>
      <dictionaryName>%s</dictionaryName>
    </dictionaryConceptPair>
  </dictionaryConceptPairs>
  <rareWordConsumer>
    <implementationName>org.apache.ctakes.dictionary.lookup2.consumer.DefaultTermConsumer</implementationName>
    <properties>
      <property key="codingScheme" value="custom"/>
    </properties>
  </rareWordConsumer>
</lookupSpecification>`,
		config.Name,
		config.Name,
		config.Filters.CaseSensitive,
		config.Name,
		config.Name,
	)
	return os.WriteFile(path, []byte(xml), 0644)
}

// generatePipelineFile generates a .piper file for the dictionary
func (bs *BuildService) generatePipelineFile(config *Config, path string) error {
	piper := fmt.Sprintf(`# Pipeline for %s dictionary
# Generated: %s
# Description: %s

# Load tokenizer
load DefaultTokenizerPipeline.piper

# Add POS tagger
add POSTagger

# Add chunker
load ChunkerSubPipe.piper

# Dictionary lookup with custom dictionary
add DefaultJCasTermAnnotator LookupXml=dictionaries/%s/dictionary.xml

# Add assertion detection
load AttributeCleartkSubPipe.piper

# Output
add pretty.plaintext.PrettyTextWriter
`,
		config.Name,
		time.Now().Format(time.RFC3339),
		config.Description,
		config.Name,
	)

	return os.WriteFile(path, []byte(piper), 0644)
}

// ValidateUMLSFiles validates that required RRF files exist in the given path
func ValidateUMLSFiles(umlsPath string) (map[string]bool, error) {
	requiredFiles := []string{
		"MRCONSO.RRF",
		"MRSTY.RRF",
	}

	optionalFiles := []string{
		"MRSAB.RRF",
		"MRMAP.RRF",
		"MRSMAP.RRF",
		"MRDEF.RRF",
		"MRRANK.RRF",
		"MRXW_ENG.RRF",
	}

	files := make(map[string]bool)

	// Check META subdirectory first
	metaPath := filepath.Join(umlsPath, "META")
	checkPath := umlsPath
	if _, err := os.Stat(metaPath); err == nil {
		checkPath = metaPath
	}

	// Check all files
	allFiles := append(requiredFiles, optionalFiles...)
	for _, file := range allFiles {
		filePath := filepath.Join(checkPath, file)
		if _, err := os.Stat(filePath); err == nil {
			files[file] = true
		}
	}

	// Verify we have minimum required files
	for _, required := range requiredFiles {
		if !files[required] {
			return files, fmt.Errorf("required file %s not found in %s", required, checkPath)
		}
	}

	return files, nil
}

// ListDictionaries lists existing built dictionaries
func ListDictionaries() ([]DictionaryInfo, error) {
	dictionariesPath := "dictionaries"

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dictionariesPath, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dictionariesPath)
	if err != nil {
		return nil, err
	}

	var dictionaries []DictionaryInfo
	for _, entry := range entries {
		if entry.IsDir() {
			configPath := filepath.Join(dictionariesPath, entry.Name(), "config.json")
			if cfg, err := LoadConfig(configPath); err == nil {
				info := DictionaryInfo{
					Name:        cfg.Name,
					Description: cfg.Description,
					Path:        filepath.Join(dictionariesPath, entry.Name()),
					CreatedAt:   cfg.CreatedAt,
					Statistics:  cfg.Statistics,
				}
				dictionaries = append(dictionaries, info)
			}
		}
	}

	return dictionaries, nil
}

// CreateDefaultConfig creates a configuration with sensible defaults
func CreateDefaultConfig(name, description string) *Config {
	return &Config{
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		Languages:   []string{"ENG"},
		TermTypes:   []string{"PT", "SY", "AB"},
		Filters: FilterConfig{
			MinTermLength:       2,
			MaxTermLength:       80,
			ExcludeSuppressible: true,
			ExcludeObsolete:     true,
			CaseSensitive:       false,
			UseNormalization:    true,
			UseMRRank:           true,
			Deduplicate:         true,
			StripPunctuation:    true,
			CollapseWhitespace:  true,
			ExcludeNumericOnly:  true,
			ExcludePunctOnly:    true,
			MinTokens:           1,
		},
		OutputFormat: "BSV",
		Outputs: Outputs{
			EmitDescriptor: true,
			EmitPipeline:   true,
			EmitManifest:   true,
			EmitTSV:        false,
			EmitJSONL:      false,
			BuildLucene:    false,
			BuildHSQLDB:    false,
			UseRareWords:   false,
		},
	}
}
