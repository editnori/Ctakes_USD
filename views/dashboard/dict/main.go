package dict

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ctakes-tui/ctakes-tui/internal/dictionary"
)

// DictionaryState represents the current dictionary builder state
type DictionaryState int

const (
	DictStateMainMenu DictionaryState = iota
	DictStateSelectUMLS
	DictStateEditingName
	DictStateSelectingTUIs
	DictStateSelectingVocabs
	DictStateMemoryConfig
	DictStateProcessingConfig
	DictStateFilterConfig
	DictStateOutputConfig
	DictStateRelationshipConfig
	DictStateBuilding
	DictStateViewingDictionaries
)

// DictController manages all dictionary-related functionality
type DictController struct {
	// State
	state     DictionaryState
	lastState DictionaryState

	// Configuration
	config  *DictionaryBuilderConfig
	options *DictOptions

	// UI Components
	table    table.Model
	viewport viewport.Model

	// Data
	vocabList  []dictionary.Vocabulary
	tuiList    []TUIOption
	builtDicts []dictionary.Info

	// Selection tracking
	cursor        int
	selectedItems map[string]bool

	// Build state
	building      bool
	buildLogs     []string
	buildProgress float64

	// Dimensions
	width  int
	height int
}

// TUIOption represents a semantic type option
type TUIOption struct {
	Code        string
	Name        string
	Description string
	Selected    bool
}

// DictionaryBuilderConfig holds all dictionary configuration
type DictionaryBuilderConfig struct {
	Name                 string
	UMLSPath             string
	Description          string
	OutputFormat         string
	SelectedVocabs       []string
	SelectedTUIs         []string
	MemorySettings       MemorySettings
	ProcessingSettings   ProcessingSettings
	FilterSettings       FilterSettings
	OutputSettings       OutputSettings
	RelationshipSettings RelationshipSettings
}

type MemorySettings struct {
	MaxMemory      string
	UseCompression bool
	ChunkSize      string
}

type ProcessingSettings struct {
	NumThreads    int
	BatchSize     int
	EnableLogging bool
}

type FilterSettings struct {
	MinTermLength   int
	MaxTermLength   int
	ExcludeObsolete bool
	ExcludeSuppress bool
	LanguageFilter  []string
}

type OutputSettings struct {
	Format       string
	Compression  bool
	IncludeStats bool
}

type RelationshipSettings struct {
	IncludeRels bool
	RelTypes    []string
	MaxDepth    int
}

// DictOptions holds UI selection options
type DictOptions struct {
	Languages []LanguageOption
	TermTypes []TermTypeOption
	Formats   []FormatOption
}

type LanguageOption struct {
	Code     string
	Name     string
	Selected bool
}

type TermTypeOption struct {
	Code     string
	Name     string
	Selected bool
}

type FormatOption struct {
	Code        string
	Name        string
	Description string
	Selected    bool
}

// NewDictController creates a new dictionary controller
func NewDictController() *DictController {
	return &DictController{
		state:         DictStateMainMenu,
		config:        NewDefaultConfig(),
		options:       NewDefaultOptions(),
		selectedItems: make(map[string]bool),
	}
}

// NewDefaultConfig creates a default dictionary configuration
func NewDefaultConfig() *DictionaryBuilderConfig {
	return &DictionaryBuilderConfig{
		Name:           "",
		UMLSPath:       "",
		Description:    "",
		OutputFormat:   "BSV",
		SelectedVocabs: []string{},
		SelectedTUIs:   []string{},
		MemorySettings: MemorySettings{
			MaxMemory:      "4GB",
			UseCompression: true,
			ChunkSize:      "10000",
		},
		ProcessingSettings: ProcessingSettings{
			NumThreads:    4,
			BatchSize:     1000,
			EnableLogging: true,
		},
		FilterSettings: FilterSettings{
			MinTermLength:   2,
			MaxTermLength:   100,
			ExcludeObsolete: true,
			ExcludeSuppress: true,
			LanguageFilter:  []string{"ENG"},
		},
		OutputSettings: OutputSettings{
			Format:       "BSV",
			Compression:  false,
			IncludeStats: true,
		},
		RelationshipSettings: RelationshipSettings{
			IncludeRels: false,
			RelTypes:    []string{},
			MaxDepth:    2,
		},
	}
}

// NewDefaultOptions creates default UI options
func NewDefaultOptions() *DictOptions {
	return &DictOptions{
		Languages: []LanguageOption{
			{Code: "ENG", Name: "English", Selected: true},
			{Code: "SPA", Name: "Spanish", Selected: false},
			{Code: "FRE", Name: "French", Selected: false},
		},
		TermTypes: []TermTypeOption{
			{Code: "PF", Name: "Preferred Form", Selected: true},
			{Code: "VC", Name: "Variant", Selected: false},
		},
		Formats: []FormatOption{
			{Code: "BSV", Name: "Bar-Separated Values", Description: "cTAKES default format", Selected: true},
			{Code: "HSQLDB", Name: "HSQLDB", Description: "Database format", Selected: false},
			{Code: "LUCENE", Name: "Lucene Index", Description: "Search index", Selected: false},
		},
	}
}

// State management
func (dc *DictController) GetState() DictionaryState {
	return dc.state
}

func (dc *DictController) SetState(state DictionaryState) {
	dc.lastState = dc.state
	dc.state = state
}

func (dc *DictController) GoBack() {
	if dc.lastState != dc.state {
		dc.SetState(dc.lastState)
	} else {
		dc.SetState(DictStateMainMenu)
	}
}

// Configuration management
func (dc *DictController) GetConfig() *DictionaryBuilderConfig {
	return dc.config
}

func (dc *DictController) UpdateConfig(config *DictionaryBuilderConfig) {
	dc.config = config
}

// UI management
func (dc *DictController) SetDimensions(width, height int) {
	dc.width = width
	dc.height = height
}

func (dc *DictController) GetDimensions() (int, int) {
	return dc.width, dc.height
}

// Main render function
func (dc *DictController) Render() string {
	switch dc.state {
	case DictStateMainMenu:
		return RenderMainMenu(dc, dc.width, dc.height)
	case DictStateSelectUMLS:
		return RenderUMLSBrowser(dc, dc.width, dc.height)
	case DictStateEditingName:
		return RenderNameEditor(dc, dc.width, dc.height)
	case DictStateSelectingTUIs:
		return RenderTUISelector(dc, dc.width, dc.height)
	case DictStateSelectingVocabs:
		return RenderVocabSelector(dc, dc.width, dc.height)
	case DictStateMemoryConfig:
		return RenderMemoryConfig(dc, dc.width, dc.height)
	case DictStateProcessingConfig:
		return RenderProcessingConfig(dc, dc.width, dc.height)
	case DictStateFilterConfig:
		return RenderFilterConfig(dc, dc.width, dc.height)
	case DictStateOutputConfig:
		return RenderOutputConfig(dc, dc.width, dc.height)
	case DictStateRelationshipConfig:
		return RenderRelationshipConfig(dc, dc.width, dc.height)
	case DictStateBuilding:
		return RenderBuildProgress(dc, dc.width, dc.height)
	case DictStateViewingDictionaries:
		return RenderDictViewer(dc, dc.width, dc.height)
	default:
		return RenderMainMenu(dc, dc.width, dc.height)
	}
}

// Handle key events
func (dc *DictController) Update(msg tea.Msg) tea.Cmd {
	switch dc.state {
	case DictStateMainMenu:
		return HandleMainMenuUpdate(dc, msg)
	case DictStateSelectUMLS:
		return HandleUMLSBrowserUpdate(dc, msg)
	case DictStateEditingName:
		return HandleNameEditorUpdate(dc, msg)
	case DictStateSelectingTUIs:
		return HandleTUISelectorUpdate(dc, msg)
	case DictStateSelectingVocabs:
		return HandleVocabSelectorUpdate(dc, msg)
	case DictStateMemoryConfig:
		return HandleMemoryConfigUpdate(dc, msg)
	case DictStateProcessingConfig:
		return HandleProcessingConfigUpdate(dc, msg)
	case DictStateFilterConfig:
		return HandleFilterConfigUpdate(dc, msg)
	case DictStateOutputConfig:
		return HandleOutputConfigUpdate(dc, msg)
	case DictStateRelationshipConfig:
		return HandleRelationshipConfigUpdate(dc, msg)
	case DictStateBuilding:
		return HandleBuildProgressUpdate(dc, msg)
	case DictStateViewingDictionaries:
		return HandleDictViewerUpdate(dc, msg)
	default:
		return HandleMainMenuUpdate(dc, msg)
	}
}
