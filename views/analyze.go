package views

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

type AnalyzeView struct {
	textarea textarea.Model
	viewport viewport.Model
	results  string
	width    int
	height   int
}

func NewAnalyzeView() AnalyzeView {
	ta := textarea.New()
	ta.Placeholder = "Enter clinical text to analyze..."
	ta.Focus()
	ta.CharLimit = 10000
	ta.SetHeight(10)

	vp := viewport.New(50, 10)
	vp.SetContent("Results will appear here after analysis...")

	return AnalyzeView{
		textarea: ta,
		viewport: vp,
	}
}

func (a AnalyzeView) Init() tea.Cmd {
	return textarea.Blink
}

func (a AnalyzeView) Update(msg tea.Msg) (AnalyzeView, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.textarea.SetWidth(msg.Width - 4)
		a.viewport.Width = msg.Width - 4
		a.viewport.Height = (msg.Height / 2) - 8

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+a":
			a.analyzeText()
		case "tab":
			if a.textarea.Focused() {
				a.textarea.Blur()
				a.viewport.GotoTop()
			} else {
				a.textarea.Focus()
			}
		}
	}

	if a.textarea.Focused() {
		var cmd tea.Cmd
		a.textarea, cmd = a.textarea.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		var cmd tea.Cmd
		a.viewport, cmd = a.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

func (a *AnalyzeView) analyzeText() {
	text := a.textarea.Value()
	if text == "" {
		a.results = "No text to analyze"
		a.viewport.SetContent(a.results)
		return
	}

	a.results = "🔍 Analysis Results:\n\n" +
		"• Detected Entities: [Mock Results]\n" +
		"  - Medications: aspirin, lisinopril\n" +
		"  - Conditions: hypertension, diabetes\n" +
		"  - Procedures: blood test, x-ray\n\n" +
		"• UMLS Concepts: [Mock Results]\n" +
		"  - C0004057: Aspirin\n" +
		"  - C0065374: Lisinopril\n\n" +
		"Note: cTAKES integration pending"

	a.viewport.SetContent(a.results)
}

func (a AnalyzeView) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginBottom(1)

	b.WriteString(titleStyle.Render("🔍 Text Analysis"))
	b.WriteString("\n\n")

	b.WriteString("Input Text:\n")
	b.WriteString(a.textarea.View())
	b.WriteString("\n\n")

	resultsTitle := lipgloss.NewStyle().
		Bold(true).
		Render("Results:")
	b.WriteString(resultsTitle + "\n")
	b.WriteString(a.viewport.View())

	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("Ctrl+A: Analyze • Tab: Switch Focus • Esc: Back"))

	return b.String()
}