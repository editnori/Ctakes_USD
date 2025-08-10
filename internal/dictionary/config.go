package dictionary

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config represents a dictionary configuration
type Config struct {
	Name          string             `json:"name"`
	Description   string             `json:"description"`
	SemanticTypes []string           `json:"semantic_types"`
	Vocabularies  []string           `json:"vocabularies"`
	Languages     []string           `json:"languages"`
	TermTypes     []string           `json:"term_types"`
	Relationships RelationshipConfig `json:"relationships"`
	Filters       FilterConfig       `json:"filters"`
	OutputFormat  string             `json:"output_format"`
	CreatedAt     time.Time          `json:"created_at"`
	UMLSVersion   string             `json:"umls_version"`
	BuildTime     time.Duration      `json:"build_time"`
	Statistics    DictionaryStats    `json:"statistics"`
	Outputs       Outputs            `json:"outputs"`
	Memory        MemoryConfig       `json:"memory"`
	Processing    ProcessingConfig   `json:"processing"`
}

// RelationshipConfig defines relationship settings
type RelationshipConfig struct {
	Enabled bool     `json:"enabled"`
	Types   []string `json:"types"`
	Depth   int      `json:"depth"`
}

// MemoryConfig defines memory allocation settings
type MemoryConfig struct {
	InitialHeapMB int `json:"initial_heap_mb"` // -Xms setting (512-3072 MB)
	MaxHeapMB     int `json:"max_heap_mb"`     // -Xmx setting (512-3072 MB)
	StackSizeMB   int `json:"stack_size_mb"`   // -Xss setting (1-64 MB)
}

// ProcessingConfig defines text processing options
type ProcessingConfig struct {
	ThreadCount       int    `json:"thread_count"`       // Number of processing threads (1-16)
	BatchSize         int    `json:"batch_size"`         // Processing batch size (100-10000)
	CacheSize         int    `json:"cache_size"`         // Cache size in MB (64-512)
	TempDirectory     string `json:"temp_directory"`     // Temporary files location
	PreserveCase      bool   `json:"preserve_case"`      // Preserve original case
	HandlePunctuation bool   `json:"handle_punctuation"` // Process punctuation
	MinWordLength     int    `json:"min_word_length"`    // Minimum word length (1-10)
	MaxWordLength     int    `json:"max_word_length"`    // Maximum word length (10-256)
}

// FilterConfig defines filtering options
type FilterConfig struct {
	MinTermLength       int      `json:"min_term_length"`
	MaxTermLength       int      `json:"max_term_length"`
	ExcludeSuppressible bool     `json:"exclude_suppressible"`
	ExcludeObsolete     bool     `json:"exclude_obsolete"`
	CaseSensitive       bool     `json:"case_sensitive"`
	UseNormalization    bool     `json:"use_normalization"`
	UseMRRank           bool     `json:"use_mrrank"`
	Deduplicate         bool     `json:"deduplicate"`
	PreferredOnly       bool     `json:"preferred_only"`
	StripPunctuation    bool     `json:"strip_punctuation"`
	CollapseWhitespace  bool     `json:"collapse_whitespace"`
	ExcludeNumericOnly  bool     `json:"exclude_numeric_only"`
	ExcludePunctOnly    bool     `json:"exclude_punct_only"`
	MinTokens           int      `json:"min_tokens"`
	MaxTokens           int      `json:"max_tokens"`
	SABPriority         []string `json:"sab_priority"`
	TTYPriority         []string `json:"tty_priority"`
	BlacklistTerms      []string `json:"blacklist_terms"`
	WhitelistTerms      []string `json:"whitelist_terms"`
	BlacklistRegex      []string `json:"blacklist_regex"`
}

// Outputs controls which auxiliary files to emit
type Outputs struct {
	EmitDescriptor bool   `json:"emit_descriptor"`
	EmitPipeline   bool   `json:"emit_pipeline"`
	EmitManifest   bool   `json:"emit_manifest"`
	EmitTSV        bool   `json:"emit_tsv"`
	EmitJSONL      bool   `json:"emit_jsonl"`
	BuildLucene    bool   `json:"build_lucene"`
	BuildHSQLDB    bool   `json:"build_hsqldb"`
	UseRareWords   bool   `json:"use_rare_words"`
	LuceneVersion  string `json:"lucene_version,omitempty"`
}

// DictionaryStats holds statistics about the built dictionary
type DictionaryStats struct {
	TotalConcepts int    `json:"total_concepts"`
	TotalTerms    int    `json:"total_terms"`
	IndexSizeMB   int64  `json:"index_size_mb"`
	BuildDate     string `json:"build_date"`
	// Optional: build duration recorded by builder
	BuildTime time.Duration `json:"build_time"`
}

// SaveConfig saves the configuration to a JSON file
func SaveConfig(config *Config, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}

	return nil
}

// LoadConfig loads a configuration from a JSON file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return &config, nil
}

// GetPresetConfigs returns preset dictionary configurations
func GetPresetConfigs() map[string]*Config {
	return map[string]*Config{
		"clinical": {
			Name:          "clinical_dictionary",
			Description:   "General clinical text processing dictionary",
			SemanticTypes: GetClinicalTUIs(),
			Vocabularies:  GetClinicalVocabularies(),
			Languages:     []string{"ENG"},
			TermTypes:     []string{"PT", "SY", "AB", "ACR"},
			Relationships: RelationshipConfig{
				Enabled: true,
				Types:   []string{"PAR", "CHD", "RB", "RN"},
				Depth:   2,
			},
			Filters: FilterConfig{
				MinTermLength:       3,
				MaxTermLength:       80,
				ExcludeSuppressible: true,
				ExcludeObsolete:     true,
				CaseSensitive:       false,
				UseNormalization:    true,
				UseMRRank:           true,
			},
			OutputFormat: "BSV",
			Memory: MemoryConfig{
				InitialHeapMB: 1024,
				MaxHeapMB:     2048,
				StackSizeMB:   8,
			},
			Processing: ProcessingConfig{
				ThreadCount:       4,
				BatchSize:         1000,
				CacheSize:         128,
				PreserveCase:      false,
				HandlePunctuation: true,
				MinWordLength:     2,
				MaxWordLength:     80,
			},
			Outputs: Outputs{
				EmitDescriptor: true,
				EmitPipeline:   true,
				EmitManifest:   true,
				UseRareWords:   true,
				LuceneVersion:  "8.11.0",
			},
		},
		"medications": {
			Name:          "medications_dictionary",
			Description:   "Medication extraction and normalization dictionary",
			SemanticTypes: GetMedicationTUIs(),
			Vocabularies:  GetMedicationVocabularies(),
			Languages:     []string{"ENG"},
			TermTypes:     []string{"PT", "SY", "BN", "GN", "IN"},
			Relationships: RelationshipConfig{
				Enabled: true,
				Types:   []string{"PAR", "CHD", "SY"},
				Depth:   1,
			},
			Filters: FilterConfig{
				MinTermLength:       2,
				MaxTermLength:       100,
				ExcludeSuppressible: true,
				ExcludeObsolete:     true,
				CaseSensitive:       false,
			},
			OutputFormat: "BSV",
		},
		"radiology": {
			Name:          "radiology_dictionary",
			Description:   "Radiology report processing dictionary",
			SemanticTypes: GetRadiologyTUIs(),
			Vocabularies:  GetRadiologyVocabularies(),
			Languages:     []string{"ENG"},
			TermTypes:     []string{"PT", "SY", "AB"},
			Relationships: RelationshipConfig{
				Enabled: true,
				Types:   []string{"PAR", "CHD", "part_of"},
				Depth:   2,
			},
			Filters: FilterConfig{
				MinTermLength:       3,
				MaxTermLength:       80,
				ExcludeSuppressible: true,
				ExcludeObsolete:     true,
				CaseSensitive:       false,
			},
			OutputFormat: "BSV",
		},
		"minimal": {
			Name:          "minimal_dictionary",
			Description:   "Minimal dictionary for fast processing",
			SemanticTypes: GetMinimalTUIs(),
			Vocabularies:  []string{"SNOMEDCT_US"},
			Languages:     []string{"ENG"},
			TermTypes:     []string{"PT"},
			Relationships: RelationshipConfig{
				Enabled: false,
			},
			Filters: FilterConfig{
				MinTermLength:       3,
				MaxTermLength:       50,
				ExcludeSuppressible: true,
				ExcludeObsolete:     true,
				CaseSensitive:       false,
			},
			OutputFormat: "BSV",
		},
	}
}

// RelationshipType represents a UMLS relationship type
type RelationshipType struct {
	Code        string
	Name        string
	Description string
}

// GetRelationshipTypes returns common UMLS relationship types
func GetRelationshipTypes() []RelationshipType {
	return []RelationshipType{
		// Hierarchical
		{Code: "PAR", Name: "Parent", Description: "Has parent relationship"},
		{Code: "CHD", Name: "Child", Description: "Has child relationship"},
		{Code: "AQ", Name: "Allowed Qualifier", Description: "Allowed qualifier for"},
		{Code: "QB", Name: "Qualified By", Description: "Can be qualified by"},

		// Associative
		{Code: "RB", Name: "Broader", Description: "Has a broader relationship"},
		{Code: "RN", Name: "Narrower", Description: "Has a narrower relationship"},
		{Code: "RO", Name: "Other", Description: "Has other relationship"},
		{Code: "RL", Name: "Similar", Description: "Similar or alike"},
		{Code: "RQ", Name: "Related and Possibly Synonymous", Description: "Related, possibly synonymous"},
		{Code: "RU", Name: "Related Unspecified", Description: "Related, unspecified"},
		{Code: "SY", Name: "Synonym", Description: "Source asserted synonymy"},

		// Specific Relationships (RELA)
		{Code: "isa", Name: "Is A", Description: "Is a type of"},
		{Code: "inverse_isa", Name: "Inverse Is A", Description: "Has subtype"},
		{Code: "part_of", Name: "Part Of", Description: "Is part of"},
		{Code: "has_part", Name: "Has Part", Description: "Has as a part"},
		{Code: "member_of", Name: "Member Of", Description: "Is member of"},
		{Code: "has_member", Name: "Has Member", Description: "Has member"},
		{Code: "branch_of", Name: "Branch Of", Description: "Is branch of"},
		{Code: "has_branch", Name: "Has Branch", Description: "Has branch"},
		{Code: "tributary_of", Name: "Tributary Of", Description: "Is tributary of"},
		{Code: "has_tributary", Name: "Has Tributary", Description: "Has tributary"},

		// Clinical Relationships
		{Code: "may_treat", Name: "May Treat", Description: "May be used to treat"},
		{Code: "may_prevent", Name: "May Prevent", Description: "May prevent"},
		{Code: "may_diagnose", Name: "May Diagnose", Description: "May be used to diagnose"},
		{Code: "occurs_in", Name: "Occurs In", Description: "Occurs in"},
		{Code: "process_of", Name: "Process Of", Description: "Is a process of"},
		{Code: "causative_agent_of", Name: "Causative Agent Of", Description: "Is causative agent of"},
		{Code: "finding_site_of", Name: "Finding Site Of", Description: "Is finding site of"},
		{Code: "manifestation_of", Name: "Manifestation Of", Description: "Is manifestation of"},
		{Code: "associated_with", Name: "Associated With", Description: "Is associated with"},
	}
}

// GetCommonRelationships returns commonly used relationship types
func GetCommonRelationships() []string {
	return []string{"PAR", "CHD", "RB", "RN", "SY", "isa", "part_of"}
}

// DictionaryInfo holds information about a dictionary
type DictionaryInfo struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Path        string          `json:"path"`
	CreatedAt   time.Time       `json:"created_at"`
	Statistics  DictionaryStats `json:"statistics"`
}
