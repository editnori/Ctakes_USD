package dashboard

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/ctakes-tui/ctakes-tui/internal/dictionary"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
	"os"
	"path/filepath"
	"time"
)

// PipelineInfo represents a saved pipeline configuration
type PipelineInfo struct {
	Name        string
	Description string
	Path        string
	CreatedAt   time.Time
	Components  int
	Status      string
}

// PipelineTemplate represents a pre-configured pipeline template
type PipelineTemplate struct {
	Name        string
	Description string
	Category    string
	Icon        string
	Config      PipelineConfig
}

// DictionaryInfo for pipeline integration
type DictionaryAvailable struct {
	Name      string
	Path      string
	TermCount int
	Type      string
}

// Pipeline execution state
type PipelineExecution struct {
	StartTime   time.Time
	EndTime     time.Time
	Status      string
	CurrentStep string
	TotalSteps  int
	CurrentDoc  int
	TotalDocs   int
	Logs        []string
	Errors      []string
	OutputPath  string
}

// Initialize default pipeline templates
func (m *Model) initPipelineTemplates() {
	m.pipelineTemplates = []PipelineTemplate{
		{
			Name:        "Clinical Notes Pipeline",
			Description: "Complete pipeline for processing clinical notes with all major components",
			Category:    "Clinical",
			Icon:        theme.GetSemanticIcon("special"),
			Config: PipelineConfig{
				Name:                      "Clinical Notes",
				Description:               "Full clinical NLP pipeline",
				TokenizationEnabled:       true,
				POSTaggingEnabled:         true,
				ChunkingEnabled:           true,
				DependencyParsingEnabled:  true,
				NEREnabled:                true,
				DictionaryLookupEnabled:   true,
				AssertionEnabled:          true,
				RelationExtractionEnabled: true,
				TemporalEnabled:           true,
				DrugNEREnabled:            true,
			},
		},
		{
			Name:        "Fast Entity Extraction",
			Description: "Lightweight pipeline focused on entity extraction with dictionary lookup",
			Category:    "Performance",
			Icon:        theme.GetSemanticIcon("active"),
			Config: PipelineConfig{
				Name:                    "Fast Extraction",
				Description:             "Quick entity extraction pipeline",
				TokenizationEnabled:     true,
				POSTaggingEnabled:       false,
				ChunkingEnabled:         true,
				NEREnabled:              true,
				DictionaryLookupEnabled: true,
				AssertionEnabled:        true,
			},
		},
		{
			Name:        "Drug Analysis Pipeline",
			Description: "Specialized pipeline for medication and drug interaction analysis",
			Category:    "Specialized",
			Icon:        theme.GetSemanticIcon("special"),
			Config: PipelineConfig{
				Name:                      "Drug Analysis",
				Description:               "Medication extraction and analysis",
				TokenizationEnabled:       true,
				POSTaggingEnabled:         true,
				ChunkingEnabled:           true,
				NEREnabled:                true,
				DictionaryLookupEnabled:   true,
				DrugNEREnabled:            true,
				SideEffectEnabled:         true,
				RelationExtractionEnabled: true,
			},
		},
		{
			Name:        "Temporal Analysis",
			Description: "Pipeline focused on temporal information extraction",
			Category:    "Specialized",
			Icon:        theme.GetSemanticIcon("info"),
			Config: PipelineConfig{
				Name:                     "Temporal Analysis",
				Description:              "Time-based information extraction",
				TokenizationEnabled:      true,
				POSTaggingEnabled:        true,
				DependencyParsingEnabled: true,
				NEREnabled:               true,
				TemporalEnabled:          true,
				AssertionEnabled:         true,
			},
		},
		{
			Name:        "Minimal Pipeline",
			Description: "Basic tokenization and sentence detection only",
			Category:    "Basic",
			Icon:        theme.GetSemanticIcon("default"),
			Config: PipelineConfig{
				Name:                "Minimal",
				Description:         "Basic text processing",
				TokenizationEnabled: true,
			},
		},
		{
			Name:        "Research Pipeline",
			Description: "Comprehensive pipeline with all analysis components for research",
			Category:    "Research",
			Icon:        theme.GetSemanticIcon("info"),
			Config: PipelineConfig{
				Name:                       "Research",
				Description:                "Complete research pipeline",
				TokenizationEnabled:        true,
				POSTaggingEnabled:          true,
				ChunkingEnabled:            true,
				DependencyParsingEnabled:   true,
				ConstituencyParsingEnabled: true,
				NEREnabled:                 true,
				DictionaryLookupEnabled:    true,
				AssertionEnabled:           true,
				RelationExtractionEnabled:  true,
				TemporalEnabled:            true,
				CoreferenceEnabled:         true,
				DrugNEREnabled:             true,
				SideEffectEnabled:          true,
				SmokingStatusEnabled:       true,
				TemplateFillingEnabled:     true,
			},
		},
	}

	// Append built dictionary pipelines (dictionaries/*/pipeline.piper)
	m.appendBuiltDictionaryPipelines()
	// Append cTAKES-provided pipelines discovered from resources
	m.appendCTAKESPipelines()
}

// appendBuiltDictionaryPipelines adds templates for each built dictionary pipeline
func (m *Model) appendBuiltDictionaryPipelines() {
	dictRoot := "dictionaries"
	entries, err := os.ReadDir(dictRoot)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		p := filepath.Join(dictRoot, name, "pipeline.piper")
		if _, err := os.Stat(p); err == nil {
			m.pipelineTemplates = append(m.pipelineTemplates, PipelineTemplate{
				Name:        "Dictionary: " + name,
				Description: "Pipeline generated for built dictionary " + name,
				Category:    "Built",
				Icon:        theme.GetSemanticIcon("special"),
				Config: PipelineConfig{
					Name:          name,
					Description:   "Dictionary pipeline for " + name,
					PiperFilePath: p,
				},
			})
		}
	}
}

// appendCTAKESPipelines discovers cTAKES .piper files and adds them as templates
func (m *Model) appendCTAKESPipelines() {
	integ, err := dictionary.NewCTAKESIntegration(nil)
	if err != nil {
		return
	}
	pf, err := integ.DiscoverPipelines()
	if err != nil {
		return
	}
	for _, p := range pf {
		m.pipelineTemplates = append(m.pipelineTemplates, PipelineTemplate{
			Name:        p.Name,
			Description: "cTAKES pipeline",
			Category:    p.Category,
			Icon:        theme.GetSemanticIcon("info"),
			Config: PipelineConfig{
				Name:          p.Name,
				Description:   p.Description,
				PiperFilePath: p.Path,
			},
		})
	}
}

// Get available dictionaries for pipeline configuration
func (m *Model) getAvailableDictionaries() []DictionaryAvailable {
	// Use cached dictionaries if available to avoid I/O during rendering
	if m.availableDictsCached {
		return m.availableDictionaries
	}

	infos, err := dictionary.ListDictionaries()
	if err != nil || len(infos) == 0 {
		m.availableDictionaries = []DictionaryAvailable{}
		m.availableDictsCached = true
		return m.availableDictionaries
	}
	out := make([]DictionaryAvailable, 0, len(infos))
	for _, d := range infos {
		termCount := 0
		if d.Statistics.TotalTerms > 0 {
			termCount = int(d.Statistics.TotalTerms)
		}
		typeStr := "BSV"
		// Heuristic: if lucene or hsqldb present, note them
		if _, err := os.Stat(filepath.Join(d.Path, "lucene")); err == nil {
			typeStr = "BSV+Lucene"
		}
		if _, err := os.Stat(filepath.Join(d.Path, "hsqldb")); err == nil {
			if typeStr == "BSV" {
				typeStr = "BSV+HSQLDB"
			} else {
				typeStr += "+HSQLDB"
			}
		}
		out = append(out, DictionaryAvailable{
			Name:      d.Name,
			Path:      d.Path,
			TermCount: termCount,
			Type:      typeStr,
		})
	}
	m.availableDictionaries = out
	m.availableDictsCached = true
	return out
}

// Initialize default pipeline configuration
func (m *Model) initPipelineConfig() {
	m.pipelineConfig = PipelineConfig{
		Name:        "",
		Description: "",

		// Set reasonable defaults
		TokenizationEnabled: true,
		TokenizationConfig: TokenizationConfig{
			SentenceModelPath:   "resources/models/sentence-detector.bin",
			TokenizerModelPath:  "resources/models/tokenizer.bin",
			KeepNewlines:        false,
			SplitHyphens:        false,
			MaxTokenLength:      100,
			MinTokenLength:      1,
			PreserveWhitespace:  false,
			HandleAbbreviations: true,
		},

		POSTaggingEnabled: true,
		POSTaggingConfig: POSTaggingConfig{
			ModelPath:          "resources/models/pos-tagger.bin",
			TagSet:             "Penn Treebank",
			UseContextualCues:  true,
			HandleUnknownWords: true,
			CaseSensitive:      false,
		},

		NEREnabled: true,
		NERConfig: NERConfig{
			EntityTypes:      []string{"Diseases/Disorders", "Signs/Symptoms", "Procedures"},
			UseContextWindow: true,
			WindowSize:       10,
			MinEntityLength:  2,
			MaxEntityLength:  50,
			CaseSensitive:    false,
		},

		DictionaryLookupEnabled: true,
		DictionaryLookupConfig: DictionaryLookupConfig{
			LookupAlgorithm:   "Exact Match",
			CaseSensitive:     false,
			MinMatchLength:    3,
			MaxPermutations:   5,
			ExcludeNumbers:    true,
			MaxLookupTextSize: 1024,
			UseLuceneIndex:    false,
			LuceneIndexPath:   "",
			UseHsqlDictionary: false,
			HsqlPath:          "",
		},

		OutputConfig: OutputConfiguration{Format: "XMI", OutputDirectory: ""},

		ResourceConfig: ResourceConfiguration{
			ResourcesDirectory:    "./apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0/resources",
			ModelsDirectory:       "./apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0/resources/models",
			DictionariesDirectory: "./dictionaries",
			DownloadMissing:       false,
			CacheDirectory:        "./cache",
		},

		RuntimeConfig: RuntimeConfiguration{
			MaxHeapSize:     2048,
			InitialHeapSize: 1024,
			ThreadPoolSize:  4,
			BatchSize:       10,
			MaxDocumentSize: 1024,
			TimeoutSeconds:  300,
			EnableProfiling: false,
			LogLevel:        "INFO",
		},

		// IO defaults
		PiperFilePath:     "",
		InputDir:          "",
		OutputDir:         "",
		RunName:           "",
		IsTemplateApplied: false,
	}

	if m.pipelineSelectedInputDirs == nil {
		m.pipelineSelectedInputDirs = make(map[string]bool)
	}
	// Initialize pipeline run name input
	ti := textinput.New()
	ti.Placeholder = "Run name"
	ti.CharLimit = 64
	ti.Width = 40
	m.pipelineNameInput = ti
}

// Discover Piper files via internal/dictionary integration
func (m *Model) discoverPiperFiles() {
	// Deprecated in simplified UX: templates are the primary choice.
	m.piperFiles = nil
	m.piperCursor = 0
}
