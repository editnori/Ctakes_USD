package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
	"github.com/ctakes-tui/ctakes-tui/internal/utils"
)

// Main menu rendering and navigation
func (m *Model) renderDictMenu(width, height int) string {
	// Remove the local title - it will be handled by the main header
	rows := []string{}

	// Templates - use consistent header styling
	rows = append(rows, theme.SubHeaderStyle.Render("Quick Start:"))
	templateIcon := theme.GetSemanticIcon("special")
	templateItem := m.renderDictMenuItem(0 == m.dictMenuCursor, templateIcon, "Templates", getTemplateSummary(m), width)
	rows = append(rows, templateItem)

	// Basic configuration - use consistent header styling
	rows = append(rows, "", theme.SubHeaderStyle.Render("Basic Configuration:"))
	items := []string{
		m.renderDictMenuItem(1 == m.dictMenuCursor, theme.GetSemanticIcon("info"), "View Dictionaries", m.countBuiltDicts(), width),
		m.renderDictMenuItem(2 == m.dictMenuCursor, theme.GetSemanticIcon("folder"), "UMLS Location", baseOr(m.umlsPath, "Not selected"), width),
		m.renderDictMenuItem(3 == m.dictMenuCursor, theme.GetSemanticIcon("default"), "Name", baseOr(m.dictConfig.Name, "Unset"), width),
		m.renderDictMenuItem(4 == m.dictMenuCursor, theme.GetSemanticIcon("data"), "Semantic Types", fmt.Sprintf("%d selected", len(m.dictConfig.TUIs)), width),
		m.renderDictMenuItem(5 == m.dictMenuCursor, theme.GetSemanticIcon("info"), "Vocabularies", fmt.Sprintf("%d selected", len(m.dictConfig.Vocabularies)), width),
	}
	rows = append(rows, items...)

	// Advanced configuration - use consistent header styling
	rows = append(rows, "", theme.SubHeaderStyle.Render("Advanced Configuration:"))
	advItems := []string{
		m.renderDictMenuItem(6 == m.dictMenuCursor, theme.GetSemanticIcon("special"), "Memory Settings", fmt.Sprintf("%d MB heap", m.dictConfig.MaxHeapMB), width),
		m.renderDictMenuItem(7 == m.dictMenuCursor, theme.GetSemanticIcon("active"), "Processing Options", fmt.Sprintf("%d threads", m.dictConfig.ThreadCount), width),
		m.renderDictMenuItem(8 == m.dictMenuCursor, theme.GetSemanticIcon("config"), "Filter Configuration", getFilterSummary(m), width),
		m.renderDictMenuItem(9 == m.dictMenuCursor, theme.GetSemanticIcon("data"), "Output Formats", getOutputSummary(m), width),
		m.renderDictMenuItem(10 == m.dictMenuCursor, theme.GetSemanticIcon("config"), "Cased Dictionary", getCasedSummary(m), width),
		m.renderDictMenuItem(11 == m.dictMenuCursor, theme.GetSemanticIcon("info"), "Relationships", getRelationshipSummary(m), width),
	}
	rows = append(rows, advItems...)

	// Build action - use consistent header styling
	rows = append(rows, "", theme.SubHeaderStyle.Render("Actions:"))
	buildIcon := theme.GetSemanticIcon("success")
	buildItem := m.renderDictMenuItem(12 == m.dictMenuCursor, buildIcon, "Build Dictionary", buildStatus(m), width)
	rows = append(rows, buildItem)

	// Remove internal navigation help - it's already shown in the main footer

	// Prevent overflow: clip with a scroll window around the selected item
	// Map cursor (0..12) to its visual row index (adjusted for removed navigation)
	selMap := map[int]int{
		0:  1,  // Templates
		1:  4,  // View Dictionaries
		2:  5,  // UMLS Location
		3:  6,  // Name
		4:  7,  // TUIs
		5:  8,  // Vocabularies
		6:  11, // Memory
		7:  12, // Processing
		8:  13, // Filters
		9:  14, // Output
		10: 15, // Cased Dictionary
		11: 16, // Relationships
		12: 19, // Build
	}
	selectedRow := selMap[m.dictMenuCursor]

	maxLines := height - 2 // inner height target
	if maxLines < 5 {
		maxLines = 5
	}

	if len(rows) > maxLines {
		// Compute start so selected stays visible
		start := 0
		high := maxLines - 2 // leave room for footer
		if selectedRow >= high {
			start = selectedRow - (high - 1)
		}
		// Ensure we don't cut the pinned header if near top
		if start < 0 {
			start = 0
		}
		// Ensure footer is retained
		end := start + maxLines
		if end > len(rows) {
			end = len(rows)
			start = end - maxLines
			if start < 0 {
				start = 0
			}
		}
		// Always include last line (footer) by adjusting window if needed
		if end < len(rows) {
			// Shift window so the last line is visible
			shift := len(rows) - end
			start -= shift
			if start < 0 {
				start = 0
			}
			end = len(rows)
		}
		rows = rows[start:end]
	}

	// Use full height without additional padding since the panel already has proper dimensions
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(rows, "\n"))
}

// Helper functions for summary text
func getFilterSummary(m *Model) string {
	activeFilters := 0
	if m.dictConfig.ExcludeSuppressible {
		activeFilters++
	}
	if m.dictConfig.ExcludeObsolete {
		activeFilters++
	}
	if m.dictConfig.UseNormalization {
		activeFilters++
	}
	if m.dictConfig.UseMRRANK {
		activeFilters++
	}
	if m.dictConfig.Deduplicate {
		activeFilters++
	}
	return fmt.Sprintf("%d filters active", activeFilters)
}

func getOutputSummary(m *Model) string {
	formats := 1 // BSV is always included
	if m.dictConfig.BuildHSQLDB {
		formats++
	}
	if m.dictConfig.BuildLucene {
		formats++
	}
	if m.dictConfig.EmitTSV {
		formats++
	}
	if m.dictConfig.EmitJSONL {
		formats++
	}
	return fmt.Sprintf("%d formats", formats)
}

// getTemplateSummary returns the currently selected template display name, or a default hint
func getTemplateSummary(m *Model) string {
	// Prefer a friendly display name when dictConfig.Name matches a known template
	if strings.TrimSpace(m.dictConfig.Name) != "" {
		for _, t := range m.dictTemplates {
			if strings.EqualFold(strings.TrimSpace(t.Config.Name), strings.TrimSpace(m.dictConfig.Name)) {
				return t.Name
			}
		}
		// Fallback to config name if not found in the current template list
		return m.dictConfig.Name
	}
	return "Pre-configured setups"
}

func getRelationshipSummary(m *Model) string {
	if m.dictConfig.EnableRelationships {
		return fmt.Sprintf("Enabled (%d types)", len(m.dictConfig.RelationshipTypes))
	}
	return "Disabled"
}

func getCasedSummary(m *Model) string {
	if m.dictConfig.BuildCasedDictionary {
		options := 0
		if m.dictConfig.IncludeAcronyms {
			options++
		}
		if m.dictConfig.IncludeAbbreviations {
			options++
		}
		return fmt.Sprintf("Enabled (%s, %d opts)", m.dictConfig.CasedTermRanking, options)
	}
	return "Disabled"
}

func (m *Model) handleMenuKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.dictMenuCursor > 0 {
			m.dictMenuCursor--
		}
	case "down", "j":
		if m.dictMenuCursor < 12 { // Updated for 13 menu items (0-12)
			m.dictMenuCursor++
		}
	case "enter", " ":
		switch m.dictMenuCursor {
		case 0: // Templates
			m.dictBuilderState = DictStateSelectingTemplate
			if len(m.dictTemplates) == 0 {
				m.initTemplates()
			}
			m.templateCursor = 0
		case 1: // View Dictionaries
			m.dictBuilderState = DictStateViewingDictionaries
			m.loadBuiltDictionaries()
		case 2: // UMLS Location
			m.dictBuilderState = DictStateSelectUMLS
			// default to ./umls_loader if exists
			if m.umlsPath != "" {
				m.currentPath = m.umlsPath
			} else if fi, err := os.Stat("umls_loader"); err == nil && fi.IsDir() {
				cwd, _ := os.Getwd()
				m.currentPath = filepath.Join(cwd, "umls_loader")
			}
			// Use a lightweight directory-only listing to reduce lag in UMLS selection
			m.updateFileListUMLS()
			// Sync table with the new listing so cursor/page-up/down work consistently
			m.updateTables()
		case 3: // Name
			m.dictBuilderState = DictStateEditingName
			m.dictNameInput.SetValue(m.dictConfig.Name)
			m.dictNameInput.Focus()
		case 4: // Semantic Types
			m.dictBuilderState = DictStateSelectingTUIs
			if len(m.tuiList) == 0 {
				m.initTUIList()
			}
			if len(m.dictConfig.TUIs) == 0 {
				m.dictConfig.TUIs = []string{}
			}
		case 5: // Vocabularies
			m.dictBuilderState = DictStateSelectingVocabs
			if len(m.vocabList) == 0 {
				m.initVocabList()
			}
			if len(m.dictConfig.Vocabularies) == 0 {
				m.dictConfig.Vocabularies = []string{}
			}
		case 6: // Memory Settings
			m.dictBuilderState = DictStateMemoryConfig
			m.configField = 0
			// Set defaults if not set
			if m.dictConfig.InitialHeapMB == 0 {
				m.dictConfig.InitialHeapMB = 1024
			}
			if m.dictConfig.MaxHeapMB == 0 {
				m.dictConfig.MaxHeapMB = 2048
			}
			if m.dictConfig.StackSizeMB == 0 {
				m.dictConfig.StackSizeMB = 8
			}
		case 7: // Processing Options
			m.dictBuilderState = DictStateProcessingConfig
			m.configField = 0
			// Set defaults if not set
			if m.dictConfig.ThreadCount == 0 {
				m.dictConfig.ThreadCount = 4
			}
			if m.dictConfig.BatchSize == 0 {
				m.dictConfig.BatchSize = 1000
			}
			if m.dictConfig.CacheSize == 0 {
				m.dictConfig.CacheSize = 128
			}
			if m.dictConfig.MinWordLength == 0 {
				m.dictConfig.MinWordLength = 2
			}
			if m.dictConfig.MaxWordLength == 0 {
				m.dictConfig.MaxWordLength = 80
			}
		case 8: // Filter Configuration
			m.dictBuilderState = DictStateFilterConfig
			m.configField = 0
			// Set defaults if not set
			if m.dictConfig.MinTermLength == 0 {
				m.dictConfig.MinTermLength = 3
			}
			if m.dictConfig.MaxTermLength == 0 {
				m.dictConfig.MaxTermLength = 80
			}
		case 9: // Output Formats
			m.dictBuilderState = DictStateOutputConfig
			m.configField = 0
			// Set default outputs
			if !m.dictConfig.EmitDescriptor && !m.dictConfig.EmitPipeline && !m.dictConfig.EmitManifest {
				m.dictConfig.EmitDescriptor = true
				m.dictConfig.EmitPipeline = true
				m.dictConfig.EmitManifest = true
			}
		case 10: // Cased Dictionary
			m.dictBuilderState = DictStateCasedConfig
			m.configField = 0
			if m.dictConfig.CasedTermRanking == "" {
				m.dictConfig.CasedTermRanking = "mrrank"
			}
		case 11: // Relationships
			m.dictBuilderState = DictStateRelationshipConfig
			m.configField = 0
			if m.dictConfig.RelationshipDepth == 0 {
				m.dictConfig.RelationshipDepth = 2
			}
		case 12: // Build Dictionary
			if isReadyToBuild(m) {
				// Start build in the build view (not full log view)
				m.dictBuilderState = DictStateBuilding
				return *m, m.startBuild()
			}
		}
	case "esc":
		// stay in dashboard
	case "q", "Q":
		// Quit dictionary builder - return to main dashboard
		m.activePanel = SidebarPanel
		m.cursor = 0
	case "p", "P":
		// Switch to pipeline view
		m.activePanel = SidebarPanel
		m.cursor = 5 // Pipeline is item 5 in the sidebar (0-indexed)
	}
	return *m, nil
}

// Helper functions for menu
func menuLine(selected bool, label, value string) string {
	if selected {
		return lipgloss.NewStyle().
			Background(theme.ColorSelectionActive).
			Foreground(theme.ColorBackground).
			Bold(true).
			Render(fmt.Sprintf("%-30s %s", label, value))
	}
	return fmt.Sprintf("%-30s %s", label, value)
}

// renderDictMenuItem renders a menu item with consistent styling
func (m *Model) renderDictMenuItem(selected bool, icon, label, value string, width int) string {
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
	return theme.RenderSelectableRow(content, width, false, selected)
}
func buildStatus(m *Model) string {
	if isReadyToBuild(m) {
		return "Ready"
	}
	return "Configure first"
}

func isReadyToBuild(m *Model) bool {
	return m.umlsPath != "" && m.dictConfig.Name != "" && len(m.dictConfig.TUIs) > 0 && len(m.dictConfig.Vocabularies) > 0
}

func baseOr(path, alt string) string {
	if strings.TrimSpace(path) == "" {
		return alt
	}
	return filepath.Base(path)
}

// renderDictConfigPreview shows the current configuration in the preview panel
func (m *Model) renderDictConfigPreview(width, height int) string {
	lines := []string{
		fmt.Sprintf("%s Dictionary Builder", theme.GetSemanticIcon("special")),
		strings.Repeat("─", utils.Max(4, width-4)),
		"",
		fmt.Sprintf("UMLS: %s", baseOr(m.umlsPath, "Not selected")),
		fmt.Sprintf("Name: %s", baseOr(m.dictConfig.Name, "Unset")),
		fmt.Sprintf("TUIs: %d", len(m.dictConfig.TUIs)),
		fmt.Sprintf("Vocabularies: %d", len(m.dictConfig.Vocabularies)),
	}
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}
