package dashboard

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
	"github.com/ctakes-tui/ctakes-tui/internal/utils"
)

// Advanced configuration menus for dictionary builder

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// cbox renders a colored checkbox: green check when true, dim outline when false.
func cbox(on bool) string {
	if on {
		return lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render("[" + utils.GetIcon("check") + "]")
	}
	return lipgloss.NewStyle().Foreground(theme.ColorBorder).Render("[ ]")
}

// Configuration scroll state
type ConfigScrollState struct {
	Offset     int
	ViewHeight int
}

// Memory configuration screen
func (m *Model) renderMemoryConfig(width, height int) string {
	// Simple clean header like Semantic Types
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Render("Memory Configuration"),
		strings.Repeat("─", width-4),
		"",
	}

	fields := []struct {
		name  string
		value int
		min   int
		max   int
		desc  string
	}{
		{"Initial Heap (MB)", m.dictConfig.InitialHeapMB, 512, 3072, "Starting memory allocation"},
		{"Max Heap (MB)", m.dictConfig.MaxHeapMB, 512, 8192, "Maximum memory allocation"},
		{"Stack Size (MB)", m.dictConfig.StackSizeMB, 1, 64, "Thread stack size"},
	}

	// Render fields with clean circle indicators like TUI selector
	for i, field := range fields {
		isFocused := i == m.configField

		// Use circle indicator for focus state
		indicator := theme.CircleBlack
		if isFocused {
			indicator = theme.CircleBlue
		}

		// Format the field display cleanly
		valueStr := fmt.Sprintf("%4d MB", field.value)
		line := fmt.Sprintf("  %s  %-20s: %-10s", indicator, field.name, valueStr)

		// Apply full-width highlighting for focused item
		if isFocused {
			// Pad to full width
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
	}

	// Simple footer without boxes
	lines = append(lines, "")
	lines = append(lines, strings.Repeat("─", width-4))
	lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).
		Render("↑/↓: Select  ←/→: Adjust  Enter: Save  ESC: Back"))

	return strings.Join(lines, "\n")
}

// Processing configuration screen
func (m *Model) renderProcessingConfig(width, height int) string {
	// Simple clean header
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Render("Processing Configuration"),
		strings.Repeat("─", width-4),
		"",
	}

	fields := []struct {
		name  string
		value interface{}
		desc  string
	}{
		{"Thread Count", m.dictConfig.ThreadCount, "Number of parallel threads (1-16)"},
		{"Batch Size", m.dictConfig.BatchSize, "Records per batch (100-10000)"},
		{"Cache Size (MB)", m.dictConfig.CacheSize, "In-memory cache size"},
		{"Temp Directory", m.dictConfig.TempDirectory, "Temporary file location"},
		{"Preserve Case", m.dictConfig.PreserveCase, "Keep original text casing"},
		{"Handle Punctuation", m.dictConfig.HandlePunctuation, "Process punctuation marks"},
		{"Min Word Length", m.dictConfig.MinWordLength, "Minimum word length (1-10)"},
		{"Max Word Length", m.dictConfig.MaxWordLength, "Maximum word length (10-256)"},
	}

	// Render fields with clean circle indicators
	for i, field := range fields {
		isFocused := i == m.configField

		// Use circle indicator
		indicator := theme.CircleBlack
		if isFocused {
			indicator = theme.CircleBlue
		}

		// Format value with circles for booleans
		valueStr := fmt.Sprintf("%v", field.value)
		if boolVal, ok := field.value.(bool); ok {
			if boolVal {
				valueStr = theme.CircleGreen
			} else {
				valueStr = theme.CircleBlack
			}
		}

		line := fmt.Sprintf("  %s  %-20s: %-12s", indicator, field.name, valueStr)

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
	}

	// Simple footer
	lines = append(lines, "")
	lines = append(lines, strings.Repeat("─", width-4))
	lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).
		Render("↑/↓: Select  Space: Toggle  ←/→: Adjust  Enter: Save  ESC: Back"))

	return strings.Join(lines, "\n")
}

// Filter configuration screen
func (m *Model) renderFilterConfig(width, height int) string {
	// Simple clean header
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Render("Filter Configuration"),
		strings.Repeat("─", width-4),
		"",
	}
	// Group filters by category with rounded containers
	sections := []struct {
		title   string
		filters []struct {
			name  string
			value interface{}
			desc  string
		}
	}{
		{
			"Length Filters",
			[]struct {
				name  string
				value interface{}
				desc  string
			}{
				{"Min Term Length", m.dictConfig.MinTermLength, "Minimum term length"},
				{"Max Term Length", m.dictConfig.MaxTermLength, "Maximum term length"},
				{"Min Tokens", m.dictConfig.MinTokens, "Minimum words per term"},
				{"Max Tokens", m.dictConfig.MaxTokens, "Maximum words per term"},
			},
		},
		{
			"Content Filters",
			[]struct {
				name  string
				value interface{}
				desc  string
			}{
				{"Exclude Suppressible", m.dictConfig.ExcludeSuppressible, "Skip suppressed terms"},
				{"Exclude Obsolete", m.dictConfig.ExcludeObsolete, "Skip obsolete terms"},
				{"Exclude Numeric Only", m.dictConfig.ExcludeNumericOnly, "Skip number-only terms"},
				{"Exclude Punct Only", m.dictConfig.ExcludePunctOnly, "Skip punctuation-only"},
				{"Preferred Only", m.dictConfig.PreferredOnly, "Only preferred terms"},
			},
		},
		{
			"Processing Options",
			[]struct {
				name  string
				value interface{}
				desc  string
			}{
				{"Case Sensitive", m.dictConfig.CaseSensitive, "Preserve case distinctions"},
				{"Use Normalization", m.dictConfig.UseNormalization, "Apply text normalization"},
				{"Use MRRANK", m.dictConfig.UseMRRANK, "Use MRRANK for ranking"},
				{"Deduplicate", m.dictConfig.Deduplicate, "Remove duplicate terms"},
				{"Strip Punctuation", m.dictConfig.StripPunctuation, "Remove punctuation"},
				{"Collapse Whitespace", m.dictConfig.CollapseWhitespace, "Normalize spaces"},
			},
		},
	}

	// Calculate total fields and scrolling
	totalFields := 0
	for _, section := range sections {
		totalFields += len(section.filters)
	}

	visibleHeight := height - 10
	// scrollOffset := 0 // Not used currently

	// Calculate scroll position to keep current field visible
	if m.configField > 0 {
		currentLine := 0
		for i := 0; i < m.configField; i++ {
			currentLine++ // Each field is 1 line
		}
		if currentLine >= visibleHeight/2 {
			// scrollOffset = currentLine - visibleHeight/2 // Not used currently
		}
	}

	fieldIndex := 0
	for sIdx, section := range sections {
		// Simple section header
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(theme.ColorSecondary).Render(section.title))
		lines = append(lines, "")

		for _, filter := range section.filters {
			isFocused := fieldIndex == m.configField

			// Use circle indicator
			indicator := theme.CircleBlack
			if isFocused {
				indicator = theme.CircleBlue
			}

			// Format value with circles for booleans
			valueStr := fmt.Sprintf("%v", filter.value)
			if boolVal, ok := filter.value.(bool); ok {
				if boolVal {
					valueStr = theme.CircleGreen
				} else {
					valueStr = theme.CircleBlack
				}
			}

			line := fmt.Sprintf("  %s  %-22s: %-8s", indicator, filter.name, valueStr)

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
			fieldIndex++
		}

		// Add spacing between sections
		if sIdx < len(sections)-1 {
			lines = append(lines, "")
		}
	}
	// Simple footer
	lines = append(lines, "")
	lines = append(lines, strings.Repeat("─", width-4))
	lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).
		Render("↑/↓: Navigate  Space: Toggle  ←/→: Adjust  Enter: Save  ESC: Back"))

	return strings.Join(lines, "\n")
}

// Output format configuration
func (m *Model) renderOutputConfig(width, height int) string {
	// Simple clean header
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Render("Output Configuration"),
		strings.Repeat("─", width-4),
		"",
	}
	// Primary format section
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(theme.ColorSecondary).Render("Primary Format"))
	lines = append(lines, fmt.Sprintf("  %s  BSV (Bar-Separated Values) - Default cTAKES format", theme.CircleGreen))
	lines = append(lines, "")

	// Additional formats section
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(theme.ColorSecondary).Render("Additional Formats"))

	formats := []struct {
		name    string
		enabled bool
		desc    string
		field   *bool
	}{
		{"HSQLDB", m.dictConfig.BuildHSQLDB, "Embedded database format", &m.dictConfig.BuildHSQLDB},
		{"Lucene Index", m.dictConfig.BuildLucene, "Full-text search index", &m.dictConfig.BuildLucene},
		{"TSV Export", m.dictConfig.EmitTSV, "Tab-separated values", &m.dictConfig.EmitTSV},
		{"JSON Lines", m.dictConfig.EmitJSONL, "Newline-delimited JSON", &m.dictConfig.EmitJSONL},
		{"Rare Words", m.dictConfig.UseRareWords, "Include rare word indexing", &m.dictConfig.UseRareWords},
	}

	// Calculate scrolling for output formats
	visibleHeight := height - 15
	scrollOffset := 0
	// total items computed later after metadata defined if needed

	if m.configField > 3 && m.configField*2 > visibleHeight {
		scrollOffset = m.configField - visibleHeight/2
		if scrollOffset < 0 {
			scrollOffset = 0
		}
	}

	for i, format := range formats {
		isFocused := i == m.configField

		// Use circle indicators
		indicator := theme.CircleBlack
		if isFocused {
			indicator = theme.CircleBlue
		}

		checkbox := theme.CircleBlack
		if format.enabled {
			checkbox = theme.CircleGreen
		}

		line := fmt.Sprintf("  %s  %s  %-15s - %s", indicator, checkbox, format.name, format.desc)

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
	}

	lines = append(lines, "")

	// Metadata files section
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(theme.ColorSecondary).Render("Metadata Files"))
	metadata := []struct {
		name    string
		enabled bool
		desc    string
	}{
		{"Descriptor XML", m.dictConfig.EmitDescriptor, "cTAKES descriptor file"},
		{"Pipeline Config", m.dictConfig.EmitPipeline, "Processing pipeline configuration"},
		{"Manifest JSON", m.dictConfig.EmitManifest, "Build configuration manifest"},
	}

	for i, meta := range metadata {
		isFocused := i+len(formats) == m.configField

		// Use circle indicators
		indicator := theme.CircleBlack
		if isFocused {
			indicator = theme.CircleBlue
		}

		checkbox := theme.CircleBlack
		if meta.enabled {
			checkbox = theme.CircleGreen
		}

		line := fmt.Sprintf("  %s  %s  %-15s - %s", indicator, checkbox, meta.name, meta.desc)

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
	}

	// Simple footer
	lines = append(lines, "")
	lines = append(lines, strings.Repeat("─", width-4))
	lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).
		Render("↑/↓: Navigate  Space: Toggle  Enter: Save  ESC: Back"))

	return strings.Join(lines, "\n")
}

// Relationship configuration
func (m *Model) renderRelationshipConfig(width, height int) string {
	// Simple clean header
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Render("Relationship Configuration"),
		strings.Repeat("─", width-4),
		"",
	}
	// Enable/disable relationships with circle indicators
	isFocused := m.configField == 0
	indicator := theme.CircleBlack
	if isFocused {
		indicator = theme.CircleBlue
	}

	checkbox := theme.CircleBlack
	if m.dictConfig.EnableRelationships {
		checkbox = theme.CircleGreen
	}

	line := fmt.Sprintf("  %s  Include Relationships: %s", indicator, checkbox)

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
	lines = append(lines, "")
	if m.dictConfig.EnableRelationships {
		// Relationship depth
		isFocused = m.configField == 1
		indicator = theme.CircleBlack
		if isFocused {
			indicator = theme.CircleBlue
		}

		line = fmt.Sprintf("  %s  Relationship Depth: %d levels", indicator, m.dictConfig.RelationshipDepth)

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
		lines = append(lines, "")

		// Relationship types section
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(theme.ColorSecondary).Render("Relationship Types"))
		relTypes := []struct {
			code     string
			name     string
			selected bool
		}{
			{"PAR", "Parent", contains(m.dictConfig.RelationshipTypes, "PAR")},
			{"CHD", "Child", contains(m.dictConfig.RelationshipTypes, "CHD")},
			{"RB", "Broader", contains(m.dictConfig.RelationshipTypes, "RB")},
			{"RN", "Narrower", contains(m.dictConfig.RelationshipTypes, "RN")},
			{"SY", "Synonym", contains(m.dictConfig.RelationshipTypes, "SY")},
			{"isa", "Is-A", contains(m.dictConfig.RelationshipTypes, "isa")},
			{"part_of", "Part-Of", contains(m.dictConfig.RelationshipTypes, "part_of")},
			{"may_treat", "May Treat", contains(m.dictConfig.RelationshipTypes, "may_treat")},
			{"associated_with", "Associated With", contains(m.dictConfig.RelationshipTypes, "associated_with")},
		}

		for i, rel := range relTypes {
			isFocused = i+2 == m.configField // +2 for enable and depth fields

			indicator = theme.CircleBlack
			if isFocused {
				indicator = theme.CircleBlue
			}

			checkbox = theme.CircleBlack
			if rel.selected {
				checkbox = theme.CircleGreen
			}

			line = fmt.Sprintf("  %s  %s  %-15s (%s)", indicator, checkbox, rel.name, rel.code)

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
		}
	}

	// Simple footer
	lines = append(lines, "")
	lines = append(lines, strings.Repeat("─", width-4))
	lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).
		Render("↑/↓: Navigate  Space: Toggle  ←/→: Adjust depth  Enter: Save  ESC: Back"))

	return strings.Join(lines, "\n")
}

// clipToHeight truncates a set of lines to at most max lines and
// appends an ellipsis if anything was omitted.
func clipToHeight(lines []string, max int) []string {
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

// Handle key events for configuration screens
func (m *Model) handleConfigKeys(msg tea.KeyMsg, configType DictBuilderState) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		maxFields := m.getMaxConfigFields(configType)
		if m.configField < maxFields-1 {
			m.configField++
		}
	case "left", "h":
		m.adjustConfigValue(configType, -1)
	case "right", "l":
		m.adjustConfigValue(configType, 1)
	case " ", "space":
		m.toggleConfigBool(configType)
	case "enter":
		// Save and return to main menu
		m.dictBuilderState = DictStateMainMenu
		m.configField = 0
	case "esc":
		// Cancel and return to main menu
		m.dictBuilderState = DictStateMainMenu
		m.configField = 0
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

// Helper to get max fields for each config type
func (m *Model) getMaxConfigFields(configType DictBuilderState) int {
	switch configType {
	case DictStateMemoryConfig:
		return 3 // Initial heap, max heap, stack size
	case DictStateProcessingConfig:
		return 8 // Thread count, batch size, cache, temp dir, preserve case, handle punct, min/max word
	case DictStateFilterConfig:
		return 16 // All filter options
	case DictStateOutputConfig:
		return 8 // Various output formats
	case DictStateRelationshipConfig:
		if m.dictConfig.EnableRelationships {
			return 11 // Enable + depth + 9 relationship types
		}
		return 1 // Just enable/disable
	default:
		return 1
	}
}

// Adjust numeric config values
func (m *Model) adjustConfigValue(configType DictBuilderState, delta int) {
	switch configType {
	case DictStateMemoryConfig:
		switch m.configField {
		case 0: // Initial heap
			m.dictConfig.InitialHeapMB = clamp(m.dictConfig.InitialHeapMB+delta*256, 512, 3072)
		case 1: // Max heap
			m.dictConfig.MaxHeapMB = clamp(m.dictConfig.MaxHeapMB+delta*256, 512, 8192)
		case 2: // Stack size
			m.dictConfig.StackSizeMB = clamp(m.dictConfig.StackSizeMB+delta, 1, 64)
		}
	case DictStateProcessingConfig:
		switch m.configField {
		case 0: // Thread count
			m.dictConfig.ThreadCount = clamp(m.dictConfig.ThreadCount+delta, 1, 16)
		case 1: // Batch size
			m.dictConfig.BatchSize = clamp(m.dictConfig.BatchSize+delta*100, 100, 10000)
		case 2: // Cache size
			m.dictConfig.CacheSize = clamp(m.dictConfig.CacheSize+delta*32, 64, 512)
		case 6: // Min word length
			m.dictConfig.MinWordLength = clamp(m.dictConfig.MinWordLength+delta, 1, 10)
		case 7: // Max word length
			m.dictConfig.MaxWordLength = clamp(m.dictConfig.MaxWordLength+delta*10, 10, 256)
		}
	case DictStateFilterConfig:
		// Handle numeric filter adjustments
		if m.configField < 4 { // Length filters section
			switch m.configField {
			case 0: // Min term length
				m.dictConfig.MinTermLength = clamp(m.dictConfig.MinTermLength+delta, 1, 100)
			case 1: // Max term length
				m.dictConfig.MaxTermLength = clamp(m.dictConfig.MaxTermLength+delta*10, 10, 1000)
			case 2: // Min tokens
				m.dictConfig.MinTokens = clamp(m.dictConfig.MinTokens+delta, 0, 20)
			case 3: // Max tokens
				m.dictConfig.MaxTokens = clamp(m.dictConfig.MaxTokens+delta, 1, 50)
			}
		}
	case DictStateRelationshipConfig:
		if m.configField == 1 && m.dictConfig.EnableRelationships {
			// Adjust relationship depth
			m.dictConfig.RelationshipDepth = clamp(m.dictConfig.RelationshipDepth+delta, 0, 5)
		}
	}
}

// Toggle boolean config values
func (m *Model) toggleConfigBool(configType DictBuilderState) {
	switch configType {
	case DictStateProcessingConfig:
		switch m.configField {
		case 4:
			m.dictConfig.PreserveCase = !m.dictConfig.PreserveCase
		case 5:
			m.dictConfig.HandlePunctuation = !m.dictConfig.HandlePunctuation
		}
	case DictStateFilterConfig:
		// Map field index to boolean toggles
		boolFields := map[int]*bool{
			4:  &m.dictConfig.ExcludeSuppressible,
			5:  &m.dictConfig.ExcludeObsolete,
			6:  &m.dictConfig.ExcludeNumericOnly,
			7:  &m.dictConfig.ExcludePunctOnly,
			8:  &m.dictConfig.PreferredOnly,
			9:  &m.dictConfig.CaseSensitive,
			10: &m.dictConfig.UseNormalization,
			11: &m.dictConfig.UseMRRANK,
			12: &m.dictConfig.Deduplicate,
			13: &m.dictConfig.StripPunctuation,
			14: &m.dictConfig.CollapseWhitespace,
		}
		if field, ok := boolFields[m.configField]; ok {
			*field = !*field
		}
	case DictStateOutputConfig:
		switch m.configField {
		case 0:
			m.dictConfig.BuildHSQLDB = !m.dictConfig.BuildHSQLDB
		case 1:
			m.dictConfig.BuildLucene = !m.dictConfig.BuildLucene
		case 2:
			m.dictConfig.EmitTSV = !m.dictConfig.EmitTSV
		case 3:
			m.dictConfig.EmitJSONL = !m.dictConfig.EmitJSONL
		case 4:
			m.dictConfig.UseRareWords = !m.dictConfig.UseRareWords
		case 5:
			m.dictConfig.EmitDescriptor = !m.dictConfig.EmitDescriptor
		case 6:
			m.dictConfig.EmitPipeline = !m.dictConfig.EmitPipeline
		case 7:
			m.dictConfig.EmitManifest = !m.dictConfig.EmitManifest
		}
	case DictStateRelationshipConfig:
		if m.configField == 0 {
			m.dictConfig.EnableRelationships = !m.dictConfig.EnableRelationships
		} else if m.configField >= 2 && m.dictConfig.EnableRelationships {
			// Toggle relationship types
			relTypes := []string{"PAR", "CHD", "RB", "RN", "SY", "isa", "part_of", "may_treat", "associated_with"}
			if m.configField-2 < len(relTypes) {
				relType := relTypes[m.configField-2]
				if contains(m.dictConfig.RelationshipTypes, relType) {
					// Remove
					newTypes := []string{}
					for _, t := range m.dictConfig.RelationshipTypes {
						if t != relType {
							newTypes = append(newTypes, t)
						}
					}
					m.dictConfig.RelationshipTypes = newTypes
				} else {
					// Add
					m.dictConfig.RelationshipTypes = append(m.dictConfig.RelationshipTypes, relType)
				}
			}
		}
	}
}

// Helper function to clamp values
func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// Helper to parse integer from string
func parseInt(s string, defaultVal int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return defaultVal
}

// Cased dictionary configuration screen
func (m *Model) renderCasedConfig(width, height int) string {
	// Simple clean header
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).Render("Cased Dictionary Configuration"),
		strings.Repeat("─", width-4),
		"",
	}
	// Enable/disable cased dictionary with circle indicators
	isFocused := m.configField == 0
	indicator := theme.CircleBlack
	if isFocused {
		indicator = theme.CircleBlue
	}

	checkbox := theme.CircleBlack
	if m.dictConfig.BuildCasedDictionary {
		checkbox = theme.CircleGreen
	}

	line := fmt.Sprintf("  %s  Build Cased Dictionary: %s", indicator, checkbox)

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
	lines = append(lines, "")
	if m.dictConfig.BuildCasedDictionary {
		// Options section - simple header
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(theme.ColorSecondary).Render("Configuration Options"))
		lines = append(lines, "")

		// Ranking method
		isFocused = m.configField == 1
		indicator = theme.CircleBlack
		if isFocused {
			indicator = theme.CircleBlue
		}

		rankMethod := m.dictConfig.CasedTermRanking
		if rankMethod == "" {
			rankMethod = "mrrank"
		}

		line = fmt.Sprintf("  %s  Term Ranking: <%s>", indicator, rankMethod)

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

		// Include acronyms
		isFocused = m.configField == 2
		indicator = theme.CircleBlack
		if isFocused {
			indicator = theme.CircleBlue
		}

		checkbox = theme.CircleBlack
		if m.dictConfig.IncludeAcronyms {
			checkbox = theme.CircleGreen
		}

		line = fmt.Sprintf("  %s  %s  Include Acronyms", indicator, checkbox)

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

		// Include abbreviations
		isFocused = m.configField == 3
		indicator = theme.CircleBlack
		if isFocused {
			indicator = theme.CircleBlue
		}

		checkbox = theme.CircleBlack
		if m.dictConfig.IncludeAbbreviations {
			checkbox = theme.CircleGreen
		}

		line = fmt.Sprintf("  %s  %s  Include Abbreviations", indicator, checkbox)

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

		// Output cased BSV
		isFocused = m.configField == 4
		indicator = theme.CircleBlack
		if isFocused {
			indicator = theme.CircleBlue
		}

		checkbox = theme.CircleBlack
		if m.dictConfig.EmitCasedBSV {
			checkbox = theme.CircleGreen
		}

		line = fmt.Sprintf("  %s  %s  Emit Cased BSV", indicator, checkbox)

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
		lines = append(lines, "")

		// Info section - simple note
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorInfo).Render("Note: Cased dictionaries improve recognition of:"))
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render("  • Drug names (Aspirin vs aspirin)"))
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render("  • Proper nouns (Mayo Clinic)"))
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render("  • Acronyms (AIDS, COPD, CHF)"))
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).Render("  • Anatomical terms (Heart vs heart)"))
	}

	// Simple footer
	lines = append(lines, "")
	lines = append(lines, strings.Repeat("─", width-4))
	lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorForegroundDim).
		Render("↑/↓: Navigate  Space: Toggle  ←/→: Switch ranking  Enter: Save  ESC: Back"))

	return strings.Join(lines, "\n")
}

// Handle key events for cased configuration screen
func (m *Model) handleCasedKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	maxFields := 1 // Just enable/disable when disabled
	if m.dictConfig.BuildCasedDictionary {
		maxFields = 5 // All options when enabled
	}

	switch msg.String() {
	case "up", "k":
		if m.configField > 0 {
			m.configField--
		}
	case "down", "j":
		if m.configField < maxFields-1 {
			m.configField++
		}
	case " ", "space":
		switch m.configField {
		case 0: // Enable/disable
			m.dictConfig.BuildCasedDictionary = !m.dictConfig.BuildCasedDictionary
			if m.dictConfig.BuildCasedDictionary && m.dictConfig.CasedTermRanking == "" {
				m.dictConfig.CasedTermRanking = "mrrank"
			}
		case 2: // Acronyms
			m.dictConfig.IncludeAcronyms = !m.dictConfig.IncludeAcronyms
		case 3: // Abbreviations
			m.dictConfig.IncludeAbbreviations = !m.dictConfig.IncludeAbbreviations
		case 4: // Emit cased BSV
			m.dictConfig.EmitCasedBSV = !m.dictConfig.EmitCasedBSV
		}
	case "left", "h", "right", "l":
		if m.configField == 1 && m.dictConfig.BuildCasedDictionary {
			// Toggle ranking method
			if m.dictConfig.CasedTermRanking == "mrrank" {
				m.dictConfig.CasedTermRanking = "frequency"
			} else {
				m.dictConfig.CasedTermRanking = "mrrank"
			}
		}
	case "enter":
		// Save and return to main menu
		m.dictBuilderState = DictStateMainMenu
		m.configField = 0
	case "esc":
		// Cancel and return to main menu
		m.dictBuilderState = DictStateMainMenu
		m.configField = 0
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
