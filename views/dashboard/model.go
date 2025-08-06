package dashboard

import (
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
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
	fileTable       table.Model
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

type tickMsg time.Time

func tickEvery() tea.Cmd {
	return tea.Every(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
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

	return Model{
		activePanel: SidebarPanel,
		cursor:      0,
		sidebarItems: []MenuItem{
			{Icon: "◉", Title: "System Monitor", Action: "system"},
			{Icon: "◎", Title: "File Browser", Action: "files"},
			{Icon: "◈", Title: "Processes", Action: "processes"},
			{Icon: "◆", Title: "Documents", Action: "document_view"},
			{Icon: "◇", Title: "Analyze", Action: "analyze_view"},
			{Icon: "▷", Title: "Pipeline", Action: "pipeline_view"},
		},
		spinner:     s,
		currentPath: currentDir,
		files:       []FileInfo{},
		processes:   []ProcessInfo{},
		lastUpdate:  time.Now(),
		keys:        defaultKeyMap(),
	}
}
