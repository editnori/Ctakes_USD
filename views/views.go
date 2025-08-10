package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ctakes-tui/ctakes-tui/views/dashboard"
)

// DashboardView wraps the dashboard model to provide a consistent interface
type DashboardView struct {
	model dashboard.Model
}

// NewDashboardView creates a new dashboard view
func NewDashboardView() DashboardView {
	return DashboardView{
		model: dashboard.New(),
	}
}

// Init initializes the dashboard view
func (d DashboardView) Init() tea.Cmd {
	return d.model.Init()
}

// Update updates the dashboard view
func (d DashboardView) Update(msg tea.Msg) (DashboardView, tea.Cmd) {
	newModel, cmd := d.model.Update(msg)
	return DashboardView{model: newModel}, cmd
}

// View renders the dashboard view
func (d DashboardView) View() string {
	return d.model.View()
}

// GetCursor returns the current cursor position (for compatibility with main.go)
func (d DashboardView) GetCursor() int {
	return d.model.GetCursor()
}

// DocumentView represents the document processing view
type DocumentView struct {
	// Add document view fields as needed
}

// NewDocumentView creates a new document view
func NewDocumentView() DocumentView {
	return DocumentView{}
}

// Init initializes the document view
func (d DocumentView) Init() tea.Cmd {
	return nil
}

// Update updates the document view
func (d DocumentView) Update(msg tea.Msg) (DocumentView, tea.Cmd) {
	return d, nil
}

// View renders the document view
func (d DocumentView) View() string {
	return "Document Processing View\n\nThis feature is under development."
}

// AnalyzeView represents the text analysis view
type AnalyzeView struct {
	// Add analyze view fields as needed
}

// NewAnalyzeView creates a new analyze view
func NewAnalyzeView() AnalyzeView {
	return AnalyzeView{}
}

// Init initializes the analyze view
func (a AnalyzeView) Init() tea.Cmd {
	return nil
}

// Update updates the analyze view
func (a AnalyzeView) Update(msg tea.Msg) (AnalyzeView, tea.Cmd) {
	return a, nil
}

// View renders the analyze view
func (a AnalyzeView) View() string {
	return "Text Analysis View\n\nThis feature is under development."
}

// PipelineView represents the pipeline configuration view
type PipelineView struct {
	// Add pipeline view fields as needed
}

// NewPipelineView creates a new pipeline view
func NewPipelineView() PipelineView {
	return PipelineView{}
}

// Init initializes the pipeline view
func (p PipelineView) Init() tea.Cmd {
	return nil
}

// Update updates the pipeline view
func (p PipelineView) Update(msg tea.Msg) (PipelineView, tea.Cmd) {
	return p, nil
}

// View renders the pipeline view
func (p PipelineView) View() string {
	return "Pipeline Configuration View\n\nThis feature is under development."
}
