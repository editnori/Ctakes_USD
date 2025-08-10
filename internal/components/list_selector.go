package components

import (
	"fmt"
	"strings"

	"github.com/ctakes-tui/ctakes-tui/internal/theme"
)

// ListSelector - Reusable list selection component
type ListSelector struct {
	// Items
	items         []ListItem
	cursor        int
	selectedItems map[int]bool

	// Configuration
	multiSelect bool
	showIcons   bool
	title       string

	// Display
	width      int
	height     int
	startIndex int

	// Callbacks
	onSelect      func(item ListItem, index int)
	onMultiSelect func(items []ListItem, indices []int)

	// Styling
	showCheckboxes bool
}

type ListItem struct {
	Label       string
	Value       interface{}
	Icon        string
	Disabled    bool
	Separator   bool // Render as separator/header
	Description string
}

// NewListSelector creates a new list selector component
func NewListSelector(title string, multiSelect bool) *ListSelector {
	return &ListSelector{
		title:          title,
		multiSelect:    multiSelect,
		selectedItems:  make(map[int]bool),
		showIcons:      true,
		showCheckboxes: multiSelect,
		width:          60,
		height:         15,
	}
}

// Configuration methods
func (ls *ListSelector) SetSize(width, height int) *ListSelector {
	ls.width = width
	ls.height = height
	return ls
}

func (ls *ListSelector) SetItems(items []ListItem) *ListSelector {
	ls.items = items
	ls.cursor = 0
	ls.startIndex = 0
	ls.selectedItems = make(map[int]bool)
	return ls
}

func (ls *ListSelector) AddItem(item ListItem) *ListSelector {
	ls.items = append(ls.items, item)
	return ls
}

func (ls *ListSelector) AddSeparator(label string) *ListSelector {
	ls.items = append(ls.items, ListItem{
		Label:     label,
		Separator: true,
	})
	return ls
}

func (ls *ListSelector) SetShowIcons(show bool) *ListSelector {
	ls.showIcons = show
	return ls
}

func (ls *ListSelector) SetShowCheckboxes(show bool) *ListSelector {
	ls.showCheckboxes = show
	return ls
}

func (ls *ListSelector) OnSelect(callback func(item ListItem, index int)) *ListSelector {
	ls.onSelect = callback
	return ls
}

func (ls *ListSelector) OnMultiSelect(callback func(items []ListItem, indices []int)) *ListSelector {
	ls.onMultiSelect = callback
	return ls
}

// Navigation methods
func (ls *ListSelector) MoveUp() {
	if ls.cursor > 0 {
		ls.cursor--
		// Skip separators
		for ls.cursor >= 0 && ls.items[ls.cursor].Separator {
			ls.cursor--
		}
		if ls.cursor < 0 {
			ls.cursor = 0
		}
		ls.updateScrollPosition()
	}
}

func (ls *ListSelector) MoveDown() {
	if ls.cursor < len(ls.items)-1 {
		ls.cursor++
		// Skip separators
		for ls.cursor < len(ls.items) && ls.items[ls.cursor].Separator {
			ls.cursor++
		}
		if ls.cursor >= len(ls.items) {
			ls.cursor = len(ls.items) - 1
		}
		ls.updateScrollPosition()
	}
}

func (ls *ListSelector) MoveToFirst() {
	ls.cursor = 0
	for ls.cursor < len(ls.items) && ls.items[ls.cursor].Separator {
		ls.cursor++
	}
	ls.updateScrollPosition()
}

func (ls *ListSelector) MoveToLast() {
	ls.cursor = len(ls.items) - 1
	for ls.cursor >= 0 && ls.items[ls.cursor].Separator {
		ls.cursor--
	}
	ls.updateScrollPosition()
}

func (ls *ListSelector) updateScrollPosition() {
	visibleHeight := ls.height - 3 // Header + footer

	if ls.cursor < ls.startIndex {
		ls.startIndex = ls.cursor
	} else if ls.cursor >= ls.startIndex+visibleHeight {
		ls.startIndex = ls.cursor - visibleHeight + 1
	}

	// Ensure we don't scroll past the end
	maxStart := len(ls.items) - visibleHeight
	if maxStart < 0 {
		maxStart = 0
	}
	if ls.startIndex > maxStart {
		ls.startIndex = maxStart
	}
}

// Selection methods
func (ls *ListSelector) ToggleSelection() {
	if !ls.multiSelect || ls.cursor >= len(ls.items) {
		return
	}

	item := ls.items[ls.cursor]
	if item.Disabled || item.Separator {
		return
	}

	ls.selectedItems[ls.cursor] = !ls.selectedItems[ls.cursor]
}

func (ls *ListSelector) SelectAll() {
	if !ls.multiSelect {
		return
	}

	for i, item := range ls.items {
		if !item.Disabled && !item.Separator {
			ls.selectedItems[i] = true
		}
	}
}

func (ls *ListSelector) ClearSelection() {
	ls.selectedItems = make(map[int]bool)
}

func (ls *ListSelector) GetSelectedItems() ([]ListItem, []int) {
	items := []ListItem{}
	indices := []int{}

	for i, selected := range ls.selectedItems {
		if selected && i < len(ls.items) {
			items = append(items, ls.items[i])
			indices = append(indices, i)
		}
	}

	return items, indices
}

func (ls *ListSelector) GetSelectedCount() int {
	count := 0
	for _, selected := range ls.selectedItems {
		if selected {
			count++
		}
	}
	return count
}

func (ls *ListSelector) GetCurrentItem() (ListItem, int) {
	if ls.cursor < len(ls.items) {
		return ls.items[ls.cursor], ls.cursor
	}
	return ListItem{}, -1
}

// Action methods
func (ls *ListSelector) SelectCurrent() {
	if ls.cursor >= len(ls.items) {
		return
	}

	item := ls.items[ls.cursor]
	if item.Disabled || item.Separator {
		return
	}

	if ls.multiSelect {
		ls.ToggleSelection()
	} else if ls.onSelect != nil {
		ls.onSelect(item, ls.cursor)
	}
}

func (ls *ListSelector) SaveSelection() {
	if ls.multiSelect && ls.onMultiSelect != nil {
		items, indices := ls.GetSelectedItems()
		ls.onMultiSelect(items, indices)
	} else if ls.onSelect != nil {
		item, index := ls.GetCurrentItem()
		if index >= 0 && !item.Disabled && !item.Separator {
			ls.onSelect(item, index)
		}
	}
}

// Render the list selector
func (ls *ListSelector) Render() string {
	lines := make([]string, 0, ls.height)

	// Header
	if ls.title != "" {
		selectedInfo := ""
		if ls.multiSelect {
			selectedInfo = fmt.Sprintf(" (%d selected)", ls.GetSelectedCount())
		}
		header := theme.RenderHeader(ls.title+selectedInfo, ls.width)
		lines = append(lines, header)
		lines = append(lines, theme.RenderDivider(ls.width))
	}

	// Empty list
	if len(ls.items) == 0 {
		emptyMsg := theme.RenderTextDim("No items available")
		lines = append(lines, emptyMsg)
		return ls.padToHeight(lines)
	}

	// Item list
	visibleHeight := ls.height - 3 // Header + divider + footer
	if ls.title == "" {
		visibleHeight = ls.height - 1 // Just footer
	}

	endIndex := ls.startIndex + visibleHeight
	if endIndex > len(ls.items) {
		endIndex = len(ls.items)
	}

	for i := ls.startIndex; i < endIndex; i++ {
		item := ls.items[i]
		focused := i == ls.cursor
		selected := ls.selectedItems[i]

		line := ls.renderListItem(item, selected, focused, i)
		lines = append(lines, line)
	}

	// Footer with help
	footer := ls.renderFooter()
	lines = append(lines, footer)

	return ls.padToHeight(lines)
}

func (ls *ListSelector) renderListItem(item ListItem, selected, focused bool, index int) string {
	// Separator/header items
	if item.Separator {
		return theme.RenderTextBold(item.Label)
	}

	parts := []string{}

	// Checkbox/selection indicator
	if ls.showCheckboxes {
		indicator := theme.RenderIcon("inactive")
		if selected {
			indicator = theme.RenderIcon("success")
		}
		parts = append(parts, indicator)
	}

	// Icon
	if ls.showIcons && item.Icon != "" {
		parts = append(parts, theme.RenderIcon(item.Icon))
	}

	// Label
	label := item.Label
	if item.Disabled {
		label = theme.RenderTextDim(label)
	}
	parts = append(parts, label)

	// Description (if fits)
	if item.Description != "" {
		maxDescWidth := ls.width - 20 // Rough estimate for other parts
		if len(item.Description) <= maxDescWidth {
			desc := theme.RenderTextDim(" - " + item.Description)
			parts = append(parts, desc)
		}
	}

	content := strings.Join(parts, " ")

	// Apply styling
	if item.Disabled {
		return theme.RenderTextDim(content)
	} else if focused {
		return theme.RenderSelection(content, ls.width)
	}

	return theme.RenderText(content)
}

func (ls *ListSelector) renderFooter() string {
	helpItems := []string{}

	if ls.multiSelect {
		helpItems = append(helpItems, "[Space] Toggle")
	}

	helpItems = append(helpItems, "[Enter] Select")

	if ls.multiSelect {
		helpItems = append(helpItems, "[A] All", "[C] Clear")
	}

	helpItems = append(helpItems, "[Esc] Cancel")

	return theme.RenderHelpBar(helpItems, ls.width)
}

func (ls *ListSelector) padToHeight(lines []string) string {
	// Pad with empty lines to reach desired height
	for len(lines) < ls.height {
		lines = append(lines, "")
	}

	// Truncate if too many lines
	if len(lines) > ls.height {
		lines = lines[:ls.height]
	}

	return strings.Join(lines, "\n")
}

// Utility methods
func (ls *ListSelector) FindItemByValue(value interface{}) int {
	for i, item := range ls.items {
		if item.Value == value {
			return i
		}
	}
	return -1
}

func (ls *ListSelector) SetSelectedByValue(value interface{}) {
	index := ls.FindItemByValue(value)
	if index >= 0 {
		ls.cursor = index
		if ls.multiSelect {
			ls.selectedItems[index] = true
		}
	}
}

func (ls *ListSelector) SetSelectedByValues(values []interface{}) {
	if !ls.multiSelect {
		return
	}

	ls.ClearSelection()
	for _, value := range values {
		index := ls.FindItemByValue(value)
		if index >= 0 {
			ls.selectedItems[index] = true
		}
	}
}

// Filter items by a predicate function
func (ls *ListSelector) FilterItems(predicate func(ListItem) bool) *ListSelector {
	filtered := []ListItem{}
	for _, item := range ls.items {
		if predicate(item) {
			filtered = append(filtered, item)
		}
	}
	ls.items = filtered
	ls.cursor = 0
	ls.startIndex = 0
	ls.selectedItems = make(map[int]bool)
	return ls
}
