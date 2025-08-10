package dashboard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/dictionary"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
	"github.com/ctakes-tui/ctakes-tui/internal/utils"
)

// Template definitions for quick dictionary configuration
type DictTemplate struct {
	Name        string
	Description string
	Config      DictionaryConfig
}

// Initialize available templates
func (m *Model) initTemplates() {
	m.dictTemplates = []DictTemplate{
		{
			Name:        "Full Medical Dictionary",
			Description: "Comprehensive medical terminology with all major vocabularies",
			Config: DictionaryConfig{
				Name:        "FullMedical",
				Description: "Complete medical dictionary with all vocabularies",
				TUIs:        dictionary.GetClinicalTUIs(),
				Vocabularies: []string{
					"SNOMEDCT_US", "RXNORM", "ICD10CM", "ICD10PCS", "LOINC", "CPT",
				},
				Languages:           []string{"ENG"},
				TermTypes:           []string{"PT", "SY", "AB", "ACR"},
				InitialHeapMB:       1024,
				MaxHeapMB:           2048,
				StackSizeMB:         8,
				ThreadCount:         4,
				BatchSize:           1000,
				CacheSize:           128,
				PreserveCase:        false,
				HandlePunctuation:   true,
				MinWordLength:       2,
				MaxWordLength:       80,
				MinTermLength:       3,
				MaxTermLength:       80,
				ExcludeSuppressible: true,
				ExcludeObsolete:     true,
				UseNormalization:    true,
				UseMRRANK:           true,
				Deduplicate:         true,
				EmitDescriptor:      true,
				EmitPipeline:        true,
				EmitManifest:        true,
				BuildLucene:         true,
				BuildHSQLDB:         false,
			},
		},
		{
			Name:        "Medication Dictionary",
			Description: "Focus on drugs, medications, and pharmaceutical terms",
			Config: DictionaryConfig{
				Name:        "Medications",
				Description: "Drug and medication terminology",
				TUIs:        dictionary.GetMedicationTUIs(),
				Vocabularies: []string{
					"RXNORM", "NDC", "VANDF", "NDDF",
				},
				Languages:           []string{"ENG"},
				TermTypes:           []string{"PT", "SY", "AB"},
				InitialHeapMB:       512,
				MaxHeapMB:           1024,
				StackSizeMB:         4,
				ThreadCount:         2,
				BatchSize:           500,
				CacheSize:           64,
				PreserveCase:        false,
				HandlePunctuation:   true,
				MinWordLength:       2,
				MaxWordLength:       80,
				MinTermLength:       3,
				MaxTermLength:       80,
				ExcludeSuppressible: true,
				ExcludeObsolete:     true,
				UseNormalization:    true,
				UseMRRANK:           true,
				Deduplicate:         true,
				EmitDescriptor:      true,
				EmitPipeline:        true,
				EmitManifest:        true,
				BuildLucene:         false,
				BuildHSQLDB:         true,
			},
		},
		{
			Name:        "Procedure Dictionary",
			Description: "Medical procedures, surgeries, and interventions",
			Config: DictionaryConfig{
				Name:        "Procedures",
				Description: "Medical procedures and interventions",
				TUIs:        dictionary.GetProcedureTUIs(),
				Vocabularies: []string{
					"CPT", "HCPCS", "ICD10PCS", "SNOMEDCT_US",
				},
				Languages:           []string{"ENG"},
				TermTypes:           []string{"PT", "SY"},
				InitialHeapMB:       768,
				MaxHeapMB:           1536,
				StackSizeMB:         6,
				ThreadCount:         3,
				BatchSize:           750,
				CacheSize:           96,
				PreserveCase:        false,
				HandlePunctuation:   true,
				MinWordLength:       2,
				MaxWordLength:       80,
				MinTermLength:       3,
				MaxTermLength:       80,
				ExcludeSuppressible: true,
				ExcludeObsolete:     true,
				UseNormalization:    true,
				UseMRRANK:           true,
				Deduplicate:         true,
				EmitDescriptor:      true,
				EmitPipeline:        true,
				EmitManifest:        true,
				BuildLucene:         false,
				BuildHSQLDB:         true,
			},
		},
		{
			Name:        "Diagnosis Dictionary",
			Description: "Diseases, conditions, and diagnostic terminology",
			Config: DictionaryConfig{
				Name:        "Diagnoses",
				Description: "Disease and condition terminology",
				TUIs:        dictionary.GetDiagnosisTUIs(),
				Vocabularies: []string{
					"ICD10CM", "ICD9CM", "SNOMEDCT_US", "NCI",
				},
				Languages:           []string{"ENG"},
				TermTypes:           []string{"PT", "SY", "AB"},
				InitialHeapMB:       768,
				MaxHeapMB:           1536,
				StackSizeMB:         6,
				ThreadCount:         3,
				BatchSize:           750,
				CacheSize:           96,
				PreserveCase:        false,
				HandlePunctuation:   true,
				MinWordLength:       2,
				MaxWordLength:       80,
				MinTermLength:       3,
				MaxTermLength:       80,
				ExcludeSuppressible: true,
				ExcludeObsolete:     true,
				UseNormalization:    true,
				UseMRRANK:           true,
				Deduplicate:         true,
				EmitDescriptor:      true,
				EmitPipeline:        true,
				EmitManifest:        true,
				BuildLucene:         true,
				BuildHSQLDB:         false,
			},
		},
		{
			Name:        "Laboratory Dictionary",
			Description: "Lab tests, measurements, and clinical observations",
			Config: DictionaryConfig{
				Name:        "Laboratory",
				Description: "Laboratory and clinical measurement terminology",
				TUIs:        dictionary.GetLaboratoryTUIs(),
				Vocabularies: []string{
					"LOINC", "SNOMEDCT_US",
				},
				Languages:           []string{"ENG"},
				TermTypes:           []string{"PT", "SY", "AB"},
				InitialHeapMB:       512,
				MaxHeapMB:           1024,
				StackSizeMB:         4,
				ThreadCount:         2,
				BatchSize:           500,
				CacheSize:           64,
				PreserveCase:        false,
				HandlePunctuation:   true,
				MinWordLength:       2,
				MaxWordLength:       80,
				MinTermLength:       3,
				MaxTermLength:       80,
				ExcludeSuppressible: true,
				ExcludeObsolete:     true,
				UseNormalization:    true,
				UseMRRANK:           true,
				Deduplicate:         true,
				EmitDescriptor:      true,
				EmitPipeline:        true,
				EmitManifest:        true,
				BuildLucene:         false,
				BuildHSQLDB:         true,
				EmitTSV:             true,
			},
		},
		{
			Name:        "Minimal Dictionary",
			Description: "Lightweight configuration for testing and development",
			Config: DictionaryConfig{
				Name:        "TestDict",
				Description: "Minimal dictionary for testing",
				TUIs: []string{
					"T047", // Disease or Syndrome
					"T121", // Pharmacologic Substance
					"T061", // Therapeutic Procedure
				},
				Vocabularies: []string{
					"SNOMEDCT_US", "RXNORM",
				},
				Languages:           []string{"ENG"},
				TermTypes:           []string{"PT"},
				InitialHeapMB:       256,
				MaxHeapMB:           512,
				StackSizeMB:         2,
				ThreadCount:         1,
				BatchSize:           100,
				CacheSize:           32,
				PreserveCase:        false,
				HandlePunctuation:   false,
				MinWordLength:       3,
				MaxWordLength:       50,
				MinTermLength:       3,
				MaxTermLength:       50,
				ExcludeSuppressible: true,
				ExcludeObsolete:     true,
				UseNormalization:    false,
				UseMRRANK:           false,
				Deduplicate:         true,
				EmitDescriptor:      true,
				EmitPipeline:        true,
				EmitManifest:        true,
				BuildLucene:         false,
				BuildHSQLDB:         false,
			},
		},
	}
}

// Render template selection screen with consistent design matching TUI/Vocab selectors
func (m *Model) renderTemplateSelector(width, height int) string {
	// Simple clean header like Semantic Types
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Render("Dictionary Templates"),
		strings.Repeat("─", width-4),
		fmt.Sprintf("%s %d templates available", theme.CircleBlue, len(m.dictTemplates)),
		"",
	}
	// Calculate visible window
	headerLines := 4 // Title + divider + count + blank
	footerLines := 3 // Divider + help + blank
	itemHeight := 4  // Each template takes 4 lines (name, desc, summary, blank)
	visibleHeight := height - headerLines - footerLines
	maxVisibleItems := visibleHeight / itemHeight
	if maxVisibleItems < 1 {
		maxVisibleItems = 1
	}

	// Calculate scroll position with proper scrolling
	startIdx := 0
	if m.templateCursor >= maxVisibleItems {
		// Keep cursor in middle of view when possible
		startIdx = m.templateCursor - (maxVisibleItems / 2)
		if startIdx < 0 {
			startIdx = 0
		}
	}

	// Ensure we don't go past the end
	if startIdx+maxVisibleItems > len(m.dictTemplates) {
		startIdx = len(m.dictTemplates) - maxVisibleItems
		if startIdx < 0 {
			startIdx = 0
		}
	}

	endIdx := startIdx + maxVisibleItems
	if endIdx > len(m.dictTemplates) {
		endIdx = len(m.dictTemplates)
	}

	// Render visible templates with clean circle indicators
	for i := startIdx; i < endIdx; i++ {
		t := m.dictTemplates[i]
		isFocused := i == m.templateCursor

		// Use circle indicator for focus
		indicator := theme.CircleBlack
		if isFocused {
			indicator = theme.CircleBlue
		}

		// Get template type indicator
		typeIndicator := theme.CirclePurple // Default for templates
		if strings.Contains(strings.ToLower(t.Name), "full") {
			typeIndicator = theme.CircleGreen
		} else if strings.Contains(strings.ToLower(t.Name), "minimal") {
			typeIndicator = theme.CircleOrange
		} else if strings.Contains(strings.ToLower(t.Name), "clinical") {
			typeIndicator = theme.CircleBlue
		}

		// Format template info
		summary := summarizeTemplate(t)
		line := fmt.Sprintf("  %s  %s  %-30s", indicator, typeIndicator, t.Name)

		// Apply full-width highlighting for focused item
		if isFocused {
			if len(line) < width {
				line = line + strings.Repeat(" ", width-len(line))
			}
			line = lipgloss.NewStyle().
				Background(theme.ColorSelectionActive).
				Foreground(theme.ColorBackground).
				Bold(true).
				Render(line)
		}

		lines = append(lines, line)

		// Add description on next line with indent
		descLine := fmt.Sprintf("        %s", t.Description)
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render(descLine))

		// Add summary on next line with indent
		summaryLine := fmt.Sprintf("        %s", summary)
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render(summaryLine))
		lines = append(lines, "") // Spacing between templates
	}

	// Add scroll indicator if needed
	if len(m.dictTemplates) > maxVisibleItems {
		lines = append(lines, theme.RenderScrollIndicator(startIdx, endIdx, len(m.dictTemplates), width))
	}

	// Simple footer
	lines = append(lines, strings.Repeat("─", width-4))
	lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).
		Render("Enter: Select  ↑/↓: Navigate  Tab: Details  ESC: Cancel"))
	// Return without additional padding since rows handle their own padding
	return strings.Join(lines, "\n")
}

// Handle template selection keys
func (m *Model) handleTemplateKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.templateCursor > 0 {
			m.templateCursor--
		}
	case "down", "j":
		if m.templateCursor < len(m.dictTemplates)-1 {
			m.templateCursor++
		}
	case "enter":
		// Apply selected template
		if m.templateCursor < len(m.dictTemplates) {
			template := m.dictTemplates[m.templateCursor]
			m.dictConfig = template.Config
			// Keep existing UMLS path if set
			if m.umlsPath == "" {
				// Could set a default or leave empty
			}
		}
		m.dictBuilderState = DictStateMainMenu
	case "esc":
		m.dictBuilderState = DictStateMainMenu
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

// Helper function to get template preview for the preview panel
func (m *Model) renderTemplatePreview(width, height int) string {
	if m.templateCursor >= len(m.dictTemplates) {
		return ""
	}

	template := m.dictTemplates[m.templateCursor]

	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Render(template.Name),
		strings.Repeat("─", width-4),
		"",
		template.Description,
		"",
		lipgloss.NewStyle().Bold(true).Render("Configuration:"),
		"",
		fmt.Sprintf("Name: %s", template.Config.Name),
		fmt.Sprintf("Semantic Types: %d", len(template.Config.TUIs)),
		fmt.Sprintf("Vocabularies: %d", len(template.Config.Vocabularies)),
		"",
		lipgloss.NewStyle().Bold(true).Render("Memory Settings:"),
		fmt.Sprintf("Max Heap: %d MB", template.Config.MaxHeapMB),
		fmt.Sprintf("Threads: %d", template.Config.ThreadCount),
		fmt.Sprintf("Batch Size: %d", template.Config.BatchSize),
		"",
		lipgloss.NewStyle().Bold(true).Render("Vocabularies:"),
	}

	// Add vocabulary list
	for _, vocab := range template.Config.Vocabularies {
		lines = append(lines, fmt.Sprintf("  - %s", vocab))
	}

	if len(template.Config.TUIs) > 0 {
		lines = append(lines, "", lipgloss.NewStyle().Bold(true).Render("Semantic Types:"))
		maxTUIs := 8 // Limit displayed TUIs
		for i, tui := range template.Config.TUIs {
			if i >= maxTUIs {
				lines = append(lines, fmt.Sprintf("  ... and %d more", len(template.Config.TUIs)-maxTUIs))
				break
			}
			lines = append(lines, fmt.Sprintf("  - %s", tui))
		}
	}

	// Clip preview to fit panel height to avoid layout breakage
	lines = clipTemplateHeight(lines, height)
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// Local clip helper to avoid import cycles and keep preview within bounds
func clipTemplateHeight(lines []string, max int) []string {
	if max <= 0 || len(lines) <= max {
		return lines
	}
	if max < 2 {
		return lines[:max]
	}
	clipped := append([]string{}, lines[:max-1]...)
	clipped = append(clipped, "…")
	return clipped
}

// summarizeTemplate builds a concise, one-line summary suitable for the list view
func summarizeTemplate(t DictTemplate) string {
	luc := checkBool(t.Config.BuildLucene)
	hsq := checkBool(t.Config.BuildHSQLDB)
	tsv := checkBool(t.Config.EmitTSV)
	jsn := checkBool(t.Config.EmitJSONL)
	return fmt.Sprintf("TUIs:%d Voc:%d Heap:%dMB Thr:%d Out:L[%s]H[%s]T[%s]J[%s]",
		len(t.Config.TUIs), len(t.Config.Vocabularies), t.Config.MaxHeapMB, t.Config.ThreadCount,
		luc, hsq, tsv, jsn,
	)
}

func checkBool(b bool) string {
	if b {
		return lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render(utils.GetIcon("check"))
	}
	return " "
}
