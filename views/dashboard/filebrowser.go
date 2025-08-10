package dashboard

import (
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
	"github.com/ctakes-tui/ctakes-tui/internal/utils"
)

// updateFileList now returns a tea.Cmd for async loading
func (m *Model) updateFileList() tea.Cmd {
	// Clear any existing error
	m.err = nil

	// Set loading state
	m.isLoadingDir = true

	// Return async load command
	m.dirRequestID++
	return LoadDirectoryAsync(m.currentPath, m.currentDirPage, m.dirRequestID)
}

// updateFileListSync is the synchronous fallback (for compatibility)
func (m *Model) updateFileListSync() {
	// Use cached data if available
	if cached := getCachedDirectory(m.currentPath); cached != nil && !isCacheExpired(cached) {
		m.files = cached.Files
		m.err = cached.Error
		m.totalDirItems = cached.TotalCount
		m.isLoadingDir = false

		// Fix cursor bounds
		if m.fileTable.Cursor() >= len(m.files) && len(m.files) > 0 {
			m.fileTable.SetCursor(len(m.files) - 1)
		}
		return
	}

	// Fallback to synchronous load (should rarely happen)
	files, err := loadDirectoryWithContext(nil, m.currentPath)
	if err != nil {
		m.err = err
		m.files = []FileInfo{}
	} else {
		m.files = getPaginatedFiles(files, m.currentDirPage)
		m.totalDirItems = len(files)
	}
	m.isLoadingDir = false

	// Cache the result
	cacheDirectory(m.currentPath, m.files, m.err, m.totalDirItems)
}

// updateDirList populates m.files with directories only (async version)
func (m *Model) updateDirList() tea.Cmd {
	m.isLoadingDir = true
	m.dirRequestID++
	return LoadDirectoryAsync(m.currentPath, m.currentDirPage, m.dirRequestID)
}

// updateDirListSync is the synchronous fallback for directory-only listing
func (m *Model) updateDirListSync() {
	// Check cache first
	if cached := getCachedDirectory(m.currentPath); cached != nil && !isCacheExpired(cached) {
		// Filter to directories only
		m.files = filterDirectories(cached.Files)
		m.err = cached.Error
		m.isLoadingDir = false
		return
	}

	// Load and filter directories
	files, err := loadDirectoryWithContext(nil, m.currentPath)
	if err != nil {
		m.err = err
		m.files = []FileInfo{}
	} else {
		m.files = filterDirectories(files)
	}
	m.isLoadingDir = false

	// Cache the full result
	cacheDirectory(m.currentPath, files, err, len(files))
}

// filterDirectories returns only directories from a file list
func filterDirectories(files []FileInfo) []FileInfo {
	dirs := []FileInfo{}
	for _, f := range files {
		if f.IsDir {
			dirs = append(dirs, f)
		}
	}
	return dirs
}

func (m *Model) handleFileAction() tea.Cmd {
	// Validate cursor bounds
	if m.fileTable.Cursor() < 0 || m.fileTable.Cursor() >= len(m.files) {
		return nil
	}

	selected := m.files[m.fileTable.Cursor()]
	// Toggle selection on Space key is handled in Update; this is Enter action
	if selected.IsDir {
		// Clear cache for the old path if navigating away
		oldPath := m.currentPath

		if selected.Name == ".." {
			m.currentPath = filepath.Dir(m.currentPath)
		} else {
			m.currentPath = filepath.Join(m.currentPath, selected.Name)
		}

		// Reset page when changing directories
		m.currentDirPage = 0

		// Clear old path cache if it's getting stale
		if oldPath != m.currentPath {
			go func() {
				ClearCache(oldPath)
			}()
		}

		// Update tables after navigation
		cmd := m.updateFileList()

		// Force table update to reflect new directory contents
		go func() {
			m.updateTables()
		}()

		return cmd
	} else {
		m.loadFilePreview(selected)
		// Update preview panel
		m.showPreview = true
		m.updateTables()
	}
	return nil
}

// Toggle selection for current row
func (m *Model) toggleSelection() {
	if m.fileTable.Cursor() < 0 || m.fileTable.Cursor() >= len(m.files) {
		return
	}
	idx := m.fileTable.Cursor()
	// Do not toggle ".." pseudo-entry
	if m.files[idx].Name == ".." && m.files[idx].IsDir {
		return
	}
	m.files[idx].Selected = !m.files[idx].Selected
	m.updateTables()
}

// getSelectedFiles returns all currently selected files
func (m *Model) getSelectedFiles() []FileInfo {
	selected := []FileInfo{}
	for _, file := range m.files {
		if file.Selected && file.Name != ".." {
			selected = append(selected, file)
		}
	}
	return selected
}

// clearAllSelections clears all file selections
func (m *Model) clearAllSelections() {
	for i := range m.files {
		m.files[i].Selected = false
	}
	m.updateTables()
}

// isGoFile checks if a file is a Go source file
func isGoFile(filename string) bool {
	return filepath.Ext(filename) == ".go"
}

// filterGoFiles returns only Go files from the file list
func (m *Model) filterGoFiles() []FileInfo {
	goFiles := []FileInfo{}
	for _, file := range m.files {
		if !file.IsDir && isGoFile(file.Name) {
			goFiles = append(goFiles, file)
		}
	}
	return goFiles
}

func (m *Model) renderFileBrowser(width, height int) string {
	// Ensure minimum dimensions
	width = utils.Max(30, width)
	height = utils.Max(5, height)

	// Show loading indicator if loading
	if m.isLoadingDir {
		// Use consistent header styling
		path := utils.TruncateString(m.currentPath, width-theme.SpacingSM*2)
		header := theme.RenderHeader(path, width)

		loadingView := RenderLoadingIndicator(m.spinner, width)

		return lipgloss.JoinVertical(lipgloss.Left, header, loadingView)
	}

	if m.err != nil {
		// Use consistent error styling with semantic icon
		return theme.RenderStatusMessage("error", fmt.Sprintf("Error: %v", m.err))
	}

	// Build path with consistent styling
	path := utils.TruncateString(m.currentPath, width-theme.SpacingSM*2)

	// Add pagination info if there are multiple pages
	if m.totalDirItems > MaxItemsPerPage {
		totalPages, _ := GetPageInfo(m.totalDirItems, m.currentDirPage)
		path = fmt.Sprintf("%s (Page %d/%d, %d items)",
			utils.TruncateString(m.currentPath, width-30),
			m.currentDirPage+1, totalPages, m.totalDirItems)
	}

	// Use consistent header styling
	header := theme.RenderHeader(path, width)

	// Apply consistent spacing for table
	tableHeight := height - theme.HeightHeader - theme.SpacingXS
	m.fileTable.SetHeight(tableHeight)
	m.fileTable.SetWidth(width)

	content := m.fileTable.View()

	// Apply consistent content padding
	return theme.ContentCompactStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, header, content),
	)
}

func (m *Model) renderFileBrowserCompact(width, height int) string {
	// Ensure minimum dimensions
	width = utils.Max(30, width)
	height = utils.Max(5, height)

	// Show loading indicator if loading
	if m.isLoadingDir {
		pathStyle := lipgloss.NewStyle().
			Foreground(theme.ColorSecondary).
			Bold(true).
			MaxWidth(width).
			Width(width)

		path := utils.TruncateString(m.currentPath, width)
		header := pathStyle.Render(path)

		loadingView := RenderLoadingIndicator(m.spinner, width)

		result := lipgloss.JoinVertical(lipgloss.Left, header, loadingView)
		return utils.ConstrainBox(result, width, height)
	}

	if m.err != nil {
		errMsg := utils.TruncateString(fmt.Sprintf("Error: %v", m.err), width)
		return theme.ErrorStyle.MaxWidth(width).Render(errMsg)
	}

	// Path header with strict width constraint
	pathStyle := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Bold(true).
		MaxWidth(width).
		Width(width)

	path := utils.TruncateString(m.currentPath, width)

	// Add page info for large directories
	if m.totalDirItems > MaxItemsPerPage {
		path = fmt.Sprintf("%s [%d/%d]",
			utils.TruncateString(m.currentPath, width-10),
			m.currentDirPage+1, (m.totalDirItems+MaxItemsPerPage-1)/MaxItemsPerPage)
	}

	header := pathStyle.Render(path)

	// Use the table widget for proper scrolling
	tableHeight := utils.Max(2, height-2) // Account for header
	tableWidth := utils.Max(20, width)

	m.fileTable.SetHeight(tableHeight)
	m.fileTable.SetWidth(tableWidth)

	// Render the table view which handles scrolling internally
	tableView := m.fileTable.View()

	// Ensure table view doesn't exceed width
	tableView = utils.SafeRender(tableView, width)

	result := lipgloss.JoinVertical(lipgloss.Left, header, tableView)
	return utils.ConstrainBox(result, width, height)
}

// handleFileBrowserPagination handles page navigation
func (m *Model) handleFileBrowserPagination(forward bool) tea.Cmd {
	if m.totalDirItems <= MaxItemsPerPage {
		return nil // No pagination needed
	}

	totalPages := (m.totalDirItems + MaxItemsPerPage - 1) / MaxItemsPerPage

	if forward {
		if m.currentDirPage < totalPages-1 {
			m.currentDirPage++
			m.dirRequestID++
			return LoadDirectoryPage(m.currentPath, m.currentDirPage, m.dirRequestID)
		}
	} else {
		if m.currentDirPage > 0 {
			m.currentDirPage--
			m.dirRequestID++
			return LoadDirectoryPage(m.currentPath, m.currentDirPage, m.dirRequestID)
		}
	}

	return nil
}
