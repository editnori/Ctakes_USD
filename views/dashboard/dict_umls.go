package dashboard

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/dictionary"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
	"github.com/ctakes-tui/ctakes-tui/internal/utils"
)

// UMLS directory selection
func (m *Model) renderUMLSSelector(width, height int) string {
	// Ensure sane minimums and clamp inner width
	width = utils.Max(30, width)
	height = utils.Max(8, height)

	header := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorAccent).MaxWidth(width - 2).Render("Select UMLS Directory")
	help := lipgloss.NewStyle().Foreground(theme.ColorInfo).MaxWidth(width - 2).Render("Enter: Open • Space: Select • Backspace/←: Up • ESC: Back")
	curPath := utils.TruncateString("Path: "+m.currentPath, width-2)

	lines := []string{
		header,
		help,
		"",
		curPath,
		"",
	}

	// Show loading indicator if loading
	if m.isLoadingDir {
		lines = append(lines, RenderLoadingIndicator(m.spinner, width))
		return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
	}

	// Optional hint if current directory clearly has RRFs (fast path)
	if cached := m.detectRRFFilesCached(m.currentPath); len(cached) > 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render(fmt.Sprintf("Found %d RRF files in current directory", len(cached))))
		lines = append(lines, "")
	}

	// Render simple list from m.files with scrolling
	maxRows := utils.Max(3, height-10)
	cursor := m.fileTable.Cursor()

	// Calculate the scroll window
	startIdx := 0
	if cursor >= maxRows {
		startIdx = cursor - maxRows + 1
	}

	// Ensure startIdx doesn't go negative
	if startIdx < 0 {
		startIdx = 0
	}

	// Calculate end index
	endIdx := startIdx + maxRows
	if endIdx > len(m.files) {
		endIdx = len(m.files)
	}

	// Render visible items (avoid expensive per-row RRF scanning here)
	for i := startIdx; i < endIdx; i++ {
		f := m.files[i]
		name := f.Name
		if f.IsDir {
			name = utils.TruncateString(""+utils.GetIcon("folder")+" "+name, width-2)
		} else {
			name = utils.TruncateString(""+utils.GetIcon("file")+" "+name, width-2)
		}
		if i == m.fileTable.Cursor() {
			name = lipgloss.NewStyle().Background(theme.ColorSelectionActive).Foreground(theme.ColorBackground).Render(name)
		}
		lines = append(lines, utils.TruncateString(name, width-2))
	}
	lines = append(lines, "", utils.TruncateString("ESC: Cancel", width-2))

	content := strings.Join(lines, "\n")
	// Constrain to panel size to prevent overflow/wrap
	return lipgloss.NewStyle().Width(width).Height(height).Render(utils.ConstrainBox(content, width, height))
}

func (m *Model) handleUMLSKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k", "down", "j":
		// Validate cursor bounds before updating
		if len(m.files) > 0 {
			m.fileTable, _ = m.fileTable.Update(msg)
			// Ensure cursor stays in bounds
			if m.fileTable.Cursor() >= len(m.files) {
				m.fileTable.SetCursor(len(m.files) - 1)
			}
		}
	case "pgup":
		if cmd := m.handleFileBrowserPagination(false); cmd != nil {
			return *m, cmd
		}
	case "pgdown":
		if cmd := m.handleFileBrowserPagination(true); cmd != nil {
			return *m, cmd
		}
	case "enter", "right", "l":
		if m.fileTable.Cursor() >= 0 && m.fileTable.Cursor() < len(m.files) {
			sel := m.files[m.fileTable.Cursor()]
			if sel.IsDir {
				var target string
				if sel.Name == ".." {
					target = filepath.Dir(m.currentPath)
				} else {
					target = filepath.Join(m.currentPath, sel.Name)
				}
				// Navigate into the directory
				m.currentPath = target
				m.currentDirPage = 0 // Reset page
				return *m, m.updateFileListUMLS()
			}
		}
	case "backspace", "left", "h":
		m.currentPath = filepath.Dir(m.currentPath)
		m.currentDirPage = 0 // Reset page
		return *m, m.updateFileListUMLS()
	case "s", "S", " ", "space":
		// Try to create RRF parser to validate the UMLS directory
		if m.validateUMLSDirectory(m.currentPath) {
			m.umlsPath = m.currentPath
			m.rrfFiles = m.detectRRFFilesCached(m.currentPath)
			m.dictBuilderState = DictStateMainMenu
		}
	case "esc":
		m.dictBuilderState = DictStateMainMenu
	case "q", "Q":
		// Quit dictionary builder - return to main dashboard
		m.activePanel = SidebarPanel
		m.cursor = 0
	case "p", "P":
		// Switch to pipeline view
		m.activePanel = SidebarPanel
		m.cursor = 5 // Pipeline is item 5 in the sidebar (0-indexed)
	}
	return *m, nil
}

// validateUMLSDirectory validates that a directory contains proper UMLS files using RRF parser
func (m *Model) validateUMLSDirectory(path string) bool {
	// First check for META subdirectory (standard UMLS layout)
	metaPath := filepath.Join(path, "META")
	if _, err := os.Stat(metaPath); err == nil {
		// Try to create RRF parser for META directory
		if parser, err := dictionary.NewRRFParser(metaPath); err == nil {
			// Validate that required files are present
			files := parser.GetAvailableFiles()
			return len(files) >= 2 // At least MRCONSO.RRF and MRSTY.RRF
		}
	}

	// If no META directory, check if current directory has RRF files
	if parser, err := dictionary.NewRRFParser(path); err == nil {
		files := parser.GetAvailableFiles()
		return len(files) >= 2
	}

	return false
}

// Helper function to detect RRF files in a directory using the parser
func detectRRFFiles(path string) []string {
	// Check META subdirectory first
	metaPath := filepath.Join(path, "META")
	if parser, err := dictionary.NewRRFParser(metaPath); err == nil {
		return parser.GetAvailableFiles()
	}

	// If no META, check current directory
	if parser, err := dictionary.NewRRFParser(path); err == nil {
		return parser.GetAvailableFiles()
	}

	// Fallback to simple file listing
	var out []string
	if entries, err := os.ReadDir(path); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(strings.ToUpper(e.Name()), ".RRF") {
				out = append(out, e.Name())
			}
		}
	}
	return out
}

// Cached detection to reduce lag when browsing large folders
func (m *Model) detectRRFFilesCached(path string) []string {
	if m.rrfCache == nil {
		m.rrfCache = make(map[string][]string)
	}
	if v, ok := m.rrfCache[path]; ok {
		return v
	}
	res := detectRRFFiles(path)
	// Cache the result; keep it lightweight, no invalidation needed for loader use
	m.rrfCache[path] = res
	return res
}

// updateFileListUMLS uses async loading for UMLS directory selection
func (m *Model) updateFileListUMLS() tea.Cmd {
	m.isLoadingDir = true
	m.dirRequestID++
	return LoadDirectoryAsync(m.currentPath, 0, m.dirRequestID)
}

// updateFileListUMLSSync is the synchronous fallback
func (m *Model) updateFileListUMLSSync() {
	// Check cache first
	if cached := getCachedDirectory(m.currentPath); cached != nil && !isCacheExpired(cached) {
		// Filter to directories only
		m.files = filterDirectories(cached.Files)
		m.err = cached.Error
		m.isLoadingDir = false

		// Keep cursor in-bounds
		if m.fileTable.Cursor() >= len(m.files) && len(m.files) > 0 {
			m.fileTable.SetCursor(len(m.files) - 1)
		}
		return
	}

	// Load directories with timeout protection
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	files, err := loadDirectoryWithContext(ctx, m.currentPath)
	if err != nil {
		m.err = err
		m.files = []FileInfo{}
	} else {
		m.files = filterDirectories(files)
	}
	m.isLoadingDir = false

	// Keep cursor in-bounds
	if m.fileTable.Cursor() >= len(m.files) && len(m.files) > 0 {
		m.fileTable.SetCursor(len(m.files) - 1)
	}

	// Cache the result
	cacheDirectory(m.currentPath, files, err, len(files))
}
