package components

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ctakes-tui/ctakes-tui/internal/theme"
	"github.com/ctakes-tui/ctakes-tui/internal/utils"
)

// FileBrowser - Reusable async file browser component
type FileBrowser struct {
	// Current state
	currentPath   string
	entries       []FileEntry
	cursor        int
	selectedFiles map[string]bool

	// Configuration
	multiSelect     bool
	directoriesOnly bool
	showHidden      bool
	sortBy          SortMode

	// Display
	width      int
	height     int
	startIndex int

	// Callbacks
	onSelect      func(path string)
	onMultiSelect func(paths []string)

	// Async loading
	loading   bool
	loadError error
}

type FileEntry struct {
	Name       string
	Path       string
	IsDir      bool
	Size       int64
	ModTime    time.Time
	Permission os.FileMode
}

type SortMode int

const (
	SortByName SortMode = iota
	SortBySize
	SortByModTime
	SortByType
)

// NewFileBrowser creates a new file browser component
func NewFileBrowser(path string, multiSelect bool) *FileBrowser {
	fb := &FileBrowser{
		currentPath:   path,
		multiSelect:   multiSelect,
		selectedFiles: make(map[string]bool),
		sortBy:        SortByName,
		width:         80,
		height:        20,
	}

	// Load initial directory
	fb.LoadDirectory(path)

	return fb
}

// Configure the file browser
func (fb *FileBrowser) SetSize(width, height int) *FileBrowser {
	fb.width = width
	fb.height = height
	return fb
}

func (fb *FileBrowser) SetDirectoriesOnly(dirOnly bool) *FileBrowser {
	fb.directoriesOnly = dirOnly
	return fb
}

func (fb *FileBrowser) SetShowHidden(show bool) *FileBrowser {
	fb.showHidden = show
	fb.LoadDirectory(fb.currentPath) // Reload with new setting
	return fb
}

func (fb *FileBrowser) SetSortMode(mode SortMode) *FileBrowser {
	fb.sortBy = mode
	fb.sortEntries()
	return fb
}

func (fb *FileBrowser) OnSelect(callback func(path string)) *FileBrowser {
	fb.onSelect = callback
	return fb
}

func (fb *FileBrowser) OnMultiSelect(callback func(paths []string)) *FileBrowser {
	fb.onMultiSelect = callback
	return fb
}

// LoadDirectory loads a directory asynchronously
func (fb *FileBrowser) LoadDirectory(path string) error {
	fb.loading = true
	fb.loadError = nil
	fb.currentPath = path

	// Load directory contents
	entries, err := os.ReadDir(path)
	if err != nil {
		fb.loading = false
		fb.loadError = err
		return err
	}

	// Convert to FileEntry structs
	fb.entries = make([]FileEntry, 0, len(entries))

	// Add parent directory entry if not at root
	if path != "/" && path != "." {
		parent := filepath.Dir(path)
		fb.entries = append(fb.entries, FileEntry{
			Name:  "..",
			Path:  parent,
			IsDir: true,
		})
	}

	for _, entry := range entries {
		// Skip hidden files if not showing them
		if !fb.showHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Skip files if directories only
		if fb.directoriesOnly && !info.IsDir() {
			continue
		}

		fb.entries = append(fb.entries, FileEntry{
			Name:       entry.Name(),
			Path:       filepath.Join(path, entry.Name()),
			IsDir:      info.IsDir(),
			Size:       info.Size(),
			ModTime:    info.ModTime(),
			Permission: info.Mode(),
		})
	}

	fb.sortEntries()
	fb.loading = false

	// Reset cursor and scroll
	fb.cursor = 0
	fb.startIndex = 0

	return nil
}

// sortEntries sorts the file entries based on the current sort mode
func (fb *FileBrowser) sortEntries() {
	sort.Slice(fb.entries, func(i, j int) bool {
		// Always sort directories first (except ..)
		if fb.entries[i].Name != ".." && fb.entries[j].Name != ".." {
			if fb.entries[i].IsDir != fb.entries[j].IsDir {
				return fb.entries[i].IsDir
			}
		}

		switch fb.sortBy {
		case SortBySize:
			return fb.entries[i].Size < fb.entries[j].Size
		case SortByModTime:
			return fb.entries[i].ModTime.After(fb.entries[j].ModTime)
		case SortByType:
			extI := filepath.Ext(fb.entries[i].Name)
			extJ := filepath.Ext(fb.entries[j].Name)
			if extI != extJ {
				return extI < extJ
			}
			return fb.entries[i].Name < fb.entries[j].Name
		default: // SortByName
			return fb.entries[i].Name < fb.entries[j].Name
		}
	})
}

// Navigation methods
func (fb *FileBrowser) MoveUp() {
	if fb.cursor > 0 {
		fb.cursor--
		fb.updateScrollPosition()
	}
}

func (fb *FileBrowser) MoveDown() {
	if fb.cursor < len(fb.entries)-1 {
		fb.cursor++
		fb.updateScrollPosition()
	}
}

func (fb *FileBrowser) PageUp() {
	fb.cursor -= fb.height - 3 // Leave room for header/footer
	if fb.cursor < 0 {
		fb.cursor = 0
	}
	fb.updateScrollPosition()
}

func (fb *FileBrowser) PageDown() {
	fb.cursor += fb.height - 3
	if fb.cursor >= len(fb.entries) {
		fb.cursor = len(fb.entries) - 1
	}
	fb.updateScrollPosition()
}

func (fb *FileBrowser) updateScrollPosition() {
	visibleHeight := fb.height - 3 // Header + footer

	if fb.cursor < fb.startIndex {
		fb.startIndex = fb.cursor
	} else if fb.cursor >= fb.startIndex+visibleHeight {
		fb.startIndex = fb.cursor - visibleHeight + 1
	}

	// Ensure we don't scroll past the end
	maxStart := len(fb.entries) - visibleHeight
	if maxStart < 0 {
		maxStart = 0
	}
	if fb.startIndex > maxStart {
		fb.startIndex = maxStart
	}
}

// Selection methods
func (fb *FileBrowser) ToggleSelection() {
	if !fb.multiSelect || fb.cursor >= len(fb.entries) {
		return
	}

	entry := fb.entries[fb.cursor]
	if entry.Name == ".." {
		return // Don't allow selecting parent directory
	}

	fb.selectedFiles[entry.Path] = !fb.selectedFiles[entry.Path]
}

func (fb *FileBrowser) SelectAll() {
	if !fb.multiSelect {
		return
	}

	for _, entry := range fb.entries {
		if entry.Name != ".." {
			fb.selectedFiles[entry.Path] = true
		}
	}
}

func (fb *FileBrowser) ClearSelection() {
	fb.selectedFiles = make(map[string]bool)
}

func (fb *FileBrowser) GetSelectedPaths() []string {
	paths := make([]string, 0, len(fb.selectedFiles))
	for path, selected := range fb.selectedFiles {
		if selected {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}

// Action methods
func (fb *FileBrowser) Enter() {
	if fb.cursor >= len(fb.entries) {
		return
	}

	entry := fb.entries[fb.cursor]

	if entry.IsDir {
		// Navigate to directory
		fb.LoadDirectory(entry.Path)
	} else {
		// Select file
		if fb.onSelect != nil {
			fb.onSelect(entry.Path)
		}
	}
}

func (fb *FileBrowser) SaveSelection() {
	if fb.multiSelect && fb.onMultiSelect != nil {
		paths := fb.GetSelectedPaths()
		fb.onMultiSelect(paths)
	} else if fb.cursor < len(fb.entries) && fb.onSelect != nil {
		fb.onSelect(fb.entries[fb.cursor].Path)
	}
}

// Render the file browser
func (fb *FileBrowser) Render() string {
	lines := make([]string, 0, fb.height)

	// Header with current path and status
	header := fb.renderHeader()
	lines = append(lines, header)

	// Loading or error state
	if fb.loading {
		loadingMsg := theme.RenderText("Loading...")
		lines = append(lines, loadingMsg)
		return fb.padToHeight(lines)
	}

	if fb.loadError != nil {
		errorMsg := theme.RenderStatus("error", fb.loadError.Error())
		lines = append(lines, errorMsg)
		return fb.padToHeight(lines)
	}

	// File list
	visibleHeight := fb.height - 3 // Header + footer + divider
	endIndex := fb.startIndex + visibleHeight
	if endIndex > len(fb.entries) {
		endIndex = len(fb.entries)
	}

	for i := fb.startIndex; i < endIndex; i++ {
		entry := fb.entries[i]
		focused := i == fb.cursor
		selected := fb.selectedFiles[entry.Path]

		line := fb.renderFileEntry(entry, selected, focused)
		lines = append(lines, line)
	}

	// Add scroll indicator if needed
	if len(fb.entries) > visibleHeight {
		scrollInfo := theme.RenderNavigationHelp([]string{
			fmt.Sprintf("Showing %d-%d of %d", fb.startIndex+1, endIndex, len(fb.entries)),
		}, fb.width)
		lines = append(lines, scrollInfo)
	}

	// Footer with help
	footer := fb.renderFooter()
	lines = append(lines, footer)

	return fb.padToHeight(lines)
}

func (fb *FileBrowser) renderHeader() string {
	// Breadcrumb path
	pathParts := strings.Split(fb.currentPath, string(os.PathSeparator))
	if len(pathParts) > 3 {
		// Truncate long paths
		pathParts = append([]string{"..."}, pathParts[len(pathParts)-2:]...)
	}

	return theme.RenderBreadcrumb(pathParts, fb.width)
}

func (fb *FileBrowser) renderFileEntry(entry FileEntry, selected, focused bool) string {
	// Icon
	icon := theme.RenderIcon("file")
	if entry.IsDir {
		icon = theme.RenderIcon("folder")
	}

	// Selection indicator
	selectionIcon := " "
	if fb.multiSelect {
		if selected {
			selectionIcon = theme.RenderIcon("success")
		} else {
			selectionIcon = theme.RenderIcon("inactive")
		}
	}

	// Name (truncate if too long)
	nameWidth := fb.width - 20 // Leave space for icon, selection, size, etc.
	name := entry.Name
	if len(name) > nameWidth {
		name = name[:nameWidth-1] + "…"
	}

	// Size (for files)
	sizeStr := ""
	if !entry.IsDir && entry.Name != ".." {
		sizeStr = utils.FormatFileSizeShort(entry.Size)
	}

	// Build line
	content := fmt.Sprintf("%s %s %-*s %8s", selectionIcon, icon, nameWidth, name, sizeStr)

	// Apply selection styling
	if focused {
		return theme.RenderSelection(content, fb.width)
	}

	return theme.RenderText(content)
}

func (fb *FileBrowser) renderFooter() string {
	helpItems := []string{"[Enter] Open", "[Esc] Back"}

	if fb.multiSelect {
		helpItems = append([]string{"[Space] Select"}, helpItems...)
		helpItems = append(helpItems, "[A] All", "[C] Clear", "[S] Save")
	}

	return theme.RenderHelpBar(helpItems, fb.width)
}

func (fb *FileBrowser) padToHeight(lines []string) string {
	// Pad with empty lines to reach desired height
	for len(lines) < fb.height {
		lines = append(lines, "")
	}

	// Truncate if too many lines
	if len(lines) > fb.height {
		lines = lines[:fb.height]
	}

	return strings.Join(lines, "\n")
}

// Utility functions (using common utilities to avoid duplication)

// GetCurrentPath returns the current directory path
func (fb *FileBrowser) GetCurrentPath() string {
	return fb.currentPath
}

// GetSelectedCount returns the number of selected files
func (fb *FileBrowser) GetSelectedCount() int {
	count := 0
	for _, selected := range fb.selectedFiles {
		if selected {
			count++
		}
	}
	return count
}
