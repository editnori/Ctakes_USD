package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ctakes-tui/ctakes-tui/views"
	"github.com/ctakes-tui/ctakes-tui/views/dashboard"
)

const Version = "0.1.0"

type model struct {
	currentView   string
	dashboardView dashboard.Model
	documentView  *views.DocumentModel
	analyzeView   *views.AnalyzeModel
	pipelineView  *views.PipelineModel
	width         int
	height        int
	initialized   map[string]bool
}

func initialModel() model {
	return model{
		currentView:   "dashboard",
		dashboardView: dashboard.New(),
		initialized:   make(map[string]bool),
	}
}

func (m model) Init() tea.Cmd {
	m.initialized["dashboard"] = true
	return m.dashboardView.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = size.Width
		m.height = size.Height
	}

	if strMsg, ok := msg.(string); ok {
		switch strMsg {
		case "main_menu", "dashboard_view":
			m.currentView = "dashboard"
			if !m.initialized["dashboard"] {
				m.initialized["dashboard"] = true
				return m, m.dashboardView.Init()
			}
		case "document_view":
			m.currentView = "document"
			if m.documentView == nil {
				doc := views.NewDocumentModel()
				m.documentView = &doc
				m.initialized["document"] = true
				return m, m.documentView.Init()
			}
		case "analyze_view":
			m.currentView = "analyze"
			if m.analyzeView == nil {
				analyze := views.NewAnalyzeModel()
				m.analyzeView = &analyze
				m.initialized["analyze"] = true
				return m, m.analyzeView.Init()
			}
		case "pipeline_view":
			m.currentView = "pipeline"
			if m.pipelineView == nil {
				pipeline := views.NewPipelineModel()
				m.pipelineView = &pipeline
				m.initialized["pipeline"] = true
				return m, m.pipelineView.Init()
			}
		}
	}

	var cmd tea.Cmd
	switch m.currentView {
	case "dashboard":
		m.dashboardView, cmd = m.dashboardView.Update(msg)
	case "document":
		if m.documentView != nil {
			*m.documentView, cmd = m.documentView.Update(msg)
		}
	case "analyze":
		if m.analyzeView != nil {
			*m.analyzeView, cmd = m.analyzeView.Update(msg)
		}
	case "pipeline":
		if m.pipelineView != nil {
			*m.pipelineView, cmd = m.pipelineView.Update(msg)
		}
	}

	return m, cmd
}

func (m model) View() string {
	switch m.currentView {
	case "dashboard":
		return m.dashboardView.View()
	case "document":
		if m.documentView != nil {
			return m.documentView.View()
		}
	case "analyze":
		if m.analyzeView != nil {
			return m.analyzeView.View()
		}
	case "pipeline":
		if m.pipelineView != nil {
			return m.pipelineView.View()
		}
	}
	return m.dashboardView.View()
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("cTAKES TUI v%s\n", Version)
		os.Exit(0)
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
