package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
	"github.com/ctakes-tui/ctakes-tui/internal/utils"
)

func (m *Model) updateFileList() {
	files, err := os.ReadDir(m.currentPath)
	if err != nil {
		m.err = err
		return
	}

	m.files = []FileInfo{}
	m.err = nil

	// Add parent directory navigation
	if m.currentPath != "/" && m.currentPath != "" {
		m.files = append(m.files, FileInfo{
			Name:    "..",
			Size:    "",
			Mode:    "drwxr-xr-x",
			ModTime: "",
			IsDir:   true,
			Icon:    "⬆",
		})
	}

	// Process all files including hidden ones
	for _, file := range files {
		// Get file info - if error, still add the file with basic info
		info, err := file.Info()

		var fileInfo FileInfo
		if err != nil {
			// Still add the file even if we can't get full info
			fileInfo = FileInfo{
				Name:    file.Name(),
				Size:    "?",
				Mode:    "?",
				ModTime: "?",
				IsDir:   file.IsDir(),
				Icon:    utils.GetFileIcon(file.Name(), file.IsDir()),
			}
		} else {
			fileInfo = FileInfo{
				Name:    file.Name(),
				Size:    utils.FormatFileSize(info.Size()),
				Mode:    info.Mode().String(),
				ModTime: info.ModTime().Format("Jan 02 15:04"),
				IsDir:   file.IsDir(),
				Icon:    utils.GetFileIcon(file.Name(), file.IsDir()),
			}

			if file.IsDir() {
				fileInfo.Size = "-"
			}
		}

		m.files = append(m.files, fileInfo)
	}

	// Sort: directories first, then alphabetically
	sort.Slice(m.files[1:], func(i, j int) bool {
		if m.files[i+1].IsDir != m.files[j+1].IsDir {
			return m.files[i+1].IsDir
		}
		return strings.ToLower(m.files[i+1].Name) < strings.ToLower(m.files[j+1].Name)
	})
}

func (m *Model) handleFileAction() tea.Cmd {
	if m.fileTable.Cursor() >= 0 && m.fileTable.Cursor() < len(m.files) {
		selected := m.files[m.fileTable.Cursor()]
		if selected.IsDir {
			if selected.Name == ".." {
				m.currentPath = filepath.Dir(m.currentPath)
			} else {
				m.currentPath = filepath.Join(m.currentPath, selected.Name)
			}
			m.updateFileList()
			m.updateTables()
		} else {
			m.loadFilePreview(selected)
		}
	}
	return nil
}

func (m *Model) renderFileBrowser(width, height int) string {
	if m.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	pathStyle := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Bold(true)

	path := utils.TruncateString(m.currentPath, width-2)
	header := pathStyle.Render(path)

	tableHeight := height - 3
	m.fileTable.SetHeight(tableHeight)
	m.fileTable.SetWidth(width)

	content := m.fileTable.View()

	return lipgloss.JoinVertical(lipgloss.Left, header, content)
}

func (m *Model) renderFileBrowserCompact(width, height int) string {
	if m.err != nil {
		return theme.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	// Path header
	pathStyle := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Bold(true).
		MaxWidth(width)

	path := utils.TruncateString(m.currentPath, width)
	header := pathStyle.Render(path)

	// Use the table widget for proper scrolling
	tableHeight := height - 2 // Account for header
	m.fileTable.SetHeight(tableHeight)

	// Render the table view which handles scrolling internally
	tableView := m.fileTable.View()

	return lipgloss.JoinVertical(lipgloss.Left, header, tableView)
}
