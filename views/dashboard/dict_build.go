package dashboard

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/dictionary"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
)

// BuildState represents the current state of the build process
type BuildState struct {
	Stage              string
	Progress           float64
	CurrentStep        string
	StartTime          time.Time
	ElapsedTime        time.Duration
	EstimatedRemaining time.Duration
	ProcessedItems     int
	TotalItems         int
	MemoryUsage        int64
	CPUUsage           float64
	Errors             []string
	Warnings           []string
	IsComplete         bool
	IsCancelled        bool
}

// Build process with enhanced logging and monitoring
func (m *Model) startBuild() tea.Cmd {
	// Reset UI/build state so a new session streams logs properly
	m.buildProgress = 0
	m.buildLogs = []string{"=== Dictionary Build Started ==="}
	m.buildStartTime = time.Now()
	m.buildError = nil
	m.lastLogIndex = 0
	m.buildState = BuildState{
		StartTime: time.Now(),
		Stage:     "Initializing",
	}
	// Prepare a viewport for logs (full log view and preview reuse it)
	if m.buildViewport.Width == 0 {
		m.buildViewport.Width = 80
	}
	if m.buildViewport.Height == 0 {
		m.buildViewport.Height = 20
	}
	m.buildViewport.HighPerformanceRendering = false
	m.buildViewport.SetContent(strings.Join(m.buildLogs, "\n"))

	// Start logger
	logPath := filepath.Join("dictionaries", baseOr(m.dictConfig.Name, "Dictionary"), "build.log")
	var logErr error
	m.buildLogger, logErr = dictionary.NewBuildLogger(logPath)
	if logErr != nil {
		m.buildLogs = append(m.buildLogs, fmt.Sprintf("Warning: Failed to create logger: %v", logErr))
	} else {
		m.buildLogs = append(m.buildLogs, fmt.Sprintf("Logger initialized: %s", logPath))
	}
	// No direct UI mutation from logger callbacks; we poll the logger on ticks

	// Start the async build first
	go m.runBuildAsync()

	// Then start the ticker
	return buildTickEvery()
}

func (m *Model) runBuildAsync() {
	outDir := filepath.Join("dictionaries", baseOr(m.dictConfig.Name, "Dictionary"))

	// Create enhanced configuration with all options
	cfg := m.createEnhancedConfig()

	// Capture the logger reference to avoid closure issues
	logger := m.buildLogger

	// Add progress callback that updates both logs and logger
	var lastStage string
	progressCallback := func(stage, message string, progress float64) {
		// Only log; UI polls the logger on a timer to update state
		if logger != nil {
			// Only start a new stage if it's different from the last one
			if stage != lastStage && stage != "" {
				logger.StartStage(stage)
				lastStage = stage
			}
			logger.Info(message, progress)
		}
	}

	// Add error callback
	errorCallback := func(err error) {
		// Only log; UI will reflect completion/failure via polling
		if logger != nil {
			if err != nil {
				logger.Error("Build failed", err)
			} else {
				logger.StartStage("done")
				logger.Info("Build completed successfully", 1.0)
			}
		}
	}

	// Create build service with callbacks
	bs := dictionary.NewBuildService(progressCallback, errorCallback)

	// Start async build (this will block until complete)
	bs.BuildDictionaryAsync(cfg, m.umlsPath, outDir)
}

// createEnhancedConfig creates a fully configured dictionary config
func (m *Model) createEnhancedConfig() *dictionary.Config {
	cfg := dictionary.CreateDefaultConfig(m.dictConfig.Name, m.dictConfig.Description)

	// Apply user selections
	cfg.SemanticTypes = append([]string{}, m.dictConfig.TUIs...)
	cfg.Vocabularies = append([]string{}, m.dictConfig.Vocabularies...)
	cfg.Languages = append([]string{}, m.dictConfig.Languages...)

	// Set defaults if unset
	if len(cfg.Languages) == 0 {
		cfg.Languages = []string{"ENG"}
	}
	if len(cfg.TermTypes) == 0 {
		cfg.TermTypes = []string{"PT", "SY", "AB", "ACR"}
	}

	// Memory
	initHeap := m.dictConfig.InitialHeapMB
	maxHeap := m.dictConfig.MaxHeapMB
	stack := m.dictConfig.StackSizeMB
	if initHeap == 0 {
		initHeap = 1024
	}
	if maxHeap == 0 {
		maxHeap = 2048
	}
	if stack == 0 {
		stack = 8
	}
	cfg.Memory = dictionary.MemoryConfig{
		InitialHeapMB: initHeap,
		MaxHeapMB:     maxHeap,
		StackSizeMB:   stack,
	}

	// Processing
	thr := m.dictConfig.ThreadCount
	if thr == 0 {
		thr = 4
	}
	bs := m.dictConfig.BatchSize
	if bs == 0 {
		bs = 1000
	}
	cache := m.dictConfig.CacheSize
	if cache == 0 {
		cache = 128
	}
	minW := m.dictConfig.MinWordLength
	if minW == 0 {
		minW = 2
	}
	maxW := m.dictConfig.MaxWordLength
	if maxW == 0 {
		maxW = 80
	}
	cfg.Processing = dictionary.ProcessingConfig{
		ThreadCount:       thr,
		BatchSize:         bs,
		CacheSize:         cache,
		TempDirectory:     m.dictConfig.TempDirectory,
		PreserveCase:      m.dictConfig.PreserveCase,
		HandlePunctuation: m.dictConfig.HandlePunctuation,
		MinWordLength:     minW,
		MaxWordLength:     maxW,
	}

	// Filters
	minT := m.dictConfig.MinTermLength
	if minT == 0 {
		minT = 3
	}
	maxT := m.dictConfig.MaxTermLength
	if maxT == 0 {
		maxT = 80
	}
	minTok := m.dictConfig.MinTokens
	maxTok := m.dictConfig.MaxTokens
	if maxTok == 0 {
		maxTok = 10
	}
	cfg.Filters = dictionary.FilterConfig{
		MinTermLength:       minT,
		MaxTermLength:       maxT,
		ExcludeSuppressible: m.dictConfig.ExcludeSuppressible,
		ExcludeObsolete:     m.dictConfig.ExcludeObsolete,
		CaseSensitive:       m.dictConfig.CaseSensitive,
		UseNormalization:    m.dictConfig.UseNormalization,
		UseMRRank:           m.dictConfig.UseMRRANK,
		Deduplicate:         m.dictConfig.Deduplicate,
		PreferredOnly:       m.dictConfig.PreferredOnly,
		StripPunctuation:    m.dictConfig.StripPunctuation,
		CollapseWhitespace:  m.dictConfig.CollapseWhitespace,
		ExcludeNumericOnly:  m.dictConfig.ExcludeNumericOnly,
		ExcludePunctOnly:    m.dictConfig.ExcludePunctOnly,
		MinTokens:           minTok,
		MaxTokens:           maxTok,
	}

	// Outputs
	cfg.Outputs = dictionary.Outputs{
		EmitDescriptor: m.dictConfig.EmitDescriptor,
		EmitPipeline:   m.dictConfig.EmitPipeline,
		EmitManifest:   m.dictConfig.EmitManifest,
		EmitTSV:        m.dictConfig.EmitTSV,
		EmitJSONL:      m.dictConfig.EmitJSONL,
		BuildLucene:    m.dictConfig.BuildLucene,
		BuildHSQLDB:    m.dictConfig.BuildHSQLDB,
		UseRareWords:   m.dictConfig.UseRareWords,
		LuceneVersion:  "8.11.0",
	}

	// Relationships
	if m.dictConfig.EnableRelationships {
		cfg.Relationships = dictionary.RelationshipConfig{
			Enabled: true,
			Types:   append([]string{}, m.dictConfig.RelationshipTypes...),
			Depth:   m.dictConfig.RelationshipDepth,
		}
	}

	return cfg
}

func (m *Model) renderBuildView(width, height int) string {
	pct := int(m.buildProgress * 100)

	// Header with stage info
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent)
	header := headerStyle.Render(fmt.Sprintf("Building Dictionary - %s", m.buildState.Stage))

	// Enhanced progress bar with better visibility
	barWidth := width - 12
	filled := int(float64(barWidth) * m.buildProgress)

	// Create more visible progress bar
	var progressBar string
	if filled > 0 {
		progressBar = lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render(strings.Repeat("█", filled)) +
			lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render(strings.Repeat("░", barWidth-filled))
	} else {
		progressBar = lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render(strings.Repeat("░", barWidth))
	}

	// Progress text with stage information
	progressText := fmt.Sprintf("%d%% Complete", pct)
	if m.buildState.Stage != "" && m.buildState.Stage != "Initializing" {
		progressText = fmt.Sprintf("%d%% - %s", pct, m.buildState.Stage)
	}

	// Time information
	elapsed := time.Since(m.buildStartTime)
	timeInfo := fmt.Sprintf("Elapsed: %s", elapsed.Round(time.Second))

	// Current operation
	currentOp := ""
	if m.buildState.CurrentStep != "" {
		currentOp = fmt.Sprintf("Current: %s", m.buildState.CurrentStep)
	}

	// Statistics
	statsLines := []string{}
	if m.buildState.ProcessedItems > 0 {
		statsLines = append(statsLines, fmt.Sprintf("Processed: %d / %d items",
			m.buildState.ProcessedItems, m.buildState.TotalItems))
	}

	// Build the display
	lines := []string{
		header,
		"",
		progressBar,
		progressText,
		"",
		timeInfo,
	}

	if currentOp != "" {
		lines = append(lines, currentOp)
	}

	if len(statsLines) > 0 {
		lines = append(lines, "")
		lines = append(lines, statsLines...)
	}

	// Errors and warnings
	if len(m.buildState.Errors) > 0 {
		errorStyle := lipgloss.NewStyle().Foreground(theme.ColorError)
		lines = append(lines, "", errorStyle.Render(fmt.Sprintf("Errors: %d", len(m.buildState.Errors))))
		for i, err := range m.buildState.Errors {
			if i >= 3 {
				lines = append(lines, errorStyle.Render("..."))
				break
			}
			lines = append(lines, errorStyle.Render(fmt.Sprintf("  - %s", err)))
		}
	}

	if len(m.buildState.Warnings) > 0 {
		warnStyle := lipgloss.NewStyle().Foreground(theme.ColorWarning)
		lines = append(lines, "", warnStyle.Render(fmt.Sprintf("Warnings: %d", len(m.buildState.Warnings))))
		for i, warn := range m.buildState.Warnings {
			if i >= 3 {
				lines = append(lines, warnStyle.Render("..."))
				break
			}
			lines = append(lines, warnStyle.Render(fmt.Sprintf("  - %s", warn)))
		}
	}

	// Status line
	statusLine := ""
	if m.buildState.IsComplete {
		statusLine = lipgloss.NewStyle().Foreground(theme.ColorSuccess).Bold(true).Render("Build Complete")
	} else if m.buildState.IsCancelled {
		statusLine = lipgloss.NewStyle().Foreground(theme.ColorWarning).Bold(true).Render("Build Cancelled")
	} else if m.buildError != nil {
		statusLine = lipgloss.NewStyle().Foreground(theme.ColorError).Bold(true).Render("Build Failed")
	}

	if statusLine != "" {
		lines = append(lines, "", statusLine)
	}

	lines = append(lines, "", "ESC: Cancel/Back | L: View Full Log | Q: Quit")

	return lipgloss.NewStyle().Padding(1).Width(width - 2).Height(height - 2).Render(strings.Join(lines, "\n"))
}

// Full terminal log viewer
func (m *Model) renderFullLogView(width, height int) string {
	// Header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent)
	progressPercent := int(m.buildProgress * 100)
	title := fmt.Sprintf("Build Logs - %s (%d%%)", m.buildState.Stage, progressPercent)
	logPath := filepath.Join("dictionaries", baseOr(m.dictConfig.Name, "Dictionary"), "build.log")

	// Size the viewport to the available space
	vpHeight := height - 6 // room for header and footer
	if vpHeight < 3 {
		vpHeight = 3
	}
	m.buildViewport.Width = width - 4
	m.buildViewport.Height = vpHeight

	// Compose footer status
	status := []string{}
	if m.buildState.IsComplete {
		status = append(status, lipgloss.NewStyle().Foreground(theme.ColorSuccess).Bold(true).Render("Build Complete"))
	} else if m.buildState.IsCancelled {
		status = append(status, lipgloss.NewStyle().Foreground(theme.ColorWarning).Bold(true).Render("Build Cancelled"))
	} else if m.buildError != nil {
		status = append(status, lipgloss.NewStyle().Foreground(theme.ColorError).Bold(true).Render("Build Failed"))
	} else {
		status = append(status, lipgloss.NewStyle().Foreground(theme.ColorAccent).Render("Building..."))
	}
	elapsed := time.Since(m.buildStartTime)
	status = append(status, fmt.Sprintf("Elapsed: %s", elapsed.Round(time.Second)))
	if m.buildState.CurrentStep != "" {
		status = append(status, fmt.Sprintf("Step: %s", m.buildState.CurrentStep))
	}

	content := []string{
		headerStyle.Render(title),
		strings.Repeat("─", width-2),
		fmt.Sprintf("Log file: %s", logPath),
		m.buildViewport.View(),
		"",
		strings.Join(status, " | "),
		"",
		"ESC: Back | Q: Quit",
	}

	return lipgloss.NewStyle().Padding(1).Width(width - 2).Height(height - 2).Render(strings.Join(content, "\n"))
}

// Handle keys for full log view
func (m *Model) handleFullLogKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Return to build view
		m.dictBuilderState = DictStateBuilding
	case "q", "Q":
		// Quit dictionary builder - return to main dashboard
		if m.buildProgress >= 1.0 || m.buildError != nil {
			if m.buildLogger != nil {
				m.buildLogger.Close()
			}
			m.activePanel = SidebarPanel
			m.cursor = 0
		} else {
			// Build in progress - cancel and quit
			m.buildState.IsCancelled = true
			m.buildError = fmt.Errorf("Build cancelled by user")
			if m.buildLogger != nil {
				m.buildLogger.Warning("Build cancelled by user", nil)
				m.buildLogger.Close()
			}
			m.activePanel = SidebarPanel
			m.cursor = 0
		}
	}
	return *m, nil
}

func (m *Model) renderBuildLogs(width, height int) string {
	// This renders detailed logs in the preview panel
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Render("Build Details"),
		strings.Repeat("─", width-4),
		"",
	}

	// Configuration summary
	configSection := []string{
		"Configuration:",
		fmt.Sprintf("  Name: %s", m.dictConfig.Name),
		fmt.Sprintf("  UMLS: %s", filepath.Base(m.umlsPath)),
		fmt.Sprintf("  TUIs: %d selected", len(m.dictConfig.TUIs)),
		fmt.Sprintf("  Vocabularies: %d selected", len(m.dictConfig.Vocabularies)),
		"",
	}
	lines = append(lines, configSection...)

	// Build statistics
	if m.buildState.ProcessedItems > 0 || m.buildState.IsComplete {
		statsSection := []string{
			"Statistics:",
			fmt.Sprintf("  Stage: %s", m.buildState.Stage),
			fmt.Sprintf("  Progress: %.1f%%", m.buildState.Progress*100),
			fmt.Sprintf("  Elapsed: %v", m.buildState.ElapsedTime.Round(time.Second)),
		}

		if m.buildState.ProcessedItems > 0 {
			statsSection = append(statsSection, fmt.Sprintf("  Processed: %d items", m.buildState.ProcessedItems))
		}

		lines = append(lines, statsSection...)
		lines = append(lines, "")
	}

	// Output formats being generated
	if m.dictConfig.BuildLucene || m.dictConfig.BuildHSQLDB || m.dictConfig.EmitTSV || m.dictConfig.EmitJSONL {
		outputSection := []string{"Output Formats:"}
		outputSection = append(outputSection, "  - BSV (Default)")
		if m.dictConfig.BuildLucene {
			outputSection = append(outputSection, "  - Lucene Index")
		}
		if m.dictConfig.BuildHSQLDB {
			outputSection = append(outputSection, "  - HSQLDB")
		}
		if m.dictConfig.EmitTSV {
			outputSection = append(outputSection, "  - TSV Export")
		}
		if m.dictConfig.EmitJSONL {
			outputSection = append(outputSection, "  - JSON Export")
		}
		lines = append(lines, outputSection...)
		lines = append(lines, "")
	}

	// Recent activity: show via a small viewport-like slice for stability
	lines = append(lines, "Recent Activity:")
	maxLogs := height - len(lines) - 6
	if maxLogs < 3 {
		maxLogs = 3
	}
	start := 0
	if len(m.buildLogs) > maxLogs {
		start = len(m.buildLogs) - maxLogs
	}
	for _, log := range m.buildLogs[start:] {
		if strings.Contains(log, "ERROR") {
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorError).Render("  "+log))
		} else if strings.Contains(log, "WARN") {
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorWarning).Render("  "+log))
		} else {
			lines = append(lines, "  "+log)
		}
	}

	return lipgloss.NewStyle().Padding(1).Width(width - 2).Height(height - 2).Render(strings.Join(lines, "\n"))
}

func (m *Model) handleBuildKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "p", "P":
		// Switch to pipeline view (if build is complete or failed)
		if m.buildProgress >= 1.0 || m.buildError != nil {
			m.activePanel = SidebarPanel
			m.cursor = 5 // Pipeline is item 5 in the sidebar (0-indexed)
		}
	case "esc":
		if m.buildProgress >= 1.0 || m.buildError != nil {
			// Build complete or failed, go back to menu
			if m.buildLogger != nil {
				m.buildLogger.Close()
			}
			m.dictBuilderState = DictStateMainMenu
		} else {
			// Build in progress, cancel it
			m.buildState.IsCancelled = true
			m.buildError = fmt.Errorf("Build cancelled by user")
			m.buildLogs = append(m.buildLogs, "=== Build Cancelled ===")
			if m.buildLogger != nil {
				m.buildLogger.Warning("Build cancelled by user", nil)
				m.buildLogger.Close()
			}
			m.dictBuilderState = DictStateMainMenu
		}
	case "l", "L":
		// Switch to full log view
		m.dictBuilderState = DictStateBuildingFullLogs
	case "q", "Q":
		// Quit dictionary builder - return to main dashboard (if build is complete or failed)
		if m.buildProgress >= 1.0 || m.buildError != nil {
			if m.buildLogger != nil {
				m.buildLogger.Close()
			}
			m.activePanel = SidebarPanel
			m.cursor = 0
		} else {
			// Build in progress - cancel and quit
			m.buildState.IsCancelled = true
			m.buildError = fmt.Errorf("Build cancelled by user")
			if m.buildLogger != nil {
				m.buildLogger.Warning("Build cancelled by user", nil)
				m.buildLogger.Close()
			}
			m.activePanel = SidebarPanel
			m.cursor = 0
		}
	}
	return *m, nil
}

// Note: Model struct is defined in model.go with these additions:
// - buildState BuildState
// - buildLogger *dictionary.BuildLogger
// - mu sync.Mutex

// Extended config fields are added to DictionaryConfig in model.go
