package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DocumentView struct {
	files    []string
	cursor   int
	selected map[int]struct{}
	width    int
	height   int
}

func NewDocumentView() DocumentView {
	return DocumentView{
		files: []string{
			"sample1.txt",
			"patient_notes.txt",
			"clinical_report.pdf",
		},
		selected: make(map[int]struct{}),
	}
}

func (d DocumentView) Init() tea.Cmd {
	return nil
}

func (d DocumentView) Update(msg tea.Msg) (DocumentView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if d.cursor > 0 {
				d.cursor--
			}
		case "down", "j":
			if d.cursor < len(d.files)-1 {
				d.cursor++
			}
		case " ", "enter":
			if _, ok := d.selected[d.cursor]; ok {
				delete(d.selected, d.cursor)
			} else {
				d.selected[d.cursor] = struct{}{}
			}
		}
	}
	return d, nil
}

func (d DocumentView) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	b.WriteString(titleStyle.Render("► Document Processing"))
	b.WriteString("\n\n")

	if len(d.files) == 0 {
		b.WriteString("No documents loaded. Press 'a' to add documents.\n")
		b.WriteString("\n[Placeholder: cTAKES document processing integration pending]\n")
	} else {
		b.WriteString("Select documents to process:\n\n")
		for i, file := range d.files {
			cursor := "  "
			if d.cursor == i {
				cursor = "▶ "
			}

			checked := "☐"
			if _, ok := d.selected[i]; ok {
				checked = "☑"
			}

			line := fmt.Sprintf("%s%s %s", cursor, checked, file)
			if d.cursor == i {
				line = lipgloss.NewStyle().
					Foreground(lipgloss.Color("86")).
					Bold(true).
					Render(line)
			}
			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("␣ Select • ⏎ Process • ESC Back"))

	return b.String()
}