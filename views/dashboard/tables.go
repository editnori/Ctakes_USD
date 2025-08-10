package dashboard

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
)

func (m *Model) updateTables() {
	// Calculate available space more accurately
	sidebarWidth := m.width / 4
	remainingWidth := m.width - sidebarWidth - 2

	if m.showPreview {
		mainWidth := (remainingWidth * 3) / 5
		m.updateTablesCompact(mainWidth-4, m.height-8)
	} else {
		m.updateTablesCompact(remainingWidth-4, m.height-8)
	}
}

func (m *Model) updateTablesCompact(availableWidth, tableHeight int) {
	columns := []table.Column{
		{Title: "", Width: 2},
		{Title: "Name", Width: availableWidth / 2},
		{Title: "Size", Width: 10},
		{Title: "Modified", Width: 15},
	}

	if availableWidth < 60 {
		columns = []table.Column{
			{Title: "", Width: 2},
			{Title: "Name", Width: availableWidth - 15},
			{Title: "Size", Width: 10},
		}
	}

	rows := []table.Row{}
	for _, file := range m.files {
		// Add selection indicator to the icon
		icon := file.Icon
		if file.Selected {
			icon = "✓" // Checkmark for selected files
		}

		if availableWidth < 60 {
			rows = append(rows, table.Row{
				icon,
				file.Name,
				file.Size,
			})
		} else {
			rows = append(rows, table.Row{
				icon,
				file.Name,
				file.Size,
				file.ModTime,
			})
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(m.activePanel == MainPanel),
		table.WithHeight(tableHeight),
		table.WithWidth(availableWidth),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.ColorBorder).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(theme.ColorBackground).
		Background(theme.ColorAccent).
		Bold(false)
	t.SetStyles(s)

	m.fileTable = t
}
