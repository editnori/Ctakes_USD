package views

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
	"github.com/ctakes-tui/ctakes-tui/internal/utils"
)

type DocumentModel struct {
	width       int
	height      int
	currentPath string
	files       []DocumentFile
	selected    map[string]bool
	fileTable   table.Model
	ready       bool
	showPreview bool
	previewText string
	err         error
}

type DocumentFile struct {
	Path     string
	Name     string
	Size     string
	Modified string
	IsDir    bool
	Icon     string
	Selected bool
}

func NewDocumentModel() DocumentModel {
	m := DocumentModel{
		currentPath: ".",
		selected:    make(map[string]bool),
	}
	m.loadFiles()
	return m
}

func (m DocumentModel) Init() tea.Cmd {
	return nil
}

func (m DocumentModel) Update(msg tea.Msg) (DocumentModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.updateTable()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, docKeys.Back):
			return m, func() tea.Msg { return "main_menu" }

		case key.Matches(msg, docKeys.SelectAll):
			for i := range m.files {
				if !m.files[i].IsDir {
					m.files[i].Selected = true
					m.selected[m.files[i].Path] = true
				}
			}
			m.updateTable()

		case key.Matches(msg, docKeys.SelectNone):
			for i := range m.files {
				m.files[i].Selected = false
				delete(m.selected, m.files[i].Path)
			}
			m.updateTable()

		case key.Matches(msg, docKeys.ToggleSelect):
			if m.fileTable.Cursor() < len(m.files) {
				idx := m.fileTable.Cursor()
				m.files[idx].Selected = !m.files[idx].Selected
				if m.files[idx].Selected {
					m.selected[m.files[idx].Path] = true
				} else {
					delete(m.selected, m.files[idx].Path)
				}
				m.updateTable()
			}

		case key.Matches(msg, docKeys.Enter):
			if m.fileTable.Cursor() < len(m.files) {
				file := m.files[m.fileTable.Cursor()]
				if file.IsDir {
					m.currentPath = file.Path
					m.loadFiles()
				} else {
					m.files[m.fileTable.Cursor()].Selected = !m.files[m.fileTable.Cursor()].Selected
					if m.files[m.fileTable.Cursor()].Selected {
						m.selected[file.Path] = true
					} else {
						delete(m.selected, file.Path)
					}
					m.updateTable()
				}
			}

		case key.Matches(msg, docKeys.Process):
			if len(m.selected) > 0 {
				return m, m.processDocuments()
			}

		case key.Matches(msg, docKeys.Preview):
			m.showPreview = !m.showPreview
			if m.showPreview && m.fileTable.Cursor() < len(m.files) {
				m.loadPreview(m.files[m.fileTable.Cursor()].Path)
			}

		case key.Matches(msg, docKeys.Up):
			if m.fileTable.Cursor() > 0 {
				newTable, cmd := m.fileTable.Update(msg)
				m.fileTable = newTable
				if m.showPreview && m.fileTable.Cursor() < len(m.files) {
					m.loadPreview(m.files[m.fileTable.Cursor()].Path)
				}
				return m, cmd
			}

		case key.Matches(msg, docKeys.Down):
			if m.fileTable.Cursor() < len(m.files)-1 {
				newTable, cmd := m.fileTable.Update(msg)
				m.fileTable = newTable
				if m.showPreview && m.fileTable.Cursor() < len(m.files) {
					m.loadPreview(m.files[m.fileTable.Cursor()].Path)
				}
				return m, cmd
			}
		}

		newTable, cmd := m.fileTable.Update(msg)
		m.fileTable = newTable
		return m, cmd
	}

	return m, cmd
}

func (m DocumentModel) View() string {
	if !m.ready {
		return theme.BaseStyle.Render("Loading documents...")
	}

	if m.showPreview {
		return m.renderSplitView()
	}
	return m.renderFullView()
}

func (m *DocumentModel) renderFullView() string {
	topPadding := "\n\n"
	header := m.renderHeader()
	content := m.renderFileList(m.width-4, m.height-12)
	footer := m.renderFooter()

	main := lipgloss.JoinVertical(
		lipgloss.Left,
		topPadding,
		header,
		content,
		footer,
	)

	return theme.BaseStyle.
		Width(m.width).
		Height(m.height).
		Render(main)
}

func (m *DocumentModel) renderSplitView() string {
	topPadding := "\n\n"
	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth - 2

	header := m.renderHeader()

	leftPanel := m.renderFileList(leftWidth-2, m.height-12)
	rightPanel := m.renderPreview(rightWidth-2, m.height-12)

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanel,
		lipgloss.NewStyle().Width(2).Render("  "),
		rightPanel,
	)

	footer := m.renderFooter()

	main := lipgloss.JoinVertical(
		lipgloss.Left,
		topPadding,
		header,
		content,
		footer,
	)

	return theme.BaseStyle.
		Width(m.width).
		Height(m.height).
		Render(main)
}

func (m *DocumentModel) renderHeader() string {
	title := theme.RenderTitle(theme.IconDocument, "Document Processor")

	selectedCount := len(m.selected)
	status := fmt.Sprintf("%d files selected", selectedCount)
	if selectedCount == 0 {
		status = "No files selected"
	}

	statusBar := lipgloss.NewStyle().
		Foreground(theme.ColorForegroundDim).
		Render(status)

	path := lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Render(fmt.Sprintf("%s %s", theme.IconFolder, m.currentPath))

	header := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			path,
			strings.Repeat(" ", m.width-lipgloss.Width(path)-lipgloss.Width(statusBar)-4),
			statusBar,
		),
		strings.Repeat(theme.BorderDividerH, m.width-4),
	)

	return header
}

func (m *DocumentModel) renderFileList(width, height int) string {
	if len(m.files) == 0 {
		empty := lipgloss.NewStyle().
			Foreground(theme.ColorForegroundDim).
			Italic(true).
			Align(lipgloss.Center).
			Width(width).
			Height(height).
			Render("No clinical documents found in this directory")

		return theme.PanelStyle.
			Width(width).
			Height(height).
			Render(empty)
	}

	return theme.PanelActiveStyle.
		Width(width).
		Height(height).
		Render(m.fileTable.View())
}

func (m *DocumentModel) renderPreview(width, height int) string {
	header := theme.RenderTitle(theme.IconFile, "Preview")

	content := m.previewText
	if content == "" {
		content = lipgloss.NewStyle().
			Foreground(theme.ColorForegroundDim).
			Italic(true).
			Render("Select a file to preview")
	}

	preview := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		strings.Repeat(theme.BorderDividerH, width-4),
		"",
		content,
	)

	return theme.PanelStyle.
		Width(width).
		Height(height).
		Render(preview)
}

func (m *DocumentModel) renderFooter() string {
	keys := []string{
		theme.RenderKeyHelp("↑↓", "Navigate"),
		theme.RenderKeyHelp("Space", "Select"),
		theme.RenderKeyHelp("a", "Select All"),
		theme.RenderKeyHelp("n", "Select None"),
		theme.RenderKeyHelp("p", "Preview"),
		theme.RenderKeyHelp("Enter", "Process"),
		theme.RenderKeyHelp("Esc", "Back"),
	}

	return theme.FooterStyle.
		Width(m.width).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, keys...))
}

func (m *DocumentModel) loadFiles() {
	m.files = []DocumentFile{}

	// Add parent dir
	if m.currentPath != "/" && m.currentPath != "." {
		m.files = append(m.files, DocumentFile{
			Path:  filepath.Dir(m.currentPath),
			Name:  "..",
			Icon:  theme.IconArrowUp,
			IsDir: true,
		})
	}

	entries, err := os.ReadDir(m.currentPath)
	if err != nil {
		m.err = err
		return
	}

	var dirs, docs []DocumentFile

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		fullPath := filepath.Join(m.currentPath, entry.Name())

		file := DocumentFile{
			Path:     fullPath,
			Name:     entry.Name(),
			Modified: info.ModTime().Format("2006-01-02 15:04"),
			Selected: m.selected[fullPath],
		}

		if entry.IsDir() {
			file.IsDir = true
			file.Icon = theme.IconFolder
			file.Size = "-"
			dirs = append(dirs, file)
		} else if isClinicalDocument(entry.Name()) {
			file.Icon = utils.GetFileIcon(entry.Name(), false)
			file.Size = utils.FormatFileSize(info.Size())
			docs = append(docs, file)
		}
	}

	// Sort entries
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	sort.Slice(docs, func(i, j int) bool {
		return strings.ToLower(docs[i].Name) < strings.ToLower(docs[j].Name)
	})

	m.files = append(m.files, dirs...)
	m.files = append(m.files, docs...)

	m.updateTable()
}

func (m *DocumentModel) updateTable() {
	columns := []table.Column{
		{Title: "", Width: 2},
		{Title: "", Width: 2},
		{Title: "Name", Width: m.width/2 - 20},
		{Title: "Size", Width: 10},
		{Title: "Modified", Width: 16},
	}

	var rows []table.Row
	for _, f := range m.files {
		checkbox := theme.IconCheck
		if !f.Selected {
			checkbox = " "
		}
		if f.IsDir {
			checkbox = ""
		}

		rows = append(rows, table.Row{
			checkbox,
			f.Icon,
			f.Name,
			f.Size,
			f.Modified,
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(m.height-14),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.ColorBorderInactive).
		BorderBottom(true).
		Bold(true).
		Foreground(theme.ColorForegroundDim)
	// Use centralized theme selection styles for consistency
	s.Selected = theme.RowFocusedStyle
	s.Cell = s.Cell.
		Foreground(theme.ColorForeground)

	t.SetStyles(s)
	m.fileTable = t
}

func (m *DocumentModel) loadPreview(path string) {
	if fs.ValidPath(path) {
		content, err := os.ReadFile(path)
		if err != nil {
			m.previewText = fmt.Sprintf("Error reading file: %v", err)
			return
		}

		// Limit to 1000 chars
		preview := string(content)
		if len(preview) > 1000 {
			preview = preview[:1000] + "\n\n... (truncated)"
		}

		m.previewText = preview
	}
}

func (m *DocumentModel) processDocuments() tea.Cmd {
	return func() tea.Msg {
		// Trigger cTAKES
		var files []string
		for path := range m.selected {
			files = append(files, path)
		}
		// Process with cTAKES
		return "results_view"
	}
}

func isClinicalDocument(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	clinicalExts := []string{".txt", ".doc", ".docx", ".pdf", ".rtf", ".xml", ".hl7", ".cda"}
	for _, validExt := range clinicalExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

type docKeyMap struct {
	Up           key.Binding
	Down         key.Binding
	Enter        key.Binding
	Back         key.Binding
	SelectAll    key.Binding
	SelectNone   key.Binding
	ToggleSelect key.Binding
	Process      key.Binding
	Preview      key.Binding
}

var docKeys = docKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "process selected"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("esc", "back"),
	),
	SelectAll: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "select all"),
	),
	SelectNone: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "select none"),
	),
	ToggleSelect: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle selection"),
	),
	Process: key.NewBinding(
		key.WithKeys("ctrl+p"),
		key.WithHelp("ctrl+p", "process documents"),
	),
	Preview: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "toggle preview"),
	),
}
