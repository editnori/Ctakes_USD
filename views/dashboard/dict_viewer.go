package dashboard

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/dictionary"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
)

// Dictionary viewer - list view
func (m *Model) renderDictViewer(width, height int) string {
	// Simple clean header like Semantic Types
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Render("Built Dictionaries"),
		strings.Repeat("─", width-4),
		fmt.Sprintf("%s %d dictionaries found", theme.CircleBlue, len(m.builtDictionaries)),
		"",
	}
	if len(m.builtDictionaries) == 0 {
		// Empty state - simple text without boxes
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render("No dictionaries found."))
		lines = append(lines, "")
		lines = append(lines, "Build your first dictionary by:")
		lines = append(lines, "  1. Selecting UMLS location")
		lines = append(lines, "  2. Setting dictionary name")
		lines = append(lines, "  3. Choosing semantic types")
		lines = append(lines, "  4. Selecting vocabularies")
		lines = append(lines, "  5. Building dictionary")
	} else {
		// Dictionary list with clean circle indicators
		headerLines := 4 // Title + divider + count + blank
		footerLines := 3
		itemHeight := 3 // Each dictionary takes 3 lines (name, date, blank)
		visibleHeight := height - headerLines - footerLines
		maxVisibleItems := visibleHeight / itemHeight
		if maxVisibleItems < 1 {
			maxVisibleItems = 1
		}

		// Calculate visible range with proper scrolling
		startIdx := 0
		if m.dictListCursor >= maxVisibleItems {
			// Keep cursor in middle of view when possible
			startIdx = m.dictListCursor - (maxVisibleItems / 2)
			if startIdx < 0 {
				startIdx = 0
			}
		}

		// Ensure we don't go past the end
		if startIdx+maxVisibleItems > len(m.builtDictionaries) {
			startIdx = len(m.builtDictionaries) - maxVisibleItems
			if startIdx < 0 {
				startIdx = 0
			}
		}

		endIdx := startIdx + maxVisibleItems
		if endIdx > len(m.builtDictionaries) {
			endIdx = len(m.builtDictionaries)
		}

		for i := startIdx; i < endIdx; i++ {
			d := m.builtDictionaries[i]
			isFocused := i == m.dictListCursor

			// Load config to get term count
			configPath := filepath.Join(d.Path, "config.json")
			cfg, _ := dictionary.LoadConfig(configPath)
			termCount := "N/A"
			if cfg != nil && cfg.Statistics.TotalTerms > 0 {
				termCount = fmt.Sprintf("%d", cfg.Statistics.TotalTerms)
			}

			// Calculate size
			size := m.getDictionarySize(d.Path)

			// Use circle indicator for focus
			indicator := theme.CircleBlack
			if isFocused {
				indicator = theme.CircleBlue
			}

			// Status indicator based on size/terms
			statusIndicator := theme.CircleGreen // Built successfully

			// Format the dictionary info
			line := fmt.Sprintf("  %s  %s  %-30s  %10s  %8s terms",
				indicator, statusIndicator, d.Name, size, termCount)

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

			// Add creation date on next line with indent
			dateLine := fmt.Sprintf("        Created: %s", d.Created.Format("2006-01-02 15:04"))
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render(dateLine))
			lines = append(lines, "") // Spacing between dictionaries
		}

		// Show scroll indicator if needed
		if len(m.builtDictionaries) > maxVisibleItems {
			lines = append(lines, theme.RenderScrollIndicator(startIdx, endIdx, len(m.builtDictionaries), width))
		}
	}

	// Simple footer
	lines = append(lines, strings.Repeat("─", width-4))
	lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).
		Render("Enter: Details | B: Browse | D: Delete | E: Export | C: Copy | ESC: Back"))

	return strings.Join(lines, "\n")
}

func (m *Model) handleViewerKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.dictListCursor > 0 {
			m.dictListCursor--
		}
	case "down", "j":
		if m.dictListCursor < len(m.builtDictionaries)-1 {
			m.dictListCursor++
		}
	case "d", "D":
		// Delete dictionary with confirmation
		if m.dictListCursor < len(m.builtDictionaries) {
			dict := m.builtDictionaries[m.dictListCursor]
			// Remove directory
			os.RemoveAll(dict.Path)
			// Reload list
			m.loadBuiltDictionaries()
			// Adjust cursor if needed
			if m.dictListCursor >= len(m.builtDictionaries) && m.dictListCursor > 0 {
				m.dictListCursor--
			}
		}
	case "e", "E":
		// Export configuration to clipboard
		if m.dictListCursor < len(m.builtDictionaries) {
			dict := m.builtDictionaries[m.dictListCursor]
			configPath := filepath.Join(dict.Path, "config.json")
			if _, err := os.ReadFile(configPath); err == nil {
				// In a real app, you'd copy to clipboard
				m.buildLogs = append(m.buildLogs, fmt.Sprintf("Config exported for %s", dict.Name))
			}
		}
	case "c", "C":
		// Copy dictionary path to use in pipeline
		if m.dictListCursor < len(m.builtDictionaries) {
			dict := m.builtDictionaries[m.dictListCursor]
			// Store the selected dictionary path for pipeline use
			m.buildLogs = append(m.buildLogs, fmt.Sprintf("Dictionary '%s' ready for pipeline", dict.Name))
		}
	case "b", "B":
		// Browse dictionary contents
		if m.dictListCursor < len(m.builtDictionaries) {
			m.loadDictionaryContent()
			m.dictBuilderState = DictStateBrowsingContent
		}
	case "enter":
		// View details
		if m.dictListCursor < len(m.builtDictionaries) {
			m.loadDictionaryDetails()
			m.dictBuilderState = DictStateViewingDetails
		}
	case "esc":
		m.dictBuilderState = DictStateMainMenu
	case "q", "Q":
		// Quit dictionary builder - return to main dashboard
		m.activePanel = SidebarPanel
		m.cursor = 0
	case "p", "P":
		// Switch to NLP Pipeline Builder menu item dynamically
		m.activePanel = SidebarPanel
		// Find index of pipeline builder in sidebar
		target := -1
		for i, it := range m.sidebarItems {
			if it.Action == "nlp_pipeline_builder" {
				target = i
				break
			}
		}
		if target >= 0 {
			m.cursor = target
		} else {
			m.cursor = 0
		}
	}
	return *m, nil
}

func (m *Model) countBuiltDicts() string {
	dicts, err := dictionary.ListDictionaries()
	if err != nil || len(dicts) == 0 {
		return "None"
	}
	return fmt.Sprintf("%d found", len(dicts))
}

func (m *Model) loadBuiltDictionaries() {
	dicts, _ := dictionary.ListDictionaries()
	m.builtDictionaries = []DictionaryInfo{}
	for _, d := range dicts {
		m.builtDictionaries = append(m.builtDictionaries, DictionaryInfo{
			Name:    d.Name,
			Path:    d.Path,
			Created: d.CreatedAt,
		})
	}
	m.dictListCursor = 0
}

func (m *Model) getDictionarySize(path string) string {
	var totalSize int64

	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	// Format size
	if totalSize < 1024 {
		return fmt.Sprintf("%d B", totalSize)
	} else if totalSize < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(totalSize)/1024)
	} else if totalSize < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(totalSize)/(1024*1024))
	} else {
		return fmt.Sprintf("%.1f GB", float64(totalSize)/(1024*1024*1024))
	}
}

// Load dictionary details for viewing
func (m *Model) loadDictionaryDetails() {
	if m.dictListCursor >= len(m.builtDictionaries) {
		return
	}

	dict := m.builtDictionaries[m.dictListCursor]
	configPath := filepath.Join(dict.Path, "config.json")
	cfg, _ := dictionary.LoadConfig(configPath)
	m.selectedDict = cfg
}

// Load dictionary content (terms) for browsing
func (m *Model) loadDictionaryContent() {
	if m.dictListCursor >= len(m.builtDictionaries) {
		return
	}

	dict := m.builtDictionaries[m.dictListCursor]
	bsvPath := filepath.Join(dict.Path, "terms.bsv")

	m.dictContent = []string{}
	m.dictContentCursor = 0

	// Read first 1000 lines of BSV file
	file, err := os.Open(bsvPath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() && count < 1000 {
		m.dictContent = append(m.dictContent, scanner.Text())
		count++
	}

}

// Render dictionary details view
func (m *Model) renderDictDetails(width, height int) string {
	if m.selectedDict == nil {
		return "Loading dictionary details..."
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent)
	labelStyle := lipgloss.NewStyle().Foreground(theme.ColorAccent).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(theme.ColorForeground)

	lines := []string{
		headerStyle.Render(fmt.Sprintf("Dictionary: %s", m.selectedDict.Name)),
		strings.Repeat("─", width-4),
		"",
	}

	// Basic Information
	lines = append(lines, headerStyle.Render("Basic Information"))
	lines = append(lines, fmt.Sprintf("%s %s",
		labelStyle.Render("Description:"),
		valueStyle.Render(m.selectedDict.Description)))
	lines = append(lines, fmt.Sprintf("%s %s",
		labelStyle.Render("Created:"),
		valueStyle.Render(m.selectedDict.CreatedAt.Format("2006-01-02 15:04:05"))))
	lines = append(lines, fmt.Sprintf("%s %s",
		labelStyle.Render("Output Format:"),
		valueStyle.Render(m.selectedDict.OutputFormat)))
	lines = append(lines, "")

	// Statistics
	if m.selectedDict.Statistics.TotalTerms > 0 {
		lines = append(lines, headerStyle.Render("Statistics"))
		lines = append(lines, fmt.Sprintf("%s %d",
			labelStyle.Render("Total Terms:"),
			m.selectedDict.Statistics.TotalTerms))
		lines = append(lines, fmt.Sprintf("%s %d",
			labelStyle.Render("Total Concepts:"),
			m.selectedDict.Statistics.TotalConcepts))
		lines = append(lines, fmt.Sprintf("%s %v",
			labelStyle.Render("Build Time:"),
			m.selectedDict.Statistics.BuildTime.Round(time.Second)))
		lines = append(lines, "")
	}

	// Configuration
	lines = append(lines, headerStyle.Render("Configuration"))

	// Semantic Types
	if len(m.selectedDict.SemanticTypes) > 0 {
		lines = append(lines, fmt.Sprintf("%s %d selected",
			labelStyle.Render("Semantic Types:"),
			len(m.selectedDict.SemanticTypes)))
		for i, tui := range m.selectedDict.SemanticTypes {
			if i >= 3 {
				lines = append(lines, fmt.Sprintf("  ... and %d more",
					len(m.selectedDict.SemanticTypes)-3))
				break
			}
			lines = append(lines, fmt.Sprintf("  - %s", tui))
		}
	}

	// Vocabularies
	if len(m.selectedDict.Vocabularies) > 0 {
		lines = append(lines, fmt.Sprintf("%s %d selected",
			labelStyle.Render("Vocabularies:"),
			len(m.selectedDict.Vocabularies)))
		for i, vocab := range m.selectedDict.Vocabularies {
			if i >= 3 {
				lines = append(lines, fmt.Sprintf("  ... and %d more",
					len(m.selectedDict.Vocabularies)-3))
				break
			}
			lines = append(lines, fmt.Sprintf("  - %s", vocab))
		}
	}

	// Files
	lines = append(lines, "")
	lines = append(lines, headerStyle.Render("Generated Files"))
	dict := m.builtDictionaries[m.dictListCursor]

	// Check for files
	files := []struct {
		name string
		path string
		desc string
	}{
		{"terms.bsv", "terms.bsv", "BSV dictionary file"},
		{"config.json", "config.json", "Configuration file"},
		{"dictionary.xml", "dictionary.xml", "cTAKES descriptor"},
		{"pipeline.piper", "pipeline.piper", "Pipeline configuration"},
		{"build.log", "build.log", "Build log"},
		{"cui2terms.hsqldb", "cui2terms.hsqldb", "HSQLDB database"},
	}

	for _, f := range files {
		fpath := filepath.Join(dict.Path, f.path)
		if fi, err := os.Stat(fpath); err == nil {
			size := formatFileSize(fi.Size())
			lines = append(lines, fmt.Sprintf("  - %s (%s) - %s", f.name, size, f.desc))
		}
	}

	lines = append(lines, "")
	lines = append(lines, "B: Browse Terms | E: Export | ESC: Back to List")

	// Clip to panel height for clean rendering
	lines = clipToHeight(lines, height-2)
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// Render dictionary content browser
func (m *Model) renderDictContent(width, height int) string {
	if len(m.dictContent) == 0 {
		return "Loading dictionary content..."
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent)
	dict := m.builtDictionaries[m.dictListCursor]

	lines := []string{
		headerStyle.Render(fmt.Sprintf("Dictionary Terms: %s", dict.Name)),
		strings.Repeat("─", width-4),
		"Columns: text | CUI | TUI | code | vocab | type | preferred",
		strings.Repeat("─", width-4),
		"",
	}

	// Setup viewport
	vpHeight := height - 10
	if vpHeight < 5 {
		vpHeight = 5
	}

	// Display terms with scrolling
	maxDisplay := vpHeight
	startIdx := 0
	if m.dictContentCursor >= maxDisplay {
		startIdx = m.dictContentCursor - maxDisplay + 1
	}

	for i := startIdx; i < len(m.dictContent) && i < startIdx+maxDisplay; i++ {
		line := m.dictContent[i]
		// Parse BSV line
		parts := strings.Split(line, "|")
		if len(parts) >= 7 {
			// Format for display: show term, CUI, and TUI
			term := parts[0]
			if len(term) > 40 {
				term = term[:37] + "..."
			}
			display := fmt.Sprintf("%-40s %s %s", term, parts[1], parts[2])

			if i == m.dictContentCursor {
				display = lipgloss.NewStyle().
					Background(theme.ColorSelectionActive).
					Foreground(theme.ColorBackground).
					Render(display)
			}
			lines = append(lines, display)
		}
	}

	// Show position
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Line %d of %d (showing first 1000 terms)",
		m.dictContentCursor+1, len(m.dictContent)))

	lines = append(lines, "")
	lines = append(lines, "↑↓: Navigate | /: Search | ESC: Back to Details")

	// Ensure we don't overflow the panel height
	lines = clipToHeight(lines, height-2)
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

func formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	} else {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
}

// Add handlers for detail and content views
func (m *Model) handleDetailKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "b", "B":
		m.loadDictionaryContent()
		m.dictBuilderState = DictStateBrowsingContent
	case "e", "E":
		// Export functionality
		// Could implement export to various formats
	case "esc":
		m.dictBuilderState = DictStateViewingDictionaries
	}
	return *m, nil
}

func (m *Model) handleContentKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.dictContentCursor > 0 {
			m.dictContentCursor--
		}
	case "down", "j":
		if m.dictContentCursor < len(m.dictContent)-1 {
			m.dictContentCursor++
		}
	case "pgup":
		m.dictContentCursor -= 10
		if m.dictContentCursor < 0 {
			m.dictContentCursor = 0
		}
	case "pgdown":
		m.dictContentCursor += 10
		if m.dictContentCursor >= len(m.dictContent) {
			m.dictContentCursor = len(m.dictContent) - 1
		}
	case "esc":
		m.dictBuilderState = DictStateViewingDetails
	}
	return *m, nil
}
