package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
)

type AnalyzeModel struct {
	textarea   textarea.Model
	viewport   viewport.Model
	spinner    spinner.Model
	width      int
	height     int
	results    []AnalysisResult
	analyzing  bool
	focusInput bool
	ready      bool
}

type AnalysisResult struct {
	Type     string
	Text     string
	CUI      string
	Score    float64
	Polarity string
	Ontology string
	Begin    int
	End      int
}

func NewAnalyzeModel() AnalyzeModel {
	ta := textarea.New()
	ta.Placeholder = "Enter or paste clinical text for real-time analysis..."
	ta.SetWidth(50)
	ta.SetHeight(10)
	ta.Focus()
	ta.CharLimit = 10000

	// Style the textarea
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(theme.ColorBackgroundLighter)
	ta.FocusedStyle.Base = lipgloss.NewStyle().
		BorderForeground(theme.ColorBorderActive).
		Foreground(theme.ColorForeground)
	ta.BlurredStyle.Base = lipgloss.NewStyle().
		BorderForeground(theme.ColorBorderInactive).
		Foreground(theme.ColorForegroundDim)

	vp := viewport.New(50, 10)
	vp.SetContent("")

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.ColorAccent)

	return AnalyzeModel{
		textarea:   ta,
		viewport:   vp,
		spinner:    s,
		focusInput: true,
		results:    []AnalysisResult{},
	}
}

func (m AnalyzeModel) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
	)
}

func (m AnalyzeModel) Update(msg tea.Msg) (AnalyzeModel, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Update component sizes
		textWidth := m.width/2 - 4
		textHeight := m.height - 14 // Adjusted for top padding

		m.textarea.SetWidth(textWidth)
		m.textarea.SetHeight(textHeight)
		m.viewport.Width = textWidth
		m.viewport.Height = textHeight

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, analyzeKeys.Back):
			return m, func() tea.Msg { return "main_menu" }

		case key.Matches(msg, analyzeKeys.Tab):
			// Toggle focus between input and results
			m.focusInput = !m.focusInput
			if m.focusInput {
				cmd = m.textarea.Focus()
			} else {
				m.textarea.Blur()
			}
			cmds = append(cmds, cmd)

		case key.Matches(msg, analyzeKeys.Analyze):
			if !m.analyzing && m.textarea.Value() != "" {
				m.analyzing = true
				cmds = append(cmds, m.performAnalysis())
			}

		case key.Matches(msg, analyzeKeys.Clear):
			m.textarea.SetValue("")
			m.results = []AnalysisResult{}
			m.viewport.SetContent("")
		}

	case analysisCompleteMsg:
		m.analyzing = false
		m.results = msg.results
		m.updateResultsView()

	case spinner.TickMsg:
		if m.analyzing {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Update textarea or viewport based on focus
	if m.focusInput {
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m AnalyzeModel) View() string {
	if !m.ready {
		return theme.BaseStyle.Render("Initializing analyzer...")
	}

	topPadding := "\n\n"
	header := m.renderHeader()
	content := m.renderContent()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		topPadding,
		header,
		content,
		footer,
	)
}

func (m *AnalyzeModel) renderHeader() string {
	title := theme.RenderTitle(theme.IconAnalyze, "Clinical Text Analyzer")

	status := "Ready"
	statusStyle := theme.StatusStyle

	if m.analyzing {
		status = m.spinner.View() + " Analyzing..."
		statusStyle = theme.StatusInfoStyle
	} else if len(m.results) > 0 {
		status = fmt.Sprintf("%d entities found", len(m.results))
		statusStyle = theme.StatusSuccessStyle
	}

	statusBar := statusStyle.Render(status)

	header := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			theme.SubtitleStyle.Render("Real-time NLP analysis powered by cTAKES"),
			strings.Repeat(" ", m.width-60-lipgloss.Width(statusBar)),
			statusBar,
		),
		strings.Repeat(theme.BorderDividerH, m.width-4),
	)

	return header
}

func (m *AnalyzeModel) renderContent() string {
	leftWidth := m.width/2 - 2
	rightWidth := m.width - leftWidth - 2

	// Input panel
	inputTitle := theme.RenderTitle(theme.IconDocument, "Input Text")
	inputStyle := theme.PanelStyle.
		Width(leftWidth).
		Height(m.height - 10) // Adjusted for top padding

	if m.focusInput {
		inputStyle = theme.PanelFocusedStyle.
			Width(leftWidth).
			Height(m.height - 10) // Adjusted for top padding
	}

	inputContent := lipgloss.JoinVertical(
		lipgloss.Left,
		inputTitle,
		strings.Repeat(theme.BorderDividerH, leftWidth-4),
		m.textarea.View(),
	)

	inputPanel := inputStyle.Render(inputContent)

	// Results panel
	resultsTitle := theme.RenderTitle(theme.IconResults, "Analysis Results")
	resultsStyle := theme.PanelStyle.
		Width(rightWidth).
		Height(m.height - 10) // Adjusted for top padding

	if !m.focusInput {
		resultsStyle = theme.PanelFocusedStyle.
			Width(rightWidth).
			Height(m.height - 10) // Adjusted for top padding
	}

	resultsContent := lipgloss.JoinVertical(
		lipgloss.Left,
		resultsTitle,
		strings.Repeat(theme.BorderDividerH, rightWidth-4),
		m.viewport.View(),
	)

	resultsPanel := resultsStyle.Render(resultsContent)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		inputPanel,
		lipgloss.NewStyle().Width(2).Render("  "),
		resultsPanel,
	)
}

func (m *AnalyzeModel) renderFooter() string {
	keys := []string{
		theme.RenderKeyHelp("Tab", "Switch Panel"),
		theme.RenderKeyHelp("Ctrl+Enter", "Analyze"),
		theme.RenderKeyHelp("Ctrl+L", "Clear"),
		theme.RenderKeyHelp("Esc", "Back"),
	}

	charCount := fmt.Sprintf("%d/%d chars", len(m.textarea.Value()), m.textarea.CharLimit)

	leftStr := lipgloss.JoinHorizontal(lipgloss.Left, keys...)
	rightStr := theme.FooterDescStyle.Render(charCount)

	gap := m.width - lipgloss.Width(leftStr) - lipgloss.Width(rightStr)
	if gap < 0 {
		gap = 0
	}

	return theme.FooterStyle.
		Width(m.width).
		Render(leftStr + strings.Repeat(" ", gap) + rightStr)
}

func (m *AnalyzeModel) updateResultsView() {
	if len(m.results) == 0 {
		m.viewport.SetContent(lipgloss.NewStyle().
			Foreground(theme.ColorForegroundDim).
			Italic(true).
			Render("No analysis results yet. Enter text and press Ctrl+Enter to analyze."))
		return
	}

	var content strings.Builder

	// Group results by type
	byType := make(map[string][]AnalysisResult)
	for _, r := range m.results {
		byType[r.Type] = append(byType[r.Type], r)
	}

	// Render each type
	types := []string{"Disease/Disorder", "Medication", "Sign/Symptom", "Procedure", "Anatomy"}
	for _, t := range types {
		if results, ok := byType[t]; ok && len(results) > 0 {
			header := lipgloss.NewStyle().
				Foreground(theme.ColorAccent).
				Bold(true).
				Render(fmt.Sprintf("%s %s (%d)", getTypeIcon(t), t, len(results)))

			content.WriteString(header + "\n")
			content.WriteString(strings.Repeat("─", 40) + "\n")

			for _, r := range results {
				entity := lipgloss.NewStyle().
					Foreground(theme.ColorForegroundBright).
					Render(r.Text)

				cui := lipgloss.NewStyle().
					Foreground(theme.ColorForegroundDim).
					Render(fmt.Sprintf("[%s]", r.CUI))

				score := lipgloss.NewStyle().
					Foreground(theme.ColorSuccess).
					Render(fmt.Sprintf("%.2f", r.Score))

				content.WriteString(fmt.Sprintf("  • %s %s %s\n", entity, cui, score))

				if r.Ontology != "" {
					onto := lipgloss.NewStyle().
						Foreground(theme.ColorInfo).
						Italic(true).
						Render(fmt.Sprintf("    %s", r.Ontology))
					content.WriteString(onto + "\n")
				}
			}
			content.WriteString("\n")
		}
	}

	m.viewport.SetContent(content.String())
}

func (m *AnalyzeModel) performAnalysis() tea.Cmd {
	return func() tea.Msg {
		text := m.textarea.Value()

		if text == "" {
			return analysisCompleteMsg{results: []AnalysisResult{}}
		}

		// TODO: Connect to actual cTAKES manager
		// manager := ctakes.GetManager()
		// results, err := manager.AnalyzeText(text)
		// if err != nil {
		//     return analysisErrorMsg{err: err}
		// }

		// Placeholder until cTAKES integration is complete
		results := []AnalysisResult{
			{
				Type:     "Info",
				Text:     "cTAKES integration pending",
				CUI:      "N/A",
				Score:    0.0,
				Ontology: "Analysis will be available once cTAKES is connected",
			},
		}

		return analysisCompleteMsg{results: results}
	}
}

type analysisCompleteMsg struct {
	results []AnalysisResult
}

func getTypeIcon(t string) string {
	switch t {
	case "Disease/Disorder":
		return "◈"
	case "Medication":
		return "◉"
	case "Sign/Symptom":
		return "◆"
	case "Procedure":
		return "◇"
	case "Anatomy":
		return "◎"
	default:
		return "•"
	}
}

type analyzeKeyMap struct {
	Back    key.Binding
	Tab     key.Binding
	Analyze key.Binding
	Clear   key.Binding
}

var analyzeKeys = analyzeKeyMap{
	Back: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("esc", "back"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch panel"),
	),
	Analyze: key.NewBinding(
		key.WithKeys("ctrl+enter"),
		key.WithHelp("ctrl+enter", "analyze"),
	),
	Clear: key.NewBinding(
		key.WithKeys("ctrl+l"),
		key.WithHelp("ctrl+l", "clear"),
	),
}
