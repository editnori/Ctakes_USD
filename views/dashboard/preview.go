package dashboard

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
	"github.com/ctakes-tui/ctakes-tui/internal/utils"
)

func (m *Model) loadFilePreview(file FileInfo) {
	if file.IsDir {
		m.previewContent = "Directory selected"
		m.previewReady = true
		m.previewViewport.SetContent(m.previewContent)
		return
	}

	if !m.isPreviewable(file.Name) {
		m.previewContent = fmt.Sprintf("Preview not available for %s files", filepath.Ext(file.Name))
		m.previewReady = true
		m.previewViewport.SetContent(m.previewContent)
		return
	}

	fullPath := filepath.Join(m.currentPath, file.Name)

	info, err := os.Stat(fullPath)
	if err != nil {
		m.previewContent = fmt.Sprintf("Error: %v", err)
		m.previewReady = true
		m.previewViewport.SetContent(m.previewContent)
		return
	}

	maxSize := int64(500 * 1024) // 500KB limit for preview
	if info.Size() > maxSize {
		m.previewContent = fmt.Sprintf("File too large to preview (%.1f KB > 500 KB)", float64(info.Size())/1024)
		m.previewReady = true
		m.previewViewport.SetContent(m.previewContent)
		return
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		m.previewContent = fmt.Sprintf("Error reading file: %v", err)
		m.previewReady = true
		m.previewViewport.SetContent(m.previewContent)
		return
	}

	text := string(content)
	lines := strings.Split(text, "\n")

	maxLines := 50
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, fmt.Sprintf("\n... (%d more lines)", len(strings.Split(text, "\n"))-maxLines))
	}

	if m.shouldHighlight(file.Name) {
		m.previewContent = m.applySyntaxHighlighting(lines, file.Name)
	} else {
		m.previewContent = strings.Join(lines, "\n")
	}

	m.previewReady = true
	m.previewViewport.SetContent(m.previewContent)
}
func (m *Model) isPreviewable(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	previewableExts := []string{
		// Text files
		".txt", ".md", ".markdown", ".rst", ".log",
		// Code files
		".go", ".py", ".js", ".ts", ".java", ".c", ".cpp", ".rs", ".rb", ".php",
		".html", ".css", ".scss", ".sass", ".less",
		// Config files
		".json", ".xml", ".yaml", ".yml", ".toml", ".ini", ".conf", ".cfg",
		// Shell scripts
		".sh", ".bash", ".zsh", ".fish", ".ps1", ".bat", ".cmd",
		// SQL
		".sql",
		// Documentation
		".doc", ".docx", ".pdf", // Note: These would need special handling
		// Medical formats
		".hl7", ".cda", ".fhir",
	}

	for _, e := range previewableExts {
		if ext == e {
			return true
		}
	}
	return false
}

func (m *Model) shouldHighlight(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	highlightExts := []string{
		".go", ".py", ".js", ".ts", ".java", ".c", ".cpp", ".rs",
		".html", ".css", ".json", ".xml", ".yaml", ".yml", ".toml",
		".sh", ".bash", ".zsh", ".fish", ".sql", ".md",
	}

	for _, e := range highlightExts {
		if ext == e {
			return true
		}
	}
	return false
}
func (m *Model) applySyntaxHighlighting(lines []string, filename string) string {
	text := strings.Join(lines, "\n")

	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, text)
	if err != nil {
		return text
	}

	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return text
	}

	return buf.String()
}

func (m *Model) renderPreviewPanel(width, height int) string {
	// Always show header
	headerStyle := lipgloss.NewStyle().
		Foreground(theme.ColorPrimary).
		Bold(true).
		Width(width)

	dividerStyle := lipgloss.NewStyle().
		Foreground(theme.ColorBorder).
		Width(width)

	// Check if we have a selected file
	if m.fileTable.Cursor() >= 0 && m.fileTable.Cursor() < len(m.files) {
		selectedFile := m.files[m.fileTable.Cursor()]
		header := headerStyle.Render(utils.TruncateString(selectedFile.Name, width))
		divider := dividerStyle.Render(strings.Repeat("─", width))

		// Content area height (account for header and divider)
		contentHeight := height - 3

		if !m.previewReady || selectedFile.IsDir {
			// Show appropriate message
			message := "Directory selected"
			if !selectedFile.IsDir && !m.isPreviewable(selectedFile.Name) {
				message = "Preview not available"
			} else if !selectedFile.IsDir && !m.previewReady {
				message = "Loading..."
			}

			messageStyle := lipgloss.NewStyle().
				Foreground(theme.ColorSecondary).
				Width(width).
				Height(contentHeight).
				AlignHorizontal(lipgloss.Center).
				AlignVertical(lipgloss.Center)

			content := messageStyle.Render(message)
			return lipgloss.JoinVertical(lipgloss.Left, header, divider, content)
		}

		// Render actual preview content
		m.previewViewport.Width = width
		m.previewViewport.Height = contentHeight

		content := m.previewViewport.View()
		return lipgloss.JoinVertical(lipgloss.Left, header, divider, content)
	}

	// No file selected
	header := headerStyle.Render("Preview")
	divider := dividerStyle.Render(strings.Repeat("─", width))

	messageStyle := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Width(width).
		Height(height - 3).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center)

	content := messageStyle.Render("Select a file to preview")
	return lipgloss.JoinVertical(lipgloss.Left, header, divider, content)
}
