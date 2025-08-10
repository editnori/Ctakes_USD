package utils

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// Removed FormatFileSize - now in common.go
// Removed TruncateString - now in common.go

// truncateToWidth truncates string to exact display width, handling unicode properly
func truncateToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}

	var result strings.Builder
	currentWidth := 0

	for _, r := range s {
		runeWidth := runeDisplayWidth(r)
		if currentWidth+runeWidth > width {
			break
		}
		result.WriteRune(r)
		currentWidth += runeWidth
	}

	return result.String()
}

// runeDisplayWidth returns the display width of a rune
func runeDisplayWidth(r rune) int {
	if r == '\t' {
		return 4 // Tab width
	}
	if !utf8.ValidRune(r) || r < 32 {
		return 0 // Control characters
	}
	// Simple approximation - could use runewidth package for accuracy
	if r < 127 {
		return 1 // ASCII
	}
	// Wide characters (CJK, etc)
	if r >= 0x1100 && r <= 0x115F || // Hangul
		r >= 0x2E80 && r <= 0x9FFF || // CJK
		r >= 0xAC00 && r <= 0xD7AF || // Hangul
		r >= 0xF900 && r <= 0xFAFF || // CJK Compatibility
		r >= 0xFE30 && r <= 0xFE4F || // CJK Compatibility
		r >= 0xFF00 && r <= 0xFF60 || // Fullwidth Forms
		r >= 0xFFE0 && r <= 0xFFE6 { // Fullwidth Forms
		return 2
	}
	return 1
}

// SafeRender ensures content fits within width, truncating if necessary
func SafeRender(content string, width int) string {
	if width <= 0 {
		return ""
	}

	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		if lipgloss.Width(line) > width {
			result = append(result, TruncateString(line, width))
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// SafeRenderWithBounds ensures content fits within both width and height bounds
// and adds necessary padding to prevent terminal edge cutoff
func SafeRenderWithBounds(content string, maxWidth, maxHeight int) string {
	if maxWidth <= 0 || maxHeight <= 0 {
		return ""
	}

	// First ensure width compliance
	content = SafeRender(content, maxWidth)

	// Then ensure height compliance
	lines := strings.Split(content, "\n")
	if len(lines) > maxHeight {
		lines = lines[:maxHeight]
	}

	// Ensure each line is exactly maxWidth (pad with spaces if needed)
	for i, line := range lines {
		width := lipgloss.Width(line)
		if width < maxWidth {
			lines[i] = line + strings.Repeat(" ", maxWidth-width)
		}
	}

	// Pad to exact height if needed
	for len(lines) < maxHeight {
		lines = append(lines, strings.Repeat(" ", maxWidth))
	}

	return strings.Join(lines, "\n")
}

// ConstrainBox ensures a box-styled content fits within dimensions
func ConstrainBox(content string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	// First constrain width
	content = SafeRender(content, width)

	// Then constrain height
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

// Removed WrapText - now in common.go

// ClampWidth ensures a value is within min and max bounds
func ClampWidth(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// SafeBox creates a bordered box that guarantees all 4 borders are visible
// It reserves space for borders and ensures content doesn't overflow
func SafeBox(content string, width, height int, borderStyle lipgloss.Border, borderColor lipgloss.Color) string {
	// Minimum dimensions to show borders
	if width < 4 {
		width = 4
	}
	if height < 3 {
		height = 3
	}

	// Calculate inner dimensions (accounting for borders)
	innerWidth := Max(1, width-2)   // -2 for left and right borders
	innerHeight := Max(1, height-2) // -2 for top and bottom borders

	// Constrain content to inner dimensions
	constrainedContent := ConstrainBox(content, innerWidth, innerHeight)

	// Create the box with exact dimensions
	box := lipgloss.NewStyle().
		Border(borderStyle).
		BorderForeground(borderColor).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		Width(innerWidth).      // Set inner width
		Height(innerHeight).    // Set inner height
		MaxWidth(innerWidth).   // Enforce max width
		MaxHeight(innerHeight). // Enforce max height
		Render(constrainedContent)

	return box
}

// CalculateSafeContentArea returns the safe content area dimensions
// accounting for terminal borders and padding
func CalculateSafeContentArea(termWidth, termHeight int, padding int) (width, height int) {
	// CRITICAL: Be very conservative with terminal dimensions
	// Many terminals don't render content at exact boundaries
	// Reserve space for:
	// - Terminal edges (1 char on each side minimum)
	// - Padding (configurable, default 2 for safety)
	// - Additional buffer for border rendering
	// - Scroll buffer space
	minPadding := Max(3, padding) // Increased to 3 for extra safety

	// Calculate safe width and height with more conservative margins
	safeWidth := Max(10, termWidth-(minPadding*2))
	safeHeight := Max(5, termHeight-(minPadding*2)-1) // Extra -1 for bottom scroll buffer

	return safeWidth, safeHeight
}

// BorderCharacters defines the characters used for drawing borders
type BorderCharacters struct {
	TopLeft     string
	TopRight    string
	BottomLeft  string
	BottomRight string
	Horizontal  string
	Vertical    string
	TJoinTop    string // ┬
	TJoinBottom string // ┴
	TJoinLeft   string // ├
	TJoinRight  string // ┤
	Cross       string // ┼
}

// GetSharedBorderChars returns the characters for shared borders
func GetSharedBorderChars() BorderCharacters {
	return BorderCharacters{
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
		Horizontal:  "─",
		Vertical:    "│",
		TJoinTop:    "┬",
		TJoinBottom: "┴",
		TJoinLeft:   "├",
		TJoinRight:  "┤",
		Cross:       "┼",
	}
}

// JoinHorizontalWithSharedBorder joins panels horizontally with a shared vertical border
func JoinHorizontalWithSharedBorder(panels []string, widths []int, height int, activePanel int, activeBorderColor, inactiveBorderColor lipgloss.Color) string {
	if len(panels) == 0 {
		return ""
	}

	chars := GetSharedBorderChars()

	// Build the complete layout line by line
	lines := make([]string, height)

	for row := 0; row < height; row++ {
		var lineBuilder strings.Builder

		for panelIdx, panel := range panels {
			panelLines := strings.Split(panel, "\n")

			// Get the content for this row (or empty if beyond panel height)
			content := ""
			if row < len(panelLines) {
				content = panelLines[row]
			}

			// Ensure content fits the panel width
			panelWidth := widths[panelIdx]
			if lipgloss.Width(content) < panelWidth {
				content = content + strings.Repeat(" ", panelWidth-lipgloss.Width(content))
			} else if lipgloss.Width(content) > panelWidth {
				content = TruncateString(content, panelWidth)
			}

			// Determine border color for this panel
			borderCol := inactiveBorderColor
			if panelIdx == activePanel {
				borderCol = activeBorderColor
			}

			// Add left border for first panel or shared border for others
			if panelIdx == 0 {
				// First panel - add left border
				if row == 0 {
					lineBuilder.WriteString(lipgloss.NewStyle().Foreground(borderCol).Render(chars.TopLeft))
				} else if row == height-1 {
					lineBuilder.WriteString(lipgloss.NewStyle().Foreground(borderCol).Render(chars.BottomLeft))
				} else {
					lineBuilder.WriteString(lipgloss.NewStyle().Foreground(borderCol).Render(chars.Vertical))
				}
			}

			// Add top/bottom border or content
			if row == 0 {
				// Top border
				lineBuilder.WriteString(lipgloss.NewStyle().Foreground(borderCol).Render(strings.Repeat(chars.Horizontal, panelWidth)))
			} else if row == height-1 {
				// Bottom border
				lineBuilder.WriteString(lipgloss.NewStyle().Foreground(borderCol).Render(strings.Repeat(chars.Horizontal, panelWidth)))
			} else {
				// Content
				lineBuilder.WriteString(content)
			}

			// Add right border or shared border
			if panelIdx == len(panels)-1 {
				// Last panel - add right border
				if row == 0 {
					lineBuilder.WriteString(lipgloss.NewStyle().Foreground(borderCol).Render(chars.TopRight))
				} else if row == height-1 {
					lineBuilder.WriteString(lipgloss.NewStyle().Foreground(borderCol).Render(chars.BottomRight))
				} else {
					lineBuilder.WriteString(lipgloss.NewStyle().Foreground(borderCol).Render(chars.Vertical))
				}
			} else {
				// Shared border between panels
				// Use the active color if either panel is active
				sharedBorderCol := inactiveBorderColor
				if panelIdx == activePanel || panelIdx+1 == activePanel {
					sharedBorderCol = activeBorderColor
				}

				if row == 0 {
					lineBuilder.WriteString(lipgloss.NewStyle().Foreground(sharedBorderCol).Render(chars.TJoinTop))
				} else if row == height-1 {
					lineBuilder.WriteString(lipgloss.NewStyle().Foreground(sharedBorderCol).Render(chars.TJoinBottom))
				} else {
					lineBuilder.WriteString(lipgloss.NewStyle().Foreground(sharedBorderCol).Render(chars.Vertical))
				}
			}
		}

		lines[row] = lineBuilder.String()
	}

	return strings.Join(lines, "\n")
}

// CreateSharedBorderLayout creates a layout with shared borders between panels
func CreateSharedBorderLayout(sidebar, main, preview string, sidebarWidth, mainWidth, previewWidth, height int, activePanel int, showPreview bool, activeBorderColor, inactiveBorderColor lipgloss.Color) string {
	chars := GetSharedBorderChars()

	// Split content into lines
	sidebarLines := strings.Split(sidebar, "\n")
	mainLines := strings.Split(main, "\n")
	previewLines := strings.Split(preview, "\n")

	// Ensure all panels have enough lines
	for len(sidebarLines) < height-2 {
		sidebarLines = append(sidebarLines, "")
	}
	for len(mainLines) < height-2 {
		mainLines = append(mainLines, "")
	}
	for len(previewLines) < height-2 {
		previewLines = append(previewLines, "")
	}

	lines := make([]string, height)

	for row := 0; row < height; row++ {
		var lineBuilder strings.Builder

		// Determine colors for each section
		sidebarColor := inactiveBorderColor
		if activePanel == 0 { // SidebarPanel
			sidebarColor = activeBorderColor
		}

		mainColor := inactiveBorderColor
		if activePanel == 1 { // MainPanel
			mainColor = activeBorderColor
		}

		previewColor := inactiveBorderColor
		if activePanel == 3 { // PreviewPanel
			previewColor = activeBorderColor
		}

		if row == 0 {
			// Top border
			lineBuilder.WriteString(lipgloss.NewStyle().Foreground(sidebarColor).Render(chars.TopLeft))
			lineBuilder.WriteString(lipgloss.NewStyle().Foreground(sidebarColor).Render(strings.Repeat(chars.Horizontal, sidebarWidth)))

			// Junction between sidebar and main
			junctionColor := sidebarColor
			if activePanel == 1 {
				junctionColor = mainColor
			}
			lineBuilder.WriteString(lipgloss.NewStyle().Foreground(junctionColor).Render(chars.TJoinTop))

			lineBuilder.WriteString(lipgloss.NewStyle().Foreground(mainColor).Render(strings.Repeat(chars.Horizontal, mainWidth)))

			if showPreview {
				// Junction between main and preview
				junctionColor = mainColor
				if activePanel == 3 {
					junctionColor = previewColor
				}
				lineBuilder.WriteString(lipgloss.NewStyle().Foreground(junctionColor).Render(chars.TJoinTop))
				lineBuilder.WriteString(lipgloss.NewStyle().Foreground(previewColor).Render(strings.Repeat(chars.Horizontal, previewWidth)))
				lineBuilder.WriteString(lipgloss.NewStyle().Foreground(previewColor).Render(chars.TopRight))
			} else {
				lineBuilder.WriteString(lipgloss.NewStyle().Foreground(mainColor).Render(chars.TopRight))
			}
		} else if row == height-1 {
			// Bottom border
			lineBuilder.WriteString(lipgloss.NewStyle().Foreground(sidebarColor).Render(chars.BottomLeft))
			lineBuilder.WriteString(lipgloss.NewStyle().Foreground(sidebarColor).Render(strings.Repeat(chars.Horizontal, sidebarWidth)))

			// Junction between sidebar and main
			junctionColor := sidebarColor
			if activePanel == 1 {
				junctionColor = mainColor
			}
			lineBuilder.WriteString(lipgloss.NewStyle().Foreground(junctionColor).Render(chars.TJoinBottom))

			lineBuilder.WriteString(lipgloss.NewStyle().Foreground(mainColor).Render(strings.Repeat(chars.Horizontal, mainWidth)))

			if showPreview {
				// Junction between main and preview
				junctionColor = mainColor
				if activePanel == 3 {
					junctionColor = previewColor
				}
				lineBuilder.WriteString(lipgloss.NewStyle().Foreground(junctionColor).Render(chars.TJoinBottom))
				lineBuilder.WriteString(lipgloss.NewStyle().Foreground(previewColor).Render(strings.Repeat(chars.Horizontal, previewWidth)))
				lineBuilder.WriteString(lipgloss.NewStyle().Foreground(previewColor).Render(chars.BottomRight))
			} else {
				lineBuilder.WriteString(lipgloss.NewStyle().Foreground(mainColor).Render(chars.BottomRight))
			}
		} else {
			// Content rows
			contentRow := row - 1

			// Left border
			lineBuilder.WriteString(lipgloss.NewStyle().Foreground(sidebarColor).Render(chars.Vertical))

			// Sidebar content
			sidebarContent := ""
			if contentRow < len(sidebarLines) {
				sidebarContent = sidebarLines[contentRow]
			}
			sidebarContent = ensureWidth(sidebarContent, sidebarWidth)
			lineBuilder.WriteString(sidebarContent)

			// Shared border between sidebar and main
			junctionColor := inactiveBorderColor
			if activePanel == 0 || activePanel == 1 {
				junctionColor = activeBorderColor
			}
			lineBuilder.WriteString(lipgloss.NewStyle().Foreground(junctionColor).Render(chars.Vertical))

			// Main content
			mainContent := ""
			if contentRow < len(mainLines) {
				mainContent = mainLines[contentRow]
			}
			mainContent = ensureWidth(mainContent, mainWidth)
			lineBuilder.WriteString(mainContent)

			if showPreview {
				// Shared border between main and preview
				junctionColor = inactiveBorderColor
				if activePanel == 1 || activePanel == 3 {
					junctionColor = activeBorderColor
				}
				lineBuilder.WriteString(lipgloss.NewStyle().Foreground(junctionColor).Render(chars.Vertical))

				// Preview content
				previewContent := ""
				if contentRow < len(previewLines) {
					previewContent = previewLines[contentRow]
				}
				previewContent = ensureWidth(previewContent, previewWidth)
				lineBuilder.WriteString(previewContent)

				// Right border
				lineBuilder.WriteString(lipgloss.NewStyle().Foreground(previewColor).Render(chars.Vertical))
			} else {
				// Right border
				lineBuilder.WriteString(lipgloss.NewStyle().Foreground(mainColor).Render(chars.Vertical))
			}
		}

		lines[row] = lineBuilder.String()
	}

	return strings.Join(lines, "\n")
}

// ensureWidth ensures a string is exactly the specified width
func ensureWidth(s string, width int) string {
	currentWidth := lipgloss.Width(s)
	if currentWidth < width {
		return s + strings.Repeat(" ", width-currentWidth)
	} else if currentWidth > width {
		return TruncateString(s, width)
	}
	return s
}

// CreateSimpleSharedBorderBox creates a simple box with shared border characters
func CreateSimpleSharedBorderBox(content string, width, height int, borderColor lipgloss.Color) string {
	chars := GetSharedBorderChars()

	// Ensure minimum dimensions
	if width < 4 {
		width = 4
	}
	if height < 3 {
		height = 3
	}

	// Calculate inner dimensions
	innerWidth := Max(1, width-2)
	innerHeight := Max(1, height-2)

	// Split content into lines
	contentLines := strings.Split(content, "\n")

	// Ensure we have enough lines
	for len(contentLines) < innerHeight {
		contentLines = append(contentLines, "")
	}
	if len(contentLines) > innerHeight {
		contentLines = contentLines[:innerHeight]
	}

	// Build the box
	lines := make([]string, height)

	// Top border
	lines[0] = lipgloss.NewStyle().Foreground(borderColor).Render(
		chars.TopLeft + strings.Repeat(chars.Horizontal, innerWidth) + chars.TopRight,
	)

	// Content lines with side borders
	for i := 0; i < innerHeight; i++ {
		line := contentLines[i]
		line = ensureWidth(line, innerWidth)
		lines[i+1] = lipgloss.NewStyle().Foreground(borderColor).Render(chars.Vertical) +
			line +
			lipgloss.NewStyle().Foreground(borderColor).Render(chars.Vertical)
	}

	// Bottom border
	lines[height-1] = lipgloss.NewStyle().Foreground(borderColor).Render(
		chars.BottomLeft + strings.Repeat(chars.Horizontal, innerWidth) + chars.BottomRight,
	)

	return strings.Join(lines, "\n")
}

// ClipToHeight clips a slice of strings to a maximum height
func ClipToHeight(lines []string, maxHeight int) []string {
	if len(lines) <= maxHeight {
		return lines
	}
	return lines[:maxHeight]
}

// Contains checks if a slice contains a string
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// CBox returns a checkbox character for boolean values
func CBox(checked bool) string {
	if checked {
		return "☑"
	}
	return "☐"
}
