package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PipelineComponent struct {
	Name    string
	Enabled bool
	Config  string
}

type PipelineView struct {
	components []PipelineComponent
	cursor     int
	width      int
	height     int
}

func NewPipelineView() PipelineView {
	return PipelineView{
		components: []PipelineComponent{
			{Name: "Sentence Detector", Enabled: true, Config: "Default"},
			{Name: "Tokenizer", Enabled: true, Config: "PTB"},
			{Name: "POS Tagger", Enabled: true, Config: "Default"},
			{Name: "Chunker", Enabled: true, Config: "Default"},
			{Name: "Context Dependent Tokenizer", Enabled: false, Config: "Disabled"},
			{Name: "Dictionary Lookup", Enabled: true, Config: "SNOMEDCT"},
			{Name: "Dependency Parser", Enabled: false, Config: "ClearNLP"},
			{Name: "Semantic Role Labeler", Enabled: false, Config: "ClearNLP"},
			{Name: "Named Entity Recognition", Enabled: true, Config: "Default"},
			{Name: "Assertion Module", Enabled: true, Config: "Default"},
			{Name: "Drug NER", Enabled: true, Config: "Default"},
			{Name: "Relation Extractor", Enabled: false, Config: "Default"},
		},
	}
}

func (p PipelineView) Init() tea.Cmd {
	return nil
}

func (p PipelineView) Update(msg tea.Msg) (PipelineView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if p.cursor > 0 {
				p.cursor--
			}
		case "down", "j":
			if p.cursor < len(p.components)-1 {
				p.cursor++
			}
		case " ", "enter":
			p.components[p.cursor].Enabled = !p.components[p.cursor].Enabled
		}
	}
	return p, nil
}

func (p PipelineView) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	b.WriteString(titleStyle.Render("► Pipeline Configuration"))
	b.WriteString("\n\n")

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Underline(true)

	b.WriteString(fmt.Sprintf("%-30s %-10s %s\n",
		headerStyle.Render("Component"),
		headerStyle.Render("Status"),
		headerStyle.Render("Config")))
	b.WriteString(strings.Repeat("─", 60) + "\n")

	for i, comp := range p.components {
		cursor := "  "
		if p.cursor == i {
			cursor = "▶ "
		}

		status := "✗"
		statusColor := "9"
		if comp.Enabled {
			status = "✓"
			statusColor = "10"
		}

		line := fmt.Sprintf("%s%-28s %s  %-9s %s",
			cursor,
			comp.Name,
			lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(status),
			"",
			comp.Config)

		if p.cursor == i {
			line = lipgloss.NewStyle().
				Foreground(lipgloss.Color("86")).
				Bold(true).
				Render(line)
		}

		b.WriteString(line + "\n")
	}

	b.WriteString("\n")
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	b.WriteString(footerStyle.Render("␣ Toggle • C Configure • S Save • ESC Back"))
	b.WriteString("\n\n[Placeholder: Real cTAKES pipeline configuration pending]")

	return b.String()
}