package pipeline

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ctakes-tui/ctakes-tui/internal/dictionary"
	"github.com/ctakes-tui/ctakes-tui/internal/utils"
)

// PipelineRunner manages cTAKES pipeline execution
type PipelineRunner struct {
	ctakesPath string
	workDir    string
	logger     *dictionary.BuildLogger
	ctx        context.Context
	cancel     context.CancelFunc
}

// PipelineConfig represents complete pipeline configuration
type PipelineConfig struct {
	Name        string
	Description string

	// Core NLP Components
	TokenizationEnabled        bool
	POSTaggingEnabled          bool
	ChunkingEnabled            bool
	DependencyParsingEnabled   bool
	ConstituencyParsingEnabled bool
	NEREnabled                 bool
	DictionaryLookupEnabled    bool
	AssertionEnabled           bool
	RelationExtractionEnabled  bool
	TemporalEnabled            bool
	CoreferenceEnabled         bool

	// Specialized Components
	DrugNEREnabled         bool
	SideEffectEnabled      bool
	SmokingStatusEnabled   bool
	TemplateFillingEnabled bool

	// Component Configurations
	TokenizationConfig        TokenizationConfig
	POSTaggingConfig          POSTaggingConfig
	ChunkingConfig            ChunkingConfig
	DependencyParsingConfig   DependencyParsingConfig
	ConstituencyParsingConfig ConstituencyParsingConfig
	NERConfig                 NERConfig
	DictionaryLookupConfig    DictionaryLookupConfig
	AssertionConfig           AssertionConfig
	RelationExtractionConfig  RelationExtractionConfig
	TemporalConfig            TemporalConfig
	CoreferenceConfig         CoreferenceConfig
	DrugNERConfig             DrugNERConfig
	SideEffectConfig          SideEffectConfig
	SmokingStatusConfig       SmokingStatusConfig
	TemplateFillingConfig     TemplateFillingConfig

	// Output Configuration
	OutputConfig OutputConfiguration

	// Resource Configuration
	ResourceConfig ResourceConfiguration

	// Runtime Configuration
	RuntimeConfig RuntimeConfiguration

	// IO Configuration
	PiperFilePath          string
	InputDir               string
	InputDirs              []string
	OutputDir              string
	RunName                string
	MirrorOutputStructure  bool
	IsTemplateApplied      bool
	SelectedDictionaryName string
	SelectedDictionaryPath string
}

// Component-specific configurations
type TokenizationConfig struct {
	SentenceModelPath   string
	TokenizerModelPath  string
	KeepNewlines        bool
	SplitHyphens        bool
	MaxTokenLength      int
	MinTokenLength      int
	PreserveWhitespace  bool
	HandleAbbreviations bool
}

type POSTaggingConfig struct {
	ModelPath          string
	TagSet             string
	UseContextualCues  bool
	HandleUnknownWords bool
	CaseSensitive      bool
}

type ChunkingConfig struct {
	UseShallowParsing bool
	MaxChunkLength    int
	CombineAdjacent   bool
}

type DependencyParsingConfig struct {
	UseUniversalDeps   bool
	IncludePunctuation bool
	MaxSentenceLength  int
}

type ConstituencyParsingConfig struct {
	MaxParseDepth  int
	BeamSize       int
	UseBinaryTrees bool
}

type NERConfig struct {
	EntityTypes      []string
	UseContextWindow bool
	WindowSize       int
	MinEntityLength  int
	MaxEntityLength  int
	CaseSensitive    bool
}

type DictionaryLookupConfig struct {
	DictionaryPath    string
	LookupAlgorithm   string
	CaseSensitive     bool
	MinMatchLength    int
	MaxPermutations   int
	ExcludeNumbers    bool
	MaxLookupTextSize int
	UseLuceneIndex    bool
	LuceneIndexPath   string
	UseHsqlDictionary bool
	HsqlPath          string
}

type AssertionConfig struct {
	PolarityModelPath    string
	UncertaintyModelPath string
	SubjectModelPath     string
	GenericModelPath     string
	HistoryModelPath     string
	ConditionalModelPath string
	ScopeWindowSize      int
	UseSectionHeaders    bool
}

type RelationExtractionConfig struct {
	IncludeNegatives bool
}

type TemporalConfig struct {
	IncludeTimex     bool
	IncludeEvents    bool
	IncludeRelations bool
}

type CoreferenceConfig struct {
	UseSemanticInfo bool
}

type DrugNERConfig struct {
	IncludeDosage bool
	IncludeRoute  bool
}

type SideEffectConfig struct {
	IncludeSeverity bool
}

type SmokingStatusConfig struct {
	IncludeAmount bool
}

type TemplateFillingConfig struct {
	UseConstraints bool
}

type OutputConfiguration struct {
	Format              string
	OutputDirectory     string
	IncludeMetadata     bool
	PrettyPrint         bool
	CompressOutput      bool
	SplitBySection      bool
	IncludeOriginalText bool
}

type ResourceConfiguration struct {
	ResourcesDirectory    string
	ModelsDirectory       string
	DictionariesDirectory string
	DownloadMissing       bool
	CacheDirectory        string
}

type RuntimeConfiguration struct {
	InitialHeapSize int
	MaxHeapSize     int
	ThreadPoolSize  int
	BatchSize       int
	MaxDocumentSize int
	TimeoutSeconds  int
	EnableProfiling bool
	LogLevel        string
}

// RunResult contains pipeline execution results
type RunResult struct {
	Success       bool
	ProcessedDocs int
	TotalDocs     int
	ElapsedTime   time.Duration
	OutputPath    string
	Errors        []string
	Warnings      []string
}

// ProgressCallback is called during pipeline execution
type ProgressCallback func(stage, message string, progress float64)

// NewPipelineRunner creates a new pipeline runner
func NewPipelineRunner(ctakesPath string, workDir string, logger *dictionary.BuildLogger) *PipelineRunner {
	ctx, cancel := context.WithCancel(context.Background())

	return &PipelineRunner{
		ctakesPath: ctakesPath,
		workDir:    workDir,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// RunPipeline executes the pipeline with given configuration
func (r *PipelineRunner) RunPipeline(config PipelineConfig, callback ProgressCallback) (*RunResult, error) {
	startTime := time.Now()

	if callback != nil {
		callback("initializing", "Starting pipeline execution", 0.0)
	}

	// Validate configuration
	if err := r.validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Prepare input and output directories
	inputDirs := config.InputDirs
	if len(inputDirs) == 0 && config.InputDir != "" {
		inputDirs = []string{config.InputDir}
	}

	if len(inputDirs) == 0 {
		return nil, fmt.Errorf("no input directories specified")
	}

	// Count total documents
	totalDocs := r.countDocuments(inputDirs)
	if totalDocs == 0 {
		return nil, fmt.Errorf("no documents found in input directories")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	result := &RunResult{
		TotalDocs:  totalDocs,
		OutputPath: config.OutputDir,
		Errors:     []string{},
		Warnings:   []string{},
	}

	// Process each input directory
	processedDocs := 0
	for i, inputDir := range inputDirs {
		if callback != nil {
			stage := fmt.Sprintf("processing_%d", i+1)
			message := fmt.Sprintf("Processing directory %s", filepath.Base(inputDir))
			progress := float64(i) / float64(len(inputDirs))
			callback(stage, message, progress)
		}

		outputDir := config.OutputDir
		if config.MirrorOutputStructure {
			outputDir = filepath.Join(config.OutputDir, filepath.Base(inputDir))
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to create output directory %s: %v", outputDir, err))
				continue
			}
		}

		// Run cTAKES on this directory
		docs, err := r.runCTAKESOnDirectory(inputDir, outputDir, &config, callback)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Processing failed for %s: %v", inputDir, err))
			continue
		}

		processedDocs += docs
	}

	result.ProcessedDocs = processedDocs
	result.ElapsedTime = time.Since(startTime)
	result.Success = len(result.Errors) == 0

	if callback != nil {
		callback("complete", "Pipeline execution completed", 1.0)
	}

	return result, nil
}

// validateConfig validates the pipeline configuration
func (r *PipelineRunner) validateConfig(config *PipelineConfig) error {
	if config.OutputDir == "" {
		return fmt.Errorf("output directory not specified")
	}

	// Check if cTAKES installation exists
	if r.ctakesPath != "" {
		if _, err := os.Stat(r.ctakesPath); err != nil {
			return fmt.Errorf("cTAKES installation not found at %s", r.ctakesPath)
		}
	}

	// Validate piper file if specified
	if config.PiperFilePath != "" {
		if _, err := os.Stat(config.PiperFilePath); err != nil {
			return fmt.Errorf("piper file not found: %s", config.PiperFilePath)
		}
	}

	// Validate dictionary if dictionary lookup is enabled
	if config.DictionaryLookupEnabled {
		if config.DictionaryLookupConfig.DictionaryPath == "" && config.SelectedDictionaryPath == "" {
			return fmt.Errorf("dictionary lookup enabled but no dictionary specified")
		}
	}

	return nil
}

// countDocuments counts total documents in input directories
func (r *PipelineRunner) countDocuments(inputDirs []string) int {
	total := 0
	for _, dir := range inputDirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				// Count text files
				ext := strings.ToLower(filepath.Ext(path))
				if ext == ".txt" || ext == ".text" || ext == "" {
					total++
				}
			}
			return nil
		})
	}
	return total
}

// runCTAKESOnDirectory runs cTAKES on a specific directory
func (r *PipelineRunner) runCTAKESOnDirectory(inputDir, outputDir string, config *PipelineConfig, callback ProgressCallback) (int, error) {
	// Try to use cTAKES integration
	integ, err := dictionary.NewCTAKESIntegration(r.logger)
	if err != nil {
		return 0, fmt.Errorf("failed to create cTAKES integration: %w", err)
	}

	// Build run configuration
	runConfig := dictionary.PipelineRunConfig{
		PiperFile:   config.PiperFilePath,
		InputDir:    inputDir,
		OutputDir:   outputDir,
		InitialHeap: fmt.Sprintf("%dM", utils.Max(512, config.RuntimeConfig.InitialHeapSize)),
		MaxHeap:     fmt.Sprintf("%dM", utils.Max(1024, config.RuntimeConfig.MaxHeapSize)),
	}

	// Add dictionary lookup if configured
	if config.DictionaryLookupEnabled {
		if config.SelectedDictionaryPath != "" {
			runConfig.LookupXml = config.SelectedDictionaryPath
		} else if config.DictionaryLookupConfig.DictionaryPath != "" {
			dictPath := config.DictionaryLookupConfig.DictionaryPath
			if !strings.HasSuffix(strings.ToLower(dictPath), ".xml") {
				dictPath = filepath.Join(dictPath, "dictionary.xml")
			}
			if _, err := os.Stat(dictPath); err == nil {
				runConfig.LookupXml = dictPath
			}
		}
	}

	// Use default pipeline if none specified
	if runConfig.PiperFile == "" {
		runConfig.PiperFile = filepath.Join("apache-ctakes-6.0.0-bin", "apache-ctakes-6.0.0", "resources", "org", "apache", "ctakes", "clinical", "pipeline", "DefaultFastPipeline.piper")
	}

	// Execute the pipeline
	progressCallback := func(stage, message string, progress float64) {
		if r.logger != nil {
			if progress >= 0 {
				r.logger.Info(message, progress)
			} else {
				r.logger.Info(message, -1)
			}
		}
		if callback != nil {
			callback(stage, message, progress)
		}
	}

	err = integ.RunPiper(runConfig, progressCallback)
	if err != nil {
		return 0, fmt.Errorf("pipeline execution failed: %w", err)
	}

	// Count processed documents (rough estimate)
	return r.countDocuments([]string{inputDir}), nil
}

// Stop stops the pipeline execution
func (r *PipelineRunner) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
}

// IsRunning returns whether the pipeline is currently running
func (r *PipelineRunner) IsRunning() bool {
	return r.ctx.Err() == nil
}

// SetLogger sets the logger for the runner
func (r *PipelineRunner) SetLogger(logger *dictionary.BuildLogger) {
	r.logger = logger
}

// generatePiperFile generates a custom piper file based on configuration
func (r *PipelineRunner) generatePiperFile(config *PipelineConfig) (string, error) {
	// Create a temporary piper file based on configuration
	piperContent := r.buildPiperContent(config)

	piperPath := filepath.Join(r.workDir, "custom_pipeline.piper")
	err := os.WriteFile(piperPath, []byte(piperContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write piper file: %w", err)
	}

	return piperPath, nil
}

// buildPiperContent builds piper file content based on configuration
func (r *PipelineRunner) buildPiperContent(config *PipelineConfig) string {
	var components []string

	// Add basic sentence detection and tokenization
	if config.TokenizationEnabled {
		components = append(components,
			"load org.apache.ctakes.core.pipeline.PipeBitInfo",
			"add SegmentsAndTokensFromSentencesPipeBit",
		)
	}

	// Add POS tagging
	if config.POSTaggingEnabled {
		components = append(components, "add POSTagger")
	}

	// Add chunking
	if config.ChunkingEnabled {
		components = append(components, "add Chunker")
	}

	// Add dependency parsing
	if config.DependencyParsingEnabled {
		components = append(components, "add org.apache.ctakes.dependency.parser.ae.ClearNLPDependencyParserAE")
	}

	// Add constituency parsing
	if config.ConstituencyParsingEnabled {
		components = append(components, "add org.apache.ctakes.constituency.parser.ae.ConstituencyParser")
	}

	// Add NER
	if config.NEREnabled {
		components = append(components,
			"add ContextDependentTokenizerAnnotator",
			"add org.apache.ctakes.clinicalpipeline.ClinicalPipelineFactory.getTokenProcessingPipeline",
		)
	}

	// Add dictionary lookup
	if config.DictionaryLookupEnabled {
		components = append(components, "add DictionaryLookupAnnotatorUMLS")
	}

	// Add assertion
	if config.AssertionEnabled {
		components = append(components,
			"add PolarityCleartkAnalysisEngine",
			"add UncertaintyCleartkAnalysisEngine",
			"add HistoryCleartkAnalysisEngine",
			"add ConditionalCleartkAnalysisEngine",
			"add GenericCleartkAnalysisEngine",
			"add SubjectCleartkAnalysisEngine",
		)
	}

	// Add relation extraction
	if config.RelationExtractionEnabled {
		components = append(components, "add RelationExtractorAnnotator")
	}

	// Add temporal
	if config.TemporalEnabled {
		components = append(components,
			"add BackwardsTimeAnnotator",
			"add EventAnnotator",
			"add DocTimeRelAnnotator",
		)
	}

	// Add coreference
	if config.CoreferenceEnabled {
		components = append(components, "add MentionClusterCoreferenceAnnotator")
	}

	// Add drug NER
	if config.DrugNEREnabled {
		components = append(components, "add DrugMentionAnnotator")
	}

	// Add side effect
	if config.SideEffectEnabled {
		components = append(components, "add SideEffectAnnotator")
	}

	// Add smoking status
	if config.SmokingStatusEnabled {
		components = append(components, "add SmokingStatusAnnotator")
	}

	// Add template filling
	if config.TemplateFillingEnabled {
		components = append(components, "add TemplateFillerAnnotator")
	}

	// Add output writer based on format
	switch config.OutputConfig.Format {
	case "XMI":
		components = append(components, "addLast FileWriterCasConsumer")
	case "JSON":
		components = append(components, "addLast org.apache.ctakes.core.cc.JsonCasConsumer")
	case "FHIR":
		components = append(components, "addLast org.apache.ctakes.fhir.cc.FhirJsonFileWriter")
	default:
		components = append(components, "addLast FileWriterCasConsumer")
	}

	return strings.Join(components, "\n") + "\n"
}

// GetDefaultPiperFile returns the path to a default piper file
func (r *PipelineRunner) GetDefaultPiperFile() string {
	candidates := []string{
		filepath.Join("apache-ctakes-6.0.0-bin", "apache-ctakes-6.0.0", "resources", "org", "apache", "ctakes", "clinical", "pipeline", "DefaultFastPipeline.piper"),
		filepath.Join("dictionaries", "Diagnoses", "pipeline.piper"),
		filepath.Join("dictionaries", "Laboratory", "pipeline.piper"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

// runCommand executes a system command with logging
func (r *PipelineRunner) runCommand(name string, args ...string) error {
	if r.logger != nil {
		r.logger.Debug("Executing command", map[string]interface{}{
			"command": name,
			"args":    args,
		})
	}

	cmd := exec.CommandContext(r.ctx, name, args...)
	cmd.Dir = r.workDir

	// Capture both stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = io.MultiWriter(&stdout, os.Stdout)
	cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)

	err := cmd.Run()

	if r.logger != nil {
		if err != nil {
			r.logger.Error("Command failed", fmt.Errorf("exit code: %v, stderr: %s", err, stderr.String()))
		} else {
			r.logger.Debug("Command completed successfully", map[string]interface{}{
				"stdout": stdout.String(),
			})
		}
	}

	return err
}
