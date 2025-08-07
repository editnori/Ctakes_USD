package dashboard

import (
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	lastUpdate      time.Time
	err             error
	showPreview     bool
	previewContent  string
	previewReady    bool
	keys            keyMap
	fileTable       table.Model
	processTable    table.Model

	// Dictionary Builder state
	dictBuilderState DictBuilderState
	umlsPath         string
	dictConfig       DictionaryConfig
	showDictConfig   bool
	dictTable        table.Model
	dictOptions      []DictOption
	dictPreviewMode  string
	rrfFiles         []string
	dictNameInput    textinput.Model
	editingDictName  bool
	tuiTable         table.Model
	selectingTUIs    bool
	vocabTable       table.Model
	selectingVocabs  bool

	// Dictionary viewer
	builtDictionaries []DictionaryInfo
	dictViewerTable   table.Model

	// Build progress tracking
	buildProgress       float64
	buildLogs           []string
	buildViewport       viewport.Model
	buildStartTime      time.Time
	buildCurrentStep    string
	buildTotalSteps     int
	buildCurrentStepNum int
	buildError          error
}

type MenuItem struct {
	Icon   string
	Title  string
	Action string
}

type FileInfo struct {
	Name    string
	Size    string
	Mode    string
	ModTime string
	IsDir   bool
	Icon    string
}

type ProcessInfo struct {
	PID    int32
	Name   string
	CPU    float64
	Memory float32
	Status string
}

type DictBuilderState int

const (
	DictStateSelectUMLS DictBuilderState = iota
	DictStateConfiguring
	DictStateEditingName
	DictStateSelectingTUIs
	DictStateSelectingVocabs
	DictStateConfiguringMemory
	DictStateConfiguringProcessing
	DictStateConfiguringFilters
	DictStateConfiguringOutputs
	DictStateConfiguringRelationships
	// New sub-menu states for advanced settings
	DictStateMemoryConfig
	DictStateProcessingConfig
	DictStateFilterConfig
	DictStateOutputConfig
	DictStateRelationshipConfig
	DictStateBuilding
	DictStateComplete
	DictStateViewingDictionaries
)

type DictionaryConfig struct {
	Name         string
	Description  string
	TUIs         []string
	Vocabularies []string
	Languages    []string
	TermTypes    []string
	
	// Memory Settings
	InitialHeapMB int  // 512-3072 MB
	MaxHeapMB     int  // 512-3072 MB  
	StackSizeMB   int  // 1-64 MB
	
	// Processing Options
	ThreadCount       int    // 1-16
	BatchSize         int    // 100-10000
	CacheSize         int    // 64-512 MB
	TempDirectory     string
	PreserveCase      bool
	HandlePunctuation bool
	MinWordLength     int    // 1-10
	MaxWordLength     int    // 10-256
	
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
	
	// Relationship Settings
	EnableRelationships bool
	RelationshipDepth   int      // 0-5
	RelationshipTypes   []string
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

	return Model{
		activePanel: SidebarPanel,
		cursor:      0,
		sidebarItems: []MenuItem{
			{Icon: "◉", Title: "System Monitor", Action: "system"},
			{Icon: "◎", Title: "File Browser", Action: "files"},
			{Icon: "◈", Title: "Dictionary Builder", Action: "dictionary_builder_view"},
			{Icon: "◆", Title: "Documents", Action: "document_view"},
			{Icon: "◇", Title: "Analyze", Action: "analyze_view"},
			{Icon: "▷", Title: "Pipeline", Action: "pipeline_view"},
		},
		spinner:       s,
		currentPath:   currentDir,
		files:         []FileInfo{},
		processes:     []ProcessInfo{},
		lastUpdate:    time.Now(),
		keys:          defaultKeyMap(),
		dictNameInput: ti,
	}
}
