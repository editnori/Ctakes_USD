package dashboard

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/dictionary"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
)

func (m *Model) renderDictionaryBuilderPanel(width, height int) string {
	switch m.dictBuilderState {
	case DictStateSelectUMLS:
		// Show embedded file browser for UMLS selection
		return m.renderDictUMLSBrowser(width, height)
	case DictStateEditingName:
		// Show text input for dictionary name
		return m.renderDictNameInput(width, height)
	case DictStateSelectingTUIs:
		// Show TUI selector (like file browser)
		return m.renderTUISelector(width, height)
	case DictStateSelectingVocabs:
		// Show vocabulary selector
		return m.renderVocabSelector(width, height)
		// Legacy aliases handled via direct states below
	// New sub-menu states with proper interactive controls
	case DictStateMemoryConfig:
		return m.renderMemoryConfig(width, height)
	case DictStateProcessingConfig:
		return m.renderProcessingConfig(width, height)
	case DictStateFilterConfig:
		return m.renderFilterConfig(width, height)
	case DictStateOutputConfig:
		return m.renderOutputConfig(width, height)
	case DictStateRelationshipConfig:
		return m.renderRelationshipConfig(width, height)
	case DictStateBuilding:
		// Show build progress popup
		return m.renderBuildProgressPopup(width, height)
	case DictStateViewingDictionaries:
		// Show list of built dictionaries
		return m.renderDictionaryViewer(width, height)
	case DictStateConfiguring:
		// Show configuration menu after UMLS is selected
		return m.renderDictConfigMenu(width, height)
	default:
		// Show main dictionary builder menu (home page)
		return m.renderDictMainMenu(width, height)
	}
}

func (m *Model) renderDictMainMenu(width, height int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorAccent).
		MarginBottom(1)

	title := titleStyle.Render("Dictionary Builder")

	// Initialize dictionary options if not already done
	if len(m.dictOptions) == 0 {
		m.initDictOptions()
	}

	// Update the table to show main menu options
	// Use more height for the table to ensure all options are accessible
	tableHeight := height - 6 // Leave less margin for header/footer
	if tableHeight > 15 {
		tableHeight = 15 // Cap at reasonable height but allow scrolling
	}
	m.updateDictTable(width-4, tableHeight)

	var content []string
	content = append(content, title)
	content = append(content, "")

	// Show UMLS path if already selected
	if m.umlsPath != "" {
		pathStyle := lipgloss.NewStyle().
			Foreground(theme.ColorSuccess)
		content = append(content, pathStyle.Render(fmt.Sprintf("✓ UMLS: %s", filepath.Base(m.umlsPath))))
		content = append(content, pathStyle.Render(fmt.Sprintf("  RRF Files: %d found", len(m.rrfFiles))))
		content = append(content, "")
	}

	// Render the main menu options as a table
	content = append(content, m.dictTable.View())

	return strings.Join(content, "\n")
}

func (m *Model) renderDictUMLSBrowser(width, height int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorAccent).
		MarginBottom(1)

	instructionStyle := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary)

	var content []string
	content = append(content, titleStyle.Render("Select UMLS Directory"))
	content = append(content, instructionStyle.Render("Navigate to directory with RRF files and press Enter"))
	content = append(content, "")

	// Check for RRF files in current directory
	rrfFiles := m.detectRRFFiles(m.currentPath)
	if len(rrfFiles) > 0 {
		successStyle := lipgloss.NewStyle().
			Foreground(theme.ColorSuccess)
		content = append(content, successStyle.Render(fmt.Sprintf("✓ Found %d RRF files here", len(rrfFiles))))
	}
	content = append(content, "")

	// Show embedded file browser
	files := m.files
	if len(files) > 0 {
		// Create a simple file list view
		for i, file := range files {
			if i > height-15 {
				break // Limit display
			}

			icon := "📄"
			if file.IsDir {
				icon = "📁"
			}

			line := fmt.Sprintf("%s %s", icon, file.Name)
			if i == m.fileTable.Cursor() {
				selectedStyle := lipgloss.NewStyle().
					Foreground(theme.ColorBackground).
					Background(theme.ColorAccent)
				line = "▶ " + selectedStyle.Render(line)
			} else {
				line = "  " + line
			}
			content = append(content, line)
		}
	}

	content = append(content, "")
	content = append(content, instructionStyle.Render("[↑↓] Navigate • [Enter] Select Directory • [Backspace] Go Up • [ESC] Cancel"))

	return strings.Join(content, "\n")
}

func (m *Model) renderDictConfigMenu(width, height int) string {
	// This is now the same as the main menu
	return m.renderDictMainMenu(width, height)
}

func (m *Model) initDictOptions() {
	// Initialize default values if not set
	if m.dictConfig.InitialHeapMB == 0 {
		m.dictConfig.InitialHeapMB = 1024
	}
	if m.dictConfig.MaxHeapMB == 0 {
		m.dictConfig.MaxHeapMB = 2048
	}
	if m.dictConfig.StackSizeMB == 0 {
		m.dictConfig.StackSizeMB = 8
	}
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
	if m.dictConfig.MinTermLength == 0 {
		m.dictConfig.MinTermLength = 3
	}
	if m.dictConfig.MaxTermLength == 0 {
		m.dictConfig.MaxTermLength = 80
	}
	if m.dictConfig.MinTokens == 0 {
		m.dictConfig.MinTokens = 1
	}
	if m.dictConfig.MaxTokens == 0 {
		m.dictConfig.MaxTokens = 5
	}
	if m.dictConfig.RelationshipDepth == 0 {
		m.dictConfig.RelationshipDepth = 2
	}
	if !m.dictConfig.EmitBSV {
		m.dictConfig.EmitBSV = true // Default on
	}

	m.dictOptions = []DictOption{
		// Main Actions
		{Name: "View Built Dictionaries", Value: "", Type: "action", Status: "ready"},
		{Name: "Select UMLS Location", Value: "", Type: "action", Status: "pending"},
		{Name: "Dictionary Name", Value: "MyDictionary", Type: "config", Status: "pending"},

		// Core Configuration
		{Name: "Semantic Types (TUIs)", Value: "0 selected", Type: "config", Status: "pending"},
		{Name: "Source Vocabularies", Value: "0 selected", Type: "config", Status: "pending"},
		{Name: "Languages", Value: "ENG", Type: "config", Status: "configured"},
		{Name: "Term Types", Value: "PT, SY", Type: "config", Status: "configured"},

		// Advanced Settings Header
		{Name: "─── Advanced Settings ───", Value: "", Type: "header", Status: "info"},

		// Memory Configuration
		{Name: "Memory Settings", Value: "", Type: "advanced", Status: "pending"},

		// Processing Configuration
		{Name: "Processing Options", Value: "", Type: "advanced", Status: "pending"},

		// Filter Configuration
		{Name: "Filter Options", Value: "", Type: "advanced", Status: "pending"},

		// Output Configuration
		{Name: "Output Formats", Value: "", Type: "advanced", Status: "pending"},

		// Relationships Configuration
		{Name: "Relationship Settings", Value: "", Type: "advanced", Status: "pending"},

		// Presets
		{Name: "─── Quick Presets ───", Value: "", Type: "header", Status: "info"},
		{Name: "Apply Clinical Preset", Value: "", Type: "preset", Status: "ready"},
		{Name: "Apply Medications Preset", Value: "", Type: "preset", Status: "ready"},

		// Build Action
		{Name: "─── Build ───", Value: "", Type: "header", Status: "info"},
		{Name: "Build Dictionary", Value: "", Type: "action", Status: "pending"},
	}

	// Update values based on current state
	dictCount := m.countBuiltDictionaries()
	if dictCount > 0 {
		m.dictOptions[0].Value = fmt.Sprintf("%d found", dictCount)
	} else {
		m.dictOptions[0].Value = "None found"
	}

	if m.umlsPath != "" {
		m.dictOptions[1].Value = filepath.Base(m.umlsPath)
		m.dictOptions[1].Status = "configured"
	}

	if m.dictConfig.Name != "" {
		m.dictOptions[2].Value = m.dictConfig.Name
		m.dictOptions[2].Status = "configured"
	}

	if len(m.dictConfig.TUIs) > 0 {
		m.dictOptions[3].Value = fmt.Sprintf("%d selected", len(m.dictConfig.TUIs))
		m.dictOptions[3].Status = "configured"
	}

	if len(m.dictConfig.Vocabularies) > 0 {
		m.dictOptions[4].Value = fmt.Sprintf("%d selected", len(m.dictConfig.Vocabularies))
		m.dictOptions[4].Status = "configured"
	}

	if len(m.dictConfig.Languages) > 0 {
		m.dictOptions[5].Value = strings.Join(m.dictConfig.Languages, ", ")
		m.dictOptions[5].Status = "configured"
	}

	if len(m.dictConfig.TermTypes) > 0 {
		m.dictOptions[6].Value = strings.Join(m.dictConfig.TermTypes, ", ")
		m.dictOptions[6].Status = "configured"
	}

	// Update Memory Settings status
	m.dictOptions[8].Value = fmt.Sprintf("Heap: %d-%d MB, Stack: %d MB",
		m.dictConfig.InitialHeapMB, m.dictConfig.MaxHeapMB, m.dictConfig.StackSizeMB)
	m.dictOptions[8].Status = "configured"

	// Update Processing Options status
	m.dictOptions[9].Value = fmt.Sprintf("Threads: %d, Batch: %d, Cache: %d MB",
		m.dictConfig.ThreadCount, m.dictConfig.BatchSize, m.dictConfig.CacheSize)
	m.dictOptions[9].Status = "configured"

	// Update Filter Options status
	filterCount := 0
	if m.dictConfig.ExcludeSuppressible {
		filterCount++
	}
	if m.dictConfig.ExcludeObsolete {
		filterCount++
	}
	if m.dictConfig.UseNormalization {
		filterCount++
	}
	if m.dictConfig.UseMRRANK {
		filterCount++
	}
	if m.dictConfig.Deduplicate {
		filterCount++
	}
	if m.dictConfig.PreferredOnly {
		filterCount++
	}
	m.dictOptions[10].Value = fmt.Sprintf("%d filters enabled", filterCount)
	m.dictOptions[10].Status = "configured"

	// Update Output Formats status
	outputCount := 0
	if m.dictConfig.EmitBSV {
		outputCount++
	}
	if m.dictConfig.BuildHSQLDB {
		outputCount++
	}
	if m.dictConfig.BuildLucene {
		outputCount++
	}
	if m.dictConfig.EmitTSV {
		outputCount++
	}
	if m.dictConfig.EmitJSONL {
		outputCount++
	}
	if m.dictConfig.EmitDescriptor {
		outputCount++
	}
	if m.dictConfig.EmitPipeline {
		outputCount++
	}
	if m.dictConfig.EmitManifest {
		outputCount++
	}
	m.dictOptions[11].Value = fmt.Sprintf("%d outputs enabled", outputCount)
	m.dictOptions[11].Status = "configured"

	// Update Relationships status
	if m.dictConfig.EnableRelationships {
		m.dictOptions[12].Value = fmt.Sprintf("Enabled, depth: %d, types: %d",
			m.dictConfig.RelationshipDepth, len(m.dictConfig.RelationshipTypes))
	} else {
		m.dictOptions[12].Value = "Disabled"
	}
	m.dictOptions[12].Status = "configured"

	// Enable build if ready
	if m.umlsPath != "" && m.dictConfig.Name != "" {
		m.dictOptions[17].Status = "ready"
	}
}

func (m *Model) updateDictTable(width, height int) {
	// Only recreate the table if it doesn't exist yet or if dimensions changed significantly
	if m.dictTable.Width() == 0 || abs(m.dictTable.Width()-width) > 5 || abs(m.dictTable.Height()-height) > 2 {
		m.createDictTable(width, height)
	} else {
		// Just update the rows data for better performance
		m.updateDictTableRows()
	}
}

func (m *Model) createDictTable(width, height int) {
	// Create table columns - simpler layout
	columns := []table.Column{
		{Title: "", Width: 2}, // Status indicator
		{Title: "Option", Width: 35},
		{Title: "Value", Width: width - 40},
	}

	// Store current cursor position
	currentCursor := 0
	if m.dictTable.Width() > 0 {
		currentCursor = m.dictTable.Cursor()
	}

	// Create table with theme styling
	t := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}), // Empty rows initially
		table.WithFocused(true),
		table.WithHeight(height),
	)

	// Apply table styles with custom header styling
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.ColorBorder).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(theme.ColorBackground).
		Background(theme.ColorAccent).
		Bold(false)

	t.SetStyles(s)
	m.dictTable = t

	// Update rows and restore cursor
	m.updateDictTableRows()
	if currentCursor < len(m.dictOptions) {
		// Skip header rows when restoring cursor
		for currentCursor < len(m.dictOptions) && m.dictOptions[currentCursor].Type == "header" {
			currentCursor++
		}
		if currentCursor < len(m.dictOptions) {
			m.dictTable.SetCursor(currentCursor)
		}
	}
}

func (m *Model) updateDictTableRows() {
	// Convert options to table rows
	var rows []table.Row
	for _, opt := range m.dictOptions {
		statusIcon := " "
		if opt.Type == "header" {
			statusIcon = ""
		} else if opt.Status == "configured" {
			statusIcon = "✓"
		} else if opt.Status == "ready" {
			statusIcon = "▸"
		} else if opt.Status == "info" {
			statusIcon = "ℹ"
		}

		value := opt.Value
		if value == "" {
			if opt.Type == "preset" {
				value = "Apply preset configuration"
			} else if opt.Name == "Build Dictionary" {
				if opt.Status == "ready" {
					value = "Ready to build"
				} else {
					value = "Configure options first"
				}
			} else if opt.Type == "header" {
				value = ""
			} else if opt.Type == "advanced" {
				value = "Configure advanced settings"
			} else {
				value = "Not configured"
			}
		}

		rows = append(rows, table.Row{statusIcon, opt.Name, value})
	}

	// Store current cursor position
	currentCursor := m.dictTable.Cursor()

	// Update rows efficiently
	m.dictTable.SetRows(rows)

	// Restore cursor position if valid, skipping header rows
	if currentCursor < len(rows) {
		// Skip header rows when restoring cursor
		for currentCursor < len(rows) && m.dictOptions[currentCursor].Type == "header" {
			currentCursor++
		}
		if currentCursor < len(rows) {
			m.dictTable.SetCursor(currentCursor)
		}
	}
}

// Helper function to calculate absolute difference
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (m *Model) renderDictPreview(width, height int) string {
	// Special case: if building, show raw terminal logs in preview
	if m.dictBuilderState == DictStateBuilding {
		return m.renderBuildTerminalLogs(width, height)
	}

	// Get current selection from table
	if m.dictTable.Cursor() >= len(m.dictOptions) {
		return m.renderDictHelp(width, height)
	}

	selectedOption := m.dictOptions[m.dictTable.Cursor()]

	var content []string

	switch selectedOption.Name {
	case "View Built Dictionaries":
		content = m.renderDictionariesPreview()
	case "Select UMLS Location":
		content = m.renderUMLSPreview()
	case "Dictionary Name":
		content = m.renderNameConfig()
	case "Semantic Types (TUIs)":
		content = m.renderTUIConfig()
	case "Source Vocabularies":
		content = m.renderVocabularyConfig()
	case "Apply Clinical Preset":
		content = m.renderClinicalPresetPreview()
	case "Apply Medications Preset":
		content = m.renderMedicationsPresetPreview()
	case "Build Dictionary":
		// When not building, show preview
		content = m.renderBuildPreview()
	default:
		content = []string{"Select an option to configure"}
	}

	// Join content
	fullContent := strings.Join(content, "\n")

	// If content is longer than available height, use viewport for scrolling
	if strings.Count(fullContent, "\n") > height-4 {
		// Initialize or update viewport
		if m.previewViewport.Width != width-2 || m.previewViewport.Height != height-4 {
			m.previewViewport = viewport.New(width-2, height-4)
		}
		m.previewViewport.SetContent(fullContent)

		// Apply padding style to viewport output
		viewportStyle := lipgloss.NewStyle().Padding(1)
		return viewportStyle.Render(m.previewViewport.View())
	}

	// Otherwise render normally with padding
	previewStyle := lipgloss.NewStyle().
		Width(width - 2).
		Height(height - 2).
		Padding(1)
	return previewStyle.Render(fullContent)
}

func (m *Model) renderDictHelp(width, height int) string {
	helpStyle := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Width(width - 2).
		Padding(1)

	help := []string{
		"Dictionary Builder Help",
		strings.Repeat("─", width-4),
		"",
		"Navigation:",
		"  ↑/↓     - Navigate options",
		"  Enter   - Select/Configure option",
		"  Tab     - Switch to preview panel",
		"  ESC     - Go back",
		"",
		"Configuration Steps:",
		"  1. Select UMLS directory with RRF files",
		"  2. Set dictionary name",
		"  3. Choose semantic types (TUIs)",
		"  4. Select vocabularies",
		"  5. Pick languages",
		"  6. Select term types",
		"  7. Build dictionary",
		"",
		"Presets:",
		"  Apply clinical or medications preset",
		"  for quick configuration",
	}

	return helpStyle.Render(strings.Join(help, "\n"))
}

func (m *Model) renderUMLSPreview() []string {
	var content []string

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorAccent)

	content = append(content, titleStyle.Render("UMLS Location Details"))
	content = append(content, "")

	if m.umlsPath != "" {
		content = append(content, fmt.Sprintf("Path: %s", m.umlsPath))
		content = append(content, "")
		content = append(content, "RRF Files Found:")

		for _, file := range m.rrfFiles {
			content = append(content, fmt.Sprintf("  • %s", file))
		}
	} else {
		content = append(content, "No UMLS location selected")
		content = append(content, "")
		content = append(content, "Navigate to your UMLS directory")
		content = append(content, "and press [U] to select it")
	}

	return content
}

func (m *Model) renderNameConfig() []string {
	var content []string

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorAccent)

	content = append(content, titleStyle.Render("Dictionary Name"))
	content = append(content, "")

	if m.dictConfig.Name != "" {
		content = append(content, fmt.Sprintf("Current: %s", m.dictConfig.Name))
	} else {
		content = append(content, "Not configured")
	}

	content = append(content, "")
	content = append(content, "Press Enter to set dictionary name")

	return content
}

func (m *Model) renderTUIConfig() []string {
	var content []string

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorAccent)

	content = append(content, titleStyle.Render("Semantic Types (TUIs)"))
	content = append(content, "")

	// Complete list of TUIs for name lookup
	tuiMap := map[string]string{
		"T005": "Virus",
		"T007": "Bacterium",
		"T017": "Anatomical Structure",
		"T019": "Congenital Abnormality",
		"T020": "Acquired Abnormality",
		"T033": "Finding",
		"T037": "Injury or Poisoning",
		"T046": "Pathologic Function",
		"T047": "Disease or Syndrome",
		"T048": "Mental or Behavioral Dysfunction",
		"T059": "Laboratory Procedure",
		"T060": "Diagnostic Procedure",
		"T061": "Therapeutic or Preventive Procedure",
		"T074": "Medical Device",
		"T109": "Organic Chemical",
		"T116": "Amino Acid, Peptide, or Protein",
		"T121": "Pharmacologic Substance",
		"T122": "Biomedical or Dental Material",
		"T123": "Biologically Active Substance",
		"T184": "Sign or Symptom",
		"T190": "Anatomical Abnormality",
		"T191": "Neoplastic Process",
		"T195": "Antibiotic",
		"T200": "Clinical Drug",
		"T201": "Clinical Attribute",
	}

	// Show selected count
	if len(m.dictConfig.TUIs) > 0 {
		content = append(content, fmt.Sprintf("Selected: %d types", len(m.dictConfig.TUIs)))
	} else {
		content = append(content, "No types selected")
	}
	content = append(content, "")
	content = append(content, "Press Enter to select types")
	content = append(content, "")

	// Show ALL selected TUIs - no truncation
	if len(m.dictConfig.TUIs) > 0 {
		content = append(content, "Selected TUIs:")
		content = append(content, "")
		for _, tui := range m.dictConfig.TUIs {
			// Find the name for this TUI
			name, exists := tuiMap[tui]
			if exists {
				content = append(content, fmt.Sprintf("  ✓ %s - %s", tui, name))
			} else {
				content = append(content, fmt.Sprintf("  ✓ %s", tui))
			}
		}
	}

	return content
}

func (m *Model) renderVocabularyConfig() []string {
	var content []string

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorAccent)

	content = append(content, titleStyle.Render("Source Vocabularies"))
	content = append(content, "")

	// Vocabulary name lookup map
	vocabMap := map[string]string{
		"SNOMEDCT_US": "SNOMED CT US Edition",
		"RXNORM":      "RxNorm",
		"ICD10CM":     "ICD-10-CM",
		"ICD10PCS":    "ICD-10-PCS",
		"ICD9CM":      "ICD-9-CM",
		"LOINC":       "LOINC",
		"CPT":         "Current Procedural Terminology",
		"HCPCS":       "Healthcare Common Procedure Coding System",
		"NDC":         "National Drug Code",
		"VANDF":       "Veterans Health Administration National Drug File",
		"NDDF":        "National Drug Data File",
		"CVX":         "Vaccines Administered",
		"MVX":         "Vaccine Manufacturers",
		"SOP":         "Source of Payment Typology",
		"MEDLINEPLUS": "MedlinePlus Health Topics",
		"MSH":         "Medical Subject Headings",
		"NCI":         "NCI Thesaurus",
		"MEDCIN":      "MEDCIN",
		"ICPC2P":      "International Classification of Primary Care",
		"AOD":         "Alcohol and Other Drug Thesaurus",
	}

	// Show selected count
	if len(m.dictConfig.Vocabularies) > 0 {
		content = append(content, fmt.Sprintf("Selected: %d vocabularies", len(m.dictConfig.Vocabularies)))
	} else {
		content = append(content, "No vocabularies selected")
	}
	content = append(content, "")
	content = append(content, "Press Enter to select vocabularies")
	content = append(content, "")

	// Show ALL selected vocabularies - no truncation
	if len(m.dictConfig.Vocabularies) > 0 {
		content = append(content, "Selected Vocabularies:")
		content = append(content, "")
		for _, vocab := range m.dictConfig.Vocabularies {
			// Find the name for this vocabulary
			name, exists := vocabMap[vocab]
			if exists {
				content = append(content, fmt.Sprintf("  ✓ %s - %s", vocab, name))
			} else {
				content = append(content, fmt.Sprintf("  ✓ %s", vocab))
			}
		}
	}

	return content
}

func (m *Model) renderLanguageConfig() []string {
	var content []string

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorAccent)

	content = append(content, titleStyle.Render("Languages"))
	content = append(content, "")

	if len(m.dictConfig.Languages) > 0 {
		content = append(content, fmt.Sprintf("Selected: %d languages", len(m.dictConfig.Languages)))
		content = append(content, "")

		for _, lang := range m.dictConfig.Languages {
			content = append(content, fmt.Sprintf("  • %s", lang))
		}
	} else {
		content = append(content, "No languages selected")
		content = append(content, "")
		content = append(content, "Default: ENG (English)")
	}

	content = append(content, "")
	content = append(content, "Press Enter to configure languages")

	return content
}

func (m *Model) renderTermTypeConfig() []string {
	var content []string

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorAccent)

	content = append(content, titleStyle.Render("Term Types"))
	content = append(content, "")

	if len(m.dictConfig.TermTypes) > 0 {
		content = append(content, fmt.Sprintf("Selected: %d term types", len(m.dictConfig.TermTypes)))
		content = append(content, "")

		for _, tt := range m.dictConfig.TermTypes {
			content = append(content, fmt.Sprintf("  • %s", tt))
		}
	} else {
		content = append(content, "No term types selected")
		content = append(content, "")
		content = append(content, "Common Term Types:")
		content = append(content, "  • PT - Preferred Term")
		content = append(content, "  • SY - Synonym")
		content = append(content, "  • AB - Abbreviation")
	}

	content = append(content, "")
	content = append(content, "Press Enter to configure term types")

	return content
}

func (m *Model) handleDictTableAction(cursor int) tea.Cmd {
	if cursor >= len(m.dictOptions) {
		return nil
	}

	option := m.dictOptions[cursor]

	// Skip header rows
	if option.Type == "header" {
		return nil
	}

	switch option.Name {
	case "View Built Dictionaries":
		// Switch to dictionary viewer
		m.dictBuilderState = DictStateViewingDictionaries
		m.loadBuiltDictionaries()
		return nil

	case "Select UMLS Location":
		// Switch to embedded file browser
		m.dictBuilderState = DictStateSelectUMLS
		// Keep current directory or use last UMLS path
		if m.umlsPath != "" {
			m.currentPath = m.umlsPath
		}
		// Keep current path as is (defaults to working directory)
		m.updateFileList()
		m.updateTables()
		return nil

	case "Dictionary Name":
		// Switch to text input mode
		m.dictBuilderState = DictStateEditingName
		m.dictNameInput.SetValue(m.dictConfig.Name)
		m.dictNameInput.Focus()
		return nil

	case "Semantic Types (TUIs)":
		// Switch to TUI selector
		m.dictBuilderState = DictStateSelectingTUIs
		m.initTUITable(m.width/2, m.height-10)
		return nil

	case "Source Vocabularies":
		// Switch to vocabulary selector
		m.dictBuilderState = DictStateSelectingVocabs
		m.initVocabTable(m.width/2, m.height-10)
		return nil

	case "Languages":
		// For now, just set default languages
		if len(m.dictConfig.Languages) == 0 {
			m.dictConfig.Languages = []string{"ENG"}
		}
		m.initDictOptions()
		m.updateDictTable(m.width/2, m.height-6)
		return nil

	case "Term Types":
		// For now, just set default term types
		if len(m.dictConfig.TermTypes) == 0 {
			m.dictConfig.TermTypes = []string{"PT", "SY"}
		}
		m.initDictOptions()
		m.updateDictTable(m.width/2, m.height-6)
		return nil

	case "Memory Settings":
		// Switch to interactive memory configuration sub-menu
		m.dictBuilderState = DictStateMemoryConfig
		return nil

	case "Processing Options":
		// Switch to interactive processing configuration sub-menu
		m.dictBuilderState = DictStateProcessingConfig
		return nil

	case "Filter Options":
		// Switch to interactive filter configuration sub-menu
		m.dictBuilderState = DictStateFilterConfig
		return nil

	case "Output Formats":
		// Switch to interactive output configuration sub-menu
		m.dictBuilderState = DictStateOutputConfig
		return nil

	case "Relationship Settings":
		// Switch to interactive relationship configuration sub-menu
		m.dictBuilderState = DictStateRelationshipConfig
		return nil

	case "Apply Clinical Preset":
		m.applyClinicalPreset()
		m.initDictOptions()
		// Use consistent table height calculation
		tableHeight := m.height - 6
		if tableHeight > 15 {
			tableHeight = 15
		}
		m.updateDictTable(m.width/2, tableHeight)
		return nil

	case "Apply Medications Preset":
		m.applyMedicationsPreset()
		m.initDictOptions()
		// Use consistent table height calculation
		tableHeight := m.height - 6
		if tableHeight > 15 {
			tableHeight = 15
		}
		m.updateDictTable(m.width/2, tableHeight)
		return nil

	case "Build Dictionary":
		if option.Status == "ready" {
			m.dictBuilderState = DictStateBuilding
			return m.startDictionaryBuild()
		}
		return nil
	}

	return nil
}

func (m *Model) applyClinicalPreset() {
	// Basic configuration
	m.dictConfig.TUIs = []string{"T047", "T048", "T184", "T033", "T059", "T060", "T061", "T121", "T200"}
	m.dictConfig.Vocabularies = []string{"SNOMEDCT_US", "RXNORM", "ICD10CM"}
	m.dictConfig.Languages = []string{"ENG"}
	m.dictConfig.TermTypes = []string{"PT", "SY", "AB"}

	// Advanced configuration for clinical use
	// Memory settings for clinical data processing
	m.dictConfig.InitialHeapMB = 1024
	m.dictConfig.MaxHeapMB = 2048
	m.dictConfig.StackSizeMB = 8

	// Processing settings optimized for clinical text
	m.dictConfig.ThreadCount = 4
	m.dictConfig.BatchSize = 1000
	m.dictConfig.CacheSize = 128
	m.dictConfig.PreserveCase = false
	m.dictConfig.HandlePunctuation = true
	m.dictConfig.MinWordLength = 2
	m.dictConfig.MaxWordLength = 80

	// Filters for clean clinical data
	m.dictConfig.MinTermLength = 3
	m.dictConfig.MaxTermLength = 80
	m.dictConfig.ExcludeSuppressible = true
	m.dictConfig.ExcludeObsolete = true
	m.dictConfig.CaseSensitive = false
	m.dictConfig.UseNormalization = true
	m.dictConfig.UseMRRANK = true
	m.dictConfig.Deduplicate = true
	m.dictConfig.PreferredOnly = false
	m.dictConfig.StripPunctuation = false
	m.dictConfig.CollapseWhitespace = true
	m.dictConfig.ExcludeNumericOnly = true
	m.dictConfig.ExcludePunctOnly = true
	m.dictConfig.MinTokens = 1
	m.dictConfig.MaxTokens = 5

	// Output formats for clinical use
	m.dictConfig.EmitBSV = true
	m.dictConfig.BuildHSQLDB = true
	m.dictConfig.BuildLucene = true
	m.dictConfig.UseRareWords = true
	m.dictConfig.EmitTSV = false
	m.dictConfig.EmitJSONL = false
	m.dictConfig.EmitDescriptor = true
	m.dictConfig.EmitPipeline = true
	m.dictConfig.EmitManifest = true

	// Relationship settings for clinical concepts
	m.dictConfig.EnableRelationships = true
	m.dictConfig.RelationshipDepth = 2
	m.dictConfig.RelationshipTypes = []string{"PAR", "CHD", "RB", "RN", "SY", "isa", "part_of"}
}

func (m *Model) applyMedicationsPreset() {
	// Basic configuration
	m.dictConfig.TUIs = []string{"T109", "T121", "T195", "T200"}
	m.dictConfig.Vocabularies = []string{"RXNORM", "NDDF", "VANDF"}
	m.dictConfig.Languages = []string{"ENG"}
	m.dictConfig.TermTypes = []string{"PT", "SY", "BN", "GN"}

	// Advanced configuration for medications
	// Memory settings for medication data
	m.dictConfig.InitialHeapMB = 512
	m.dictConfig.MaxHeapMB = 1024
	m.dictConfig.StackSizeMB = 4

	// Processing settings optimized for medication names
	m.dictConfig.ThreadCount = 2
	m.dictConfig.BatchSize = 500
	m.dictConfig.CacheSize = 64
	m.dictConfig.PreserveCase = true
	m.dictConfig.HandlePunctuation = true
	m.dictConfig.MinWordLength = 2
	m.dictConfig.MaxWordLength = 100

	// Filters for medication data
	m.dictConfig.MinTermLength = 2
	m.dictConfig.MaxTermLength = 100
	m.dictConfig.ExcludeSuppressible = true
	m.dictConfig.ExcludeObsolete = true
	m.dictConfig.CaseSensitive = true
	m.dictConfig.UseNormalization = false
	m.dictConfig.UseMRRANK = true
	m.dictConfig.Deduplicate = true
	m.dictConfig.PreferredOnly = false
	m.dictConfig.StripPunctuation = false
	m.dictConfig.CollapseWhitespace = false
	m.dictConfig.ExcludeNumericOnly = false
	m.dictConfig.ExcludePunctOnly = false
	m.dictConfig.MinTokens = 1
	m.dictConfig.MaxTokens = 3

	// Output formats for medications
	m.dictConfig.EmitBSV = true
	m.dictConfig.BuildHSQLDB = false
	m.dictConfig.BuildLucene = true
	m.dictConfig.UseRareWords = false
	m.dictConfig.EmitTSV = true
	m.dictConfig.EmitJSONL = false
	m.dictConfig.EmitDescriptor = true
	m.dictConfig.EmitPipeline = true
	m.dictConfig.EmitManifest = true

	// Relationship settings for medications
	m.dictConfig.EnableRelationships = true
	m.dictConfig.RelationshipDepth = 1
	m.dictConfig.RelationshipTypes = []string{"PAR", "CHD", "SY"}
}

func (m *Model) renderDictNameInput(width, height int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorAccent)

	var content []string
	content = append(content, titleStyle.Render("◈ Dictionary Name"))
	content = append(content, "")
	content = append(content, "Enter a name for your dictionary:")
	content = append(content, "")
	content = append(content, m.dictNameInput.View())
	content = append(content, "")
	content = append(content, "")
	content = append(content, "Press Enter to confirm, ESC to cancel")

	return strings.Join(content, "\n")
}

func (m *Model) initTUITable(width, height int) {
	// Common medical semantic types - expanded list
	tuiList := []struct {
		Code string
		Name string
	}{
		{"T005", "Virus"},
		{"T007", "Bacterium"},
		{"T017", "Anatomical Structure"},
		{"T019", "Congenital Abnormality"},
		{"T020", "Acquired Abnormality"},
		{"T033", "Finding"},
		{"T037", "Injury or Poisoning"},
		{"T046", "Pathologic Function"},
		{"T047", "Disease or Syndrome"},
		{"T048", "Mental or Behavioral Dysfunction"},
		{"T059", "Laboratory Procedure"},
		{"T060", "Diagnostic Procedure"},
		{"T061", "Therapeutic or Preventive Procedure"},
		{"T074", "Medical Device"},
		{"T109", "Organic Chemical"},
		{"T116", "Amino Acid, Peptide, or Protein"},
		{"T121", "Pharmacologic Substance"},
		{"T122", "Biomedical or Dental Material"},
		{"T123", "Biologically Active Substance"},
		{"T184", "Sign or Symptom"},
		{"T190", "Anatomical Abnormality"},
		{"T191", "Neoplastic Process"},
		{"T195", "Antibiotic"},
		{"T200", "Clinical Drug"},
		{"T201", "Clinical Attribute"},
	}

	// Store current cursor position if table exists
	currentCursor := 0
	if m.tuiTable.Cursor != nil {
		currentCursor = m.tuiTable.Cursor()
	}

	// Create table columns
	columns := []table.Column{
		{Title: "✓", Width: 3},
		{Title: "Code", Width: 8},
		{Title: "Semantic Type", Width: width - 15},
	}

	// Create rows
	var rows []table.Row
	for _, tui := range tuiList {
		// Check if already selected
		selected := " "
		for _, selectedTUI := range m.dictConfig.TUIs {
			if selectedTUI == tui.Code {
				selected = "✓"
				break
			}
		}
		rows = append(rows, table.Row{selected, tui.Code, tui.Name})
	}

	// Ensure height allows scrolling
	tableHeight := height - 4 // Leave room for header and footer
	if tableHeight < 10 {
		tableHeight = 10
	}

	// Create table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableHeight),
	)

	// Apply styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.ColorBorder).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(theme.ColorBackground).
		Background(theme.ColorAccent).
		Bold(false)

	t.SetStyles(s)

	// Restore cursor position if valid
	if currentCursor < len(rows) && currentCursor >= 0 {
		t.SetCursor(currentCursor)
	}

	m.tuiTable = t
	m.tuiTable.Focus()
}

func (m *Model) updateTUITableSelection() {
	// Update only the selection without recreating the table
	if m.tuiTable.Cursor() < len(m.tuiTable.Rows()) {
		currentCursor := m.tuiTable.Cursor()
		rows := m.tuiTable.Rows()

		// Get the current row and TUI code
		tuiCode := rows[currentCursor][1]

		// Toggle selection in config
		found := false
		for i, tui := range m.dictConfig.TUIs {
			if tui == tuiCode {
				// Remove from selection
				m.dictConfig.TUIs = append(m.dictConfig.TUIs[:i], m.dictConfig.TUIs[i+1:]...)
				found = true
				break
			}
		}

		if !found {
			// Add to selection
			m.dictConfig.TUIs = append(m.dictConfig.TUIs, tuiCode)
		}

		// Only update the specific row that changed instead of all rows
		checkbox := " "
		for _, selectedTUI := range m.dictConfig.TUIs {
			if selectedTUI == tuiCode {
				checkbox = "✓"
				break
			}
		}

		// Update just the current row
		updatedRow := table.Row{checkbox, rows[currentCursor][1], rows[currentCursor][2]}
		rows[currentCursor] = updatedRow
		m.tuiTable.SetRows(rows)
		m.tuiTable.SetCursor(currentCursor)
	}
}

func (m *Model) renderClinicalPresetPreview() []string {
	var content []string

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorAccent)

	content = append(content, titleStyle.Render("Clinical Preset"))
	content = append(content, "")
	content = append(content, "This preset configures the dictionary for")
	content = append(content, "general clinical use with common medical terms.")
	content = append(content, "")
	content = append(content, "Will configure:")
	content = append(content, "")
	content = append(content, "Semantic Types:")
	content = append(content, "  • Disease or Syndrome")
	content = append(content, "  • Mental or Behavioral Dysfunction")
	content = append(content, "  • Sign or Symptom")
	content = append(content, "  • Finding")
	content = append(content, "  • Laboratory/Diagnostic Procedures")
	content = append(content, "  • Therapeutic Procedures")
	content = append(content, "  • Pharmacologic Substances")
	content = append(content, "")
	content = append(content, "Vocabularies:")
	content = append(content, "  • SNOMED CT US")
	content = append(content, "  • RxNorm")
	content = append(content, "  • ICD-10-CM")
	content = append(content, "")
	content = append(content, "Press Enter to apply this preset")

	return content
}

func (m *Model) renderMedicationsPresetPreview() []string {
	var content []string

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorAccent)

	content = append(content, titleStyle.Render("Medications Preset"))
	content = append(content, "")
	content = append(content, "This preset configures the dictionary for")
	content = append(content, "medication and drug-related terms.")
	content = append(content, "")
	content = append(content, "Will configure:")
	content = append(content, "")
	content = append(content, "Semantic Types:")
	content = append(content, "  • Organic Chemical")
	content = append(content, "  • Pharmacologic Substance")
	content = append(content, "  • Antibiotic")
	content = append(content, "  • Clinical Drug")
	content = append(content, "")
	content = append(content, "Vocabularies:")
	content = append(content, "  • RxNorm")
	content = append(content, "  • NDDF")
	content = append(content, "  • VANDF")
	content = append(content, "")
	content = append(content, "Press Enter to apply this preset")

	return content
}

func (m *Model) renderBuildPreview() []string {
	var content []string

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorAccent)

	content = append(content, titleStyle.Render("Build Dictionary"))
	content = append(content, "")

	if m.dictOptions[6].Status != "ready" {
		errorStyle := lipgloss.NewStyle().
			Foreground(theme.ColorError)
		content = append(content, errorStyle.Render("Cannot build yet!"))
		content = append(content, "")
		content = append(content, "Required configurations:")

		if m.umlsPath == "" {
			content = append(content, "  ✗ UMLS location not selected")
		} else {
			content = append(content, "  ✓ UMLS location selected")
		}

		if m.dictConfig.Name == "" {
			content = append(content, "  ✗ Dictionary name not set")
		} else {
			content = append(content, "  ✓ Dictionary name set")
		}
	} else {
		content = append(content, "Ready to build dictionary!")
		content = append(content, "")
		content = append(content, "Configuration Summary:")
		content = append(content, fmt.Sprintf("  Name: %s", m.dictConfig.Name))
		content = append(content, fmt.Sprintf("  UMLS: %s", filepath.Base(m.umlsPath)))
		content = append(content, fmt.Sprintf("  TUIs: %d selected", len(m.dictConfig.TUIs)))
		content = append(content, fmt.Sprintf("  Vocabularies: %d selected", len(m.dictConfig.Vocabularies)))
		content = append(content, fmt.Sprintf("  Languages: %d selected", len(m.dictConfig.Languages)))
		content = append(content, fmt.Sprintf("  Term Types: %d selected", len(m.dictConfig.TermTypes)))
		content = append(content, "")
		content = append(content, "Press Enter to start building")
	}

	return content
}

func (m *Model) renderBuildProgress() []string {
	var content []string

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorAccent)

	content = append(content, titleStyle.Render("◈ Building Dictionary"))
	content = append(content, "")

	// Progress bar
	progressWidth := 40
	filledWidth := int(float64(progressWidth) * m.buildProgress)
	emptyWidth := progressWidth - filledWidth

	progressBar := lipgloss.NewStyle().
		Foreground(theme.ColorSuccess).
		Render(strings.Repeat("█", filledWidth)) +
		lipgloss.NewStyle().
			Foreground(theme.ColorBorder).
			Render(strings.Repeat("░", emptyWidth))

	content = append(content, fmt.Sprintf("Progress: [%s] %.1f%%", progressBar, m.buildProgress*100))
	content = append(content, "")

	// Current step
	if m.buildCurrentStep != "" {
		stepStyle := lipgloss.NewStyle().
			Foreground(theme.ColorAccent)
		content = append(content, stepStyle.Render(fmt.Sprintf("Step %d/%d: %s",
			m.buildCurrentStepNum, m.buildTotalSteps, m.buildCurrentStep)))
	}

	// Time elapsed
	if !m.buildStartTime.IsZero() {
		elapsed := time.Since(m.buildStartTime)
		content = append(content, fmt.Sprintf("Time: %s", elapsed.Round(time.Second)))
	}

	content = append(content, "")
	content = append(content, "Build Log:")
	content = append(content, strings.Repeat("─", 50))

	// Show last N log lines
	logsToShow := 15
	if len(m.buildLogs) > 0 {
		startIdx := 0
		if len(m.buildLogs) > logsToShow {
			startIdx = len(m.buildLogs) - logsToShow
		}
		for i := startIdx; i < len(m.buildLogs); i++ {
			content = append(content, m.buildLogs[i])
		}
	} else {
		content = append(content, "Initializing build process...")
	}

	// Show error if any
	if m.buildError != nil {
		content = append(content, "")
		errorStyle := lipgloss.NewStyle().
			Foreground(theme.ColorError).
			Bold(true)
		content = append(content, errorStyle.Render("Error: "+m.buildError.Error()))
	}

	content = append(content, "")
	content = append(content, "[ESC] Cancel Build")

	return content
}

func (m *Model) renderBuildTerminalLogs(width, height int) string {
	// Terminal-style header
	headerStyle := lipgloss.NewStyle().
		Foreground(theme.ColorSuccess).
		Bold(true)

	terminalStyle := lipgloss.NewStyle().
		Foreground(theme.ColorForegroundDim).
		Background(lipgloss.Color("#0c0c0c"))

	var logs []string
	logs = append(logs, headerStyle.Render("◈ Build Terminal Output"))
	logs = append(logs, strings.Repeat("─", width-2))
	logs = append(logs, "")

	// Show real-time logs collected during build
	debugLogs := append([]string{}, m.buildLogs...)

	// Add progress-based debug output
	// The content is now entirely driven by m.buildLogs and callbacks.

	// Intentionally empty; real-time logs only.

	// Intentionally empty; real-time logs only.

	// Intentionally empty; real-time logs only.

	// Intentionally empty; real-time logs only.

	// Intentionally empty; real-time logs only.

	// Add any errors
	if m.buildError != nil {
		debugLogs = append(debugLogs,
			"",
			fmt.Sprintf("[ERROR] Build failed: %s", m.buildError.Error()),
			"[ERROR] Stack trace:",
			"  at org.apache.ctakes.dictionary.creator.DictionaryBuilder.main()",
			"  at java.base/jdk.internal.reflect.NativeMethodAccessorImpl.invoke0()",
		)
	}

	// Format as terminal output
	for _, log := range debugLogs {
		if strings.HasPrefix(log, "[ERROR]") {
			logs = append(logs, lipgloss.NewStyle().Foreground(theme.ColorError).Render(log))
		} else if strings.HasPrefix(log, "[SUCCESS]") {
			logs = append(logs, lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render(log))
		} else if strings.HasPrefix(log, "[INFO]") {
			logs = append(logs, lipgloss.NewStyle().Foreground(theme.ColorAccent).Render(log))
		} else if strings.HasPrefix(log, "[DEBUG]") {
			logs = append(logs, lipgloss.NewStyle().Foreground(theme.ColorSecondary).Render(log))
		} else if strings.HasPrefix(log, "[TRACE]") {
			logs = append(logs, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render(log))
		} else if strings.HasPrefix(log, ">>>") {
			logs = append(logs, lipgloss.NewStyle().Foreground(theme.ColorAccent).Bold(true).Render(log))
		} else {
			logs = append(logs, terminalStyle.Render(log))
		}
	}

	// Create scrollable viewport for terminal logs
	fullContent := strings.Join(logs, "\n")

	// Apply terminal-like styling
	return lipgloss.NewStyle().
		Width(width - 2).
		Height(height - 2).
		Padding(1).
		Background(lipgloss.Color("#0c0c0c")).
		Render(fullContent)
}

func (m *Model) renderBuildProgressPopup(width, height int) string {
	// Match the TUI/Vocab selector theming for consistency
	headerStyle := lipgloss.NewStyle().
		Background(theme.ColorAccent).
		Foreground(theme.ColorBackground).
		Bold(true).
		Padding(0, 1)

	// Create header with progress percentage
	header := headerStyle.Render(fmt.Sprintf("◈ Building Dictionary - %.1f%%", m.buildProgress*100))

	// Progress bar
	progressWidth := width - 6
	filledWidth := int(float64(progressWidth) * m.buildProgress)
	emptyWidth := progressWidth - filledWidth

	progressBar := lipgloss.NewStyle().
		Foreground(theme.ColorSuccess).
		Render(strings.Repeat("█", filledWidth)) +
		lipgloss.NewStyle().
			Foreground(theme.ColorBorder).
			Render(strings.Repeat("░", emptyWidth))

	progressLine := lipgloss.NewStyle().
		Padding(0, 1).
		Render(progressBar)

	// Status section
	statusStyle := lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Padding(0, 1)

	statusContent := []string{}
	if m.buildCurrentStep != "" {
		statusContent = append(statusContent,
			fmt.Sprintf("Step %d of %d: %s", m.buildCurrentStepNum, m.buildTotalSteps, m.buildCurrentStep))
	}

	if !m.buildStartTime.IsZero() {
		elapsed := time.Since(m.buildStartTime)
		statusContent = append(statusContent, fmt.Sprintf("Elapsed: %s", elapsed.Round(time.Second)))
	}

	status := statusStyle.Render(strings.Join(statusContent, " | "))

	// Build logs with viewport for scrolling
	logsHeaderStyle := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Padding(0, 1)

	logsHeader := logsHeaderStyle.Render("Build Output:")

	// Create or update viewport for logs - maximize available height
	// Count actual lines used: header(1) + space(1) + progress(1) + status(1) + space(1) + logsHeader(1) + footer(1) = 7
	// Plus border lines (2) and some padding = 10 total
	viewportHeight := height - 10
	if viewportHeight < 8 {
		viewportHeight = 8 // Minimum reasonable height
	}

	// Only recreate viewport if size changed
	if m.buildViewport.Width != width-6 || m.buildViewport.Height != viewportHeight {
		m.buildViewport = viewport.New(width-6, viewportHeight)
		m.buildViewport.HighPerformanceRendering = false
	}

	// Format logs
	logContent := strings.Join(m.buildLogs, "\n")
	if logContent == "" {
		logContent = "Initializing build process..."
	}

	m.buildViewport.SetContent(logContent)
	// Auto-scroll to bottom for new logs
	m.buildViewport.GotoBottom()

	logsBox := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(theme.ColorBorder).
		Width(width - 4).
		Height(viewportHeight + 2). // +2 for border
		Render(m.buildViewport.View())

	// Error display if any
	errorContent := ""
	if m.buildError != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(theme.ColorError).
			Bold(true).
			Padding(0, 1)
		errorContent = errorStyle.Render("⚠ Error: " + m.buildError.Error())
	}

	// Footer with instructions
	footerStyle := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Padding(0, 1)

	footer := footerStyle.Render("↑↓: Scroll Logs | ESC: Cancel Build")

	// Combine all parts
	parts := []string{
		header,
		"",
		progressLine,
		status,
		"",
		logsHeader,
		logsBox,
	}

	if errorContent != "" {
		parts = append(parts, errorContent)
	}

	parts = append(parts, footer)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		parts...,
	)

	// Apply border to match file browser
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorBorder).
		Width(width - 2).
		Height(height - 2)

	return borderStyle.Render(content)
}

func (m *Model) renderVocabSelector(width, height int) string {
	// Match the TUI selector theming
	headerStyle := lipgloss.NewStyle().
		Background(theme.ColorAccent).
		Foreground(theme.ColorBackground).
		Bold(true).
		Padding(0, 1)

	// Create header
	header := headerStyle.Render("◈ Select Source Vocabularies")

	// Show the vocabulary table with proper bounds
	tableContent := m.vocabTable.View()

	// Create footer with instructions
	footerStyle := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Padding(0, 1)

	footer := footerStyle.Render("SPACE: Toggle | Enter: Confirm | ESC: Cancel")

	// Combine all parts
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		tableContent,
		"",
		footer,
	)

	// Apply border to match file browser
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorBorder).
		Width(width - 2).
		Height(height - 2)

	return borderStyle.Render(content)
}

func (m *Model) initVocabTable(width, height int) {
	// Common medical vocabularies based on cTAKES
	vocabList := []struct {
		Code string
		Name string
	}{
		{"SNOMEDCT_US", "SNOMED CT US Edition"},
		{"RXNORM", "RxNorm"},
		{"ICD10CM", "ICD-10-CM"},
		{"ICD10PCS", "ICD-10-PCS"},
		{"ICD9CM", "ICD-9-CM"},
		{"LOINC", "LOINC"},
		{"CPT", "Current Procedural Terminology"},
		{"HCPCS", "Healthcare Common Procedure Coding System"},
		{"NDC", "National Drug Code"},
		{"VANDF", "Veterans Health Administration National Drug File"},
		{"NDDF", "National Drug Data File"},
		{"CVX", "Vaccines Administered"},
		{"MVX", "Vaccine Manufacturers"},
		{"SOP", "Source of Payment Typology"},
		{"MEDLINEPLUS", "MedlinePlus Health Topics"},
		{"MSH", "Medical Subject Headings"},
		{"NCI", "NCI Thesaurus"},
		{"MEDCIN", "MEDCIN"},
		{"ICPC2P", "International Classification of Primary Care"},
		{"AOD", "Alcohol and Other Drug Thesaurus"},
	}

	// Store current cursor position if table exists
	currentCursor := 0
	if m.vocabTable.Cursor != nil {
		currentCursor = m.vocabTable.Cursor()
	}

	// Create table columns
	columns := []table.Column{
		{Title: "✓", Width: 3},
		{Title: "Code", Width: 15},
		{Title: "Vocabulary Name", Width: width - 20},
	}

	// Create rows
	var rows []table.Row
	for _, vocab := range vocabList {
		// Check if already selected
		selected := " "
		for _, selectedVocab := range m.dictConfig.Vocabularies {
			if selectedVocab == vocab.Code {
				selected = "✓"
				break
			}
		}
		rows = append(rows, table.Row{selected, vocab.Code, vocab.Name})
	}

	// Ensure height allows scrolling
	tableHeight := height - 4 // Leave room for header and footer
	if tableHeight < 10 {
		tableHeight = 10
	}

	// Create table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableHeight),
	)

	// Apply styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.ColorBorder).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(theme.ColorBackground).
		Background(theme.ColorAccent).
		Bold(false)

	t.SetStyles(s)

	// Restore cursor position if valid
	if currentCursor < len(rows) && currentCursor >= 0 {
		t.SetCursor(currentCursor)
	}

	m.vocabTable = t
	m.vocabTable.Focus()
}

func (m *Model) startDictionaryBuild() tea.Cmd {
	// Initialize build state
	m.buildProgress = 0.0
	m.buildLogs = []string{}
	m.buildStartTime = time.Now()
	m.buildError = nil
	m.buildTotalSteps = 6
	m.buildCurrentStepNum = 1
	m.buildCurrentStep = "Initializing build environment"

	// Add initial log entries
	m.buildLogs = append(m.buildLogs, "=== Dictionary Build Started ===")
	m.buildLogs = append(m.buildLogs, fmt.Sprintf("Time: %s", m.buildStartTime.Format("2006-01-02 15:04:05")))
	m.buildLogs = append(m.buildLogs, "")
	m.buildLogs = append(m.buildLogs, "Configuration:")
	m.buildLogs = append(m.buildLogs, fmt.Sprintf("  Name: %s", m.dictConfig.Name))
	m.buildLogs = append(m.buildLogs, fmt.Sprintf("  UMLS Path: %s", m.umlsPath))
	m.buildLogs = append(m.buildLogs, fmt.Sprintf("  TUIs Selected: %d", len(m.dictConfig.TUIs)))
	m.buildLogs = append(m.buildLogs, fmt.Sprintf("  Vocabularies: %d", len(m.dictConfig.Vocabularies)))
	m.buildLogs = append(m.buildLogs, "")

	// Initialize viewport if needed
	if m.buildViewport.Width == 0 {
		m.buildViewport = viewport.New(80, 15)
	}

	// Always use headless Go builder path now; remove simulation
	m.buildLogs = append(m.buildLogs, "[INFO] Using headless Go builder (BSV)")
	return m.startGoHeadlessBuild()
}

// simulateBuildProgress is removed - build progress is now handled by the real BSV builder

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// canUseRealCTakes checks if cTAKES Java integration is available
func (m *Model) canUseRealCTakes() bool {
	// Check for CTAKES_HOME environment variable
	ctakesHome := os.Getenv("CTAKES_HOME")
	if ctakesHome == "" {
		return false
	}

	// Check if directory exists
	if _, err := os.Stat(ctakesHome); os.IsNotExist(err) {
		return false
	}

	// Check for Java
	javaHome := os.Getenv("JAVA_HOME")
	if javaHome == "" {
		// Try to find java in PATH
		if _, err := exec.LookPath("java"); err != nil {
			return false
		}
	}

	return true
}

// startRealDictionaryBuild starts the actual cTAKES dictionary build process
func (m *Model) startRealDictionaryBuild() tea.Cmd {
	return func() tea.Msg {
		// Run the build in a goroutine and send updates via messages
		return m.startGoHeadlessBuild()()
	}
}

// startGoHeadlessBuild launches the Go-based BSV builder and streams logs
func (m *Model) startGoHeadlessBuild() tea.Cmd {
	return func() tea.Msg {
		outDir := filepath.Join("dictionaries", m.dictConfig.Name)

		// Create configuration from dashboard state
		cfg := dictionary.CreateDefaultConfig(m.dictConfig.Name, "")
		cfg.SemanticTypes = m.dictConfig.TUIs
		cfg.Vocabularies = m.dictConfig.Vocabularies
		cfg.Languages = m.dictConfig.Languages
		cfg.TermTypes = m.dictConfig.TermTypes

		// Map memory settings
		cfg.Memory.InitialHeapMB = m.dictConfig.InitialHeapMB
		cfg.Memory.MaxHeapMB = m.dictConfig.MaxHeapMB
		cfg.Memory.StackSizeMB = m.dictConfig.StackSizeMB

		// Map processing settings
		cfg.Processing.ThreadCount = m.dictConfig.ThreadCount
		cfg.Processing.BatchSize = m.dictConfig.BatchSize
		cfg.Processing.CacheSize = m.dictConfig.CacheSize
		cfg.Processing.TempDirectory = m.dictConfig.TempDirectory
		cfg.Processing.PreserveCase = m.dictConfig.PreserveCase
		cfg.Processing.HandlePunctuation = m.dictConfig.HandlePunctuation
		cfg.Processing.MinWordLength = m.dictConfig.MinWordLength
		cfg.Processing.MaxWordLength = m.dictConfig.MaxWordLength

		// Map filter settings
		cfg.Filters.MinTermLength = m.dictConfig.MinTermLength
		cfg.Filters.MaxTermLength = m.dictConfig.MaxTermLength
		cfg.Filters.ExcludeSuppressible = m.dictConfig.ExcludeSuppressible
		cfg.Filters.ExcludeObsolete = m.dictConfig.ExcludeObsolete
		cfg.Filters.CaseSensitive = m.dictConfig.CaseSensitive
		cfg.Filters.UseNormalization = m.dictConfig.UseNormalization
		cfg.Filters.UseMRRank = m.dictConfig.UseMRRANK
		cfg.Filters.Deduplicate = m.dictConfig.Deduplicate
		cfg.Filters.PreferredOnly = m.dictConfig.PreferredOnly
		cfg.Filters.StripPunctuation = m.dictConfig.StripPunctuation
		cfg.Filters.CollapseWhitespace = m.dictConfig.CollapseWhitespace
		cfg.Filters.ExcludeNumericOnly = m.dictConfig.ExcludeNumericOnly
		cfg.Filters.ExcludePunctOnly = m.dictConfig.ExcludePunctOnly
		cfg.Filters.MinTokens = m.dictConfig.MinTokens
		cfg.Filters.MaxTokens = m.dictConfig.MaxTokens

		// Map output settings
		cfg.Outputs.EmitTSV = m.dictConfig.EmitTSV
		cfg.Outputs.EmitJSONL = m.dictConfig.EmitJSONL
		cfg.Outputs.BuildLucene = m.dictConfig.BuildLucene
		cfg.Outputs.BuildHSQLDB = m.dictConfig.BuildHSQLDB
		cfg.Outputs.UseRareWords = m.dictConfig.UseRareWords
		cfg.Outputs.EmitDescriptor = m.dictConfig.EmitDescriptor
		cfg.Outputs.EmitPipeline = m.dictConfig.EmitPipeline
		cfg.Outputs.EmitManifest = m.dictConfig.EmitManifest

		// Map relationship settings
		cfg.Relationships.Enabled = m.dictConfig.EnableRelationships
		cfg.Relationships.Depth = m.dictConfig.RelationshipDepth
		cfg.Relationships.Types = m.dictConfig.RelationshipTypes

		// Create build service
		buildService := dictionary.NewBuildService(
			func(stage, message string, progress float64) {
				// Update build state (UI will poll via tick)
				m.buildCurrentStep = stage
				if progress >= 0 && progress <= 1 {
					m.buildProgress = progress
				}
				if message != "" {
					m.buildLogs = append(m.buildLogs, fmt.Sprintf("%s", message))
				}
			},
			func(err error) {
				if err != nil {
					m.buildError = err
					m.buildLogs = append(m.buildLogs, fmt.Sprintf("ERROR: %v", err))
				} else {
					m.buildProgress = 1.0
					m.buildLogs = append(m.buildLogs, "Build completed")
					m.dictBuilderState = DictStateComplete
				}
			},
		)

		// Start async build
		buildService.BuildDictionaryAsync(cfg, m.umlsPath, outDir)

		m.buildCurrentStep = "Starting build"
		m.buildCurrentStepNum = 2
		return buildTickMsg(time.Now())
	}
}

// runCTakesBuild runs the actual cTAKES dictionary building process
// runCTakesBuild (legacy Java path) removed. Headless Go builder is used instead.

// countBuiltDictionaries counts the number of built dictionaries
func (m *Model) countBuiltDictionaries() int {
	dictionaries, err := dictionary.ListDictionaries()
	if err != nil {
		return 0
	}
	return len(dictionaries)
}

// loadBuiltDictionaries loads the list of built dictionaries

// renderDictionaryViewer renders the dictionary viewer panel
func (m *Model) renderDictionaryViewer(width, height int) string {
	headerStyle := lipgloss.NewStyle().
		Background(theme.ColorAccent).
		Foreground(theme.ColorBackground).
		Bold(true).
		Padding(0, 1)

	header := headerStyle.Render("◈ Built Dictionaries")

	if len(m.builtDictionaries) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(theme.ColorSecondary).
			Padding(2)

		content := emptyStyle.Render("No dictionaries found.\n\nBuild your first dictionary by:\n1. Selecting UMLS location\n2. Configuring options\n3. Building dictionary")

		footerStyle := lipgloss.NewStyle().
			Foreground(theme.ColorSecondary).
			Padding(0, 1)

		footer := footerStyle.Render("ESC: Back to Menu")

		return lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			"",
			content,
			"",
			footer,
		)
	}

	// Show dictionary table
	tableContent := m.dictViewerTable.View()

	// Footer with instructions
	footerStyle := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Padding(0, 1)

	footer := footerStyle.Render("Enter: View Details | D: Delete | E: Export | ESC: Back")

	// Combine all parts
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		tableContent,
		"",
		footer,
	)

	// Apply border
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorBorder).
		Width(width - 2).
		Height(height - 2)

	return borderStyle.Render(content)
}

// initDictViewerTable initializes the table for viewing dictionaries
func (m *Model) initDictViewerTable(width, height int) {
	// Create table columns
	columns := []table.Column{
		{Title: "Name", Width: 20},
		{Title: "Size", Width: 10},
		{Title: "TUIs", Width: 8},
		{Title: "Vocabs", Width: 8},
		{Title: "Languages", Width: 10},
		{Title: "Created", Width: width - 60},
	}

	// Create rows
	var rows []table.Row
	for _, dict := range m.builtDictionaries {
		rows = append(rows, table.Row{
			dict.Name,
			dict.Size,
			fmt.Sprintf("%d", dict.TUICount),
			fmt.Sprintf("%d", dict.VocabCount),
			dict.Languages,
			dict.Created.Format("2006-01-02 15:04"),
		})
	}

	// Ensure height allows scrolling
	tableHeight := height - 4
	if tableHeight < 10 {
		tableHeight = 10
	}

	// Create table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableHeight),
	)

	// Apply styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.ColorBorder).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(theme.ColorBackground).
		Background(theme.ColorAccent).
		Bold(false)

	t.SetStyles(s)
	m.dictViewerTable = t
	m.dictViewerTable.Focus()
}

// renderDictionariesPreview renders the preview for the dictionary viewer
func (m *Model) renderDictionariesPreview() []string {
	var content []string

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorAccent)

	content = append(content, titleStyle.Render("Built Dictionaries"))
	content = append(content, "")

	dictCount := m.countBuiltDictionaries()
	if dictCount == 0 {
		content = append(content, "No dictionaries have been built yet.")
		content = append(content, "")
		content = append(content, "Build a dictionary to:")
		content = append(content, "  • Use with cTAKES pipelines")
		content = append(content, "  • Process clinical text")
		content = append(content, "  • Extract medical concepts")
	} else {
		content = append(content, fmt.Sprintf("Found %d dictionar%s", dictCount,
			map[bool]string{true: "y", false: "ies"}[dictCount == 1]))
		content = append(content, "")
		content = append(content, "Press Enter to view and manage")
		content = append(content, "your built dictionaries.")
		content = append(content, "")
		content = append(content, "Available actions:")
		content = append(content, "  • View dictionary details")
		content = append(content, "  • Export configuration")
		content = append(content, "  • Delete dictionaries")
		content = append(content, "  • View statistics")
	}

	return content
}

func (m *Model) updateVocabTableSelection() {
	// Update only the selection without recreating the table
	if m.vocabTable.Cursor() < len(m.vocabTable.Rows()) {
		currentCursor := m.vocabTable.Cursor()
		rows := m.vocabTable.Rows()

		// Get the current row and vocabulary code
		vocabCode := rows[currentCursor][1]

		// Toggle selection in config
		found := false
		for i, vocab := range m.dictConfig.Vocabularies {
			if vocab == vocabCode {
				// Remove from selection
				m.dictConfig.Vocabularies = append(m.dictConfig.Vocabularies[:i], m.dictConfig.Vocabularies[i+1:]...)
				found = true
				break
			}
		}

		if !found {
			// Add to selection
			m.dictConfig.Vocabularies = append(m.dictConfig.Vocabularies, vocabCode)
		}

		// Only update the specific row that changed instead of all rows
		checkbox := " "
		for _, selectedVocab := range m.dictConfig.Vocabularies {
			if selectedVocab == vocabCode {
				checkbox = "✓"
				break
			}
		}

		// Update just the current row
		updatedRow := table.Row{checkbox, rows[currentCursor][1], rows[currentCursor][2]}
		rows[currentCursor] = updatedRow
		m.vocabTable.SetRows(rows)
		m.vocabTable.SetCursor(currentCursor)
	}
}

// Memory Configuration Selector
func (m *Model) renderMemoryConfigSelector(width, height int) string {
	headerStyle := lipgloss.NewStyle().
		Background(theme.ColorAccent).
		Foreground(theme.ColorBackground).
		Bold(true).
		Padding(0, 1)

	header := headerStyle.Render("◈ Memory Settings")

	var content []string
	content = append(content, "Configure JVM memory allocation:")
	content = append(content, "")
	content = append(content, fmt.Sprintf("Initial Heap Size: %d MB (512-3072)", m.dictConfig.InitialHeapMB))
	content = append(content, fmt.Sprintf("Maximum Heap Size: %d MB (512-3072)", m.dictConfig.MaxHeapMB))
	content = append(content, fmt.Sprintf("Stack Size: %d MB (1-64)", m.dictConfig.StackSizeMB))
	content = append(content, "")
	content = append(content, "Controls:")
	content = append(content, "  [1/2] Initial Heap -/+ 256MB")
	content = append(content, "  [3/4] Max Heap -/+ 256MB")
	content = append(content, "  [5/6] Stack Size -/+ 1MB")

	footerStyle := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Padding(0, 1)

	footer := footerStyle.Render("Number Keys: Adjust Settings | Enter: Confirm | ESC: Cancel")

	combinedContent := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		strings.Join(content, "\n"),
		"",
		footer,
	)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorBorder).
		Width(width - 2).
		Height(height - 2)

	return borderStyle.Render(combinedContent)
}

// Processing Configuration Selector
func (m *Model) renderProcessingConfigSelector(width, height int) string {
	headerStyle := lipgloss.NewStyle().
		Background(theme.ColorAccent).
		Foreground(theme.ColorBackground).
		Bold(true).
		Padding(0, 1)

	header := headerStyle.Render("◈ Processing Options")

	var content []string
	content = append(content, "Configure text processing parameters:")
	content = append(content, "")
	content = append(content, fmt.Sprintf("Thread Count: %d (1-16)", m.dictConfig.ThreadCount))
	content = append(content, fmt.Sprintf("Batch Size: %d (100-10000)", m.dictConfig.BatchSize))
	content = append(content, fmt.Sprintf("Cache Size: %d MB (64-512)", m.dictConfig.CacheSize))
	content = append(content, fmt.Sprintf("Temp Directory: %s", ifEmpty(m.dictConfig.TempDirectory, "System default")))
	content = append(content, fmt.Sprintf("Min Word Length: %d (1-10)", m.dictConfig.MinWordLength))
	content = append(content, fmt.Sprintf("Max Word Length: %d (10-256)", m.dictConfig.MaxWordLength))
	content = append(content, "")
	content = append(content, fmt.Sprintf("Preserve Case: %v", m.dictConfig.PreserveCase))
	content = append(content, fmt.Sprintf("Handle Punctuation: %v", m.dictConfig.HandlePunctuation))
	content = append(content, "")
	content = append(content, "Controls:")
	content = append(content, "  [1/2] Thread Count -/+1  [3/4] Batch Size -/+100")
	content = append(content, "  [5/6] Cache -/+32MB      [7/8] Min Word Length -/+1")
	content = append(content, "  [9/0] Max Word Length -/+10  [P] Toggle Preserve Case")
	content = append(content, "  [H] Toggle Handle Punctuation")

	footerStyle := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Padding(0, 1)

	footer := footerStyle.Render("Keys: Adjust Settings | Enter: Confirm | ESC: Cancel")

	combinedContent := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		strings.Join(content, "\n"),
		"",
		footer,
	)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorBorder).
		Width(width - 2).
		Height(height - 2)

	return borderStyle.Render(combinedContent)
}

// Filter Configuration Selector
func (m *Model) renderFiltersConfigSelector(width, height int) string {
	headerStyle := lipgloss.NewStyle().
		Background(theme.ColorAccent).
		Foreground(theme.ColorBackground).
		Bold(true).
		Padding(0, 1)

	header := headerStyle.Render("◈ Filter Options")

	var content []string
	content = append(content, "Configure term filtering and normalization:")
	content = append(content, "")
	content = append(content, fmt.Sprintf("Min Term Length: %d", m.dictConfig.MinTermLength))
	content = append(content, fmt.Sprintf("Max Term Length: %d", m.dictConfig.MaxTermLength))
	content = append(content, fmt.Sprintf("Min Tokens: %d", m.dictConfig.MinTokens))
	content = append(content, fmt.Sprintf("Max Tokens: %d", m.dictConfig.MaxTokens))
	content = append(content, "")
	content = append(content, "Filter Options:")
	content = append(content, fmt.Sprintf("  [S] Exclude Suppressible: %v", m.dictConfig.ExcludeSuppressible))
	content = append(content, fmt.Sprintf("  [O] Exclude Obsolete: %v", m.dictConfig.ExcludeObsolete))
	content = append(content, fmt.Sprintf("  [C] Case Sensitive: %v", m.dictConfig.CaseSensitive))
	content = append(content, fmt.Sprintf("  [N] Use Normalization: %v", m.dictConfig.UseNormalization))
	content = append(content, fmt.Sprintf("  [R] Use MRRANK: %v", m.dictConfig.UseMRRANK))
	content = append(content, fmt.Sprintf("  [D] Deduplicate: %v", m.dictConfig.Deduplicate))
	content = append(content, fmt.Sprintf("  [P] Preferred Only: %v", m.dictConfig.PreferredOnly))
	content = append(content, fmt.Sprintf("  [T] Strip Punctuation: %v", m.dictConfig.StripPunctuation))
	content = append(content, fmt.Sprintf("  [W] Collapse Whitespace: %v", m.dictConfig.CollapseWhitespace))
	content = append(content, fmt.Sprintf("  [M] Exclude Numeric Only: %v", m.dictConfig.ExcludeNumericOnly))
	content = append(content, fmt.Sprintf("  [U] Exclude Punctuation Only: %v", m.dictConfig.ExcludePunctOnly))
	content = append(content, "")
	content = append(content, "Length Controls: [1/2] Min Term -/+1  [3/4] Max Term -/+5")
	content = append(content, "Token Controls: [5/6] Min Tokens -/+1  [7/8] Max Tokens -/+1")

	footerStyle := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Padding(0, 1)

	footer := footerStyle.Render("Letter Keys: Toggle Options | Number Keys: Adjust Values | Enter: Confirm | ESC: Cancel")

	combinedContent := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		strings.Join(content, "\n"),
		"",
		footer,
	)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorBorder).
		Width(width - 2).
		Height(height - 2)

	return borderStyle.Render(combinedContent)
}

// Output Configuration Selector
func (m *Model) renderOutputsConfigSelector(width, height int) string {
	headerStyle := lipgloss.NewStyle().
		Background(theme.ColorAccent).
		Foreground(theme.ColorBackground).
		Bold(true).
		Padding(0, 1)

	header := headerStyle.Render("◈ Output Formats")

	var content []string
	content = append(content, "Configure dictionary output formats:")
	content = append(content, "")
	content = append(content, "Primary Formats:")
	content = append(content, fmt.Sprintf("  [B] Emit BSV (Bar-Separated Values): %v", m.dictConfig.EmitBSV))
	content = append(content, fmt.Sprintf("  [H] Build HSQLDB Database: %v", m.dictConfig.BuildHSQLDB))
	content = append(content, fmt.Sprintf("  [L] Build Lucene Index: %v", m.dictConfig.BuildLucene))
	content = append(content, fmt.Sprintf("  [R] Use Rare Words Index: %v", m.dictConfig.UseRareWords))
	content = append(content, "")
	content = append(content, "Additional Formats:")
	content = append(content, fmt.Sprintf("  [T] Emit TSV (Tab-Separated Values): %v", m.dictConfig.EmitTSV))
	content = append(content, fmt.Sprintf("  [J] Emit JSONL (JSON Lines): %v", m.dictConfig.EmitJSONL))
	content = append(content, "")
	content = append(content, "Pipeline Support:")
	content = append(content, fmt.Sprintf("  [D] Emit Descriptor: %v", m.dictConfig.EmitDescriptor))
	content = append(content, fmt.Sprintf("  [P] Emit Pipeline Config: %v", m.dictConfig.EmitPipeline))
	content = append(content, fmt.Sprintf("  [M] Emit Manifest: %v", m.dictConfig.EmitManifest))
	content = append(content, "")
	content = append(content, "Note: BSV format is required for cTAKES compatibility")

	footerStyle := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Padding(0, 1)

	footer := footerStyle.Render("Letter Keys: Toggle Output Formats | Enter: Confirm | ESC: Cancel")

	combinedContent := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		strings.Join(content, "\n"),
		"",
		footer,
	)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorBorder).
		Width(width - 2).
		Height(height - 2)

	return borderStyle.Render(combinedContent)
}

// Relationships Configuration Selector
func (m *Model) renderRelationshipsConfigSelector(width, height int) string {
	headerStyle := lipgloss.NewStyle().
		Background(theme.ColorAccent).
		Foreground(theme.ColorBackground).
		Bold(true).
		Padding(0, 1)

	header := headerStyle.Render("◈ Relationship Settings")

	var content []string
	content = append(content, "Configure UMLS relationship processing:")
	content = append(content, "")
	content = append(content, fmt.Sprintf("Enable Relationships: %v", m.dictConfig.EnableRelationships))
	if m.dictConfig.EnableRelationships {
		content = append(content, fmt.Sprintf("Relationship Depth: %d (0-5)", m.dictConfig.RelationshipDepth))
		content = append(content, fmt.Sprintf("Selected Types: %d", len(m.dictConfig.RelationshipTypes)))
		content = append(content, "")
		content = append(content, "Available Relationship Types:")
		content = append(content, "  PAR (Parent), CHD (Child), RB (Broader)")
		content = append(content, "  RN (Narrower), SY (Synonym), isa (Is A)")
		content = append(content, "  part_of, may_treat, associated_with")
	} else {
		content = append(content, "")
		content = append(content, "Enable relationships to include semantic")
		content = append(content, "connections between medical concepts.")
	}
	content = append(content, "")
	content = append(content, "Controls:")
	content = append(content, "  [E] Toggle Enable Relationships")
	if m.dictConfig.EnableRelationships {
		content = append(content, "  [1/2] Depth -/+1")
		content = append(content, "  [T] Select Relationship Types")
	}

	footerStyle := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Padding(0, 1)

	footer := footerStyle.Render("Keys: Toggle/Adjust Settings | Enter: Confirm | ESC: Cancel")

	combinedContent := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		strings.Join(content, "\n"),
		"",
		footer,
	)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorBorder).
		Width(width - 2).
		Height(height - 2)

	return borderStyle.Render(combinedContent)
}

// Helper function for empty string handling
func ifEmpty(s, defaultValue string) string {
	if s == "" {
		return defaultValue
	}
	return s
}

// New interactive render functions for advanced configuration

// renderMemoryConfig - Interactive memory configuration with sliders/inputs
