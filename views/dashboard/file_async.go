package dashboard

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
	"github.com/ctakes-tui/ctakes-tui/internal/utils"
)

const (
	MaxItemsPerPage      = 50
	CacheExpiration      = 30 * time.Second
	MaxCacheEntries      = 100
	DirectoryLoadTimeout = 10 * time.Second
)

// Cache management
type DirCacheEntry struct {
	Files      []FileInfo
	Error      error
	Timestamp  time.Time
	TotalCount int
}

var (
	dirCache   = make(map[string]*DirCacheEntry)
	cacheMutex = sync.RWMutex{}
)

// getCachedDirectory retrieves cached directory information
func getCachedDirectory(path string) *DirCacheEntry {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()
	return dirCache[path]
}

// isCacheExpired checks if cache entry is expired
func isCacheExpired(entry *DirCacheEntry) bool {
	return time.Since(entry.Timestamp) > CacheExpiration
}

// cacheDirectory stores directory information in cache
func cacheDirectory(path string, files []FileInfo, err error, totalCount int) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// Clean up old entries if cache is getting too large
	if len(dirCache) >= MaxCacheEntries {
		// Remove oldest entries
		oldestTime := time.Now()
		oldestKey := ""
		for key, entry := range dirCache {
			if entry.Timestamp.Before(oldestTime) {
				oldestTime = entry.Timestamp
				oldestKey = key
			}
		}
		if oldestKey != "" {
			delete(dirCache, oldestKey)
		}
	}

	dirCache[path] = &DirCacheEntry{
		Files:      files,
		Error:      err,
		Timestamp:  time.Now(),
		TotalCount: totalCount,
	}
}

// ClearCache removes entries from cache
func ClearCache(path string) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	if path == "" {
		// Clear all cache
		dirCache = make(map[string]*DirCacheEntry)
	} else {
		delete(dirCache, path)
	}
}

// LoadDirectoryAsync loads directory content asynchronously
func LoadDirectoryAsync(path string, page int, requestID uint64) tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			return DirectoryLoadingMsg{Path: path}
		},
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), DirectoryLoadTimeout)
			defer cancel()

			files, err := loadDirectoryWithContext(ctx, path)
			if err != nil {
				return DirectoryLoadedMsg{
					Path:      path,
					Files:     []FileInfo{},
					Error:     err,
					RequestID: requestID,
				}
			}

			// Get paginated files
			paginatedFiles := getPaginatedFiles(files, page)
			totalCount := len(files)

			// Cache the full result
			cacheDirectory(path, files, err, totalCount)

			return DirectoryLoadedMsg{
				Path:      path,
				Files:     paginatedFiles,
				Error:     nil,
				RequestID: requestID,
			}
		},
	)
}

// LoadDirectoryPage loads a specific page of directory content
func LoadDirectoryPage(path string, page int, requestID uint64) tea.Cmd {
	return func() tea.Msg {
		// Check cache first
		if cached := getCachedDirectory(path); cached != nil && !isCacheExpired(cached) {
			paginatedFiles := getPaginatedFiles(cached.Files, page)
			return DirectoryLoadedMsg{
				Path:      path,
				Files:     paginatedFiles,
				Error:     cached.Error,
				RequestID: requestID,
			}
		}

		// Load fresh data if cache miss
		ctx, cancel := context.WithTimeout(context.Background(), DirectoryLoadTimeout)
		defer cancel()

		files, err := loadDirectoryWithContext(ctx, path)
		if err != nil {
			return DirectoryLoadedMsg{
				Path:      path,
				Files:     []FileInfo{},
				Error:     err,
				RequestID: requestID,
			}
		}

		// Get paginated files
		paginatedFiles := getPaginatedFiles(files, page)
		totalCount := len(files)

		// Cache the full result
		cacheDirectory(path, files, err, totalCount)

		return DirectoryLoadedMsg{
			Path:      path,
			Files:     paginatedFiles,
			Error:     nil,
			RequestID: requestID,
		}
	}
}

// loadDirectoryWithContext loads directory content with context cancellation
func loadDirectoryWithContext(ctx context.Context, dirPath string) ([]FileInfo, error) {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	files := make([]FileInfo, 0, len(entries))

	// Add parent directory entry if not at root
	if dirPath != "/" && dirPath != "" {
		files = append(files, FileInfo{
			Name:    "..",
			Size:    "",
			Mode:    "drwxr-xr-x",
			ModTime: "",
			IsDir:   true,
			Icon:    theme.GetSemanticIcon("directory"),
		})
	}

	for _, entry := range entries {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}

		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't stat
		}

		// Skip hidden files on Unix systems
		if strings.HasPrefix(entry.Name(), ".") && entry.Name() != ".." {
			continue
		}

		fileInfo := FileInfo{
			Name:    entry.Name(),
			IsDir:   entry.IsDir(),
			Mode:    formatFileMode(info.Mode()),
			ModTime: info.ModTime().Format("2006-01-02 15:04"),
		}

		// Set icon based on file type
		if entry.IsDir() {
			fileInfo.Icon = theme.GetSemanticIcon("directory")
			fileInfo.Size = ""
		} else {
			fileInfo.Icon = getFileIcon(entry.Name())
			fileInfo.Size = utils.FormatFileSize(info.Size())
		}

		files = append(files, fileInfo)
	}

	// Sort files: directories first, then by name
	sort.Slice(files, func(i, j int) bool {
		// Keep ".." at the top
		if files[i].Name == ".." {
			return true
		}
		if files[j].Name == ".." {
			return false
		}

		// Directories before files
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}

		// Then alphabetically by name
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	return files, nil
}

// getPaginatedFiles returns a paginated slice of files
func getPaginatedFiles(files []FileInfo, page int) []FileInfo {
	if len(files) <= MaxItemsPerPage {
		return files
	}

	start := page * MaxItemsPerPage
	if start >= len(files) {
		return []FileInfo{}
	}

	end := start + MaxItemsPerPage
	if end > len(files) {
		end = len(files)
	}

	return files[start:end]
}

// GetPageInfo returns pagination information
func GetPageInfo(totalItems, currentPage int) (int, bool) {
	if totalItems <= MaxItemsPerPage {
		return 1, false
	}

	totalPages := (totalItems + MaxItemsPerPage - 1) / MaxItemsPerPage
	hasMore := currentPage < totalPages-1

	return totalPages, hasMore
}

// formatFileMode formats file mode as string
func formatFileMode(mode fs.FileMode) string {
	return mode.String()
}

// getFileIcon returns appropriate icon for file type
func getFileIcon(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".txt", ".md", ".readme":
		return theme.GetSemanticIcon("document")
	case ".go", ".py", ".js", ".java", ".c", ".cpp", ".h":
		return theme.GetSemanticIcon("code")
	case ".json", ".yaml", ".yml", ".xml", ".toml":
		return theme.GetSemanticIcon("data")
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg":
		return theme.GetSemanticIcon("image")
	case ".mp3", ".wav", ".ogg", ".m4a":
		return theme.GetSemanticIcon("audio")
	case ".mp4", ".avi", ".mkv", ".mov":
		return theme.GetSemanticIcon("video")
	case ".zip", ".tar", ".gz", ".7z", ".rar":
		return theme.GetSemanticIcon("archive")
	case ".pdf":
		return theme.GetSemanticIcon("pdf")
	case ".exe", ".bin", ".deb", ".rpm":
		return theme.GetSemanticIcon("executable")
	default:
		return theme.GetSemanticIcon("file")
	}
}

// RenderLoadingIndicator renders a loading spinner
func RenderLoadingIndicator(spinner interface{}, width int) string {
	loadingText := "Loading..."

	// Try to get spinner view if it has a View method
	spinnerView := ""
	if v, ok := spinner.(interface{ View() string }); ok {
		spinnerView = v.View()
	} else {
		// Fallback spinning indicator
		spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		// Use simple time-based animation
		idx := int(time.Now().UnixMilli()/100) % len(spinners)
		spinnerView = spinners[idx]
	}

	content := fmt.Sprintf("%s %s", spinnerView, loadingText)

	// Center the loading indicator
	padding := (width - lipgloss.Width(content)) / 2
	if padding < 0 {
		padding = 0
	}

	paddedContent := strings.Repeat(" ", padding) + content

	return lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Width(width).
		Render(paddedContent)
}

// Helper function to detect RRF files for dictionary builder
func (m *Model) detectRRFFiles(path string) []string {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	rrfFiles := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToUpper(entry.Name()), ".RRF") {
			rrfFiles = append(rrfFiles, entry.Name())
		}
	}

	return rrfFiles
}

// loadFilePreview loads file content for preview
func (m *Model) loadFilePreview(file FileInfo) {
	if file.IsDir {
		m.previewContent = fmt.Sprintf("Directory: %s\n\nUse Enter to navigate into this directory.", file.Name)
		m.previewReady = true
		m.previewViewport.SetContent(m.previewContent)
		return
	}

	// For regular files, try to load content
	filePath := filepath.Join(m.currentPath, file.Name)

	// Check file size first
	info, err := os.Stat(filePath)
	if err != nil {
		m.previewContent = fmt.Sprintf("Error: Cannot access file %s", file.Name)
		m.previewReady = true
		m.previewViewport.SetContent(m.previewContent)
		return
	}

	// Don't preview files larger than 1MB
	if info.Size() > 1024*1024 {
		m.previewContent = fmt.Sprintf("File: %s\nSize: %s\n\nFile too large for preview (>1MB)",
			file.Name, utils.FormatFileSize(info.Size()))
		m.previewReady = true
		m.previewViewport.SetContent(m.previewContent)
		return
	}

	// Try to read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		m.previewContent = fmt.Sprintf("Error reading file %s: %v", file.Name, err)
		m.previewReady = true
		m.previewViewport.SetContent(m.previewContent)
		return
	}

	// Check if file is binary
	if isBinaryContent(content) {
		m.previewContent = fmt.Sprintf("File: %s\nSize: %s\nType: Binary file\n\nBinary files cannot be previewed as text.",
			file.Name, utils.FormatFileSize(info.Size()))
	} else {
		// Text file - show first 1000 lines
		lines := strings.Split(string(content), "\n")
		if len(lines) > 1000 {
			lines = lines[:1000]
			lines = append(lines, "", "... (file truncated for preview)")
		}

		m.previewContent = fmt.Sprintf("File: %s\nSize: %s\nLines: %d\n\n%s",
			file.Name, utils.FormatFileSize(info.Size()), len(lines), strings.Join(lines, "\n"))
	}

	m.previewReady = true
	m.previewViewport.SetContent(m.previewContent)
}

// isBinaryContent checks if content appears to be binary
func isBinaryContent(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// Check first 512 bytes for null bytes (common in binary files)
	checkLen := len(content)
	if checkLen > 512 {
		checkLen = 512
	}

	for i := 0; i < checkLen; i++ {
		if content[i] == 0 {
			return true
		}
	}

	// Check for high percentage of non-printable characters
	nonPrintable := 0
	for i := 0; i < checkLen; i++ {
		if content[i] < 32 && content[i] != '\t' && content[i] != '\n' && content[i] != '\r' {
			nonPrintable++
		}
	}

	// If more than 30% non-printable, consider binary
	return float64(nonPrintable)/float64(checkLen) > 0.3
}
