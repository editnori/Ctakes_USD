package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/views"
)

type ViewState int

const (
	MainMenu ViewState = iota
	DocumentView
	AnalyzeView
	PipelineView
	ResultsView
	DictionaryView
	SettingsView
	HelpView
)

type model struct {
	choices      []string
	cursor       int
	selected     map[int]struct{}
	ctakesStatus string
	currentView  ViewState
	documentView views.DocumentView
	analyzeView  views.AnalyzeView
	pipelineView views.PipelineView
	width        int
	height       int
}

func initialModel() model {
	return model{
		choices: []string{
			"Process Documents",
			"Analyze Text",
			"Configure Pipeline",
			"View Results",
			"Manage Dictionaries",
			"Settings",
			"Help",
			"Exit",
		},
		selected:     make(map[int]struct{}),
		ctakesStatus: "Not Connected (Placeholder)",
		currentView:  MainMenu,
		documentView: views.NewDocumentView(),
		analyzeView:  views.NewAnalyzeView(),
		pipelineView: views.NewPipelineView(),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	switch m.currentView {
	case MainMenu:
		return m.updateMainMenu(msg)
	case DocumentView:
		return m.updateDocumentView(msg)
	case AnalyzeView:
		return m.updateAnalyzeView(msg)
	case PipelineView:
		return m.updatePipelineView(msg)
	default:
		return m.updateMainMenu(msg)
	}
}

func (m model) updateMainMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.currentView == MainMenu {
				return m, tea.Quit
			}

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter", " ":
			switch m.cursor {
			case 0:
				m.currentView = DocumentView
			case 1:
				m.currentView = AnalyzeView
				return m, m.analyzeView.Init()
			case 2:
				m.currentView = PipelineView
			case 7:
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m model) updateDocumentView(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			m.currentView = MainMenu
			return m, nil
		}
	}
	
	var cmd tea.Cmd
	m.documentView, cmd = m.documentView.Update(msg)
	return m, cmd
}

func (m model) updateAnalyzeView(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			m.currentView = MainMenu
			return m, nil
		}
	}
	
	var cmd tea.Cmd
	m.analyzeView, cmd = m.analyzeView.Update(msg)
	return m, cmd
}

func (m model) updatePipelineView(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			m.currentView = MainMenu
			return m, nil
		}
	}
	
	var cmd tea.Cmd
	m.pipelineView, cmd = m.pipelineView.Update(msg)
	return m, cmd
}

func (m model) View() string {
	switch m.currentView {
	case DocumentView:
		return m.renderWithHeader(m.documentView.View())
	case AnalyzeView:
		return m.renderWithHeader(m.analyzeView.View())
	case PipelineView:
		return m.renderWithHeader(m.pipelineView.View())
	default:
		return m.viewMainMenu()
	}
}

func (m model) renderWithHeader(content string) string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Background(lipgloss.Color("0")).
		Padding(0, 2).
		Width(m.width).
		Align(lipgloss.Center)

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 2)

	header := headerStyle.Render("cTAKES Terminal Interface")
	status := statusStyle.Render(fmt.Sprintf("Status: %s", m.ctakesStatus))

	return header + "\n" + status + "\n\n" + content
}

func (m model) viewMainMenu() string {
	var headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Background(lipgloss.Color("0")).
		Padding(1, 2).
		Width(50).
		Align(lipgloss.Center)

	var statusStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 2)

	var menuStyle = lipgloss.NewStyle().
		Padding(1, 2)

	header := headerStyle.Render("cTAKES Terminal Interface")
	status := statusStyle.Render(fmt.Sprintf("Status: %s", m.ctakesStatus))

	menu := "\n"
	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = "▸"
			choice = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("86")).
				Render(choice)
		}

		menu += fmt.Sprintf("%s %s\n", cursor, choice)
	}

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("\n↑/↓: Navigate • Enter: Select • q: Quit")

	return header + "\n" + status + menuStyle.Render(menu) + footer
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}