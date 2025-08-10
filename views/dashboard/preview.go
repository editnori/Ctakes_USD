package dashboard

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
	"github.com/ctakes-tui/ctakes-tui/internal/utils"
)

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
		Foreground(theme.ColorAccent).
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
