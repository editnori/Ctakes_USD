package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ctakes-tui/ctakes-tui/internal/theme"
)

// TableView - Reusable table display component
type TableView struct {
	// Data
	columns    []TableColumn
	rows       []TableRow
	sortedRows []TableRow

	// Display state
	cursor     int
	startRow   int
	sortColumn int
	sortAsc    bool

	// Configuration
	width       int
	height      int
	title       string
	multiSelect bool
	showHeaders bool

	// Selection
	selectedRows map[int]bool

	// Callbacks
	onSelect      func(row TableRow, index int)
	onMultiSelect func(rows []TableRow, indices []int)
}

type TableColumn struct {
	Header    string
	Key       string
	Width     int    // Fixed width, 0 for auto
	MinWidth  int    // Minimum width
	MaxWidth  int    // Maximum width, 0 for no limit
	Alignment string // "left", "center", "right"
	Sortable  bool
	Format    func(value interface{}) string
}

type TableRow struct {
	ID       interface{}            // Unique identifier
	Data     map[string]interface{} // Column key -> value
	Selected bool
	Disabled bool
}

// NewTableView creates a new table view component
func NewTableView(title string) *TableView {
	return &TableView{
		title:        title,
		columns:      []TableColumn{},
		rows:         []TableRow{},
		selectedRows: make(map[int]bool),
		showHeaders:  true,
		sortColumn:   -1,
		sortAsc:      true,
		width:        80,
		height:       20,
	}
}

// Configuration methods
func (tv *TableView) SetSize(width, height int) *TableView {
	tv.width = width
	tv.height = height
	tv.calculateColumnWidths()
	return tv
}

func (tv *TableView) SetMultiSelect(multiSelect bool) *TableView {
	tv.multiSelect = multiSelect
	return tv
}

func (tv *TableView) SetShowHeaders(show bool) *TableView {
	tv.showHeaders = show
	return tv
}

func (tv *TableView) OnSelect(callback func(row TableRow, index int)) *TableView {
	tv.onSelect = callback
	return tv
}

func (tv *TableView) OnMultiSelect(callback func(rows []TableRow, indices []int)) *TableView {
	tv.onMultiSelect = callback
	return tv
}

// Column management
func (tv *TableView) AddColumn(column TableColumn) *TableView {
	tv.columns = append(tv.columns, column)
	tv.calculateColumnWidths()
	return tv
}

func (tv *TableView) SetColumns(columns []TableColumn) *TableView {
	tv.columns = columns
	tv.calculateColumnWidths()
	return tv
}

// Data management
func (tv *TableView) SetRows(rows []TableRow) *TableView {
	tv.rows = rows
	tv.sortedRows = make([]TableRow, len(rows))
	copy(tv.sortedRows, rows)
	tv.applySorting()
	tv.cursor = 0
	tv.startRow = 0
	tv.selectedRows = make(map[int]bool)
	return tv
}

func (tv *TableView) AddRow(row TableRow) *TableView {
	tv.rows = append(tv.rows, row)
	tv.sortedRows = append(tv.sortedRows, row)
	tv.applySorting()
	return tv
}

func (tv *TableView) UpdateRow(id interface{}, data map[string]interface{}) {
	for i, row := range tv.rows {
		if row.ID == id {
			tv.rows[i].Data = data
			// Update in sorted rows too
			for j, sortedRow := range tv.sortedRows {
				if sortedRow.ID == id {
					tv.sortedRows[j].Data = data
					break
				}
			}
			tv.applySorting()
			break
		}
	}
}

func (tv *TableView) RemoveRow(id interface{}) {
	// Remove from rows
	for i, row := range tv.rows {
		if row.ID == id {
			tv.rows = append(tv.rows[:i], tv.rows[i+1:]...)
			break
		}
	}

	// Remove from sorted rows
	for i, row := range tv.sortedRows {
		if row.ID == id {
			tv.sortedRows = append(tv.sortedRows[:i], tv.sortedRows[i+1:]...)
			break
		}
	}

	// Adjust cursor
	if tv.cursor >= len(tv.sortedRows) {
		tv.cursor = len(tv.sortedRows) - 1
		if tv.cursor < 0 {
			tv.cursor = 0
		}
	}

	tv.updateScrollPosition()
}

func (tv *TableView) ClearRows() *TableView {
	tv.rows = []TableRow{}
	tv.sortedRows = []TableRow{}
	tv.cursor = 0
	tv.startRow = 0
	tv.selectedRows = make(map[int]bool)
	return tv
}

// Sorting
func (tv *TableView) SortByColumn(columnIndex int) {
	if columnIndex < 0 || columnIndex >= len(tv.columns) {
		return
	}

	if !tv.columns[columnIndex].Sortable {
		return
	}

	// Toggle sort direction if same column
	if tv.sortColumn == columnIndex {
		tv.sortAsc = !tv.sortAsc
	} else {
		tv.sortColumn = columnIndex
		tv.sortAsc = true
	}

	tv.applySorting()
}

func (tv *TableView) applySorting() {
	if tv.sortColumn < 0 || tv.sortColumn >= len(tv.columns) {
		return
	}

	column := tv.columns[tv.sortColumn]

	sort.Slice(tv.sortedRows, func(i, j int) bool {
		valI := tv.sortedRows[i].Data[column.Key]
		valJ := tv.sortedRows[j].Data[column.Key]

		// Convert to strings for comparison
		strI := tv.formatCellValue(valI, column)
		strJ := tv.formatCellValue(valJ, column)

		if tv.sortAsc {
			return strI < strJ
		}
		return strI > strJ
	})
}

// Navigation
func (tv *TableView) MoveUp() {
	if tv.cursor > 0 {
		tv.cursor--
		tv.updateScrollPosition()
	}
}

func (tv *TableView) MoveDown() {
	if tv.cursor < len(tv.sortedRows)-1 {
		tv.cursor++
		tv.updateScrollPosition()
	}
}

func (tv *TableView) PageUp() {
	pageSize := tv.getVisibleRowCount()
	tv.cursor -= pageSize
	if tv.cursor < 0 {
		tv.cursor = 0
	}
	tv.updateScrollPosition()
}

func (tv *TableView) PageDown() {
	pageSize := tv.getVisibleRowCount()
	tv.cursor += pageSize
	if tv.cursor >= len(tv.sortedRows) {
		tv.cursor = len(tv.sortedRows) - 1
	}
	tv.updateScrollPosition()
}

func (tv *TableView) MoveToFirst() {
	tv.cursor = 0
	tv.updateScrollPosition()
}

func (tv *TableView) MoveToLast() {
	tv.cursor = len(tv.sortedRows) - 1
	tv.updateScrollPosition()
}

func (tv *TableView) updateScrollPosition() {
	visibleRows := tv.getVisibleRowCount()

	if tv.cursor < tv.startRow {
		tv.startRow = tv.cursor
	} else if tv.cursor >= tv.startRow+visibleRows {
		tv.startRow = tv.cursor - visibleRows + 1
	}

	// Ensure we don't scroll past the end
	maxStart := len(tv.sortedRows) - visibleRows
	if maxStart < 0 {
		maxStart = 0
	}
	if tv.startRow > maxStart {
		tv.startRow = maxStart
	}
}

func (tv *TableView) getVisibleRowCount() int {
	headerRows := 0
	if tv.title != "" {
		headerRows += 2 // Title + divider
	}
	if tv.showHeaders {
		headerRows += 2 // Header + divider
	}

	footerRows := 1 // Help

	return tv.height - headerRows - footerRows
}

// Selection
func (tv *TableView) ToggleSelection() {
	if !tv.multiSelect || tv.cursor >= len(tv.sortedRows) {
		return
	}

	row := tv.sortedRows[tv.cursor]
	if row.Disabled {
		return
	}

	tv.selectedRows[tv.cursor] = !tv.selectedRows[tv.cursor]
}

func (tv *TableView) SelectAll() {
	if !tv.multiSelect {
		return
	}

	for i, row := range tv.sortedRows {
		if !row.Disabled {
			tv.selectedRows[i] = true
		}
	}
}

func (tv *TableView) ClearSelection() {
	tv.selectedRows = make(map[int]bool)
}

func (tv *TableView) GetSelectedRows() ([]TableRow, []int) {
	rows := []TableRow{}
	indices := []int{}

	for i, selected := range tv.selectedRows {
		if selected && i < len(tv.sortedRows) {
			rows = append(rows, tv.sortedRows[i])
			indices = append(indices, i)
		}
	}

	return rows, indices
}

func (tv *TableView) GetSelectedCount() int {
	count := 0
	for _, selected := range tv.selectedRows {
		if selected {
			count++
		}
	}
	return count
}

func (tv *TableView) GetCurrentRow() (TableRow, int) {
	if tv.cursor < len(tv.sortedRows) {
		return tv.sortedRows[tv.cursor], tv.cursor
	}
	return TableRow{}, -1
}

// Actions
func (tv *TableView) SelectCurrent() {
	if tv.cursor >= len(tv.sortedRows) {
		return
	}

	row := tv.sortedRows[tv.cursor]
	if row.Disabled {
		return
	}

	if tv.multiSelect {
		tv.ToggleSelection()
	} else if tv.onSelect != nil {
		tv.onSelect(row, tv.cursor)
	}
}

func (tv *TableView) SaveSelection() {
	if tv.multiSelect && tv.onMultiSelect != nil {
		rows, indices := tv.GetSelectedRows()
		tv.onMultiSelect(rows, indices)
	} else if tv.onSelect != nil {
		row, index := tv.GetCurrentRow()
		if index >= 0 && !row.Disabled {
			tv.onSelect(row, index)
		}
	}
}

// Rendering
func (tv *TableView) Render() string {
	lines := make([]string, 0, tv.height)

	// Title
	if tv.title != "" {
		selectedInfo := ""
		if tv.multiSelect {
			selectedInfo = fmt.Sprintf(" (%d selected)", tv.GetSelectedCount())
		}
		header := theme.RenderHeader(tv.title+selectedInfo, tv.width)
		lines = append(lines, header)
		lines = append(lines, theme.RenderDivider(tv.width))
	}

	// Headers
	if tv.showHeaders {
		headerLine := tv.renderHeaders()
		lines = append(lines, headerLine)
		lines = append(lines, theme.RenderDivider(tv.width))
	}

	// Rows
	visibleRows := tv.getVisibleRowCount()
	endRow := tv.startRow + visibleRows
	if endRow > len(tv.sortedRows) {
		endRow = len(tv.sortedRows)
	}

	if len(tv.sortedRows) == 0 {
		emptyMsg := theme.RenderTextDim("No data available")
		lines = append(lines, emptyMsg)
	} else {
		for i := tv.startRow; i < endRow; i++ {
			row := tv.sortedRows[i]
			focused := i == tv.cursor
			selected := tv.selectedRows[i]

			rowLine := tv.renderRow(row, selected, focused)
			lines = append(lines, rowLine)
		}
	}

	// Footer
	footer := tv.renderFooter()
	lines = append(lines, footer)

	return tv.padToHeight(lines)
}

func (tv *TableView) calculateColumnWidths() {
	if len(tv.columns) == 0 {
		return
	}

	// Available width (subtract borders and padding)
	availableWidth := tv.width - 2
	if tv.multiSelect {
		availableWidth -= 3 // Selection indicator
	}

	// Calculate automatic widths
	totalFixedWidth := 0
	autoColumns := 0

	for i := range tv.columns {
		if tv.columns[i].Width > 0 {
			totalFixedWidth += tv.columns[i].Width
		} else {
			autoColumns++
		}
	}

	// Distribute remaining width among auto columns
	if autoColumns > 0 {
		autoWidth := (availableWidth - totalFixedWidth) / autoColumns
		for i := range tv.columns {
			if tv.columns[i].Width == 0 {
				tv.columns[i].Width = autoWidth

				// Respect min/max width constraints
				if tv.columns[i].MinWidth > 0 && tv.columns[i].Width < tv.columns[i].MinWidth {
					tv.columns[i].Width = tv.columns[i].MinWidth
				}
				if tv.columns[i].MaxWidth > 0 && tv.columns[i].Width > tv.columns[i].MaxWidth {
					tv.columns[i].Width = tv.columns[i].MaxWidth
				}
			}
		}
	}
}

func (tv *TableView) renderHeaders() string {
	parts := []string{}

	// Selection column
	if tv.multiSelect {
		parts = append(parts, " ✓ ")
	}

	// Data columns
	for i, column := range tv.columns {
		header := column.Header

		// Add sort indicator
		if tv.sortColumn == i {
			if tv.sortAsc {
				header += " ↑"
			} else {
				header += " ↓"
			}
		}

		// Format with width and alignment
		formatted := tv.formatCellContent(header, column.Width, column.Alignment)
		parts = append(parts, formatted)
	}

	line := strings.Join(parts, "│")
	return theme.RenderTextBold(line)
}

func (tv *TableView) renderRow(row TableRow, selected, focused bool) string {
	parts := []string{}

	// Selection indicator
	if tv.multiSelect {
		indicator := " "
		if selected {
			indicator = theme.RenderIcon("success")
		} else {
			indicator = theme.RenderIcon("inactive")
		}
		parts = append(parts, " "+indicator+" ")
	}

	// Data columns
	for _, column := range tv.columns {
		value := row.Data[column.Key]
		cellContent := tv.formatCellValue(value, column)
		formatted := tv.formatCellContent(cellContent, column.Width, column.Alignment)
		parts = append(parts, formatted)
	}

	content := strings.Join(parts, "│")

	// Apply styling
	if row.Disabled {
		return theme.RenderTextDim(content)
	} else if focused {
		return theme.RenderSelection(content, tv.width)
	} else if selected {
		return theme.RenderTextBold(content)
	}

	return theme.RenderText(content)
}

func (tv *TableView) formatCellValue(value interface{}, column TableColumn) string {
	if column.Format != nil {
		return column.Format(value)
	}

	if value == nil {
		return ""
	}

	return fmt.Sprintf("%v", value)
}

func (tv *TableView) formatCellContent(content string, width int, alignment string) string {
	if len(content) > width {
		content = content[:width-1] + "…"
	}

	switch alignment {
	case "center":
		padding := width - len(content)
		leftPad := padding / 2
		rightPad := padding - leftPad
		return strings.Repeat(" ", leftPad) + content + strings.Repeat(" ", rightPad)
	case "right":
		padding := width - len(content)
		return strings.Repeat(" ", padding) + content
	default: // "left"
		padding := width - len(content)
		return content + strings.Repeat(" ", padding)
	}
}

func (tv *TableView) renderFooter() string {
	helpItems := []string{}

	if tv.multiSelect {
		helpItems = append(helpItems, "[Space] Select")
	}

	helpItems = append(helpItems, "[Enter] Choose")

	if len(tv.columns) > 0 {
		helpItems = append(helpItems, "[S] Sort")
	}

	if tv.multiSelect {
		helpItems = append(helpItems, "[A] All", "[C] Clear")
	}

	helpItems = append(helpItems, "[Esc] Back")

	return theme.RenderHelpBar(helpItems, tv.width)
}

func (tv *TableView) padToHeight(lines []string) string {
	// Pad with empty lines to reach desired height
	for len(lines) < tv.height {
		lines = append(lines, "")
	}

	// Truncate if too many lines
	if len(lines) > tv.height {
		lines = lines[:tv.height]
	}

	return strings.Join(lines, "\n")
}
