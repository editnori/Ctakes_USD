package theme

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Bubblegum Pink Theme - Light and playful colors
	ColorBackground        = lipgloss.Color("#2D2A2E") // Soft dark purple-gray
	ColorBackgroundDarker  = lipgloss.Color("#221F22") // Darker purple-gray
	ColorBackgroundLighter = lipgloss.Color("#403E41") // Lighter purple-gray

	// Text colors
	ColorForeground       = lipgloss.Color("#FCE4EC") // Light pink white
	ColorForegroundDim    = lipgloss.Color("#C48B9F") // Muted pink
	ColorForegroundBright = lipgloss.Color("#FFFFFF") // Pure white

	// Border colors
	ColorBorderInactive = lipgloss.Color("#C48B9F") // Muted pink
	ColorBorderActive   = lipgloss.Color("#FF6DB6") // Bright pink
	ColorBorderFocused  = lipgloss.Color("#FF8DC7") // Lighter bright pink

	// Bubblegum accent colors
	ColorAccent    = lipgloss.Color("#FF6DB6") // Hot pink
	ColorSecondary = lipgloss.Color("#FFB6D9") // Light pink
	ColorSuccess   = lipgloss.Color("#98E4D6") // Mint green
	ColorWarning   = lipgloss.Color("#FFDAB9") // Peach
	ColorError     = lipgloss.Color("#FF9999") // Light red
	ColorInfo      = lipgloss.Color("#B6D7FF") // Baby blue
	ColorHighlight = lipgloss.Color("#FFEB3B") // Yellow

	// Special colors for different elements
	ColorDirectory  = lipgloss.Color("#FF8DC7") // Pink for folders
	ColorExecutable = lipgloss.Color("#98E4D6") // Mint for executables
	ColorSymlink    = lipgloss.Color("#B6D7FF") // Blue for symlinks
	ColorArchive    = lipgloss.Color("#FFDAB9") // Peach for archives
	ColorDocument   = lipgloss.Color("#FCE4EC") // Light pink for docs
	ColorMedia      = lipgloss.Color("#FFB6D9") // Medium pink for media
	ColorCode       = lipgloss.Color("#98E4D6") // Mint for code

	// Common colors
	ColorPrimary = lipgloss.Color("#FF6DB6")
	ColorText    = lipgloss.Color("#FCE4EC")
	ColorBorder  = lipgloss.Color("#C48B9F")

	// System monitor colors
	ColorCPU    = lipgloss.Color("#FF6DB6") // Hot pink for CPU
	ColorMemory = lipgloss.Color("#B6D7FF") // Baby blue for Memory
	ColorDisk   = lipgloss.Color("#98E4D6") // Mint green for Disk
)

// Border characters - modern rounded style
const (
	BorderTop          = "─"
	BorderBottom       = "─"
	BorderLeft         = "│"
	BorderRight        = "│"
	BorderTopLeft      = "╭"
	BorderTopRight     = "╮"
	BorderBottomLeft   = "╰"
	BorderBottomRight  = "╯"
	BorderMiddleLeft   = "├"
	BorderMiddleRight  = "┤"
	BorderMiddleTop    = "┬"
	BorderMiddleBottom = "┴"
	BorderCross        = "┼"
	BorderDividerH     = "─"
	BorderDividerV     = "│"
)

// Icons - Professional Unicode symbols
const (
	IconFolder       = "▤"
	IconFile         = "▫"
	IconFolderOpen   = "▥"
	IconHome         = "▣"
	IconSearch       = "◎"
	IconSettings     = "●"
	IconHelp         = "◔"
	IconAnalyze      = "◈"
	IconPipeline     = "▥"
	IconDictionary   = "▦"
	IconDocument     = "▫"
	IconResults      = "◉"
	IconCheck        = "✓"
	IconCross        = "✗"
	IconArrowRight   = "→"
	IconArrowLeft    = "←"
	IconArrowUp      = "↑"
	IconArrowDown    = "↓"
	IconDot          = "•"
	IconChevronRight = "›"
	IconChevronDown  = "v"
	IconClinical     = "◆"
	IconMedical      = "◇"
	IconProcess      = "▥"
	IconCPU          = "◓"
	IconMemory       = "◒"
	IconDisk         = "◐"
	IconCode         = "◈"
	IconMedia        = "◉"
)

// Base styles
var (
	BaseStyle = lipgloss.NewStyle().
			Background(ColorBackground).
			Foreground(ColorForeground)

	// Panel styles with soft rounded borders
	PanelStyle = lipgloss.NewStyle().
			Background(ColorBackground).
			Foreground(ColorForeground).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorderInactive).
			Padding(0, 1)

	PanelActiveStyle = PanelStyle.Copy().
				BorderForeground(ColorBorderActive)

	PanelFocusedStyle = PanelStyle.Copy().
				BorderForeground(ColorBorderFocused)

	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true).
			Padding(0, 1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorForegroundDim).
			Italic(true)

	// Error style
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	// List and menu styles
	MenuItemStyle = lipgloss.NewStyle().
			Foreground(ColorForeground).
			Padding(0, 1)

	MenuItemSelectedStyle = lipgloss.NewStyle().
				Background(ColorBackgroundLighter).
				Foreground(ColorSecondary).
				Bold(true).
				Padding(0, 1)

	MenuItemFocusedStyle = lipgloss.NewStyle().
				Background(ColorAccent).
				Foreground(ColorBackground).
				Bold(true).
				Padding(0, 1)

	// Status styles
	StatusStyle = lipgloss.NewStyle().
			Background(ColorBackgroundDarker).
			Foreground(ColorForeground).
			Padding(0, 1)

	StatusSuccessStyle = StatusStyle.Copy().
				Foreground(ColorSuccess)

	StatusErrorStyle = StatusStyle.Copy().
				Foreground(ColorError)

	StatusWarningStyle = StatusStyle.Copy().
				Foreground(ColorWarning)

	StatusInfoStyle = StatusStyle.Copy().
			Foreground(ColorInfo)

	// Footer styles
	FooterStyle = lipgloss.NewStyle().
			Background(ColorBackgroundDarker).
			Foreground(ColorForegroundDim).
			Padding(0, 1)

	FooterKeyStyle = lipgloss.NewStyle().
			Background(ColorBackgroundLighter).
			Foreground(ColorAccent).
			Padding(0, 1).
			Margin(0, 1)

	FooterDescStyle = lipgloss.NewStyle().
			Foreground(ColorForeground)

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true).
				BorderBottom(true).
				BorderForeground(ColorBorderInactive)

	TableRowStyle = lipgloss.NewStyle().
			Foreground(ColorForeground)

	TableRowSelectedStyle = lipgloss.NewStyle().
				Background(ColorBackgroundLighter).
				Foreground(ColorForegroundBright)

	// Input styles
	InputStyle = lipgloss.NewStyle().
			Background(ColorBackgroundDarker).
			Foreground(ColorForeground).
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorBorderInactive).
			Padding(0, 1)

	InputFocusedStyle = InputStyle.Copy().
				BorderForeground(ColorBorderActive)

	// Button styles
	ButtonStyle = lipgloss.NewStyle().
			Background(ColorBackgroundLighter).
			Foreground(ColorForeground).
			Padding(0, 2).
			Margin(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorderInactive)

	ButtonActiveStyle = ButtonStyle.Copy().
				Background(ColorAccent).
				Foreground(ColorBackground).
				BorderForeground(ColorAccent).
				Bold(true)

	// Progress bar styles
	ProgressBarStyle = lipgloss.NewStyle().
				Background(ColorBackgroundDarker).
				Foreground(ColorAccent)

	ProgressBarEmptyStyle = lipgloss.NewStyle().
				Background(ColorBackgroundDarker).
				Foreground(ColorForegroundDim)

	// System stat styles
	CPUStyle = lipgloss.NewStyle().
			Foreground(ColorCPU).
			Bold(true)

	MemoryStyle = lipgloss.NewStyle().
			Foreground(ColorMemory).
			Bold(true)

	DiskStyle = lipgloss.NewStyle().
			Foreground(ColorDisk).
			Bold(true)
)

// Helper functions
func RenderBorder(width, height int, title string, active bool) string {
	style := PanelStyle
	if active {
		style = PanelActiveStyle
	}

	return style.Width(width).Height(height).Render("")
}

func RenderTitle(icon, text string) string {
	if icon != "" {
		return TitleStyle.Render(icon + " " + text)
	}
	return TitleStyle.Render(text)
}

func RenderMenuItem(icon, text string, selected, focused bool) string {
	content := text
	if icon != "" {
		content = icon + "  " + text
	}

	if focused {
		return MenuItemFocusedStyle.Render(content)
	}
	if selected {
		return MenuItemSelectedStyle.Render(content)
	}
	return MenuItemStyle.Render(content)
}

func RenderKeyHelp(key, desc string) string {
	return FooterKeyStyle.Render(key) + FooterDescStyle.Render(desc)
}

func RenderStatusBar(items ...string) string {
	var result string
	for i, item := range items {
		if i > 0 {
			result += StatusStyle.Render(" " + BorderDividerV + " ")
		}
		result += StatusStyle.Render(item)
	}
	return result
}
