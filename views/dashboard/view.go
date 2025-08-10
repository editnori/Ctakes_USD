package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
	"github.com/ctakes-tui/ctakes-tui/internal/utils"
)

func (m Model) View() string {
	if m.width < 80 {
		return m.renderCompactLayout()
	}
	return m.renderFullLayout()
}

func (m *Model) renderCompactLayout() string {
	banner := m.renderCompactBanner(m.width)

	mainContent := ""
	switch m.activePanel {
	case SidebarPanel:
		mainContent = m.renderSidebar(m.width-4, m.height-6)
	case MainPanel:
		mainContent = m.renderFileBrowserCompact(m.width-4, m.height-6)
	case SystemPanel:
		mainContent = m.renderSystemPanel(m.width-4, m.height-6)
	}

	contentBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorBorder).
		Width(m.width - 2).
		Height(m.height - 5).
		Padding(1).
		Render(mainContent)

	footer := m.renderCompactFooter(m.width)

	return lipgloss.JoinVertical(lipgloss.Left, banner, contentBox, footer)
}

func (m *Model) renderFullLayout() string {
	banner := m.renderFullBanner(m.width)

	// Calculate panel widths - account for borders taking space
	totalWidth := m.width
	sidebarWidth := totalWidth / 4
	// The key: subtract space for the borders between panels
	remainingWidth := totalWidth - sidebarWidth - 1 // -1 for gap between panels

	mainWidth := remainingWidth
	previewWidth := 0

	if m.showPreview {
		mainWidth = (remainingWidth * 3) / 5
		previewWidth = remainingWidth - mainWidth - 1 // -1 for gap
	}

	// Calculate content height - account for ASCII art (6 lines) + subtitle + padding (3) + footer
	contentHeight := m.height - 11

	// Render sidebar content
	sidebar := m.renderSidebar(sidebarWidth-2, contentHeight-2)

	// Render main content based on selected sidebar item
	mainContent := ""
	if m.cursor < len(m.sidebarItems) {
		switch m.sidebarItems[m.cursor].Action {
		case "system":
			mainContent = m.renderSystemPanel(mainWidth-2, contentHeight-2)
		case "files":
			mainContent = m.renderFileBrowserCompact(mainWidth-2, contentHeight-2)
		case "dictionary_builder_view":
			mainContent = m.renderDictionaryBuilderPanel(mainWidth-2, contentHeight-2)
		case "pipeline":
			mainContent = m.renderPipelineConfigPanel(mainWidth-2, contentHeight-2)
		default:
			mainContent = m.renderFileBrowserCompact(mainWidth-2, contentHeight-2)
		}
	} else {
		mainContent = m.renderFileBrowserCompact(mainWidth-2, contentHeight-2)
	}

	// Create boxes with active panel highlighting
	sidebarBorderColor := theme.ColorBorder
	if m.activePanel == SidebarPanel {
		sidebarBorderColor = theme.ColorAccent // Highlight active panel
	}

	mainBorderColor := theme.ColorBorder
	if m.activePanel == MainPanel {
		mainBorderColor = theme.ColorAccent // Highlight active panel
	}

	sidebarBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(sidebarBorderColor).
		Width(sidebarWidth - 2). // -2 for left and right border chars
		Height(contentHeight).
		Render(sidebar)

	mainBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(mainBorderColor).
		Width(mainWidth - 2). // -2 for left and right border chars
		Height(contentHeight).
		Render(mainContent)

	var content string
	if m.showPreview && previewWidth > 0 {
		previewBorderColor := theme.ColorBorder
		if m.activePanel == PreviewPanel {
			previewBorderColor = theme.ColorAccent // Highlight active panel
		}

		previewContent := m.renderPreviewPanel(previewWidth-4, contentHeight-2)
		previewBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(previewBorderColor).
			Width(previewWidth - 2). // -2 for border chars
			Height(contentHeight).
			Render(previewContent)

		content = lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, mainBox, previewBox)
	} else {
		content = lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, mainBox)
	}

	footer := m.renderFooter(m.width)

	// Join everything vertically
	return lipgloss.JoinVertical(lipgloss.Left, banner, content, footer)
}

func (m *Model) renderCompactBanner(width int) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorAccent).
		Render("cTAKES CLI")

	author := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Italic(true).
		Render("by Dr. Layth M Qassem")

	status := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Render(fmt.Sprintf("CPU: %.1f%% | MEM: %.1f%%", m.cpuPercent, m.memPercent))

	leftPart := title + " " + author
	spacerWidth := width - lipgloss.Width(leftPart) - lipgloss.Width(status) - 2
	spacer := strings.Repeat(" ", utils.Max(0, spacerWidth))

	return lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Render(leftPart + spacer + status)
}

func (m *Model) renderFullBanner(width int) string {
	// Clean, properly aligned ASCII art - fixed first row alignment
	asciiArt := []string{
		"██████╗ ████████╗ █████╗ ██╗  ██╗███████╗███████╗      ██████╗██╗     ██╗",
		"██╔════╝╚══██╔══╝██╔══██╗██║ ██╔╝██╔════╝██╔════╝     ██╔════╝██║     ██║",
		"██║        ██║   ███████║█████╔╝ █████╗  ███████╗████╗██║     ██║     ██║",
		"██║        ██║   ██╔══██║██╔═██╗ ██╔══╝  ╚════██║╚═══╝██║     ██║     ██║",
		"╚██████╗   ██║   ██║  ██║██║  ██╗███████╗███████║     ╚██████╗███████╗██║",
		" ╚═════╝   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝╚══════╝      ╚═════╝╚══════╝╚═╝",
	}

	subtitle := "Dr. Layth M Qassem PharmD, MS"

	// Status info to be right-aligned
	status := fmt.Sprintf("CPU: %.1f%% | MEM: %.1f%% | DISK: %.1f%%",
		m.cpuPercent, m.memPercent, m.diskPercent)

	// Build the banner
	lines := make([]string, 0)

	// Add more top padding for visibility
	lines = append(lines, "")
	lines = append(lines, "")
	// Render ASCII art centered with status on the right
	if len(asciiArt) > 0 {
		artStyle := lipgloss.NewStyle().
			Foreground(theme.ColorAccent).
			Bold(true)

		statusStyle := lipgloss.NewStyle().
			Foreground(theme.ColorForegroundDim)

		// Calculate centering for ASCII art only (not including status)
		artWidth := lipgloss.Width(asciiArt[0])
		centerPadding := (width - artWidth) / 2
		if centerPadding < 0 {
			centerPadding = 0
		}
		leftPad := strings.Repeat(" ", centerPadding)

		// First line with status positioned at the right edge
		statusWidth := lipgloss.Width(status)
		// Calculate space between centered art and right-aligned status
		middleSpace := width - centerPadding - artWidth - statusWidth - 2
		if middleSpace < 2 {
			middleSpace = 2
		}

		line1 := leftPad + artStyle.Render(asciiArt[0]) + strings.Repeat(" ", middleSpace) + statusStyle.Render(status)
		lines = append(lines, line1)

		// Remaining ASCII art lines (centered the same way)
		for i := 1; i < len(asciiArt); i++ {
			lines = append(lines, leftPad+artStyle.Render(asciiArt[i]))
		}
	}

	// Add subtitle centered below ASCII art
	lines = append(lines, lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Italic(true).
		Width(width).
		Align(lipgloss.Center).
		Render(subtitle))

	lines = append(lines, "") // Bottom padding

	return strings.Join(lines, "\n")
}

func (m *Model) renderSidebar(width, height int) string {
	// Ensure minimum width to prevent negative values
	if width <= 0 {
		width = 20 // Minimum sidebar width
	}
	if height <= 0 {
		height = 10 // Minimum height
	}

	// Add a header for the navigation menu that follows the highlighting scheme
	headerForeground := theme.ColorForegroundDim
	if m.activePanel == SidebarPanel {
		headerForeground = theme.ColorAccent
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(headerForeground).
		Bold(false).   // Match file explorer styling - no bold
		Padding(0, 1). // Reduced padding to match file explorer
		Width(width)

	header := headerStyle.Render("Navigation")

	// Add a subtle divider that also follows the color scheme
	dividerColor := theme.ColorBorder
	if m.activePanel == SidebarPanel {
		dividerColor = theme.ColorAccent
	}

	divider := lipgloss.NewStyle().
		Foreground(dividerColor).
		Width(width).
		Render(strings.Repeat("─", utils.Max(0, width)))

	items := make([]string, 0)
	items = append(items, header)
	items = append(items, divider)

	// Add minimal vertical spacing
	items = append(items, "")

	for i, item := range m.sidebarItems {
		// Check if this item is selected based on cursor position and active panel
		selected := (m.activePanel == SidebarPanel && i == m.cursor)
		renderedItem := m.renderSidebarItem(item, selected, width)
		items = append(items, renderedItem)
		// Removed spacing between items to match file explorer's compact style
	}

	// Create a viewport-like container for consistent rendering
	content := lipgloss.JoinVertical(lipgloss.Left, items...)

	// Ensure the content fills the available height
	contentHeight := lipgloss.Height(content)
	if contentHeight < height && height > contentHeight {
		paddingLines := utils.Max(0, height-contentHeight)
		padding := strings.Repeat("\n", paddingLines)
		content = content + padding
	}

	return content
}

func (m *Model) renderSidebarItem(item MenuItem, selected bool, width int) string {
	// Build the content string with proper spacing
	icon := item.Icon
	title := item.Title

	// Calculate padding to fill the entire width
	// Account for: icon (1-2 chars) + space + title + padding
	contentLength := lipgloss.Width(icon) + 1 + lipgloss.Width(title)
	paddingNeeded := width - contentLength - 2 // -2 for left/right margins (reduced from -4)
	if paddingNeeded < 0 {
		paddingNeeded = 0
	}

	// Create the full content line with padding (single space indent to match file explorer)
	content := " " + icon + " " + title + strings.Repeat(" ", paddingNeeded)

	if selected {
		// Selected state: Pink background with dark text (matching file explorer table)
		return lipgloss.NewStyle().
			Width(width).
			Background(theme.ColorAccent).
			Foreground(theme.ColorBackground).
			Bold(false). // No bold, matching table style
			Render(content)
	}

	// Normal state: No background, regular text (not dimmed when panel is active)
	normalForeground := theme.ColorForegroundDim
	if m.activePanel == SidebarPanel {
		normalForeground = theme.ColorForeground // Use normal foreground when panel is active
	}

	return lipgloss.NewStyle().
		Width(width).
		Foreground(normalForeground).
		Render(content)
}
func (m *Model) renderFooter(width int) string {
	help := []string{
		"↑↓: Navigate",
		"Tab: Switch Panel",
		"Enter: Select",
		"p: Preview",
		"q: Quit",
	}

	helpText := strings.Join(help, " | ")

	return lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Width(width).
		Align(lipgloss.Center).
		Render(helpText)
}

func (m *Model) renderCompactFooter(width int) string {
	help := "Tab: Switch | Enter: Select | q: Quit"

	return lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Width(width).
		Align(lipgloss.Center).
		Render(help)
}
