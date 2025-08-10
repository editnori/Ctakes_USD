package dashboard

import (
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/dictionary"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
)

type Panel int

const (
	SidebarPanel Panel = iota
	MainPanel
	SystemPanel
	PreviewPanel
)

type Model struct {
	width           int
	height          int
	activePanel     Panel
	cursor          int
	sidebarItems    []MenuItem
	spinner         spinner.Model
	viewport        viewport.Model
	previewViewport viewport.Model
	ready           bool
	cpuPercent      float64
	memPercent      float64
	diskPercent     float64
	cpuCores        int
	totalMem        uint64
	usedMem         uint64
	totalDisk       uint64
	usedDisk        uint64
	currentPath     string
	files           []FileInfo
	processes       []ProcessInfo
	systemCursor    int // Cursor for system panel navigation
	lastUpdate      time.Time
	err             error
	showPreview     bool
	previewContent  string
	previewReady    bool
	// Preview state per context to avoid spillover
	previewStates map[string]PreviewState
	keys          keyMap
	fileTable     table.Model
	processTable  table.Model

	// Async file browser state
	isLoadingDir   bool
	currentDirPage int
	totalDirItems  int
	dirRequestID   uint64

	// Dictionary Builder state
	dictBuilderState DictBuilderState
	umlsPath         string
	dictConfig       DictionaryConfig
	dictTable        table.Model
	dictOptions      []DictOption
	rrfFiles         []string
	dictNameInput    textinput.Model
	tuiTable         table.Model
	vocabTable       table.Model

	// Dictionary viewer
	builtDictionaries []DictionaryInfo
	dictViewerTable   table.Model
	dictListCursor    int
	selectedDict      *dictionary.Config
	dictContent       []string // For browsing dictionary terms
	dictContentCursor int
	dictSearchQuery   string
	dictViewport      viewport.Model

	// Dictionary templates
	dictTemplates []DictTemplate

	// Build progress tracking
	buildProgress       float64
	buildLogs           []string
	buildViewport       viewport.Model
	buildStartTime      time.Time
	buildCurrentStep    string
	buildTotalSteps     int
	buildCurrentStepNum int
	buildError          error

	// In-config field focus (for Memory/Processing screens)
	configField int

	// Simplified Dictionary Builder UI state
	dictMenuCursor int
	tuiList        []string
	vocabList      []string
	tuiCursor      int
	vocabCursor    int
	fileCursor     int

	// Template support (cursor placeholders retained; templates list removed)
	templateCursor          int
	dictTemplateScrollStart int

	// Enhanced build support
	buildState  BuildState
	buildLogger *dictionary.BuildLogger
	mu          sync.Mutex
	// Logger polling state
	lastLogIndex int

	// Caches
	rrfCache map[string][]string

	// Pipeline Configuration
	pipelineState               PipelineState
	pipelineConfig              PipelineConfig
	pipelineMenuCursor          int
	savedPipelines              []PipelineInfo
	pipelineTemplates           []PipelineTemplate
	pipelineTemplateCursor      int
	pipelineTemplateScrollStart int

	// cTAKES Piper discovery
	piperFiles  []dictionary.PiperFile
	piperCursor int

	// Pipeline directory selection state
	pipelineSelectedInputDirs map[string]bool
	pipelineNameInput         textinput.Model

	// Cache for available dictionaries to avoid I/O during rendering
	availableDictionaries []DictionaryAvailable
	availableDictsCached  bool
}

type MenuItem struct {
	Icon   string
	Title  string
	Action string
}

type FileInfo struct {
	Name     string
	Size     string
	Mode     string
	ModTime  string
	IsDir    bool
	Icon     string
	Selected bool
}

type ProcessInfo struct {
	PID    int32
	Name   string
	CPU    float64
	Memory float32
	Status string
}

// PreviewState holds preview content for a specific context
type PreviewState struct {
	Content  string
	Ready    bool
	FilePath string
	Error    error
}

type DictBuilderState int

const (
	DictStateMainMenu DictBuilderState = iota // Main dictionary builder menu
	DictStateSelectingTemplate
	DictStateSelectUMLS
	DictStateEditingName
	DictStateSelectingTUIs
	DictStateSelectingVocabs
	// Interactive sub-menu states for advanced settings
	DictStateMemoryConfig
	DictStateProcessingConfig
	DictStateFilterConfig
	DictStateOutputConfig
	DictStateRelationshipConfig
	DictStateBuilding
	DictStateBuildingFullLogs
	DictStateCasedConfig // New state for cased dictionary config
	DictStateComplete
	DictStateViewingDictionaries
	DictStateViewingDetails  // View details of a selected dictionary
	DictStateBrowsingContent // Browse dictionary content (terms)
	// Legacy alias used by some handlers
	DictStateConfiguring
)

// Backwards-compatibility aliases for refactored state names
const (
	DictStateConfiguringMemory        = DictStateMemoryConfig
	DictStateConfiguringProcessing    = DictStateProcessingConfig
	DictStateConfiguringFilters       = DictStateFilterConfig
	DictStateConfiguringOutputs       = DictStateOutputConfig
	DictStateConfiguringRelationships = DictStateRelationshipConfig
)

type DictionaryConfig struct {
	Name         string
	Description  string
	TUIs         []string
	Vocabularies []string
	Languages    []string
	TermTypes    []string

	// Memory Settings
	InitialHeapMB int // 512-3072 MB
	MaxHeapMB     int // 512-3072 MB
	StackSizeMB   int // 1-64 MB

	// Processing Options
	ThreadCount       int // 1-16
	BatchSize         int // 100-10000
	CacheSize         int // 64-512 MB
	TempDirectory     string
	PreserveCase      bool
	HandlePunctuation bool
	MinWordLength     int // 1-10
	MaxWordLength     int // 10-256

	// Filter Options
	MinTermLength       int
	MaxTermLength       int
	ExcludeSuppressible bool
	ExcludeObsolete     bool
	CaseSensitive       bool
	UseNormalization    bool
	UseMRRANK           bool
	Deduplicate         bool
	PreferredOnly       bool
	StripPunctuation    bool
	CollapseWhitespace  bool
	ExcludeNumericOnly  bool
	ExcludePunctOnly    bool
	MinTokens           int
	MaxTokens           int

	// Cased Dictionary Support (from cTAKES 6.0)
	BuildCasedDictionary bool   // Build case-sensitive variant
	CasedTermRanking     string // "frequency" or "mrrank"
	IncludeAcronyms      bool   // Include acronyms in cased dictionary
	IncludeAbbreviations bool   // Include abbreviations in cased dictionary

	// Output Formats
	EmitBSV        bool // default on
	BuildHSQLDB    bool
	BuildLucene    bool
	UseRareWords   bool
	EmitTSV        bool
	EmitJSONL      bool
	EmitDescriptor bool
	EmitPipeline   bool
	EmitManifest   bool
	EmitCasedBSV   bool // Emit case-sensitive BSV variant

	// Relationship Settings
	EnableRelationships bool
	RelationshipDepth   int // 0-5
	RelationshipTypes   []string

	// Extended build options
	ExportTSV            bool
	ExportJSON           bool
	BuildHSQL            bool // Note: BuildHSQLDB already exists above
	IncludeRelationships bool
	UseCustomFilters     bool
	CustomMinLength      int
	CustomMaxLength      int
	// Description field already exists above on line 144

	// Additional fields for new config screens
	QueueSize          int
	BufferSize         int
	OptimizeMemory     bool
	UseMemoryMapping   bool
	EmitXML            bool
	EmitBinary         bool
	EmitSQLite         bool
	BuildCased         bool
	IncludeLowercase   bool
	IncludeUppercase   bool
	ParallelProcessing bool
	Timeout            int
	ExcludeNumbers     bool
	ExcludeSymbols     bool
}

type DictOption struct {
	Name   string
	Value  string
	Type   string // "action", "config", "info"
	Status string // "pending", "configured", "ready"
}

type DictionaryInfo struct {
	Name       string
	Path       string
	Size       string
	TUICount   int
	VocabCount int
	Languages  string
	Created    time.Time
}

type tickMsg time.Time
type buildTickMsg time.Time

// Message types for directory operations
type DirectoryLoadedMsg struct {
	Path      string
	Files     []FileInfo
	Error     error
	RequestID uint64
}

type DirectoryLoadingMsg struct {
	Path string
}

func tickEvery() tea.Cmd {
	return tea.Every(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func buildTickEvery() tea.Cmd {
	return tea.Every(500*time.Millisecond, func(t time.Time) tea.Msg {
		return buildTickMsg(t)
	})
}

func New() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.ColorAccent)

	// Get current working directory instead of home directory
	currentDir, err := os.Getwd()
	if err != nil {
		// Fallback to home directory if we can't get current directory
		currentDir, _ = os.UserHomeDir()
	}

	// Initialize text input for dictionary name
	ti := textinput.New()
	ti.Placeholder = "Enter dictionary name"
	ti.CharLimit = 50
	ti.Width = 40

	// Initialize file table
	fileTable := table.New(
		table.WithColumns([]table.Column{
			{Title: "", Width: 2},
			{Title: "Name", Width: 30},
			{Title: "Size", Width: 10},
			{Title: "Modified", Width: 15},
		}),
		table.WithRows([]table.Row{}),
		table.WithFocused(false),
		table.WithHeight(10),
	)

	model := Model{
		activePanel: SidebarPanel,
		cursor:      0,
		sidebarItems: []MenuItem{
			{Icon: theme.GetSemanticIcon("info"), Title: "System Monitor", Action: "system"},
			{Icon: theme.GetSemanticIcon("browse"), Title: "File Browser", Action: "files"},
			{Icon: theme.GetSemanticIcon("special"), Title: "Dictionary Builder", Action: "dictionary_builder_view"},
			{Icon: theme.GetSemanticIcon("data"), Title: "Pipeline Configuration", Action: "pipeline"},
		},
		spinner:       s,
		currentPath:   currentDir,
		files:         []FileInfo{},
		processes:     []ProcessInfo{},
		lastUpdate:    time.Now(),
		keys:          defaultKeyMap(),
		dictNameInput: ti,
		rrfCache:      make(map[string][]string),
		fileTable:     fileTable,
		// Initialize async file browser state
		isLoadingDir:   false,
		currentDirPage: 0,
		totalDirItems:  0,
	}

	// Initialize dictionary templates
	model.initTemplates()

	return model
}

// GetCursor returns the current cursor position for compatibility
func (m Model) GetCursor() int {
	return m.cursor
}
