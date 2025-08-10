package theme

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"os"
	"runtime"
	"strings"
)

// GLOBAL THEME SYSTEM - Claude Code Aesthetic
// Single source of truth for all application styling
// Dark/Light contrast with clean lines, NO blue hues

var (
	// CORE COLORS - Claude Code Dark/Light Aesthetic
	// Light theme colors (primary)
	ColorLightBG     = lipgloss.Color("#FFFFFF") // Pure white background
	ColorLightFG     = lipgloss.Color("#1A1A1A") // Near-black text
	ColorLightFGDim  = lipgloss.Color("#666666") // Medium gray for dim text
	ColorLightBorder = lipgloss.Color("#E5E5E5") // Light gray borders
	ColorLightAccent = lipgloss.Color("#2A2A2A") // Dark accent for selections

	// Dark theme colors (alternative)
	ColorDarkBG     = lipgloss.Color("#1A1A1A") // Near-black background
	ColorDarkFG     = lipgloss.Color("#FFFFFF") // White text
	ColorDarkFGDim  = lipgloss.Color("#AAAAAA") // Light gray for dim text
	ColorDarkBorder = lipgloss.Color("#333333") // Dark gray borders
	ColorDarkAccent = lipgloss.Color("#E5E5E5") // Light accent for selections

	// ADDITIONAL COLORS - Used by dashboard views
	ColorBackground      = ColorLightBG              // Alias for current background
	ColorForeground      = ColorLightFG              // Alias for current foreground
	ColorForegroundDim   = ColorLightFGDim           // Alias for current dim foreground
	ColorSelectionActive = ColorLightAccent          // Alias for current selection
	ColorSecondary       = lipgloss.Color("#888888") // Secondary text color
	ColorBorderInactive  = ColorLightBorder          // Alias for inactive borders
	ColorText            = ColorLightFG              // Alias for text color

	// SEMANTIC COLORS - Minimal, high contrast
	ColorSuccess = lipgloss.Color("#00AA00") // Green for success/active
	ColorError   = lipgloss.Color("#CC0000") // Red for errors/critical
	ColorWarning = lipgloss.Color("#FF8800") // Orange for warnings
	ColorInfo    = lipgloss.Color("#333333") // Dark gray for info (no blue!)

	// ACTIVE COLOR SCHEME (defaults to light)
	ColorBG        = ColorLightBG
	ColorFG        = ColorLightFG
	ColorFGDim     = ColorLightFGDim
	ColorBorder    = ColorLightBorder
	ColorAccent    = ColorLightAccent
	ColorSelection = ColorLightAccent // Unified selection color

	// TYPOGRAPHY SYSTEM - OpenDyslexic-compatible
	FontFamily     = "OpenDyslexic, monospace" // Clean, readable font
	FontSizeNormal = 14
	FontSizeSmall  = 12
	FontSizeLarge  = 16

	// SPACING SYSTEM - Consistent throughout
	SpaceXS = 1
	SpaceSM = 2
	SpaceMD = 4 // Default spacing
	SpaceLG = 8
	SpaceXL = 12

	// Additional spacing aliases used by dashboard
	SpacingXS = SpaceXS
	SpacingSM = SpaceSM
	SpacingMD = SpaceMD
	SpacingLG = SpaceLG

	// COMPONENT DIMENSIONS
	HeightRow    = 1 // Single line
	HeightPanel  = 3 // Panel headers/footers
	HeightButton = 1 // Buttons
	HeightHeader = 3 // Header height

	WidthMin   = 20
	WidthPanel = 40
	WidthFull  = -1 // Use full available width
)

// BORDER SYSTEM - Clean, minimal borders
var (
	BorderLight  = "─"
	BorderHeavy  = "━"
	BorderVert   = "│"
	BorderCorner = "┌┐└┘"

	// ASCII fallbacks
	BorderLightASCII  = "-"
	BorderHeavyASCII  = "="
	BorderVertASCII   = "|"
	BorderCornerASCII = "++++"
)

// ICON SYSTEM - Clean, minimal icons
var (
	IconActive   = "●" // Solid dot for active/selected
	IconInactive = "○" // Hollow dot for inactive
	IconFolder   = "▶" // Triangle for folders
	IconFile     = "▫" // Small square for files
	IconSuccess  = "✓" // Check mark
	IconError    = "✗" // X mark
	IconWarning  = "!" // Exclamation
	IconInfo     = "i" // Information

	// Circle symbols for semantic status
	CircleBlue   = "🔵" // Blue circle for info/primary
	CircleGreen  = "🟢" // Green circle for success/active
	CircleRed    = "🔴" // Red circle for error/critical
	CircleYellow = "🟡" // Yellow circle for warning
	CircleWhite  = "⚪" // White circle for inactive/default
	CircleBlack  = "⚫" // Black circle for disabled
	CirclePurple = "🟣" // Purple circle for special/advanced
	CircleOrange = "🟠" // Orange circle for config/settings

	// ASCII fallbacks
	IconActiveASCII   = "*"
	IconInactiveASCII = "o"
	IconFolderASCII   = ">"
	IconFileASCII     = "-"
	IconSuccessASCII  = "+"
	IconErrorASCII    = "x"
	IconWarningASCII  = "!"
	IconInfoASCII     = "i"
	CircleBlueASCII   = "(i)"
	CircleGreenASCII  = "(+)"
	CircleRedASCII    = "(-)"
	CircleYellowASCII = "(!)"
	CircleWhiteASCII  = "( )"
	CircleBlackASCII  = "(x)"
	CirclePurpleASCII = "(*)"
	CircleOrangeASCII = "(o)"
)

// GLOBAL STYLES - All components use these
var (
	// Base container style
	BaseStyle = lipgloss.NewStyle().
			Background(ColorBG).
			Foreground(ColorFG)

	// Text styles
	TextStyle = lipgloss.NewStyle().
			Foreground(ColorFG)

	TextDimStyle = lipgloss.NewStyle().
			Foreground(ColorFGDim)

	TextBoldStyle = lipgloss.NewStyle().
			Foreground(ColorFG).
			Bold(true)

	// Selection styles - unified across all components
	SelectionStyle = lipgloss.NewStyle().
			Background(ColorSelection).
			Foreground(ColorBG).
			Bold(true)

	// Border styles
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder)

	BorderActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(ColorAccent)

	// Panel styles
	PanelStyle = lipgloss.NewStyle().
			Background(ColorBG).
			Foreground(ColorFG).
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	PanelActiveStyle = PanelStyle.Copy().
				BorderForeground(ColorAccent)

	// Header styles
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorFG).
			Padding(0, 1)

	SubHeaderStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Padding(0, 1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorFGDim).
			Italic(true)

	// Footer/status styles
	FooterStyle = lipgloss.NewStyle().
			Foreground(ColorFGDim).
			Padding(0, 1)

	// Button styles
	ButtonStyle = lipgloss.NewStyle().
			Background(ColorBG).
			Foreground(ColorFG).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 2)

	ButtonActiveStyle = lipgloss.NewStyle().
				Background(ColorAccent).
				Foreground(ColorBG).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorAccent).
				Bold(true).
				Padding(0, 2)

	// Status message styles
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(ColorInfo)

	// Content styles
	ContentCompactStyle = lipgloss.NewStyle().
				Padding(0, SpacingXS)
)

// INITIALIZATION
var asciiMode bool

func init() {
	// Detect ASCII mode (no env override needed for clean implementation)
	asciiMode = runtime.GOOS == "windows" || os.Getenv("CTAKES_ASCII") == "1"

	if asciiMode {
		setupASCII()
	}
}

// setupASCII switches all visual elements to ASCII-safe alternatives
func setupASCII() {
	// Switch borders
	BorderLight = BorderLightASCII
	BorderHeavy = BorderHeavyASCII
	BorderVert = BorderVertASCII

	// Switch icons
	IconActive = IconActiveASCII
	IconInactive = IconInactiveASCII
	IconFolder = IconFolderASCII
	IconFile = IconFileASCII
	IconSuccess = IconSuccessASCII
	IconError = IconErrorASCII
	IconWarning = IconWarningASCII
	IconInfo = IconInfoASCII

	// Switch circles
	CircleBlue = CircleBlueASCII
	CircleGreen = CircleGreenASCII
	CircleRed = CircleRedASCII
	CircleYellow = CircleYellowASCII
	CircleWhite = CircleWhiteASCII
	CircleBlack = CircleBlackASCII
	CirclePurple = CirclePurpleASCII
	CircleOrange = CircleOrangeASCII

	// Update border styles
	asciiBorder := lipgloss.Border{
		Top: BorderLight, Bottom: BorderLight,
		Left: BorderVert, Right: BorderVert,
		TopLeft: "+", TopRight: "+",
		BottomLeft: "+", BottomRight: "+",
	}

	PanelStyle = PanelStyle.Copy().Border(asciiBorder)
	PanelActiveStyle = PanelActiveStyle.Copy().Border(asciiBorder)
	BorderStyle = BorderStyle.Copy().Border(asciiBorder)
	BorderActiveStyle = BorderActiveStyle.Copy().Border(asciiBorder)
	ButtonStyle = ButtonStyle.Copy().Border(asciiBorder)
	ButtonActiveStyle = ButtonActiveStyle.Copy().Border(asciiBorder)
}

// THEME SWITCHING (for future dark mode support)
func SetDarkTheme() {
	ColorBG = ColorDarkBG
	ColorFG = ColorDarkFG
	ColorFGDim = ColorDarkFGDim
	ColorBorder = ColorDarkBorder
	ColorAccent = ColorDarkAccent
	ColorSelection = ColorDarkAccent
	updateAllStyles()
}

func SetLightTheme() {
	ColorBG = ColorLightBG
	ColorFG = ColorLightFG
	ColorFGDim = ColorLightFGDim
	ColorBorder = ColorLightBorder
	ColorAccent = ColorLightAccent
	ColorSelection = ColorLightAccent
	updateAllStyles()
}

func updateAllStyles() {
	// Update all global styles with new colors
	BaseStyle = BaseStyle.Background(ColorBG).Foreground(ColorFG)
	TextStyle = TextStyle.Foreground(ColorFG)
	TextDimStyle = TextDimStyle.Foreground(ColorFGDim)
	TextBoldStyle = TextBoldStyle.Foreground(ColorFG)
	SelectionStyle = SelectionStyle.Background(ColorSelection).Foreground(ColorBG)
	BorderStyle = BorderStyle.BorderForeground(ColorBorder)
	BorderActiveStyle = BorderActiveStyle.BorderForeground(ColorAccent)
	PanelStyle = PanelStyle.Background(ColorBG).Foreground(ColorFG).BorderForeground(ColorBorder)
	PanelActiveStyle = PanelActiveStyle.BorderForeground(ColorAccent)
	HeaderStyle = HeaderStyle.Foreground(ColorFG)
	FooterStyle = FooterStyle.Foreground(ColorFGDim)
	ButtonStyle = ButtonStyle.Background(ColorBG).Foreground(ColorFG).BorderForeground(ColorBorder)
	ButtonActiveStyle = ButtonActiveStyle.Background(ColorAccent).Foreground(ColorBG).BorderForeground(ColorAccent)
}

// UTILITY FUNCTIONS - Core theme functions used throughout the app

// RenderText renders text with consistent styling
func RenderText(text string) string {
	return TextStyle.Render(text)
}

func RenderTextDim(text string) string {
	return TextDimStyle.Render(text)
}

func RenderTextBold(text string) string {
	return TextBoldStyle.Render(text)
}

// RenderSelection renders selected content with unified selection style
func RenderSelection(content string, width int) string {
	if width > 0 {
		return SelectionStyle.Width(width).Render(content)
	}
	return SelectionStyle.Render(content)
}

// RenderPanel renders a bordered panel
func RenderPanel(content string, width, height int, active bool) string {
	style := PanelStyle
	if active {
		style = PanelActiveStyle
	}

	if width > 0 && height > 0 {
		return style.Width(width).Height(height).Render(content)
	}
	return style.Render(content)
}

// RenderHeader renders a consistent header
func RenderHeader(text string, width int) string {
	if width > 0 {
		return HeaderStyle.Width(width).Render(text)
	}
	return HeaderStyle.Render(text)
}

// RenderFooter renders a consistent footer
func RenderFooter(text string, width int) string {
	if width > 0 {
		return FooterStyle.Width(width).Render(text)
	}
	return FooterStyle.Render(text)
}

// RenderButton renders a button
func RenderButton(text string, active bool) string {
	style := ButtonStyle
	if active {
		style = ButtonActiveStyle
	}
	return style.Render(text)
}

// RenderStatus renders status messages with appropriate colors
func RenderStatus(msgType, text string) string {
	icon := IconInfo
	style := InfoStyle

	switch msgType {
	case "success":
		icon = IconSuccess
		style = SuccessStyle
	case "error":
		icon = IconError
		style = ErrorStyle
	case "warning":
		icon = IconWarning
		style = WarningStyle
	}

	return style.Render(icon + " " + text)
}

// RenderIcon gets the appropriate icon for the current mode
func RenderIcon(iconType string) string {
	switch iconType {
	case "active":
		return IconActive
	case "inactive":
		return IconInactive
	case "folder":
		return IconFolder
	case "file":
		return IconFile
	case "success":
		return IconSuccess
	case "error":
		return IconError
	case "warning":
		return IconWarning
	case "info":
		return IconInfo
	default:
		return IconInfo
	}
}

// RenderDivider creates a horizontal divider line
func RenderDivider(width int) string {
	if width <= 0 {
		return ""
	}

	line := ""
	for i := 0; i < width; i++ {
		line += BorderLight
	}
	return TextDimStyle.Render(line)
}

// GetSemanticIcon returns appropriate icons for semantic meaning
func GetSemanticIcon(iconType string) string {
	switch iconType {
	case "success", "active", "running":
		return CircleGreen
	case "error", "critical", "danger":
		return CircleRed
	case "warning", "attention":
		return CircleYellow
	case "info", "primary", "data":
		return CircleBlue
	case "inactive", "disabled":
		return CircleBlack
	case "default", "neutral":
		return CircleWhite
	case "special", "advanced", "memory":
		return CirclePurple
	case "config", "settings", "preferences":
		return CircleOrange
	default:
		return RenderIcon(iconType) // Fall back to standard icons
	}
}

// RenderSelectableRow renders a row with full-width selection highlighting
func RenderSelectableRow(content string, width int, selected bool, focused bool) string {
	if width <= 0 {
		return content
	}

	// Pad content to full width
	paddedContent := content
	if len(content) < width {
		paddedContent = content + strings.Repeat(" ", width-len(content))
	} else if len(content) > width {
		paddedContent = content[:width]
	}

	if focused {
		return SelectionStyle.Width(width).Bold(true).Render(paddedContent)
	} else if selected {
		return SelectionStyle.Width(width).Render(paddedContent)
	}

	return TextStyle.Width(width).Render(paddedContent)
}

// RenderCheckboxItem renders a checkbox item with full-width highlighting
func RenderCheckboxItem(label string, checked bool, width int, focused bool) string {
	checkbox := GetSemanticIcon("inactive")
	if checked {
		checkbox = GetSemanticIcon("success")
	}

	content := fmt.Sprintf(" %s  %s", checkbox, label)
	return RenderSelectableRow(content, width, checked, focused)
}

// RenderSelectionHeader renders a consistent header for selection views
func RenderSelectionHeader(title string, selectedCount, totalCount int, width int) string {
	header := RenderHeader(title, width)

	if totalCount > 0 {
		info := fmt.Sprintf("%s %d of %d selected", GetSemanticIcon("info"), selectedCount, totalCount)
		header += "\n" + InfoStyle.Render(info)
	}

	return header
}

// RenderSelectionFooter renders a consistent footer with help text
func RenderSelectionFooter(width int, customHelp ...string) string {
	helpText := "[Space] Toggle  [A] All  [C] Clear  [Enter] Save  [Esc] Cancel"
	if len(customHelp) > 0 {
		helpText = strings.Join(customHelp, "  ")
	}
	return FooterStyle.Width(width).Render(helpText)
}

// RenderStatusMessage renders status messages with appropriate colors and icons
func RenderStatusMessage(msgType, text string) string {
	icon := GetSemanticIcon("info")
	style := InfoStyle

	switch msgType {
	case "success":
		icon = GetSemanticIcon("success")
		style = SuccessStyle
	case "error":
		icon = GetSemanticIcon("error")
		style = ErrorStyle
	case "warning":
		icon = GetSemanticIcon("warning")
		style = WarningStyle
	}

	return style.Render(icon + " " + text)
}

// RenderScrollIndicator renders a scroll position indicator
func RenderScrollIndicator(startIdx, endIdx, total int, width int) string {
	if total <= endIdx-startIdx {
		return ""
	}

	scrollInfo := fmt.Sprintf("Showing %d-%d of %d", startIdx+1, endIdx, total)
	return InfoStyle.Render(scrollInfo)
}
