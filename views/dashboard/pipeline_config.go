package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
	"github.com/ctakes-tui/ctakes-tui/internal/utils"
)

// Pipeline Configuration State
type PipelineState int

const (
	PipelineMainMenu PipelineState = iota
	PipelineSelectingTemplate
	PipelineTemplateEditor
	PipelineDictionaryConfig
	PipelineSelectingInputDirs
	PipelineSelectingOutputDir
	PipelineEditingRunName
	PipelineOutputConfig
	PipelineRuntimeConfig
	PipelineRunning
)

// PipelineConfig holds all pipeline configuration
type PipelineConfig struct {
	Name        string
	Description string

	// Core Components
	TokenizationEnabled        bool
	TokenizationConfig         TokenizationConfig
	POSTaggingEnabled          bool
	POSTaggingConfig           POSTaggingConfig
	ChunkingEnabled            bool
	ChunkingConfig             ChunkingConfig
	DependencyParsingEnabled   bool
	DependencyParsingConfig    DependencyParsingConfig
	ConstituencyParsingEnabled bool
	ConstituencyParsingConfig  ConstituencyParsingConfig

	// Clinical NLP
	NEREnabled              bool
	NERConfig               NERConfig
	DictionaryLookupEnabled bool
	DictionaryLookupConfig  DictionaryLookupConfig
	AssertionEnabled        bool
	AssertionConfig         AssertionConfig

	// Advanced
	RelationExtractionEnabled bool
	RelationExtractionConfig  RelationExtractionConfig
	TemporalEnabled           bool
	TemporalConfig            TemporalConfig
	CoreferenceEnabled        bool
	CoreferenceConfig         CoreferenceConfig

	// Specialized
	DrugNEREnabled         bool
	DrugNERConfig          DrugNERConfig
	SideEffectEnabled      bool
	SideEffectConfig       SideEffectConfig
	SmokingStatusEnabled   bool
	SmokingStatusConfig    SmokingStatusConfig
	TemplateFillingEnabled bool
	TemplateFillingConfig  TemplateFillingConfig

	// Output & Resources
	OutputConfig   OutputConfiguration
	ResourceConfig ResourceConfiguration
	RuntimeConfig  RuntimeConfiguration

	// Piper integration & IO
	PiperFilePath string
	InputDir      string
	OutputDir     string
	// Enhanced IO
	InputDirs             []string
	MirrorOutputStructure bool
	RunName               string

	// Flow state: whether a user explicitly applied a template
	IsTemplateApplied bool

	// Selected dictionary (descriptor) for pipeline runs
	SelectedDictionaryPath string
	SelectedDictionaryName string
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
	TagSet             string // Penn Treebank, Universal, etc.
	UseContextualCues  bool
	HandleUnknownWords bool
	CaseSensitive      bool
}

type ChunkingConfig struct {
	ModelPath         string
	ChunkTypes        []string // NP, VP, PP, etc.
	UseShallowParsing bool
	MaxChunkLength    int
	CombineAdjacent   bool
}

type DependencyParsingConfig struct {
	ModelPath          string
	ParserType         string // ClearNLP, Stanford, etc.
	UseUniversalDeps   bool
	IncludePunctuation bool
	MaxSentenceLength  int
}

type ConstituencyParsingConfig struct {
	ModelPath      string
	GrammarPath    string
	MaxParseDepth  int
	BeamSize       int
	UseBinaryTrees bool
}

type NERConfig struct {
	ModelPaths       map[string]string // entity type -> model path
	EntityTypes      []string
	DictionaryPaths  []string
	UseContextWindow bool
	WindowSize       int
	MinEntityLength  int
	MaxEntityLength  int
	CaseSensitive    bool
}

type DictionaryLookupConfig struct {
	DictionaryPath    string
	LookupAlgorithm   string // Exact, Fuzzy, Permutation
	CaseSensitive     bool
	MinMatchLength    int
	MaxPermutations   int
	ExcludeNumbers    bool
	WindowAnnotations []string
	MaxLookupTextSize int
	// Indexing options
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
	ModelPath            string
	RelationTypes        []string
	MaxEntityDistance    int
	UseDependencyPaths   bool
	UseConstituencyPaths bool
	IncludeNegatives     bool
}

type TemporalConfig struct {
	ModelPath          string
	DocumentTime       string
	IncludeTimex       bool
	IncludeEvents      bool
	IncludeRelations   bool
	NormalizationRules string
}

type CoreferenceConfig struct {
	ModelPath          string
	MaxMentionDistance int
	UseSemanticInfo    bool
	UseGenderInfo      bool
	UsePlurality       bool
}

type DrugNERConfig struct {
	ModelPath          string
	DrugDictionaryPath string
	IncludeDosage      bool
	IncludeRoute       bool
	IncludeFrequency   bool
	IncludeDuration    bool
	NormalizeToCUI     bool
}

type SideEffectConfig struct {
	ModelPath           string
	SideEffectLexicon   string
	IncludeSeverity     bool
	IncludeBodyLocation bool
	LinkToDrugs         bool
}

type SmokingStatusConfig struct {
	ModelPath           string
	ClassificationRules string
	IncludeAmount       bool
	IncludeDuration     bool
	IncludeQuitDate     bool
}

type TemplateFillingConfig struct {
	TemplatePath    string
	SlotTypes       []string
	UseConstraints  bool
	MaxSlotDistance int
}

type OutputConfiguration struct {
	Format              string // XMI, JSON, FHIR, TSV, etc.
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
	CustomResourcePaths   map[string]string
	DownloadMissing       bool
	CacheDirectory        string
}

type RuntimeConfiguration struct {
	MaxHeapSize     int // MB
	InitialHeapSize int // MB
	ThreadPoolSize  int
	BatchSize       int
	MaxDocumentSize int // KB
	TimeoutSeconds  int
	EnableProfiling bool
	LogLevel        string
}

// Render simplified pipeline menu - just select a pipeline and run it
func (m *Model) renderPipelineMenu(width, height int) string {
	// Simple header
	header := theme.RenderSelectionHeader("Pipeline Configuration", 0, 0, width)
	headerLines := strings.Count(header, "\n") + 1

	// Simple menu items - focused, minimal
	type menuItem struct {
		label string
		icon  string
		value string // For showing current value
	}

	// Current template name
	tmplName := m.pipelineConfig.Name
	if tmplName == "" {
		tmplName = "Not selected"
	}

	// Get input/output dirs
	inputDir := m.pipelineConfig.InputDir
	if inputDir == "" {
		inputDir = "Not set"
	}
	outputDir := m.pipelineConfig.OutputDir
	if outputDir == "" {
		outputDir = "Not set"
	}

	// Build input dirs summary
	inputSummary := inputDir
	if len(m.pipelineConfig.InputDirs) > 0 {
		if len(m.pipelineConfig.InputDirs) == 1 {
			inputSummary = utils.TruncateString(m.pipelineConfig.InputDirs[0], 40)
		} else {
			inputSummary = fmt.Sprintf("%d selected", len(m.pipelineConfig.InputDirs))
		}
	}

	items := []menuItem{
		{label: "Run Name", icon: theme.GetSemanticIcon("default"), value: utils.TruncateString(m.pipelineConfig.RunName, 30)},
		{label: "Choose Template", icon: theme.GetSemanticIcon("special"), value: tmplName},
		{label: "Edit Components", icon: theme.GetSemanticIcon("edit"), value: fmt.Sprintf("%d enabled", m.countEnabledComponents())},
		{label: "Dictionary", icon: theme.GetSemanticIcon("special"), value: utils.TruncateString(m.pipelineConfig.SelectedDictionaryName, 30)},
		{label: "Input Directories", icon: theme.GetSemanticIcon("folder"), value: inputSummary},
		{label: "Output Directory", icon: theme.GetSemanticIcon("folder"), value: outputDir},
		{label: "Mirror Output Structure", icon: theme.GetSemanticIcon("info"), value: map[bool]string{true: "Enabled", false: "Disabled"}[m.pipelineConfig.MirrorOutputStructure]},
		{label: "Output Settings", icon: theme.GetSemanticIcon("info"), value: m.pipelineConfig.OutputConfig.Format},
		{label: "Runtime Settings", icon: theme.GetSemanticIcon("info"), value: getRuntimeSummary(m)},
		{label: "Run", icon: theme.GetSemanticIcon("active"), value: ""},
	}

	// Calculate visible window
	footerLines := 2
	visibleHeight := height - headerLines - footerLines
	if visibleHeight < 5 {
		visibleHeight = 5
	}

	// Render items
	lines := []string{header}

	for i, item := range items {
		focused := i == m.pipelineMenuCursor

		// Circle indicator for focus
		indicator := "  "
		if focused {
			indicator = "🔵"
		}

		// Format the line
		var line string
		if item.value != "" {
			line = fmt.Sprintf("%s %s %-20s: %s", indicator, item.icon, item.label, item.value)
		} else {
			line = fmt.Sprintf("%s %s %s", indicator, item.icon, item.label)
		}

		// Use unified selection system for consistent highlighting
		renderedLine := theme.RenderSelectableRow(line, width, false, focused)
		lines = append(lines, renderedLine)
	}

	// Status summary
	hasInputs := len(m.pipelineConfig.InputDirs) > 0 || m.pipelineConfig.InputDir != ""
	hasTemplate := tmplName != "Not selected"
	ready := hasTemplate && hasInputs && m.pipelineConfig.OutputDir != ""
	statusLine := "Status: "
	if ready {
		statusLine += lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render("✓ Ready to run")
	} else {
		missing := []string{}
		if !hasTemplate {
			missing = append(missing, "template")
		}
		if !hasInputs {
			missing = append(missing, "input")
		}
		if m.pipelineConfig.OutputDir == "" {
			missing = append(missing, "output")
		}
		statusLine += lipgloss.NewStyle().Foreground(theme.ColorWarning).Render("missing " + strings.Join(missing, ", "))
	}
	lines = append(lines, "", statusLine)

	// Footer
	lines = append(lines, theme.RenderSelectionFooter(width, "[Enter] Select  [Esc] Back"))

	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// Render simplified Piper selector with preview
func (m *Model) renderPiperSelector(width, height int) string {
	// Split view: left for list, right for preview
	leftWidth := width / 2
	rightWidth := width - leftWidth - 1 // -1 for divider

	// Left panel - pipeline list
	header := theme.RenderSelectionHeader("Select Pipeline", m.piperCursor+1, len(m.piperFiles), leftWidth)
	headerLines := strings.Count(header, "\n") + 1

	// Calculate visible area for left panel
	footerLines := 2
	visibleHeight := height - headerLines - footerLines
	if visibleHeight < 5 {
		visibleHeight = 5
	}

	leftLines := []string{header}

	if len(m.piperFiles) == 0 {
		leftLines = append(leftLines, theme.WarningStyle.Render("No .piper files found"))
	} else {
		// Build all items first
		items := make([]string, 0, len(m.piperFiles))
		for i, pf := range m.piperFiles {
			focused := i == m.piperCursor

			indicator := "  "
			if focused {
				indicator = "🔵"
			}

			// Just show the name, no extra info
			line := fmt.Sprintf("%s %s", indicator, pf.Name)

			// Use unified selection system
			renderedLine := theme.RenderSelectableRow(line, leftWidth, false, focused)
			items = append(items, renderedLine)
		}

		// Apply scrolling window
		startIdx := 0
		if m.piperCursor >= visibleHeight-1 {
			// Keep cursor in middle of view when possible
			startIdx = m.piperCursor - (visibleHeight / 2)
			if startIdx < 0 {
				startIdx = 0
			}
		}

		// Ensure we don't go past the end
		if startIdx+visibleHeight > len(items) {
			startIdx = len(items) - visibleHeight
			if startIdx < 0 {
				startIdx = 0
			}
		}

		endIdx := startIdx + visibleHeight
		if endIdx > len(items) {
			endIdx = len(items)
		}

		// Add visible items
		leftLines = append(leftLines, items[startIdx:endIdx]...)

		// Add scroll indicator if needed
		if len(items) > visibleHeight {
			scrollIndicator := theme.RenderScrollIndicator(startIdx, endIdx, len(items), leftWidth)
			leftLines = append(leftLines, scrollIndicator)
		}
	}
	// Right panel - preview
	rightLines := []string{
		lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.ColorAccent).
			Render("Pipeline Details"),
		"",
	}

	if m.piperCursor < len(m.piperFiles) {
		pf := m.piperFiles[m.piperCursor]

		// Show pipeline info
		rightLines = append(rightLines,
			fmt.Sprintf("Name: %s", pf.Name),
			fmt.Sprintf("Category: %s", pf.Category),
			"",
			"Components:",
		)

		// Parse and show what the pipeline does (simplified)
		if strings.Contains(pf.Name, "Fast") {
			rightLines = append(rightLines,
				"  • Fast dictionary lookup",
				"  • Tokenization",
				"  • POS tagging",
				"  • Chunking",
				"  • Entity attributes",
			)
		} else if strings.Contains(pf.Name, "Default") {
			rightLines = append(rightLines,
				"  • Standard tokenization",
				"  • Full NLP pipeline",
				"  • Dictionary lookup",
				"  • Assertion analysis",
			)
		} else if strings.Contains(pf.Name, "Coref") {
			rightLines = append(rightLines,
				"  • Coreference resolution",
				"  • Relation extraction",
				"  • Advanced NLP",
			)
		} else if strings.Contains(pf.Name, "Temporal") {
			rightLines = append(rightLines,
				"  • Temporal extraction",
				"  • Event detection",
				"  • Time normalization",
			)
		}

		rightLines = append(rightLines,
			"",
			lipgloss.NewStyle().
				Foreground(theme.ColorForegroundDim).
				Render(fmt.Sprintf("Path: %s", utils.TruncateString(pf.Path, rightWidth-6))),
		)
	} else {
		rightLines = append(rightLines,
			lipgloss.NewStyle().
				Foreground(theme.ColorForegroundDim).
				Render("Select a pipeline to see details"))
	}

	// Combine panels
	maxLines := height - 2 // for footer
	combined := make([]string, 0, maxLines)

	// Pad both panels to same height
	for len(leftLines) < maxLines {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxLines {
		rightLines = append(rightLines, "")
	}

	// Combine side by side
	divider := lipgloss.NewStyle().Foreground(theme.ColorBorderInactive).Render("│")
	for i := 0; i < maxLines && i < len(leftLines) && i < len(rightLines); i++ {
		left := lipgloss.NewStyle().Width(leftWidth).Render(leftLines[i])
		right := lipgloss.NewStyle().Width(rightWidth).Render(rightLines[i])
		combined = append(combined, left+divider+right)
	}

	// Footer
	footer := theme.RenderSelectionFooter(width, "[Enter] Select  [Esc] Back")
	combined = append(combined, footer)

	return strings.Join(combined, "\n")
}

// Render input directory selector (directories only, multi-select with Space)
func (m *Model) renderInputDirSelector(width, height int) string {
	header := theme.RenderSelectionHeader("Select Input Directories (Space select, Enter open, S save, Esc cancel)", 0, 0, width)
	headerLines := strings.Count(header, "\n") + 1
	lines := []string{header}

	if m.currentPath == "" {
		m.currentPath = "."
	}

	// Ensure listing is loaded
	// The file table is already rendered by tables.go; just wrap
	pathInfo := lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).
		Render(fmt.Sprintf("Path: %s", utils.TruncateString(m.currentPath, width-8)))
	lines = append(lines, pathInfo, "")

	// Ensure file table has correct dimensions before rendering
	m.fileTable.SetWidth(width - 2)
	m.fileTable.SetHeight(height - headerLines - 4)
	tableView := m.fileTable.View()
	lines = append(lines, tableView)

	// Footer
	footer := theme.RenderSelectionFooter(width, "[Space] Select  [Enter] Open  [S] Save  [Esc] Cancel")
	lines = append(lines, footer)
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// Render output directory selector (directories only, single select)
func (m *Model) renderOutputDirSelector(width, height int) string {
	header := theme.RenderSelectionHeader("Select Output Directory (Enter open folder, S save highlighted/current, Esc cancel)", 0, 0, width)
	headerLines := strings.Count(header, "\n") + 1
	lines := []string{header}

	if m.currentPath == "" {
		m.currentPath = "."
	}
	// Show current selection
	current := m.pipelineConfig.OutputDir
	if current == "" {
		current = "(not set)"
	}
	info := fmt.Sprintf("Current: %s", utils.TruncateString(current, width-10))
	lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render(info), "")

	// Ensure file table has correct dimensions before rendering
	m.fileTable.SetWidth(width - 2)
	m.fileTable.SetHeight(height - headerLines - 4)
	tableView := m.fileTable.View()
	lines = append(lines, tableView)

	footer := theme.RenderSelectionFooter(width, "[Enter] Open  [S] Save selection  [Esc] Cancel")
	lines = append(lines, footer)
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// Render run name editor
func (m *Model) renderRunNameEditor(width, height int) string {
	header := theme.RenderSelectionHeader("Name This Run", 0, 0, width)
	lines := []string{header, ""}

	hint := lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render("Enter a name to label this pipeline run (e.g., 2025-08-09-ctakes-notes)")
	lines = append(lines, hint, "")

	// Render input field centered
	inputView := m.pipelineNameInput.View()
	lines = append(lines, inputView)

	footer := theme.RenderSelectionFooter(width, "[Enter] Save  [Esc] Cancel")
	lines = append(lines, "", footer)
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// collectSelectedInputDirs flattens selection map to slice
func (m *Model) collectSelectedInputDirs() []string {
	dirs := make([]string, 0, len(m.pipelineSelectedInputDirs))
	for d, sel := range m.pipelineSelectedInputDirs {
		if sel {
			dirs = append(dirs, filepath.Clean(d))
		}
	}
	return dirs
}

// Helper for menu status text
func (m *Model) getSelectedPiperStatus() string {
	// Deprecated: piper selection hidden in simplified UX; use template status instead
	if !m.pipelineConfig.IsTemplateApplied {
		return lipgloss.NewStyle().Foreground(theme.ColorWarning).Render("Template: Not selected")
	}
	return lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render("Template: Selected")
}

// Render tokenization configuration with clean circle-based design
func (m *Model) renderTokenizationConfig(width, height int) string {
	header := theme.RenderSelectionHeader("Tokenization Configuration", 0, 0, width)
	headerLines := strings.Count(header, "\n") + 1
	lines := []string{header}

	fields := []struct {
		name   string
		value  interface{}
		desc   string
		isBool bool
	}{
		{"Enabled", m.pipelineConfig.TokenizationEnabled, "Enable tokenization component", true},
		{"Sentence Model", m.pipelineConfig.TokenizationConfig.SentenceModelPath, "Path to sentence detection model", false},
		{"Tokenizer Model", m.pipelineConfig.TokenizationConfig.TokenizerModelPath, "Path to tokenizer model", false},
		{"Keep Newlines", m.pipelineConfig.TokenizationConfig.KeepNewlines, "Preserve newline characters", true},
		{"Split Hyphens", m.pipelineConfig.TokenizationConfig.SplitHyphens, "Split hyphenated words", true},
		{"Min Token Length", m.pipelineConfig.TokenizationConfig.MinTokenLength, "Minimum token length (1-10)", false},
		{"Max Token Length", m.pipelineConfig.TokenizationConfig.MaxTokenLength, "Maximum token length (10-100)", false},
		{"Preserve Whitespace", m.pipelineConfig.TokenizationConfig.PreserveWhitespace, "Keep original whitespace", true},
		{"Handle Abbreviations", m.pipelineConfig.TokenizationConfig.HandleAbbreviations, "Smart abbreviation handling", true},
	}

	for i, field := range fields {
		focused := i == m.configField

		// Circle indicator for boolean fields
		var indicator string
		if field.isBool {
			if boolVal, ok := field.value.(bool); ok {
				if focused {
					indicator = "🔵" // Blue circle when focused
				} else if boolVal {
					indicator = "🟢" // Green circle when enabled
				} else {
					indicator = "⚫" // Black circle when disabled
				}
			}
		} else {
			if focused {
				indicator = "🔵"
			} else {
				indicator = "  "
			}
		}

		// Format value
		valueStr := fmt.Sprintf("%v", field.value)
		if field.isBool {
			if boolVal, ok := field.value.(bool); ok {
				if boolVal {
					valueStr = "Enabled"
				} else {
					valueStr = "Disabled"
				}
			}
		}

		// Build the line
		line := fmt.Sprintf("%s %-25s: %-20s", indicator, field.name, valueStr)

		// Use unified selection system
		renderedLine := theme.RenderSelectableRow(line, width, false, focused)
		lines = append(lines, renderedLine)

		// Description (indented)
		desc := "    " + field.desc
		lines = append(lines,
			lipgloss.NewStyle().
				Foreground(theme.ColorForegroundDim).
				Render(desc))
	}

	footer := theme.RenderSelectionFooter(width, "[↑/↓] Select  [Space] Toggle  [←/→] Adjust  [Enter] Save  [Esc] Back")
	footerLines := strings.Count(footer, "\n") + 1
	visible := height - headerLines - footerLines
	if visible < 5 {
		visible = 5
	}

	if len(lines) > visible {
		lines = lines[:visible]
	}
	lines = append(lines, footer)

	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// Render chunking configuration with clean circle-based design
func (m *Model) renderChunkingConfig(width, height int) string {
	header := theme.RenderSelectionHeader("Chunking Configuration", 0, 0, width)
	headerLines := strings.Count(header, "\n") + 1
	lines := []string{header}

	fields := []struct {
		name   string
		value  interface{}
		desc   string
		isBool bool
	}{
		{"Enabled", m.pipelineConfig.ChunkingEnabled, "Enable chunking component", true},
		{"Model Path", m.pipelineConfig.ChunkingConfig.ModelPath, "Path to chunker model", false},
		{"Use Shallow Parsing", m.pipelineConfig.ChunkingConfig.UseShallowParsing, "Shallow parse mode", true},
		{"Max Chunk Length", m.pipelineConfig.ChunkingConfig.MaxChunkLength, "Maximum tokens per chunk (2-200)", false},
		{"Combine Adjacent", m.pipelineConfig.ChunkingConfig.CombineAdjacent, "Merge adjacent compatible chunks", true},
	}

	for i, field := range fields {
		focused := i == m.configField

		// Circle indicator
		var indicator string
		if field.isBool {
			if boolVal, ok := field.value.(bool); ok {
				if focused {
					indicator = "🔵"
				} else if boolVal {
					indicator = "🟢"
				} else {
					indicator = "⚫"
				}
			}
		} else {
			if focused {
				indicator = "🔵"
			} else {
				indicator = "  "
			}
		}

		// Format value
		valueStr := fmt.Sprintf("%v", field.value)
		if field.isBool {
			if boolVal, ok := field.value.(bool); ok {
				if boolVal {
					valueStr = "Enabled"
				} else {
					valueStr = "Disabled"
				}
			}
		}

		line := fmt.Sprintf("%s %-22s: %-20s", indicator, field.name, valueStr)

		// Use unified selection system
		renderedLine := theme.RenderSelectableRow(line, width, false, focused)
		lines = append(lines, renderedLine)

		// Description
		lines = append(lines,
			lipgloss.NewStyle().
				Foreground(theme.ColorForegroundDim).
				Render("    "+field.desc))
	}

	footer := theme.RenderSelectionFooter(width, "[↑/↓] Select  [←/→] Adjust  [Space] Toggle  [Enter] Save  [Esc] Back")
	footerLines := strings.Count(footer, "\n") + 1
	visible := height - headerLines - footerLines
	if visible < 5 {
		visible = 5
	}

	if len(lines) > visible {
		lines = lines[:visible]
	}
	lines = append(lines, footer)
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// Render dependency parsing configuration with clean circle-based design
func (m *Model) renderDependencyParsingConfig(width, height int) string {
	header := theme.RenderSelectionHeader("Dependency Parsing Configuration", 0, 0, width)
	headerLines := strings.Count(header, "\n") + 1
	lines := []string{header}

	fields := []struct {
		name   string
		value  interface{}
		desc   string
		isBool bool
	}{
		{"Enabled", m.pipelineConfig.DependencyParsingEnabled, "Enable dependency parser", true},
		{"Parser Type", m.pipelineConfig.DependencyParsingConfig.ParserType, "ClearNLP / Stanford", false},
		{"Universal Deps", m.pipelineConfig.DependencyParsingConfig.UseUniversalDeps, "Map to UD", true},
		{"Include Punctuation", m.pipelineConfig.DependencyParsingConfig.IncludePunctuation, "Keep punctuation tokens", true},
		{"Max Sentence Length", m.pipelineConfig.DependencyParsingConfig.MaxSentenceLength, "Maximum sentence length (10-200)", false},
	}

	for i, field := range fields {
		focused := i == m.configField

		var indicator string
		if field.isBool {
			if boolVal, ok := field.value.(bool); ok {
				if focused {
					indicator = "🔵"
				} else if boolVal {
					indicator = "🟢"
				} else {
					indicator = "⚫"
				}
			}
		} else {
			if focused {
				indicator = "🔵"
			} else {
				indicator = "  "
			}
		}

		valueStr := fmt.Sprintf("%v", field.value)
		if field.isBool {
			if boolVal, ok := field.value.(bool); ok {
				valueStr = map[bool]string{true: "Enabled", false: "Disabled"}[boolVal]
			}
		}

		line := fmt.Sprintf("%s %-22s: %-20s", indicator, field.name, valueStr)

		// Use unified selection system
		renderedLine := theme.RenderSelectableRow(line, width, false, focused)
		lines = append(lines, renderedLine)

		lines = append(lines,
			lipgloss.NewStyle().
				Foreground(theme.ColorForegroundDim).
				Render("    "+field.desc))
	}

	footer := theme.RenderSelectionFooter(width, "[↑/↓] Select  [Space] Toggle  [←/→] Adjust  [Enter] Save  [Esc] Back")
	visible := height - headerLines - (strings.Count(footer, "\n") + 1)
	if visible < 5 {
		visible = 5
	}
	if len(lines) > visible {
		lines = lines[:visible]
	}
	lines = append(lines, footer)
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// Render constituency parsing configuration with clean circle-based design
func (m *Model) renderConstituencyParsingConfig(width, height int) string {
	header := theme.RenderSelectionHeader("Constituency Parsing Configuration", 0, 0, width)
	headerLines := strings.Count(header, "\n") + 1
	lines := []string{header}

	fields := []struct {
		name   string
		value  interface{}
		desc   string
		isBool bool
	}{
		{"Enabled", m.pipelineConfig.ConstituencyParsingEnabled, "Enable constituency parser", true},
		{"Model Path", m.pipelineConfig.ConstituencyParsingConfig.ModelPath, "Path to parser model", false},
		{"Grammar Path", m.pipelineConfig.ConstituencyParsingConfig.GrammarPath, "CFG grammar path", false},
		{"Max Parse Depth", m.pipelineConfig.ConstituencyParsingConfig.MaxParseDepth, "Maximum tree depth (5-200)", false},
		{"Beam Size", m.pipelineConfig.ConstituencyParsingConfig.BeamSize, "Decoder beam size (1-100)", false},
		{"Use Binary Trees", m.pipelineConfig.ConstituencyParsingConfig.UseBinaryTrees, "Binarize trees", true},
	}

	for i, field := range fields {
		focused := i == m.configField

		var indicator string
		if field.isBool {
			if boolVal, ok := field.value.(bool); ok {
				if focused {
					indicator = "🔵"
				} else if boolVal {
					indicator = "🟢"
				} else {
					indicator = "⚫"
				}
			}
		} else {
			if focused {
				indicator = "🔵"
			} else {
				indicator = "  "
			}
		}

		valueStr := fmt.Sprintf("%v", field.value)
		if field.isBool {
			if boolVal, ok := field.value.(bool); ok {
				valueStr = map[bool]string{true: "Enabled", false: "Disabled"}[boolVal]
			}
		}

		line := fmt.Sprintf("%s %-22s: %-20s", indicator, field.name, valueStr)

		// Use unified selection system
		renderedLine := theme.RenderSelectableRow(line, width, false, focused)
		lines = append(lines, renderedLine)

		lines = append(lines,
			lipgloss.NewStyle().
				Foreground(theme.ColorForegroundDim).
				Render("    "+field.desc))
	}

	footer := theme.RenderSelectionFooter(width, "[↑/↓] Select  [Space] Toggle  [←/→] Adjust  [Enter] Save  [Esc] Back")
	visible := height - headerLines - (strings.Count(footer, "\n") + 1)
	if visible < 5 {
		visible = 5
	}
	if len(lines) > visible {
		lines = lines[:visible]
	}
	lines = append(lines, footer)
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// Render resource configuration with clean circle-based design
func (m *Model) renderResourceConfig(width, height int) string {
	header := theme.RenderSelectionHeader("Resource Paths Configuration", 0, 0, width)
	headerLines := strings.Count(header, "\n") + 1
	lines := []string{header}

	fields := []struct {
		name   string
		value  interface{}
		desc   string
		isBool bool
	}{
		{"Resources Dir", m.pipelineConfig.ResourceConfig.ResourcesDirectory, "CTAKES_HOME/resources", false},
		{"Models Dir", m.pipelineConfig.ResourceConfig.ModelsDirectory, "Models directory", false},
		{"Dictionaries Dir", m.pipelineConfig.ResourceConfig.DictionariesDirectory, "Dictionaries root", false},
		{"Cache Dir", m.pipelineConfig.ResourceConfig.CacheDirectory, "Cache location", false},
		{"Download Missing", m.pipelineConfig.ResourceConfig.DownloadMissing, "Fetch models when missing", true},
	}

	for i, field := range fields {
		focused := i == m.configField

		var indicator string
		if field.isBool {
			if boolVal, ok := field.value.(bool); ok {
				if focused {
					indicator = "🔵"
				} else if boolVal {
					indicator = "🟢"
				} else {
					indicator = "⚫"
				}
			}
		} else {
			if focused {
				indicator = "🔵"
			} else {
				indicator = "  "
			}
		}

		valueStr := fmt.Sprintf("%v", field.value)
		if field.isBool {
			if boolVal, ok := field.value.(bool); ok {
				valueStr = map[bool]string{true: "Enabled", false: "Disabled"}[boolVal]
			}
		} else {
			// Truncate long paths
			if str, ok := field.value.(string); ok {
				valueStr = utils.TruncateString(str, 40)
			}
		}

		line := fmt.Sprintf("%s %-22s: %s", indicator, field.name, valueStr)

		// Use unified selection system
		renderedLine := theme.RenderSelectableRow(line, width, false, focused)
		lines = append(lines, renderedLine)

		lines = append(lines,
			lipgloss.NewStyle().
				Foreground(theme.ColorForegroundDim).
				Render("    "+field.desc))
	}

	footer := theme.RenderSelectionFooter(width, "[↑/↓] Select  [Space] Toggle  [Enter] Save  [Esc] Back")
	visible := height - headerLines - (strings.Count(footer, "\n") + 1)
	if visible < 5 {
		visible = 5
	}
	if len(lines) > visible {
		lines = lines[:visible]
	}
	lines = append(lines, footer)
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// Render relation extraction with clean circle-based design
func (m *Model) renderRelationExtractionConfig(width, height int) string {
	header := theme.RenderSelectionHeader("Relation Extraction Configuration", 0, 0, width)
	lines := []string{header}

	fields := []struct {
		name  string
		value bool
	}{
		{"Enabled", m.pipelineConfig.RelationExtractionEnabled},
		{"Include Negatives", m.pipelineConfig.RelationExtractionConfig.IncludeNegatives},
	}

	for i, f := range fields {
		focused := i == m.configField

		var indicator string
		if focused {
			indicator = "🔵"
		} else if f.value {
			indicator = "🟢"
		} else {
			indicator = "⚫"
		}

		valueStr := map[bool]string{true: "Enabled", false: "Disabled"}[f.value]
		line := fmt.Sprintf("%s %-20s: %s", indicator, f.name, valueStr)

		// Use unified selection system
		renderedLine := theme.RenderSelectableRow(line, width, false, focused)
		lines = append(lines, renderedLine)
	}

	footer := theme.RenderSelectionFooter(width, "[↑/↓] Select  [Space] Toggle  [Esc] Back")
	lines = append(lines, "", footer)
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// Render temporal extraction with clean circle-based design
func (m *Model) renderTemporalExtractionConfig(width, height int) string {
	header := theme.RenderSelectionHeader("Temporal Extraction Configuration", 0, 0, width)
	lines := []string{header}

	fields := []struct {
		name  string
		value bool
	}{
		{"Enabled", m.pipelineConfig.TemporalEnabled},
		{"Include TIMEX", m.pipelineConfig.TemporalConfig.IncludeTimex},
		{"Include Events", m.pipelineConfig.TemporalConfig.IncludeEvents},
		{"Include Relations", m.pipelineConfig.TemporalConfig.IncludeRelations},
	}

	for i, f := range fields {
		focused := i == m.configField

		var indicator string
		if focused {
			indicator = "🔵"
		} else if f.value {
			indicator = "🟢"
		} else {
			indicator = "⚫"
		}

		valueStr := map[bool]string{true: "Enabled", false: "Disabled"}[f.value]
		line := fmt.Sprintf("%s %-20s: %s", indicator, f.name, valueStr)

		// Use unified selection system
		renderedLine := theme.RenderSelectableRow(line, width, false, focused)
		lines = append(lines, renderedLine)
	}

	footer := theme.RenderSelectionFooter(width, "[↑/↓] Select  [Space] Toggle  [Esc] Back")
	lines = append(lines, "", footer)
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// Render coreference with clean circle-based design
func (m *Model) renderCoreferenceConfig(width, height int) string {
	header := theme.RenderSelectionHeader("Coreference Resolution Configuration", 0, 0, width)
	lines := []string{header}

	fields := []struct {
		name  string
		value bool
	}{
		{"Enabled", m.pipelineConfig.CoreferenceEnabled},
		{"Use Semantic Info", m.pipelineConfig.CoreferenceConfig.UseSemanticInfo},
	}

	for i, f := range fields {
		focused := i == m.configField

		var indicator string
		if focused {
			indicator = "🔵"
		} else if f.value {
			indicator = "🟢"
		} else {
			indicator = "⚫"
		}

		valueStr := map[bool]string{true: "Enabled", false: "Disabled"}[f.value]
		line := fmt.Sprintf("%s %-20s: %s", indicator, f.name, valueStr)

		// Use unified selection system
		renderedLine := theme.RenderSelectableRow(line, width, false, focused)
		lines = append(lines, renderedLine)
	}

	footer := theme.RenderSelectionFooter(width, "[↑/↓] Select  [Space] Toggle  [Esc] Back")
	lines = append(lines, "", footer)
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// Render Drug NER with clean circle-based design
func (m *Model) renderDrugNERConfig(width, height int) string {
	header := theme.RenderSelectionHeader("Drug NER Configuration", 0, 0, width)
	lines := []string{header}

	fields := []struct {
		name  string
		value bool
	}{
		{"Enabled", m.pipelineConfig.DrugNEREnabled},
		{"Include Dosage", m.pipelineConfig.DrugNERConfig.IncludeDosage},
		{"Include Route", m.pipelineConfig.DrugNERConfig.IncludeRoute},
	}

	for i, f := range fields {
		focused := i == m.configField

		var indicator string
		if focused {
			indicator = "🔵"
		} else if f.value {
			indicator = "🟢"
		} else {
			indicator = "⚫"
		}

		valueStr := map[bool]string{true: "Enabled", false: "Disabled"}[f.value]
		line := fmt.Sprintf("%s %-18s: %s", indicator, f.name, valueStr)

		// Use unified selection system
		renderedLine := theme.RenderSelectableRow(line, width, false, focused)
		lines = append(lines, renderedLine)
	}

	footer := theme.RenderSelectionFooter(width, "[↑/↓] Select  [Space] Toggle  [Esc] Back")
	lines = append(lines, "", footer)
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// Render Side Effect with clean circle-based design
func (m *Model) renderSideEffectConfig(width, height int) string {
	header := theme.RenderSelectionHeader("Side Effect Extraction Configuration", 0, 0, width)
	lines := []string{header}

	fields := []struct {
		name  string
		value bool
	}{
		{"Enabled", m.pipelineConfig.SideEffectEnabled},
		{"Include Severity", m.pipelineConfig.SideEffectConfig.IncludeSeverity},
	}

	for i, f := range fields {
		focused := i == m.configField

		var indicator string
		if focused {
			indicator = "🔵"
		} else if f.value {
			indicator = "🟢"
		} else {
			indicator = "⚫"
		}

		valueStr := map[bool]string{true: "Enabled", false: "Disabled"}[f.value]
		line := fmt.Sprintf("%s %-18s: %s", indicator, f.name, valueStr)

		// Use unified selection system
		renderedLine := theme.RenderSelectableRow(line, width, false, focused)
		lines = append(lines, renderedLine)
	}

	footer := theme.RenderSelectionFooter(width, "[↑/↓] Select  [Space] Toggle  [Esc] Back")
	lines = append(lines, "", footer)
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// Render Smoking Status with clean circle-based design
func (m *Model) renderSmokingStatusConfig(width, height int) string {
	header := theme.RenderSelectionHeader("Smoking Status Configuration", 0, 0, width)
	lines := []string{header}

	fields := []struct {
		name  string
		value bool
	}{
		{"Enabled", m.pipelineConfig.SmokingStatusEnabled},
		{"Include Amount", m.pipelineConfig.SmokingStatusConfig.IncludeAmount},
	}

	for i, f := range fields {
		focused := i == m.configField

		var indicator string
		if focused {
			indicator = "🔵"
		} else if f.value {
			indicator = "🟢"
		} else {
			indicator = "⚫"
		}

		valueStr := map[bool]string{true: "Enabled", false: "Disabled"}[f.value]
		line := fmt.Sprintf("%s %-18s: %s", indicator, f.name, valueStr)

		// Use unified selection system
		renderedLine := theme.RenderSelectableRow(line, width, false, focused)
		lines = append(lines, renderedLine)
	}

	footer := theme.RenderSelectionFooter(width, "[↑/↓] Select  [Space] Toggle  [Esc] Back")
	lines = append(lines, "", footer)
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// Render Template Filling with clean circle-based design
func (m *Model) renderTemplateFillingConfig(width, height int) string {
	header := theme.RenderSelectionHeader("Template Filling Configuration", 0, 0, width)
	lines := []string{header}

	fields := []struct {
		name  string
		value bool
	}{
		{"Enabled", m.pipelineConfig.TemplateFillingEnabled},
		{"Use Constraints", m.pipelineConfig.TemplateFillingConfig.UseConstraints},
	}

	for i, f := range fields {
		focused := i == m.configField

		var indicator string
		if focused {
			indicator = "🔵"
		} else if f.value {
			indicator = "🟢"
		} else {
			indicator = "⚫"
		}

		valueStr := map[bool]string{true: "Enabled", false: "Disabled"}[f.value]
		line := fmt.Sprintf("%s %-18s: %s", indicator, f.name, valueStr)

		// Use unified selection system
		renderedLine := theme.RenderSelectableRow(line, width, false, focused)
		lines = append(lines, renderedLine)
	}

	footer := theme.RenderSelectionFooter(width, "[↑/↓] Select  [Space] Toggle  [Esc] Back")
	lines = append(lines, "", footer)
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// Render Template Editor (post-template selection quick edits)
func (m *Model) renderPipelineTemplateEditor(width, height int) string {
	header := theme.RenderSelectionHeader("Edit Selected Template", 0, 0, width)
	headerLines := strings.Count(header, "\n") + 1
	lines := []string{header}

	items := []struct {
		name  string
		value bool
	}{
		{"Tokenization", m.pipelineConfig.TokenizationEnabled},
		{"POS Tagging", m.pipelineConfig.POSTaggingEnabled},
		{"Chunking", m.pipelineConfig.ChunkingEnabled},
		{"Dependency Parsing", m.pipelineConfig.DependencyParsingEnabled},
		{"Constituency Parsing", m.pipelineConfig.ConstituencyParsingEnabled},
		{"NER", m.pipelineConfig.NEREnabled},
		{"Dictionary Lookup", m.pipelineConfig.DictionaryLookupEnabled},
		{"Assertion Analysis", m.pipelineConfig.AssertionEnabled},
		{"Relation Extraction", m.pipelineConfig.RelationExtractionEnabled},
		{"Temporal Extraction", m.pipelineConfig.TemporalEnabled},
		{"Coreference Resolution", m.pipelineConfig.CoreferenceEnabled},
		{"Drug NER", m.pipelineConfig.DrugNEREnabled},
		{"Side Effect Extraction", m.pipelineConfig.SideEffectEnabled},
		{"Smoking Status", m.pipelineConfig.SmokingStatusEnabled},
		{"Template Filling", m.pipelineConfig.TemplateFillingEnabled},
	}

	for i, it := range items {
		focused := i == m.configField
		checkbox := "☐"
		if it.value {
			checkbox = "☑"
		}
		line := fmt.Sprintf("  %s %s", checkbox, it.name)
		renderedLine := theme.RenderSelectableRow(line, width, false, focused)
		lines = append(lines, renderedLine)
	}

	footer := theme.RenderSelectionFooter(width, "[Space] Toggle  [Enter] Done  [Esc] Cancel")
	visible := height - headerLines - (strings.Count(footer, "\n") + 1)
	if visible < 5 {
		visible = 5
	}
	if len(lines) > visible {
		lines = lines[:visible]
	}
	lines = append(lines, footer)
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// Render NER configuration
func (m *Model) renderNERConfig(width, height int) string {
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Render("Named Entity Recognition Configuration"),
		"",
		lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render("Configure medical entity extraction"),
		"",
	}

	// Entity types selection
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Entity Types:"))
	entityTypes := []string{
		"Diseases/Disorders", "Signs/Symptoms", "Procedures",
		"Medications", "Anatomical Sites", "Lab Values",
	}

	for i, et := range entityTypes {
		// inline contains
		selected := false
		for _, existing := range m.pipelineConfig.NERConfig.EntityTypes {
			if existing == et {
				selected = true
				break
			}
		}
		focused := i == m.configField
		checkbox := "☐"
		if selected {
			checkbox = "☑"
		}
		line := fmt.Sprintf("  %s %s", checkbox, et)
		lines = append(lines, theme.RenderSelectableRow(line, width, selected, focused))
	}

	lines = append(lines, "")

	// NER settings
	settings := []struct {
		name  string
		value interface{}
		desc  string
	}{
		{"Use Context Window", m.pipelineConfig.NERConfig.UseContextWindow, "Use surrounding context"},
		{"Window Size", m.pipelineConfig.NERConfig.WindowSize, "Context window size (5-50)"},
		{"Min Entity Length", m.pipelineConfig.NERConfig.MinEntityLength, "Minimum entity length"},
		{"Max Entity Length", m.pipelineConfig.NERConfig.MaxEntityLength, "Maximum entity length"},
		{"Case Sensitive", m.pipelineConfig.NERConfig.CaseSensitive, "Case-sensitive matching"},
	}

	offset := len(entityTypes)
	for i, setting := range settings {
		focused := i+offset == m.configField

		valueStr := fmt.Sprintf("%v", setting.value)
		if boolVal, ok := setting.value.(bool); ok {
			if boolVal {
				valueStr = "☑"
			} else {
				valueStr = "☐"
			}
		}

		line := fmt.Sprintf("%-20s: %-12s", setting.name, valueStr)
		lines = append(lines, theme.RenderSelectableRow(utils.TruncateString(line, width-4), width, false, focused))
	}

	lines = append(lines, "",
		"↑/↓: Navigate  Space: Toggle  ←/→: Adjust  Enter: Save  ESC: Back")

	if len(lines) > height-2 {
		lines = lines[:height-2]
	}
	return lipgloss.NewStyle().Width(width).Height(height).
		Render(strings.Join(lines, "\n"))
}

// Render dictionary lookup configuration
func (m *Model) renderDictionaryLookupConfig(width, height int) string {
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Render("Dictionary Lookup Configuration"),
		"",
		lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render("Configure dictionary-based entity matching"),
		"",
	}

	// Show available dictionaries
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Available Dictionaries:"))

	dictionaries := m.getAvailableDictionaries()
	for i, dict := range dictionaries {
		selected := m.pipelineConfig.DictionaryLookupConfig.DictionaryPath == dict.Path
		focused := i == m.configField

		status := ""
		if selected {
			status = lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render(" [SELECTED]")
		}

		checkbox := "☐"
		if selected {
			checkbox = "☑"
		}
		line := fmt.Sprintf("  %s %s - %d terms%s",
			checkbox, dict.Name, dict.TermCount, status)
		lines = append(lines, theme.RenderSelectableRow(utils.TruncateString(line, width-4), width, selected, focused))
	}

	lines = append(lines, "", lipgloss.NewStyle().Bold(true).Render("Lookup Settings:"))

	// Lookup algorithms
	algorithms := []string{"Exact Match", "Fuzzy Match", "Permutation Match"}
	offset := len(dictionaries)

	for i, alg := range algorithms {
		selected := m.pipelineConfig.DictionaryLookupConfig.LookupAlgorithm == alg
		focused := i+offset == m.configField
		checkbox := "☐"
		if selected {
			checkbox = "☑"
		}
		line := fmt.Sprintf("  %s %s", checkbox, alg)
		lines = append(lines, theme.RenderSelectableRow(line, width, selected, focused))
	}

	lines = append(lines, "")

	// Additional settings
	settings := []struct {
		name  string
		value interface{}
		desc  string
	}{
		{"Case Sensitive", m.pipelineConfig.DictionaryLookupConfig.CaseSensitive, "Case-sensitive matching"},
		{"Min Match Length", m.pipelineConfig.DictionaryLookupConfig.MinMatchLength, "Minimum match length"},
		{"Max Permutations", m.pipelineConfig.DictionaryLookupConfig.MaxPermutations, "Max permutations for fuzzy"},
		{"Exclude Numbers", m.pipelineConfig.DictionaryLookupConfig.ExcludeNumbers, "Skip numeric tokens"},
		{"Max Lookup Size", m.pipelineConfig.DictionaryLookupConfig.MaxLookupTextSize, "Max text size (KB)"},
	}

	offset2 := offset + len(algorithms)
	for i, setting := range settings {
		focused := i+offset2 == m.configField

		valueStr := fmt.Sprintf("%v", setting.value)
		if boolVal, ok := setting.value.(bool); ok {
			if boolVal {
				valueStr = "☑"
			} else {
				valueStr = "☐"
			}
		}

		line := fmt.Sprintf("%-20s: %-12s  %s", setting.name, valueStr, setting.desc)
		renderedLine := theme.RenderSelectableRow(utils.TruncateString(line, width-4), width, false, focused)
		lines = append(lines, renderedLine)
	}

	// Indexing options
	lines = append(lines, "", lipgloss.NewStyle().Bold(true).Render("Indexing Options:"))
	// Detect indexes under selected dictionary
	var lucenePath, hsqlPath string
	if m.pipelineConfig.DictionaryLookupConfig.DictionaryPath != "" {
		base := m.pipelineConfig.DictionaryLookupConfig.DictionaryPath
		lp := filepath.Join(base, "lucene")
		hp := filepath.Join(base, "hsqldb")
		if _, err := os.Stat(lp); err == nil {
			lucenePath = lp
		}
		if _, err := os.Stat(hp); err == nil {
			hsqlPath = hp
		}
	}
	// Lucene
	luceneSelected := m.pipelineConfig.DictionaryLookupConfig.UseLuceneIndex
	luceneLine := fmt.Sprintf("  %s Use Lucene Index", map[bool]string{true: "☑", false: "☐"}[luceneSelected])
	lines = append(lines, theme.RenderSelectableRow(utils.TruncateString(luceneLine, width-4), width, luceneSelected, m.configField == offset2+len(settings)))
	if lucenePath != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render("    "+utils.TruncateString(lucenePath, width-6)))
	}
	// HSQL
	hsqlSelected := m.pipelineConfig.DictionaryLookupConfig.UseHsqlDictionary
	hsqlLine := fmt.Sprintf("  %s Use HSQLDB Dictionary", map[bool]string{true: "☑", false: "☐"}[hsqlSelected])
	lines = append(lines, theme.RenderSelectableRow(utils.TruncateString(hsqlLine, width-4), width, hsqlSelected, m.configField == offset2+len(settings)+1))
	if hsqlPath != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render("    "+utils.TruncateString(hsqlPath, width-6)))
	}

	lines = append(lines, "",
		"↑/↓: Navigate  Space: Select  Enter: Save  ESC: Back")

	if len(lines) > height-2 {
		lines = lines[:height-2]
	}
	return lipgloss.NewStyle().Width(width).Height(height).
		Render(strings.Join(lines, "\n"))
}

// Render output configuration
func (m *Model) renderPipelineOutputConfig(width, height int) string {
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Render("Output Configuration"),
		"",
		lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render("Configure pipeline output formats and settings"),
		"",
	}

	// Output formats
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Output Format:"))
	formats := []string{"XMI", "JSON", "FHIR", "TSV", "CSV", "XML", "Plain Text"}

	for i, format := range formats {
		selected := m.pipelineConfig.OutputConfig.Format == format
		focused := i == m.configField
		checkbox := "☐"
		if selected {
			checkbox = "☑"
		}
		line := fmt.Sprintf("  %s %s", checkbox, format)
		renderedLine := theme.RenderSelectableRow(line, width, selected, focused)
		lines = append(lines, renderedLine)
	}

	lines = append(lines, "", lipgloss.NewStyle().Bold(true).Render("Output Settings:"))

	settings := []struct {
		name  string
		value interface{}
		desc  string
	}{
		{"Output Directory", m.pipelineConfig.OutputConfig.OutputDirectory, "Output file location"},
		{"Include Metadata", m.pipelineConfig.OutputConfig.IncludeMetadata, "Add processing metadata"},
		{"Pretty Print", m.pipelineConfig.OutputConfig.PrettyPrint, "Format output for readability"},
		{"Compress Output", m.pipelineConfig.OutputConfig.CompressOutput, "GZIP compression"},
		{"Split by Section", m.pipelineConfig.OutputConfig.SplitBySection, "Separate files per section"},
		{"Include Original", m.pipelineConfig.OutputConfig.IncludeOriginalText, "Keep original text"},
		{"Selected Piper", utils.TruncateString(m.pipelineConfig.PiperFilePath, 40), "Chosen .piper pipeline"},
	}

	offset := len(formats)
	for i, setting := range settings {
		focused := i+offset == m.configField

		valueStr := fmt.Sprintf("%v", setting.value)
		if boolVal, ok := setting.value.(bool); ok {
			if boolVal {
				valueStr = "☑"
			} else {
				valueStr = "☐"
			}
		}

		line := fmt.Sprintf("%-20s: %-30s", setting.name, valueStr)
		renderedLine := theme.RenderSelectableRow(utils.TruncateString(line, width-4), width, false, focused)
		lines = append(lines, renderedLine)
		if setting.desc != "" {
			desc := utils.TruncateString("  "+setting.desc, width-4)
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render(desc))
		}
	}

	lines = append(lines, "",
		"↑/↓: Navigate  Space: Toggle  Enter: Save  ESC: Back")

	if len(lines) > height-2 {
		lines = lines[:height-2]
	}
	return lipgloss.NewStyle().Width(width).Height(height).
		Render(strings.Join(lines, "\n"))
}

// Render runtime configuration
func (m *Model) renderPipelineRuntimeConfig(width, height int) string {
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Render("Runtime Configuration"),
		"",
		lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render("Configure JVM and processing parameters"),
		"",
	}

	fields := []struct {
		name  string
		value interface{}
		min   int
		max   int
		desc  string
	}{
		{"Initial Heap (MB)", m.pipelineConfig.RuntimeConfig.InitialHeapSize, 512, 4096, "JVM initial memory"},
		{"Max Heap (MB)", m.pipelineConfig.RuntimeConfig.MaxHeapSize, 1024, 16384, "JVM maximum memory"},
		{"Thread Pool Size", m.pipelineConfig.RuntimeConfig.ThreadPoolSize, 1, 32, "Parallel processing threads"},
		{"Batch Size", m.pipelineConfig.RuntimeConfig.BatchSize, 1, 1000, "Documents per batch"},
		{"Max Document Size (KB)", m.pipelineConfig.RuntimeConfig.MaxDocumentSize, 100, 10240, "Maximum document size"},
		{"Timeout (seconds)", m.pipelineConfig.RuntimeConfig.TimeoutSeconds, 30, 3600, "Processing timeout"},
	}

	for i, field := range fields {
		focused := i == m.configField

		line := fmt.Sprintf("%-25s: %6d  (min: %d, max: %d)",
			field.name, field.value, field.min, field.max)
		renderedLine := theme.RenderSelectableRow(utils.TruncateString(line, width-4), width, false, focused)
		lines = append(lines, renderedLine)
		desc := utils.TruncateString("  "+field.desc, width-4)
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render(desc))
		lines = append(lines, "")
	}

	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Additional Settings:"))

	boolSettings := []struct {
		name  string
		value bool
		desc  string
	}{
		{"Enable Profiling", m.pipelineConfig.RuntimeConfig.EnableProfiling, "Collect performance metrics"},
	}

	offset := len(fields)
	for i, setting := range boolSettings {
		focused := i+offset == m.configField

		line := fmt.Sprintf("%-25s: %s", setting.name, map[bool]string{true: "☑", false: "☐"}[setting.value])
		renderedLine := theme.RenderSelectableRow(line, width, false, focused)
		lines = append(lines, renderedLine)
		desc := utils.TruncateString("  "+setting.desc, width-4)
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render(desc))
	}

	// Log level selection
	lines = append(lines, "", lipgloss.NewStyle().Bold(true).Render("Log Level:"))
	logLevels := []string{"ERROR", "WARN", "INFO", "DEBUG", "TRACE"}

	offset2 := offset + len(boolSettings)
	for i, level := range logLevels {
		selected := m.pipelineConfig.RuntimeConfig.LogLevel == level
		focused := i+offset2 == m.configField
		checkbox := "☐"
		if selected {
			checkbox = "☑"
		}
		line := fmt.Sprintf("  %s %s", checkbox, level)
		renderedLine := theme.RenderSelectableRow(line, width, selected, focused)
		lines = append(lines, renderedLine)
	}

	lines = append(lines, "",
		"↑/↓: Select  ←/→: Adjust  Space: Toggle  Enter: Save  ESC: Back")

	if len(lines) > height-2 {
		lines = lines[:height-2]
	}
	return lipgloss.NewStyle().Width(width).Height(height).
		Render(strings.Join(lines, "\n"))
}

// Helper functions
func getPipelineStatus(enabled bool) string {
	if enabled {
		return lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render("Enabled")
	}
	return lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render("Disabled")
}

func getDictionaryInfo(m *Model) string {
	if m.pipelineConfig.DictionaryLookupConfig.DictionaryPath == "" {
		return lipgloss.NewStyle().Foreground(theme.ColorWarning).Render("No dictionary selected")
	}
	return lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render("Configured")
}

func getResourceSummary(m *Model) string {
	if m.pipelineConfig.ResourceConfig.ResourcesDirectory == "" {
		return lipgloss.NewStyle().Foreground(theme.ColorWarning).Render("Not configured")
	}
	return utils.TruncateString(m.pipelineConfig.ResourceConfig.ResourcesDirectory, 20)
}

func getRuntimeSummary(m *Model) string {
	return fmt.Sprintf("%dMB heap, %d threads",
		m.pipelineConfig.RuntimeConfig.MaxHeapSize,
		m.pipelineConfig.RuntimeConfig.ThreadPoolSize)
}

// renderPipelineMenuItem renders a menu item with consistent styling matching dictionary builder
func (m *Model) renderPipelineMenuItem(selected bool, icon, label, value string, width int) string {
	// Apply consistent padding
	paddedLabel := fmt.Sprintf("  %s %s", icon, label)

	// Calculate space for label and value
	labelWidth := 30
	if width > 60 {
		labelWidth = width / 2
	}

	// Format the content
	content := fmt.Sprintf("%-*s %s", labelWidth, paddedLabel, value)

	// Use full-width row highlighting
	return theme.RenderSelectableRow(content, width, selected, false)
}

func getRowOffset(cursor int) int {
	// Calculate row offset based on cursor position and section headers
	offset := 1 // Quick Start header
	if cursor > 0 {
		offset += 2
	} // Core NLP header
	if cursor > 5 {
		offset += 2
	} // Clinical NLP header
	if cursor > 8 {
		offset += 2
	} // Advanced header
	if cursor > 11 {
		offset += 2
	} // Specialized header
	if cursor > 15 {
		offset += 2
	} // Configuration header
	if cursor > 19 {
		offset += 2
	} // Actions header
	return offset
}

// Using contains from dict_builder.go

// countEnabledComponents mirrors helper used in enhanced preview, local copy for menu value
func (m *Model) countEnabledComponents() int {
	count := 0
	if m.pipelineConfig.TokenizationEnabled {
		count++
	}
	if m.pipelineConfig.POSTaggingEnabled {
		count++
	}
	if m.pipelineConfig.ChunkingEnabled {
		count++
	}
	if m.pipelineConfig.DependencyParsingEnabled {
		count++
	}
	if m.pipelineConfig.ConstituencyParsingEnabled {
		count++
	}
	if m.pipelineConfig.NEREnabled {
		count++
	}
	if m.pipelineConfig.DictionaryLookupEnabled {
		count++
	}
	if m.pipelineConfig.AssertionEnabled {
		count++
	}
	if m.pipelineConfig.RelationExtractionEnabled {
		count++
	}
	if m.pipelineConfig.TemporalEnabled {
		count++
	}
	if m.pipelineConfig.CoreferenceEnabled {
		count++
	}
	if m.pipelineConfig.DrugNEREnabled {
		count++
	}
	if m.pipelineConfig.SideEffectEnabled {
		count++
	}
	if m.pipelineConfig.SmokingStatusEnabled {
		count++
	}
	if m.pipelineConfig.TemplateFillingEnabled {
		count++
	}
	return count
}
