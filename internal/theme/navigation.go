package theme

import (
	"fmt"
	"strings"
)

// GLOBAL NAVIGATION SYSTEM
// Single source of truth for all navigation elements
// Consistent behavior across all UI components

// NavigationItem represents a single navigation item
type NavigationItem struct {
	Label    string
	Key      string
	Icon     string
	Active   bool
	Disabled bool
}

// NavigationBar renders a horizontal navigation bar
func RenderNavigationBar(items []NavigationItem, width int) string {
	if len(items) == 0 {
		return ""
	}

	// Calculate spacing
	totalItems := len(items)
	itemWidth := (width - (totalItems - 1)) / totalItems // -1 for separators
	if itemWidth < 5 {
		itemWidth = 5 // Minimum width
	}

	parts := make([]string, 0, totalItems)
	for _, item := range items {
		content := formatNavItem(item, itemWidth)
		parts = append(parts, content)
	}

	navBar := strings.Join(parts, " ")

	// Pad to full width
	if len(navBar) < width {
		navBar += strings.Repeat(" ", width-len(navBar))
	}

	return HeaderStyle.Width(width).Render(navBar)
}

// NavigationMenu renders a vertical navigation menu
func RenderNavigationMenu(items []NavigationItem, selectedIndex int, width int) string {
	if len(items) == 0 {
		return ""
	}

	lines := make([]string, 0, len(items))

	for i, item := range items {
		isSelected := i == selectedIndex
		line := formatMenuItem(item, isSelected, width)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// formatNavItem formats a navigation bar item
func formatNavItem(item NavigationItem, width int) string {
	icon := ""
	if item.Icon != "" {
		icon = RenderIcon(item.Icon) + " "
	}

	content := icon + item.Label

	// Truncate if too long
	if len(content) > width-2 {
		content = content[:width-3] + "…"
	}

	if item.Active {
		return ButtonActiveStyle.Width(width).Render(content)
	} else if item.Disabled {
		return TextDimStyle.Width(width).Render(content)
	}

	return ButtonStyle.Width(width).Render(content)
}

// formatMenuItem formats a menu item
func formatMenuItem(item NavigationItem, selected bool, width int) string {
	icon := RenderIcon("inactive")
	if item.Active {
		icon = RenderIcon("active")
	} else if item.Disabled {
		icon = RenderIcon("inactive")
	}

	content := fmt.Sprintf(" %s  %s", icon, item.Label)

	if item.Key != "" {
		keyHelp := fmt.Sprintf("[%s]", item.Key)
		maxLabelWidth := width - len(keyHelp) - 5 // Space for icon, spacing, and key
		if len(item.Label) > maxLabelWidth {
			label := item.Label[:maxLabelWidth-1] + "…"
			content = fmt.Sprintf(" %s  %s", icon, label)
		}
		// Right-align the key
		padding := width - len(content) - len(keyHelp) - 1
		if padding > 0 {
			content += strings.Repeat(" ", padding) + keyHelp
		}
	}

	// Apply selection styling
	if selected {
		return RenderSelection(content, width)
	} else if item.Disabled {
		return TextDimStyle.Width(width).Render(content)
	}

	return TextStyle.Width(width).Render(content)
}

// Breadcrumb navigation
func RenderBreadcrumb(parts []string, width int) string {
	if len(parts) == 0 {
		return ""
	}

	separator := " " + RenderIcon("folder") + " "
	breadcrumb := strings.Join(parts, separator)

	// Truncate if too long, keeping the last part
	if len(breadcrumb) > width-3 {
		// Try to keep last 2 parts if possible
		if len(parts) > 1 {
			lastPart := parts[len(parts)-1]
			secondLastPart := parts[len(parts)-2]
			shortBreadcrumb := "…" + separator + secondLastPart + separator + lastPart

			if len(shortBreadcrumb) <= width {
				breadcrumb = shortBreadcrumb
			} else {
				// Just keep the last part
				breadcrumb = "…" + separator + lastPart
			}
		}
	}

	return TextDimStyle.Width(width).Render(breadcrumb)
}

// Pagination navigation
type PaginationInfo struct {
	CurrentPage int
	TotalPages  int
	HasPrev     bool
	HasNext     bool
}

func RenderPagination(info PaginationInfo, width int) string {
	if info.TotalPages <= 1 {
		return ""
	}

	// Build pagination controls
	controls := []string{}

	if info.HasPrev {
		controls = append(controls, "[←] Prev")
	}

	pageInfo := fmt.Sprintf("Page %d of %d", info.CurrentPage, info.TotalPages)
	controls = append(controls, pageInfo)

	if info.HasNext {
		controls = append(controls, "Next [→]")
	}

	pagination := strings.Join(controls, "  ")

	// Center the pagination
	if len(pagination) < width {
		padding := (width - len(pagination)) / 2
		pagination = strings.Repeat(" ", padding) + pagination
	}

	return TextDimStyle.Width(width).Render(pagination)
}

// Help navigation (consistent help text across all views)
func RenderHelpBar(helpItems []string, width int) string {
	if len(helpItems) == 0 {
		return ""
	}

	helpText := strings.Join(helpItems, "  ")

	// Truncate if too long
	if len(helpText) > width-3 {
		helpText = helpText[:width-3] + "…"
	}

	return FooterStyle.Width(width).Render(helpText)
}

// Common help text shortcuts
func GetStandardHelpItems() []string {
	return []string{
		"[↑↓] Navigate",
		"[Enter] Select",
		"[Tab] Switch",
		"[Esc] Back",
		"[Q] Quit",
	}
}

func GetSelectionHelpItems() []string {
	return []string{
		"[Space] Toggle",
		"[A] All",
		"[C] Clear",
		"[Enter] Save",
		"[Esc] Cancel",
	}
}

func GetFormHelpItems() []string {
	return []string{
		"[Tab] Next Field",
		"[Shift+Tab] Prev Field",
		"[Enter] Save",
		"[Esc] Cancel",
	}
}

// Status bar with consistent layout
func RenderStatusBar(leftItems, rightItems []string, width int) string {
	leftText := strings.Join(leftItems, " | ")
	rightText := strings.Join(rightItems, " | ")

	// Calculate spacing
	totalText := len(leftText) + len(rightText)
	if totalText >= width {
		// Truncate right side if needed
		maxRight := width - len(leftText) - 3
		if maxRight > 0 && len(rightText) > maxRight {
			rightText = rightText[:maxRight-1] + "…"
		} else if maxRight <= 0 {
			rightText = ""
		}
	}

	// Build status bar
	spacing := width - len(leftText) - len(rightText)
	if spacing < 0 {
		spacing = 0
	}

	statusBar := leftText + strings.Repeat(" ", spacing) + rightText

	return FooterStyle.Width(width).Render(statusBar)
}
