package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/dictionary"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
	"github.com/ctakes-tui/ctakes-tui/internal/utils"
)

// HandlePipelineNavigation processes key events in pipeline configuration
func (m *Model) HandlePipelineNavigation(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch m.pipelineState {
	case PipelineMainMenu:
		return m.handlePipelineMenuKeys(msg)
	case PipelineSelectingTemplate:
		return m.handlePipelineTemplateKeys(msg)
	case PipelineTemplateEditor:
		return m.handlePipelineTemplateEditorKeys(msg)
	case PipelineDictionaryConfig:
		return m.handleDictionaryLookupKeys(msg)
	case PipelineSelectingInputDirs:
		return m.handleInputDirKeys(msg)
	case PipelineSelectingOutputDir:
		return m.handleOutputDirKeys(msg)
	case PipelineEditingRunName:
		return m.handleRunNameKeys(msg)
	case PipelineOutputConfig:
		return m.handlePipelineOutputKeys(msg)
	case PipelineRuntimeConfig:
		return m.handlePipelineRuntimeKeys(msg)
	case PipelineRunning:
		return m.handlePipelineRunKeys(msg)
	default:
		return m.handleGenericPipelineKeys(msg)
	}

}

// Handle simplified pipeline menu navigation
func (m *Model) handlePipelineMenuKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.pipelineMenuCursor > 0 {
			m.pipelineMenuCursor--
		}
	case "down", "j":
		if m.pipelineMenuCursor < 9 { // 10 menu items = indexes 0-9
			m.pipelineMenuCursor++
		}
	case "enter":
		switch m.pipelineMenuCursor {
		case 0: // Run Name
			m.pipelineState = PipelineEditingRunName
			m.pipelineNameInput.SetValue(m.pipelineConfig.RunName)
			m.pipelineNameInput.CursorEnd()
			m.pipelineNameInput.Focus()
		case 1: // Choose Template
			m.pipelineState = PipelineSelectingTemplate
			if len(m.pipelineTemplates) == 0 {
				m.initPipelineTemplates()
			}
		case 2: // Edit Components
			m.pipelineState = PipelineTemplateEditor
			m.configField = 0
		case 3: // Dictionary
			// Open pipeline dictionary selection/config
			m.pipelineState = PipelineDictionaryConfig
			m.configField = 0
			// Refresh dictionary cache when entering
			m.availableDictsCached = false
		case 4: // Input Directories
			m.pipelineState = PipelineSelectingInputDirs
			if m.currentPath == "" {
				m.currentPath = "."
			}
			// Ensure file list is up to date
			m.updateFileList()
			m.updateTables()
			return *m, nil
		case 5: // Output Directory
			m.pipelineState = PipelineSelectingOutputDir
			if m.currentPath == "" {
				m.currentPath = "."
			}
			m.updateFileList()
			m.updateTables()
			return *m, nil
		case 6: // Mirror Output Structure toggle
			m.pipelineConfig.MirrorOutputStructure = !m.pipelineConfig.MirrorOutputStructure
		case 7: // Output Settings
			m.pipelineState = PipelineOutputConfig
			m.configField = 0
		case 8: // Runtime Settings
			m.pipelineState = PipelineRuntimeConfig
			m.configField = 0
		case 9: // Run Pipeline
			if m.isPipelineReadyToRun() {
				m.pipelineState = PipelineRunning
				return *m, m.runPipeline()
			}
		}
	case "esc":
		// Return to main dashboard
		m.activePanel = SidebarPanel
	case "d", "D":
		// Quick switch to dictionary builder
		m.activePanel = MainPanel
		m.dictBuilderState = DictStateMainMenu
	}
	return *m, nil
}

// Handle Template Editor (toggle components quickly)
func (m *Model) handlePipelineTemplateEditorKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	max := 15 // number of quick-toggle items
	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < max-1 {
			m.configField++
		}
	case "space":
		switch m.configField {
		case 0:
			m.pipelineConfig.TokenizationEnabled = !m.pipelineConfig.TokenizationEnabled
		case 1:
			m.pipelineConfig.POSTaggingEnabled = !m.pipelineConfig.POSTaggingEnabled
		case 2:
			m.pipelineConfig.ChunkingEnabled = !m.pipelineConfig.ChunkingEnabled
		case 3:
			m.pipelineConfig.DependencyParsingEnabled = !m.pipelineConfig.DependencyParsingEnabled
		case 4:
			m.pipelineConfig.ConstituencyParsingEnabled = !m.pipelineConfig.ConstituencyParsingEnabled
		case 5:
			m.pipelineConfig.NEREnabled = !m.pipelineConfig.NEREnabled
		case 6:
			m.pipelineConfig.DictionaryLookupEnabled = !m.pipelineConfig.DictionaryLookupEnabled
		case 7:
			m.pipelineConfig.AssertionEnabled = !m.pipelineConfig.AssertionEnabled
		case 8:
			m.pipelineConfig.RelationExtractionEnabled = !m.pipelineConfig.RelationExtractionEnabled
		case 9:
			m.pipelineConfig.TemporalEnabled = !m.pipelineConfig.TemporalEnabled
		case 10:
			m.pipelineConfig.CoreferenceEnabled = !m.pipelineConfig.CoreferenceEnabled
		case 11:
			m.pipelineConfig.DrugNEREnabled = !m.pipelineConfig.DrugNEREnabled
		case 12:
			m.pipelineConfig.SideEffectEnabled = !m.pipelineConfig.SideEffectEnabled
		case 13:
			m.pipelineConfig.SmokingStatusEnabled = !m.pipelineConfig.SmokingStatusEnabled
		case 14:
			m.pipelineConfig.TemplateFillingEnabled = !m.pipelineConfig.TemplateFillingEnabled
		}
	case "enter", "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

// Handle Input Directory multi-select using file browser table
func (m *Model) handleInputDirKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k", "down", "j":
		var cmd tea.Cmd
		m.fileTable, cmd = m.fileTable.Update(msg)
		return *m, cmd
	case "pgup":
		return *m, m.handleFileBrowserPagination(false)
	case "pgdown":
		return *m, m.handleFileBrowserPagination(true)
	case " ":
		// Toggle selection in file table and mirror to map only for directories
		if m.fileTable.Cursor() >= 0 && m.fileTable.Cursor() < len(m.files) {
			sel := m.files[m.fileTable.Cursor()]
			if sel.IsDir && sel.Name != ".." {
				full := filepath.Join(m.currentPath, sel.Name)
				if m.pipelineSelectedInputDirs[full] {
					delete(m.pipelineSelectedInputDirs, full)
				} else {
					m.pipelineSelectedInputDirs[full] = true
				}
				// Reflect selection star in table row
				m.toggleSelection()
			}
		}
	case "enter":
		// Enter/open directory only
		return *m, m.handleFileAction()
	case "s", "S":
		// Save selected dirs into config and return
		m.pipelineConfig.InputDirs = m.collectSelectedInputDirs()
		m.pipelineState = PipelineMainMenu
	case "esc":
		// Cancel without saving selections
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

// Handle Output Directory single-select
func (m *Model) handleOutputDirKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k", "down", "j":
		var cmd tea.Cmd
		m.fileTable, cmd = m.fileTable.Update(msg)
		return *m, cmd
	case "pgup":
		return *m, m.handleFileBrowserPagination(false)
	case "pgdown":
		return *m, m.handleFileBrowserPagination(true)
	case "right", "l":
		// Open selected directory without choosing as output
		return *m, m.handleFileAction()
	case "enter":
		// Enter/open directory only
		return *m, m.handleFileAction()
	case "s", "S":
		// Save highlighted/current directory as output and return
		if m.fileTable.Cursor() >= 0 && m.fileTable.Cursor() < len(m.files) {
			sel := m.files[m.fileTable.Cursor()]
			if sel.IsDir {
				var chosen string
				if sel.Name == ".." {
					chosen = filepath.Dir(m.currentPath)
				} else {
					chosen = filepath.Join(m.currentPath, sel.Name)
				}
				m.pipelineConfig.OutputDir = chosen
				m.pipelineState = PipelineMainMenu
			}
		} else {
			// Fallback to current path
			m.pipelineConfig.OutputDir = m.currentPath
			m.pipelineState = PipelineMainMenu
		}
	case "esc":
		// Cancel without saving
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

// Handle run name editing
func (m *Model) handleRunNameKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.pipelineConfig.RunName = m.pipelineNameInput.Value()
		m.pipelineState = PipelineMainMenu
		return *m, nil
	case "esc":
		m.pipelineState = PipelineMainMenu
		return *m, nil
	}
	var cmd tea.Cmd
	m.pipelineNameInput, cmd = m.pipelineNameInput.Update(msg)
	return *m, cmd
}

// Handle pipeline template selection
func (m *Model) handlePipelineTemplateKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.pipelineTemplateCursor > 0 {
			m.pipelineTemplateCursor--
		}
	case "down", "j":
		if m.pipelineTemplateCursor < len(m.pipelineTemplates)-1 {
			m.pipelineTemplateCursor++
		}
	case "pgup":
		if m.pipelineTemplateCursor > 5 {
			m.pipelineTemplateCursor -= 5
		} else {
			m.pipelineTemplateCursor = 0
		}
	case "pgdown":
		if m.pipelineTemplateCursor+5 < len(m.pipelineTemplates) {
			m.pipelineTemplateCursor += 5
		} else if len(m.pipelineTemplates) > 0 {
			m.pipelineTemplateCursor = len(m.pipelineTemplates) - 1
		}
	case "enter", " ":
		// Apply selected template
		if m.pipelineTemplateCursor < len(m.pipelineTemplates) {
			// Merge template booleans into existing config; preserve paths and sub-configs
			sel := m.pipelineTemplates[m.pipelineTemplateCursor].Config
			m.pipelineConfig.Name = sel.Name
			m.pipelineConfig.Description = sel.Description
			// Core
			m.pipelineConfig.TokenizationEnabled = sel.TokenizationEnabled
			m.pipelineConfig.POSTaggingEnabled = sel.POSTaggingEnabled
			m.pipelineConfig.ChunkingEnabled = sel.ChunkingEnabled
			m.pipelineConfig.DependencyParsingEnabled = sel.DependencyParsingEnabled
			m.pipelineConfig.ConstituencyParsingEnabled = sel.ConstituencyParsingEnabled
			// Clinical
			m.pipelineConfig.NEREnabled = sel.NEREnabled
			m.pipelineConfig.DictionaryLookupEnabled = sel.DictionaryLookupEnabled
			m.pipelineConfig.AssertionEnabled = sel.AssertionEnabled
			// Advanced
			m.pipelineConfig.RelationExtractionEnabled = sel.RelationExtractionEnabled
			m.pipelineConfig.TemporalEnabled = sel.TemporalEnabled
			m.pipelineConfig.CoreferenceEnabled = sel.CoreferenceEnabled
			// Specialized
			m.pipelineConfig.DrugNEREnabled = sel.DrugNEREnabled
			m.pipelineConfig.SideEffectEnabled = sel.SideEffectEnabled
			m.pipelineConfig.SmokingStatusEnabled = sel.SmokingStatusEnabled
			m.pipelineConfig.TemplateFillingEnabled = sel.TemplateFillingEnabled
			// If template includes a Piper file path (built dict or cTAKES), use it
			if sel.PiperFilePath != "" {
				m.pipelineConfig.PiperFilePath = sel.PiperFilePath
			}
			m.pipelineConfig.IsTemplateApplied = true
			// Clear any stale PiperFilePath reliance
			// Keep existing IO selections
			// Preserve OutputDir and RunName if already set by user
			// Immediately open a lightweight editor so preconfig is editable
			m.pipelineState = PipelineMainMenu
		}
	case "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

// Handle Chunking
func (m *Model) handleChunkingKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < 4 {
			m.configField++
		}
	case "space":
		switch m.configField {
		case 0:
			m.pipelineConfig.ChunkingEnabled = !m.pipelineConfig.ChunkingEnabled
		case 2:
			m.pipelineConfig.ChunkingConfig.UseShallowParsing = !m.pipelineConfig.ChunkingConfig.UseShallowParsing
		case 4:
			m.pipelineConfig.ChunkingConfig.CombineAdjacent = !m.pipelineConfig.ChunkingConfig.CombineAdjacent
		}
	case "left", "h":
		if m.configField == 3 && m.pipelineConfig.ChunkingConfig.MaxChunkLength > 2 {
			m.pipelineConfig.ChunkingConfig.MaxChunkLength--
		}
	case "right", "l":
		if m.configField == 3 && m.pipelineConfig.ChunkingConfig.MaxChunkLength < 200 {
			m.pipelineConfig.ChunkingConfig.MaxChunkLength++
		}
	case "enter", "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

// Handle Dependency Parsing
func (m *Model) handleDependencyParsingKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < 4 {
			m.configField++
		}
	case "space":
		switch m.configField {
		case 0:
			m.pipelineConfig.DependencyParsingEnabled = !m.pipelineConfig.DependencyParsingEnabled
		case 2:
			m.pipelineConfig.DependencyParsingConfig.UseUniversalDeps = !m.pipelineConfig.DependencyParsingConfig.UseUniversalDeps
		case 3:
			m.pipelineConfig.DependencyParsingConfig.IncludePunctuation = !m.pipelineConfig.DependencyParsingConfig.IncludePunctuation
		}
	case "left", "h":
		if m.configField == 4 && m.pipelineConfig.DependencyParsingConfig.MaxSentenceLength > 10 {
			m.pipelineConfig.DependencyParsingConfig.MaxSentenceLength--
		}
	case "right", "l":
		if m.configField == 4 && m.pipelineConfig.DependencyParsingConfig.MaxSentenceLength < 200 {
			m.pipelineConfig.DependencyParsingConfig.MaxSentenceLength++
		}
	case "enter", "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

// Handle Constituency Parsing
func (m *Model) handleConstituencyParsingKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < 5 {
			m.configField++
		}
	case "space":
		switch m.configField {
		case 0:
			m.pipelineConfig.ConstituencyParsingEnabled = !m.pipelineConfig.ConstituencyParsingEnabled
		case 5:
			m.pipelineConfig.ConstituencyParsingConfig.UseBinaryTrees = !m.pipelineConfig.ConstituencyParsingConfig.UseBinaryTrees
		}
	case "left", "h":
		switch m.configField {
		case 3:
			if m.pipelineConfig.ConstituencyParsingConfig.MaxParseDepth > 5 {
				m.pipelineConfig.ConstituencyParsingConfig.MaxParseDepth--
			}
		case 4:
			if m.pipelineConfig.ConstituencyParsingConfig.BeamSize > 1 {
				m.pipelineConfig.ConstituencyParsingConfig.BeamSize--
			}
		}
	case "right", "l":
		switch m.configField {
		case 3:
			if m.pipelineConfig.ConstituencyParsingConfig.MaxParseDepth < 200 {
				m.pipelineConfig.ConstituencyParsingConfig.MaxParseDepth++
			}
		case 4:
			if m.pipelineConfig.ConstituencyParsingConfig.BeamSize < 100 {
				m.pipelineConfig.ConstituencyParsingConfig.BeamSize++
			}
		}
	case "enter", "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

// Handle Resource Config
func (m *Model) handleResourceKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < 4 {
			m.configField++
		}
	case "space":
		if m.configField == 4 {
			m.pipelineConfig.ResourceConfig.DownloadMissing = !m.pipelineConfig.ResourceConfig.DownloadMissing
		}
	case "enter", "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

// Minimal handlers for the remaining components
func (m *Model) handleRelationExtractionKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < 1 {
			m.configField++
		}
	case "space":
		if m.configField == 0 {
			m.pipelineConfig.RelationExtractionEnabled = !m.pipelineConfig.RelationExtractionEnabled
		}
		if m.configField == 1 {
			m.pipelineConfig.RelationExtractionConfig.IncludeNegatives = !m.pipelineConfig.RelationExtractionConfig.IncludeNegatives
		}
	case "enter", "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

func (m *Model) handleTemporalExtractionKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < 3 {
			m.configField++
		}
	case "space":
		switch m.configField {
		case 0:
			m.pipelineConfig.TemporalEnabled = !m.pipelineConfig.TemporalEnabled
		case 1:
			m.pipelineConfig.TemporalConfig.IncludeTimex = !m.pipelineConfig.TemporalConfig.IncludeTimex
		case 2:
			m.pipelineConfig.TemporalConfig.IncludeEvents = !m.pipelineConfig.TemporalConfig.IncludeEvents
		case 3:
			m.pipelineConfig.TemporalConfig.IncludeRelations = !m.pipelineConfig.TemporalConfig.IncludeRelations
		}
	case "enter", "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

func (m *Model) handleCoreferenceKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < 1 {
			m.configField++
		}
	case "space":
		if m.configField == 0 {
			m.pipelineConfig.CoreferenceEnabled = !m.pipelineConfig.CoreferenceEnabled
		}
		if m.configField == 1 {
			m.pipelineConfig.CoreferenceConfig.UseSemanticInfo = !m.pipelineConfig.CoreferenceConfig.UseSemanticInfo
		}
	case "enter", "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

func (m *Model) handleDrugNERKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < 2 {
			m.configField++
		}
	case "space":
		switch m.configField {
		case 0:
			m.pipelineConfig.DrugNEREnabled = !m.pipelineConfig.DrugNEREnabled
		case 1:
			m.pipelineConfig.DrugNERConfig.IncludeDosage = !m.pipelineConfig.DrugNERConfig.IncludeDosage
		case 2:
			m.pipelineConfig.DrugNERConfig.IncludeRoute = !m.pipelineConfig.DrugNERConfig.IncludeRoute
		}
	case "enter", "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

func (m *Model) handleSideEffectKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < 1 {
			m.configField++
		}
	case "space":
		if m.configField == 0 {
			m.pipelineConfig.SideEffectEnabled = !m.pipelineConfig.SideEffectEnabled
		}
		if m.configField == 1 {
			m.pipelineConfig.SideEffectConfig.IncludeSeverity = !m.pipelineConfig.SideEffectConfig.IncludeSeverity
		}
	case "enter", "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

func (m *Model) handleSmokingStatusKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < 1 {
			m.configField++
		}
	case "space":
		if m.configField == 0 {
			m.pipelineConfig.SmokingStatusEnabled = !m.pipelineConfig.SmokingStatusEnabled
		}
		if m.configField == 1 {
			m.pipelineConfig.SmokingStatusConfig.IncludeAmount = !m.pipelineConfig.SmokingStatusConfig.IncludeAmount
		}
	case "enter", "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

func (m *Model) handleTemplateFillingKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < 1 {
			m.configField++
		}
	case "space":
		if m.configField == 0 {
			m.pipelineConfig.TemplateFillingEnabled = !m.pipelineConfig.TemplateFillingEnabled
		}
		if m.configField == 1 {
			m.pipelineConfig.TemplateFillingConfig.UseConstraints = !m.pipelineConfig.TemplateFillingConfig.UseConstraints
		}
	case "enter", "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

// Handle tokenization configuration
func (m *Model) handleTokenizationKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < 8 { // Number of tokenization fields
			m.configField++
		}
	case "space":
		// Toggle boolean fields
		switch m.configField {
		case 0:
			m.pipelineConfig.TokenizationEnabled = !m.pipelineConfig.TokenizationEnabled
		case 3:
			m.pipelineConfig.TokenizationConfig.KeepNewlines = !m.pipelineConfig.TokenizationConfig.KeepNewlines
		case 4:
			m.pipelineConfig.TokenizationConfig.SplitHyphens = !m.pipelineConfig.TokenizationConfig.SplitHyphens
		case 7:
			m.pipelineConfig.TokenizationConfig.PreserveWhitespace = !m.pipelineConfig.TokenizationConfig.PreserveWhitespace
		case 8:
			m.pipelineConfig.TokenizationConfig.HandleAbbreviations = !m.pipelineConfig.TokenizationConfig.HandleAbbreviations
		}
	case "left", "h":
		// Decrease numeric values
		switch m.configField {
		case 5:
			if m.pipelineConfig.TokenizationConfig.MinTokenLength > 1 {
				m.pipelineConfig.TokenizationConfig.MinTokenLength--
			}
		case 6:
			if m.pipelineConfig.TokenizationConfig.MaxTokenLength > 10 {
				m.pipelineConfig.TokenizationConfig.MaxTokenLength--
			}
		}
	case "right", "l":
		// Increase numeric values
		switch m.configField {
		case 5:
			if m.pipelineConfig.TokenizationConfig.MinTokenLength < 10 {
				m.pipelineConfig.TokenizationConfig.MinTokenLength++
			}
		case 6:
			if m.pipelineConfig.TokenizationConfig.MaxTokenLength < 100 {
				m.pipelineConfig.TokenizationConfig.MaxTokenLength++
			}
		}
	case "enter":
		m.pipelineState = PipelineMainMenu
	case "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

// Handle POS tagging configuration
func (m *Model) handlePOSTaggingKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < 5 {
			m.configField++
		}
	case "space":
		switch m.configField {
		case 0:
			m.pipelineConfig.POSTaggingEnabled = !m.pipelineConfig.POSTaggingEnabled
		case 3:
			m.pipelineConfig.POSTaggingConfig.UseContextualCues = !m.pipelineConfig.POSTaggingConfig.UseContextualCues
		case 4:
			m.pipelineConfig.POSTaggingConfig.HandleUnknownWords = !m.pipelineConfig.POSTaggingConfig.HandleUnknownWords
		case 5:
			m.pipelineConfig.POSTaggingConfig.CaseSensitive = !m.pipelineConfig.POSTaggingConfig.CaseSensitive
		}
	case "enter":
		m.pipelineState = PipelineMainMenu
	case "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

// Handle NER configuration
func (m *Model) handleNERKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	entityTypes := []string{
		"Diseases/Disorders", "Signs/Symptoms", "Procedures",
		"Medications", "Anatomical Sites", "Lab Values",
	}

	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < len(entityTypes)+4 { // Entity types + settings
			m.configField++
		}
	case "space":
		if m.configField < len(entityTypes) {
			// Toggle entity type selection
			entityType := entityTypes[m.configField]
			if utils.Contains(m.pipelineConfig.NERConfig.EntityTypes, entityType) {
				// Remove it
				newTypes := []string{}
				for _, t := range m.pipelineConfig.NERConfig.EntityTypes {
					if t != entityType {
						newTypes = append(newTypes, t)
					}
				}
				m.pipelineConfig.NERConfig.EntityTypes = newTypes
			} else {
				// Add it
				m.pipelineConfig.NERConfig.EntityTypes = append(m.pipelineConfig.NERConfig.EntityTypes, entityType)
			}
		} else {
			// Toggle boolean settings
			offset := m.configField - len(entityTypes)
			switch offset {
			case 0:
				m.pipelineConfig.NERConfig.UseContextWindow = !m.pipelineConfig.NERConfig.UseContextWindow
			case 4:
				m.pipelineConfig.NERConfig.CaseSensitive = !m.pipelineConfig.NERConfig.CaseSensitive
			}
		}
	case "left", "h":
		offset := m.configField - len(entityTypes)
		switch offset {
		case 1: // Window size
			if m.pipelineConfig.NERConfig.WindowSize > 5 {
				m.pipelineConfig.NERConfig.WindowSize--
			}
		case 2: // Min entity length
			if m.pipelineConfig.NERConfig.MinEntityLength > 1 {
				m.pipelineConfig.NERConfig.MinEntityLength--
			}
		case 3: // Max entity length
			if m.pipelineConfig.NERConfig.MaxEntityLength > 10 {
				m.pipelineConfig.NERConfig.MaxEntityLength--
			}
		}
	case "right", "l":
		offset := m.configField - len(entityTypes)
		switch offset {
		case 1: // Window size
			if m.pipelineConfig.NERConfig.WindowSize < 50 {
				m.pipelineConfig.NERConfig.WindowSize++
			}
		case 2: // Min entity length
			if m.pipelineConfig.NERConfig.MinEntityLength < 10 {
				m.pipelineConfig.NERConfig.MinEntityLength++
			}
		case 3: // Max entity length
			if m.pipelineConfig.NERConfig.MaxEntityLength < 100 {
				m.pipelineConfig.NERConfig.MaxEntityLength++
			}
		}
	case "enter":
		m.pipelineState = PipelineMainMenu
	case "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

// Handle dictionary lookup configuration
func (m *Model) handleDictionaryLookupKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	dictionaries := m.getAvailableDictionaries()
	algorithms := []string{"Exact Match", "Fuzzy Match", "Permutation Match"}

	totalItems := len(dictionaries) + len(algorithms) + 5 + 2 // + settings + indexing toggles

	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < totalItems-1 {
			m.configField++
		}
	case "space":
		if m.configField < len(dictionaries) {
			// Select dictionary
			selected := dictionaries[m.configField]
			m.pipelineConfig.DictionaryLookupConfig.DictionaryPath = selected.Path
			// Also set SelectedDictionary fields for runPipeline
			m.pipelineConfig.SelectedDictionaryName = selected.Name
			xmlPath := selected.Path
			if !strings.HasSuffix(strings.ToLower(xmlPath), ".xml") {
				cand := filepath.Join(xmlPath, "dictionary.xml")
				if _, err := os.Stat(cand); err == nil {
					xmlPath = cand
				}
			}
			m.pipelineConfig.SelectedDictionaryPath = xmlPath
		} else if m.configField < len(dictionaries)+len(algorithms) {
			// Select algorithm
			algIndex := m.configField - len(dictionaries)
			m.pipelineConfig.DictionaryLookupConfig.LookupAlgorithm = algorithms[algIndex]
		} else {
			// Toggle boolean settings
			offset := m.configField - len(dictionaries) - len(algorithms)
			switch offset {
			case 0:
				m.pipelineConfig.DictionaryLookupConfig.CaseSensitive = !m.pipelineConfig.DictionaryLookupConfig.CaseSensitive
			case 3:
				m.pipelineConfig.DictionaryLookupConfig.ExcludeNumbers = !m.pipelineConfig.DictionaryLookupConfig.ExcludeNumbers
			case 5:
				// Use Lucene index
				m.pipelineConfig.DictionaryLookupConfig.UseLuceneIndex = !m.pipelineConfig.DictionaryLookupConfig.UseLuceneIndex
			case 6:
				// Use HSQL dictionary
				m.pipelineConfig.DictionaryLookupConfig.UseHsqlDictionary = !m.pipelineConfig.DictionaryLookupConfig.UseHsqlDictionary
			}
		}
	case "left", "h":
		offset := m.configField - len(dictionaries) - len(algorithms)
		switch offset {
		case 1: // Min match length
			if m.pipelineConfig.DictionaryLookupConfig.MinMatchLength > 1 {
				m.pipelineConfig.DictionaryLookupConfig.MinMatchLength--
			}
		case 2: // Max permutations
			if m.pipelineConfig.DictionaryLookupConfig.MaxPermutations > 1 {
				m.pipelineConfig.DictionaryLookupConfig.MaxPermutations--
			}
		case 4: // Max lookup size
			if m.pipelineConfig.DictionaryLookupConfig.MaxLookupTextSize > 100 {
				m.pipelineConfig.DictionaryLookupConfig.MaxLookupTextSize -= 100
			}
		}
		// no left/right adjustments for index toggles
	case "right", "l":
		offset := m.configField - len(dictionaries) - len(algorithms)
		switch offset {
		case 1: // Min match length
			if m.pipelineConfig.DictionaryLookupConfig.MinMatchLength < 20 {
				m.pipelineConfig.DictionaryLookupConfig.MinMatchLength++
			}
		case 2: // Max permutations
			if m.pipelineConfig.DictionaryLookupConfig.MaxPermutations < 10 {
				m.pipelineConfig.DictionaryLookupConfig.MaxPermutations++
			}
		case 4: // Max lookup size
			if m.pipelineConfig.DictionaryLookupConfig.MaxLookupTextSize < 10240 {
				m.pipelineConfig.DictionaryLookupConfig.MaxLookupTextSize += 100
			}
		}
		// no left/right adjustments for index toggles
	case "enter":
		m.pipelineState = PipelineMainMenu
	case "s", "S":
		// Save and return for consistency with other editors
		m.pipelineState = PipelineMainMenu
	case "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

// Handle assertion configuration
func (m *Model) handleAssertionKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < 8 {
			m.configField++
		}
	case "space":
		switch m.configField {
		case 0:
			m.pipelineConfig.AssertionEnabled = !m.pipelineConfig.AssertionEnabled
		case 8:
			m.pipelineConfig.AssertionConfig.UseSectionHeaders = !m.pipelineConfig.AssertionConfig.UseSectionHeaders
		}
	case "left", "h":
		if m.configField == 7 { // Scope window size
			if m.pipelineConfig.AssertionConfig.ScopeWindowSize > 5 {
				m.pipelineConfig.AssertionConfig.ScopeWindowSize--
			}
		}
	case "right", "l":
		if m.configField == 7 { // Scope window size
			if m.pipelineConfig.AssertionConfig.ScopeWindowSize < 50 {
				m.pipelineConfig.AssertionConfig.ScopeWindowSize++
			}
		}
	case "enter":
		m.pipelineState = PipelineMainMenu
	case "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

// Handle output configuration
func (m *Model) handlePipelineOutputKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	formats := []string{"XMI", "JSON", "FHIR", "TSV", "CSV", "XML", "Plain Text"}
	totalItems := len(formats) + 6 // + settings

	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < totalItems-1 {
			m.configField++
		}
	case "space":
		if m.configField < len(formats) {
			// Multi-select formats: store primary in Format, but allow toggling IncludeOriginalText et al
			m.pipelineConfig.OutputConfig.Format = formats[m.configField]
		} else {
			// Toggle boolean settings
			offset := m.configField - len(formats)
			switch offset {
			case 1:
				m.pipelineConfig.OutputConfig.IncludeMetadata = !m.pipelineConfig.OutputConfig.IncludeMetadata
			case 2:
				m.pipelineConfig.OutputConfig.PrettyPrint = !m.pipelineConfig.OutputConfig.PrettyPrint
			case 3:
				m.pipelineConfig.OutputConfig.CompressOutput = !m.pipelineConfig.OutputConfig.CompressOutput
			case 4:
				m.pipelineConfig.OutputConfig.SplitBySection = !m.pipelineConfig.OutputConfig.SplitBySection
			case 5:
				m.pipelineConfig.OutputConfig.IncludeOriginalText = !m.pipelineConfig.OutputConfig.IncludeOriginalText
			}
		}
	case "s", "S":
		// Save and return quickly
		m.pipelineState = PipelineMainMenu
		return *m, nil
	case "enter":
		m.pipelineState = PipelineMainMenu
	case "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

// Handle runtime configuration
func (m *Model) handlePipelineRuntimeKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	logLevels := []string{"ERROR", "WARN", "INFO", "DEBUG", "TRACE"}
	numericFields := 6
	boolFields := 1
	totalItems := numericFields + boolFields + len(logLevels)

	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < totalItems-1 {
			m.configField++
		}
	case "space":
		if m.configField == 6 { // Enable profiling
			m.pipelineConfig.RuntimeConfig.EnableProfiling = !m.pipelineConfig.RuntimeConfig.EnableProfiling
		} else if m.configField >= 7 {
			// Select log level
			levelIndex := m.configField - 7
			if levelIndex < len(logLevels) {
				m.pipelineConfig.RuntimeConfig.LogLevel = logLevels[levelIndex]
			}
		}
	case "left", "h":
		switch m.configField {
		case 0: // Initial heap
			if m.pipelineConfig.RuntimeConfig.InitialHeapSize > 512 {
				m.pipelineConfig.RuntimeConfig.InitialHeapSize -= 256
			}
		case 1: // Max heap
			if m.pipelineConfig.RuntimeConfig.MaxHeapSize > 1024 {
				m.pipelineConfig.RuntimeConfig.MaxHeapSize -= 256
			}
		case 2: // Thread pool
			if m.pipelineConfig.RuntimeConfig.ThreadPoolSize > 1 {
				m.pipelineConfig.RuntimeConfig.ThreadPoolSize--
			}
		case 3: // Batch size
			if m.pipelineConfig.RuntimeConfig.BatchSize > 1 {
				m.pipelineConfig.RuntimeConfig.BatchSize--
			}
		case 4: // Max doc size
			if m.pipelineConfig.RuntimeConfig.MaxDocumentSize > 100 {
				m.pipelineConfig.RuntimeConfig.MaxDocumentSize -= 100
			}
		case 5: // Timeout
			if m.pipelineConfig.RuntimeConfig.TimeoutSeconds > 30 {
				m.pipelineConfig.RuntimeConfig.TimeoutSeconds -= 30
			}
		}
	case "right", "l":
		switch m.configField {
		case 0: // Initial heap
			if m.pipelineConfig.RuntimeConfig.InitialHeapSize < 4096 {
				m.pipelineConfig.RuntimeConfig.InitialHeapSize += 256
			}
		case 1: // Max heap
			if m.pipelineConfig.RuntimeConfig.MaxHeapSize < 16384 {
				m.pipelineConfig.RuntimeConfig.MaxHeapSize += 256
			}
		case 2: // Thread pool
			if m.pipelineConfig.RuntimeConfig.ThreadPoolSize < 32 {
				m.pipelineConfig.RuntimeConfig.ThreadPoolSize++
			}
		case 3: // Batch size
			if m.pipelineConfig.RuntimeConfig.BatchSize < 1000 {
				m.pipelineConfig.RuntimeConfig.BatchSize++
			}
		case 4: // Max doc size
			if m.pipelineConfig.RuntimeConfig.MaxDocumentSize < 10240 {
				m.pipelineConfig.RuntimeConfig.MaxDocumentSize += 100
			}
		case 5: // Timeout
			if m.pipelineConfig.RuntimeConfig.TimeoutSeconds < 3600 {
				m.pipelineConfig.RuntimeConfig.TimeoutSeconds += 30
			}
		}
	case "enter", "s", "S":
		m.pipelineState = PipelineMainMenu
	case "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

// Handle saved pipeline viewer
func (m *Model) handlePipelineViewerKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.dictListCursor > 0 { // Reuse dict cursor for pipelines
			m.dictListCursor--
		}
	case "down", "j":
		if m.dictListCursor < len(m.savedPipelines)-1 {
			m.dictListCursor++
		}
	case "enter":
		// Load selected pipeline
		if m.dictListCursor < len(m.savedPipelines) {
			// Load pipeline configuration
			m.pipelineState = PipelineMainMenu
		}
	case "d":
		// Delete selected pipeline
		if m.dictListCursor < len(m.savedPipelines) {
			// Delete pipeline
			m.loadSavedPipelines()
		}
	case "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

// Handle pipeline build process
func (m *Model) handlePipelineBuildKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.pipelineState = PipelineMainMenu
	case "l":
		// View full logs
		// Implementation would show detailed build logs
	}
	return *m, nil
}

// Handle pipeline run process
func (m *Model) handlePipelineRunKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Stop pipeline run
		m.pipelineState = PipelineMainMenu
		// Clean up logger
		if m.buildLogger != nil {
			_ = m.buildLogger.CloseWithSummary()
			m.buildLogger = nil
		}
	case "up", "k":
		m.buildViewport.LineUp(1)
	case "down", "j":
		m.buildViewport.LineDown(1)
	case "pgup":
		m.buildViewport.ViewUp()
	case "pgdown":
		m.buildViewport.ViewDown()
	case "home":
		m.buildViewport.GotoTop()
	case "end":
		m.buildViewport.GotoBottom()
	case "p":
		// Pause/resume (not implemented)
	}
	return *m, nil
}

// Handle select dictionary (built descriptor picker)
func (m *Model) handleSelectDictionaryKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "s", "S":
		// Choose first available descriptor in priority order; can extend to a full list later
		candidates := []struct{ Name, Path string }{
			{"Diagnoses", filepath.Join("dictionaries", "Diagnoses", "dictionary.xml")},
			{"Laboratory", filepath.Join("dictionaries", "Laboratory", "dictionary.xml")},
		}
		for _, c := range candidates {
			if _, err := os.Stat(c.Path); err == nil {
				m.pipelineConfig.SelectedDictionaryPath = c.Path
				m.pipelineConfig.SelectedDictionaryName = c.Name
				break
			}
		}
		m.pipelineState = PipelineMainMenu
	case "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

// Generic handler for unimplemented states
func (m *Model) handleGenericPipelineKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.pipelineState = PipelineMainMenu
	}
	return *m, nil
}

// Helper functions
func (m *Model) isPipelineReadyToBuild() bool {
	// Check if at least one component is enabled
	return m.pipelineConfig.TokenizationEnabled ||
		m.pipelineConfig.POSTaggingEnabled ||
		m.pipelineConfig.NEREnabled ||
		m.pipelineConfig.DictionaryLookupEnabled
}

func (m *Model) isPipelineReadyToRun() bool {
	// Check if template, input, and output are configured
	hasInputs := len(m.pipelineConfig.InputDirs) > 0 || m.pipelineConfig.InputDir != ""
	hasTemplate := m.pipelineConfig.Name != "" && m.pipelineConfig.Name != "Not selected"
	return hasTemplate && hasInputs && m.pipelineConfig.OutputDir != ""
}

func (m *Model) loadSavedPipelines() {
	// Load saved pipeline configurations from disk
	m.savedPipelines = []PipelineInfo{
		{
			Name:        "Clinical Notes Pipeline",
			Description: "Standard clinical processing",
			Path:        "./pipelines/clinical.xml",
			Components:  12,
			Status:      "Ready",
		},
		{
			Name:        "Research Pipeline",
			Description: "Full research configuration",
			Path:        "./pipelines/research.xml",
			Components:  18,
			Status:      "Ready",
		},
	}
}

func (m *Model) buildPipeline() tea.Cmd {
	return func() tea.Msg {
		// Generate pipeline configuration XML/properties
		return pipelineBuildCompleteMsg{}
	}
}

func (m *Model) runPipeline() tea.Cmd {
	// Start async pipeline run with logger and progress polling
	// Initialize logger
	m.buildProgress = 0
	m.buildLogs = []string{"=== Pipeline Run Started ==="}
	m.buildStartTime = time.Now()
	m.buildError = nil
	m.lastLogIndex = 0
	m.buildState = BuildState{StartTime: time.Now(), Stage: "Initializing"}

	// Determine dictionary directory for logs, default to pipelines/logs
	logDir := ""
	if p := m.pipelineConfig.SelectedDictionaryPath; p != "" {
		logDir = filepath.Join(filepath.Dir(p), "logs")
	} else if p := m.pipelineConfig.DictionaryLookupConfig.DictionaryPath; p != "" {
		logDir = filepath.Join(p, "logs")
	} else if m.pipelineConfig.OutputDir != "" {
		logDir = filepath.Join(m.pipelineConfig.OutputDir, "logs")
	} else {
		logDir = filepath.Join("pipelines", "logs")
	}
	_ = os.MkdirAll(logDir, 0755)
	// Name logs by run
	runName := m.pipelineConfig.RunName
	if strings.TrimSpace(runName) == "" {
		runName = time.Now().Format("20060102-150405")
	}
	runName = strings.ReplaceAll(runName, " ", "_")
	runLogPath := filepath.Join(logDir, "pipeline-"+runName+".log")
	debugLogPath := filepath.Join(logDir, "pipeline-"+runName+".raw.log")

	// Start logger and attach files
	logger, _ := dictionary.NewBuildLogger(runLogPath)
	_ = logger.AddFile(debugLogPath)
	logger.SetMinLevel(dictionary.LogDebug)
	m.buildLogger = logger

	// Initialize viewport if not already done
	if m.buildViewport.Width == 0 || m.buildViewport.Height == 0 {
		vp := viewport.New(80, 20)
		vp.HighPerformanceRendering = false
		m.buildViewport = vp
	}
	// Initialize viewport content for live logs
	m.buildViewport.SetContent(strings.Join(m.buildLogs, "\n"))

	// Start async run
	go func() {
		defer func() {
			if m.buildLogger != nil {
				_ = m.buildLogger.CloseWithSummary()
			}
		}()

		logger.StartStage("prepare")
		logger.Info("Preparing pipeline and resources", 0)

		// Collect input directories
		inputs := m.pipelineConfig.InputDirs
		if len(inputs) == 0 && m.pipelineConfig.InputDir != "" {
			inputs = []string{m.pipelineConfig.InputDir}
		}
		// Count files
		total := 0
		for _, dir := range inputs {
			_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() {
					total++
				}
				return nil
			})
		}
		logger.Metric("input_files", total)

		integ, err := dictionary.NewCTAKESIntegration(logger)
		if err != nil {
			logger.Error("CTAKES integration failed", err)
			logger.StartStage("abort")
			logger.Warning("Aborting pipeline due to initialization failure", map[string]interface{}{"error": err.Error()})
			return
		}

		// Resolve Piper file
		piper := m.pipelineConfig.PiperFilePath
		if piper == "" {
			// Try dictionary-generated pipelines
			candidates := []string{
				filepath.Join("dictionaries", "Diagnoses", "pipeline.piper"),
				filepath.Join("dictionaries", "Laboratory", "pipeline.piper"),
			}
			for _, c := range candidates {
				if _, err := os.Stat(c); err == nil {
					piper = c
					break
				}
			}
		}
		if piper == "" {
			piper = filepath.Join("apache-ctakes-6.0.0-bin", "apache-ctakes-6.0.0", "resources", "org", "apache", "ctakes", "clinical", "pipeline", "DefaultFastPipeline.piper")
			logger.Warning("No .piper selected; falling back to DefaultFastPipeline", map[string]interface{}{"piper": piper})
		}

		// If Piper supports LookupXml via -l and we detect a built dictionary, append it
		// Prefer the dictionary whose pipeline was selected; else the first available
		var lookupXml string
		// If user selected a dictionary explicitly, use it
		if m.pipelineConfig.SelectedDictionaryPath != "" {
			if _, err := os.Stat(m.pipelineConfig.SelectedDictionaryPath); err == nil {
				lookupXml = m.pipelineConfig.SelectedDictionaryPath
			}
		}
		// Otherwise, try pipeline dictionary lookup config
		if lookupXml == "" {
			base := m.pipelineConfig.DictionaryLookupConfig.DictionaryPath
			if base != "" {
				cand := base
				if !strings.HasSuffix(strings.ToLower(cand), ".xml") {
					cand = filepath.Join(base, "dictionary.xml")
				}
				if _, err := os.Stat(cand); err == nil {
					lookupXml = cand
				}
			}
		}
		// Otherwise, try defaults
		if lookupXml == "" {
			dictCandidates := []string{
				filepath.Join("dictionaries", "Diagnoses", "dictionary.xml"),
				filepath.Join("dictionaries", "Laboratory", "dictionary.xml"),
			}
			for _, dc := range dictCandidates {
				if _, err := os.Stat(dc); err == nil {
					lookupXml = dc
					break
				}
			}
			if lookupXml == "" {
				logger.Warning("No dictionary.xml found; running without LookupXML", nil)
			}
		}

		// Build run config
		cfg := dictionary.PipelineRunConfig{
			PiperFile:   piper,
			InputDir:    "",
			OutputDir:   m.pipelineConfig.OutputDir,
			InitialHeap: fmt.Sprintf("%dM", utils.Max(512, m.pipelineConfig.RuntimeConfig.InitialHeapSize)),
			MaxHeap:     fmt.Sprintf("%dM", utils.Max(1024, m.pipelineConfig.RuntimeConfig.MaxHeapSize)),
		}
		if lookupXml != "" {
			cfg.LookupXml = lookupXml
		}

		// Ensure output base exists
		_ = os.MkdirAll(m.pipelineConfig.OutputDir, 0755)

		// Run per input (mirroring structure if enabled)
		for _, in := range inputs {
			out := m.pipelineConfig.OutputDir
			if m.pipelineConfig.MirrorOutputStructure {
				base := filepath.Base(in)
				out = filepath.Join(out, base)
				_ = os.MkdirAll(out, 0755)
			}
			cfg.InputDir = in
			cfg.OutputDir = out
			stageName := fmt.Sprintf("run_%s", filepath.Base(in))
			logger.StartStage(stageName)
			logger.Debug("Starting piper run", map[string]interface{}{"input": in, "output": out, "piper": cfg.PiperFile, "lookupXml": cfg.LookupXml})
			err := integ.RunPiper(cfg, func(stage, message string, progress float64) {
				if progress >= 0 {
					logger.Info(message, progress)
				} else {
					logger.Info(message, -1)
				}
			})
			if err != nil {
				_ = logger.Fatal("Pipeline failed", err)
				break
			}
		}

		logger.StartStage("done")
		logger.Info("Pipeline run complete", 1.0)
	}()

	// Begin polling logs/progress
	return buildTickEvery()
}

// Render a concise pipeline configuration preview for the preview panel
func (m *Model) renderPipelinePreview(width, height int) string {
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Render("Pipeline Preview"),
		strings.Repeat("─", utils.Max(10, width-4)),
		"",
	}
	// Template and components
	tmpl := m.pipelineConfig.Name
	if tmpl == "" {
		tmpl = "(not selected)"
	}
	lines = append(lines, fmt.Sprintf("Template: %s", tmpl))
	comp := []string{}
	if m.pipelineConfig.TokenizationEnabled {
		comp = append(comp, "Tokenization")
	}
	if m.pipelineConfig.POSTaggingEnabled {
		comp = append(comp, "POS")
	}
	if m.pipelineConfig.ChunkingEnabled {
		comp = append(comp, "Chunking")
	}
	if m.pipelineConfig.NEREnabled {
		comp = append(comp, "NER")
	}
	if m.pipelineConfig.DictionaryLookupEnabled {
		comp = append(comp, "Dictionary")
	}
	if m.pipelineConfig.AssertionEnabled {
		comp = append(comp, "Assertion")
	}
	if m.pipelineConfig.RelationExtractionEnabled {
		comp = append(comp, "Relation")
	}
	if m.pipelineConfig.TemporalEnabled {
		comp = append(comp, "Temporal")
	}
	if m.pipelineConfig.CoreferenceEnabled {
		comp = append(comp, "Coref")
	}
	if m.pipelineConfig.DrugNEREnabled {
		comp = append(comp, "DrugNER")
	}
	if m.pipelineConfig.SideEffectEnabled {
		comp = append(comp, "SideEffects")
	}
	if m.pipelineConfig.SmokingStatusEnabled {
		comp = append(comp, "Smoking")
	}
	if m.pipelineConfig.TemplateFillingEnabled {
		comp = append(comp, "TemplateFill")
	}
	lines = append(lines, fmt.Sprintf("Components: %s", strings.Join(comp, ", ")))
	// IO
	inCount := len(m.pipelineConfig.InputDirs)
	if inCount == 0 && m.pipelineConfig.InputDir != "" {
		inCount = 1
	}
	lines = append(lines,
		fmt.Sprintf("Inputs: %d", inCount),
		fmt.Sprintf("Output: %s", utils.TruncateString(m.pipelineConfig.OutputDir, width-12)),
		fmt.Sprintf("Mirror: %v", m.pipelineConfig.MirrorOutputStructure),
	)
	// Piper path (resolved guess)
	piper := m.pipelineConfig.PiperFilePath
	if piper == "" {
		if _, err := os.Stat(filepath.Join("dictionaries", "Diagnoses", "pipeline.piper")); err == nil {
			piper = "dictionaries/Diagnoses/pipeline.piper"
		}
	}
	if piper != "" {
		lines = append(lines, fmt.Sprintf("Piper: %s", utils.TruncateString(piper, width-10)))
	}

	// If running, add simple progress info
	if m.pipelineState == PipelineRunning {
		pct := int(m.buildProgress * 100)
		barWidth := utils.Max(10, width-12)
		filled := int(float64(barWidth) * m.buildProgress)
		bar := lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render(strings.Repeat("█", filled)) +
			lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render(strings.Repeat("░", barWidth-filled))
		lines = append(lines, "", bar, fmt.Sprintf("%d%%", pct), "", "Recent:")
		maxLogs := utils.Max(3, height-len(lines)-2)
		start := 0
		if len(m.buildLogs) > maxLogs {
			start = len(m.buildLogs) - maxLogs
		}
		for _, log := range m.buildLogs[start:] {
			lines = append(lines, utils.TruncateString("  "+log, width-2))
		}
	}

	// Constrain
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

type pipelineBuildCompleteMsg struct{}
type pipelineRunCompleteMsg struct{}

// Render pipeline configuration panel - simplified
func (m *Model) renderPipelineConfigPanel(width, height int) string {
	switch m.pipelineState {
	case PipelineMainMenu:
		return m.renderPipelineMenu(width, height)
	case PipelineTemplateEditor:
		return m.renderPipelineTemplateEditor(width, height)
	case PipelineSelectingInputDirs:
		return m.renderInputDirSelector(width, height)
	case PipelineSelectingOutputDir:
		return m.renderOutputDirSelector(width, height)
	case PipelineDictionaryConfig:
		return m.renderDictionaryLookupConfig(width, height)
	case PipelineEditingRunName:
		return m.renderRunNameEditor(width, height)
	case PipelineSelectingTemplate:
		return m.renderPipelineTemplateSelector(width, height)
	case PipelineOutputConfig:
		return m.renderPipelineOutputConfig(width, height)
	case PipelineRuntimeConfig:
		return m.renderPipelineRuntimeConfig(width, height)
	case PipelineRunning:
		return m.renderPipelineRunView(width, height)
	default:
		return m.renderPipelineMenu(width, height)
	}
}

// Render template selector with clean circle-based design
func (m *Model) renderPipelineTemplateSelector(width, height int) string {
	header := theme.RenderSelectionHeader("Pipeline Templates", m.pipelineTemplateCursor+1, len(m.pipelineTemplates), width)
	headerLines := strings.Count(header, "\n") + 1
	legend := lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render("Legend: Clinical=Blue, Performance=Green, Specialized=Purple, Basic=Orange, Research=Cyan")
	lines := []string{header, legend, ""}

	footer := theme.RenderSelectionFooter(width, "[Enter] Apply  [Esc] Back")
	footerLines := strings.Count(footer, "\n") + 1
	visible := height - headerLines - footerLines - 2
	if visible < 5 {
		visible = 5
	}

	// Group templates by category
	groups := map[string][]PipelineTemplate{}
	order := []string{}
	for _, t := range m.pipelineTemplates {
		groups[t.Category] = append(groups[t.Category], t)
	}
	// Preserve insertion order of first occurrence
	seen := map[string]bool{}
	for _, t := range m.pipelineTemplates {
		if !seen[t.Category] {
			order = append(order, t.Category)
			seen[t.Category] = true
		}
	}

	items := []string{}
	index := 0
	for _, cat := range order {
		// Category header
		color := theme.ColorSecondary
		switch strings.ToLower(cat) {
		case "clinical":
			color = theme.ColorAccent
		case "performance":
			color = theme.ColorSuccess
		case "specialized":
			color = theme.ColorAccent
		case "basic":
			color = theme.ColorWarning
		case "research":
			color = theme.ColorInfo
		}
		headerLine := lipgloss.NewStyle().Foreground(color).Bold(true).Render(cat)
		items = append(items, headerLine)
		for _, t := range groups[cat] {
			isFocused := index == m.pipelineTemplateCursor
			indicator := "⚫"
			if isFocused {
				indicator = "🔵"
			}
			line := fmt.Sprintf("  %s %s %s", indicator, t.Icon, t.Name)
			if isFocused {
				items = append(items, theme.RenderSelectableRow(line, width, false, true))
			} else {
				items = append(items, theme.RenderSelectableRow(line, width, false, false))
			}
			// Compact description line
			items = append(items, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render("    "+t.Description))
			index++
		}
		items = append(items, lipgloss.NewStyle().Foreground(theme.ColorBorderInactive).Render(strings.Repeat("─", width)))
	}

	// Map template index to its visual title line in items
	templateLineIdx := make([]int, 0, len(m.pipelineTemplates))
	idx := 0
	for _, cat := range order {
		idx++ // category header already added above for each cat
		for range groups[cat] {
			// title line at current idx
			templateLineIdx = append(templateLineIdx, idx)
			idx += 2 // title + description
		}
		idx++ // divider line
	}

	// Maintain scroll window so highlight only scrolls at edges
	if len(items) > visible && m.pipelineTemplateCursor < len(templateLineIdx) {
		target := templateLineIdx[m.pipelineTemplateCursor]
		// Scroll up if above window
		if target < m.pipelineTemplateScrollStart {
			m.pipelineTemplateScrollStart = target
		}
		// Scroll down if below window
		edge := m.pipelineTemplateScrollStart + visible - 1
		if target > edge {
			m.pipelineTemplateScrollStart = target - (visible - 1)
		}
		// Clamp
		maxStart := utils.Max(0, len(items)-visible)
		if m.pipelineTemplateScrollStart < 0 {
			m.pipelineTemplateScrollStart = 0
		}
		if m.pipelineTemplateScrollStart > maxStart {
			m.pipelineTemplateScrollStart = maxStart
		}
	} else {
		m.pipelineTemplateScrollStart = 0
	}

	startIdx := m.pipelineTemplateScrollStart
	endIdx := utils.Clamp(startIdx+visible, 0, len(items))
	lines = append(lines, items[startIdx:endIdx]...)
	if len(items) > visible {
		// Compute visible template range for indicator (not line count)
		tStart := 0
		for i, li := range templateLineIdx {
			if li >= startIdx {
				tStart = i
				break
			}
		}
		tEnd := len(m.pipelineTemplates)
		for i := len(templateLineIdx) - 1; i >= 0; i-- {
			if templateLineIdx[i] <= endIdx-1 {
				tEnd = i + 1 // exclusive
				break
			}
		}
		if tStart < 0 {
			tStart = 0
		}
		if tEnd < tStart {
			tEnd = tStart
		}
		lines = append(lines, theme.RenderScrollIndicator(tStart, tEnd, len(m.pipelineTemplates), width))
	}
	lines = append(lines, footer)
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// Render POS tagging configuration
func (m *Model) renderPOSTaggingConfig(width, height int) string {
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Render("Part-of-Speech Tagging Configuration"),
		"",
		lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render("Configure grammatical tagging"),
		"",
	}

	fields := []struct {
		name  string
		value interface{}
		desc  string
	}{
		{"Enabled", m.pipelineConfig.POSTaggingEnabled, "Enable POS tagging"},
		{"Model Path", m.pipelineConfig.POSTaggingConfig.ModelPath, "Path to POS model"},
		{"Tag Set", m.pipelineConfig.POSTaggingConfig.TagSet, "Tagging scheme"},
		{"Use Context", m.pipelineConfig.POSTaggingConfig.UseContextualCues, "Use surrounding context"},
		{"Handle Unknown", m.pipelineConfig.POSTaggingConfig.HandleUnknownWords, "Process unknown words"},
		{"Case Sensitive", m.pipelineConfig.POSTaggingConfig.CaseSensitive, "Respect case in tagging"},
	}

	for i, field := range fields {
		focused := i == m.configField

		valueStr := fmt.Sprintf("%v", field.value)
		if boolVal, ok := field.value.(bool); ok {
			valueStr = utils.CBox(boolVal)
		}

		line := fmt.Sprintf("%-20s: %-30s", field.name, valueStr)
		lines = append(lines, theme.RenderSelectableRow(utils.TruncateString(line, width-4), width, false, focused))
		if field.desc != "" {
			desc := utils.TruncateString("  "+field.desc, width-4)
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render(desc))
		}
	}

	lines = append(lines, "",
		"↑/↓: Navigate  Space: Toggle  Enter: Save  ESC: Back")

	lines = utils.ClipToHeight(lines, height-2)
	return lipgloss.NewStyle().Width(width).Height(height).
		Render(strings.Join(lines, "\n"))
}

// Render assertion configuration
func (m *Model) renderAssertionConfig(width, height int) string {
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Render("Assertion Analysis Configuration"),
		"",
		lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render("Configure negation, uncertainty, and subject detection"),
		"",
	}

	fields := []struct {
		name  string
		value interface{}
		desc  string
	}{
		{"Enabled", m.pipelineConfig.AssertionEnabled, "Enable assertion analysis"},
		{"Polarity Model", m.pipelineConfig.AssertionConfig.PolarityModelPath, "Negation detection model"},
		{"Uncertainty Model", m.pipelineConfig.AssertionConfig.UncertaintyModelPath, "Uncertainty detection"},
		{"Subject Model", m.pipelineConfig.AssertionConfig.SubjectModelPath, "Subject identification"},
		{"Generic Model", m.pipelineConfig.AssertionConfig.GenericModelPath, "Generic assertions"},
		{"History Model", m.pipelineConfig.AssertionConfig.HistoryModelPath, "Historical context"},
		{"Conditional Model", m.pipelineConfig.AssertionConfig.ConditionalModelPath, "Conditional statements"},
		{"Scope Window", m.pipelineConfig.AssertionConfig.ScopeWindowSize, "Context window size"},
		{"Use Sections", m.pipelineConfig.AssertionConfig.UseSectionHeaders, "Consider section headers"},
	}

	for i, field := range fields {
		focused := i == m.configField

		valueStr := fmt.Sprintf("%v", field.value)
		if boolVal, ok := field.value.(bool); ok {
			valueStr = utils.CBox(boolVal)
		}

		line := fmt.Sprintf("%-20s: %-30s", field.name, valueStr)
		lines = append(lines, theme.RenderSelectableRow(utils.TruncateString(line, width-4), width, false, focused))
	}

	lines = append(lines, "",
		"↑/↓: Navigate  Space: Toggle  ←/→: Adjust  Enter: Save  ESC: Back")

	lines = utils.ClipToHeight(lines, height-2)
	return lipgloss.NewStyle().Width(width).Height(height).
		Render(strings.Join(lines, "\n"))
}

// Render saved pipelines
func (m *Model) renderSavedPipelines(width, height int) string {
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Render("Saved Pipeline Configurations"),
		"",
	}

	if len(m.savedPipelines) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).
			Render("No saved pipelines found"))
	} else {
		for i, pipeline := range m.savedPipelines {
			focused := i == m.dictListCursor
			status := lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render(pipeline.Status)
			line := fmt.Sprintf("%-30s %d components  %s",
				pipeline.Name, pipeline.Components, status)
			lines = append(lines, theme.RenderSelectableRow(utils.TruncateString(line, width-4), width, false, focused))

			desc := utils.TruncateString("  "+pipeline.Description, width-4)
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render(desc))

			path := utils.TruncateString("  "+pipeline.Path, width-4)
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render(path))
			lines = append(lines, "")
		}
	}

	lines = append(lines, "",
		"↑/↓: Navigate  Enter: Load  d: Delete  ESC: Back")

	lines = utils.ClipToHeight(lines, height-2)
	return lipgloss.NewStyle().Width(width).Height(height).
		Render(strings.Join(lines, "\n"))
}

// Render pipeline build view
func (m *Model) renderPipelineBuildView(width, height int) string {
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Render("Building Pipeline Configuration"),
		"",
	}

	// Show build progress
	lines = append(lines, "Generating pipeline configuration...")
	lines = append(lines, "")

	// List enabled components
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Enabled Components:"))
	if m.pipelineConfig.TokenizationEnabled {
		lines = append(lines, fmt.Sprintf("  %s Tokenization", theme.GetSemanticIcon("success")))
	}
	if m.pipelineConfig.POSTaggingEnabled {
		lines = append(lines, fmt.Sprintf("  %s POS Tagging", theme.GetSemanticIcon("success")))
	}
	if m.pipelineConfig.ChunkingEnabled {
		lines = append(lines, fmt.Sprintf("  %s Chunking", theme.GetSemanticIcon("success")))
	}
	if m.pipelineConfig.NEREnabled {
		lines = append(lines, fmt.Sprintf("  %s Named Entity Recognition", theme.GetSemanticIcon("success")))
	}
	if m.pipelineConfig.DictionaryLookupEnabled {
		lines = append(lines, fmt.Sprintf("  %s Dictionary Lookup", theme.GetSemanticIcon("success")))
	}
	if m.pipelineConfig.AssertionEnabled {
		lines = append(lines, fmt.Sprintf("  %s Assertion Analysis", theme.GetSemanticIcon("success")))
	}

	lines = append(lines, "", lipgloss.NewStyle().Bold(true).Render("Output Configuration:"))
	lines = append(lines, fmt.Sprintf("  Format: %s", m.pipelineConfig.OutputConfig.Format))
	lines = append(lines, fmt.Sprintf("  Directory: %s", m.pipelineConfig.OutputConfig.OutputDirectory))

	lines = append(lines, "",
		"l: View Logs  ESC: Back")

	lines = utils.ClipToHeight(lines, height-2)
	return lipgloss.NewStyle().Width(width).Height(height).
		Render(strings.Join(lines, "\n"))
}

// Render pipeline run view - simplified
func (m *Model) renderPipelineRunView(width, height int) string {
	header := theme.RenderSelectionHeader("Running Pipeline", 0, 0, width)
	lines := []string{header, ""}

	// Show what's running
	if m.pipelineConfig.Name != "" {
		lines = append(lines, fmt.Sprintf("Template: %s", m.pipelineConfig.Name))
	}
	// Show inputs summary
	inputSummary := m.pipelineConfig.InputDir
	if len(m.pipelineConfig.InputDirs) > 0 {
		if len(m.pipelineConfig.InputDirs) == 1 {
			inputSummary = m.pipelineConfig.InputDirs[0]
		} else {
			inputSummary = fmt.Sprintf("%d input directories", len(m.pipelineConfig.InputDirs))
		}
	}
	lines = append(lines, fmt.Sprintf("Input:    %s", inputSummary))
	lines = append(lines, fmt.Sprintf("Output:   %s", m.pipelineConfig.OutputDir))
	lines = append(lines, "")

	// Progress bar
	pct := int(m.buildProgress * 100)
	barWidth := utils.Max(10, width-12)
	filled := int(float64(barWidth) * m.buildProgress)
	bar := lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render(strings.Repeat("░", barWidth-filled))
	lines = append(lines, bar, fmt.Sprintf("%d%%  Stage: %s", pct, m.buildState.Stage), "")

	// Live log viewport
	m.buildViewport.SetContent(strings.Join(m.buildLogs, "\n"))
	vp := m.buildViewport.View()
	lines = append(lines, vp)

	footer := theme.RenderSelectionFooter(width, "[Esc] Stop  [PgUp/PgDn] Scroll  [↑/↓] Scroll")
	lines = append(lines, "", footer)

	return lipgloss.NewStyle().Width(width).Height(height).
		Render(strings.Join(lines, "\n"))
}
