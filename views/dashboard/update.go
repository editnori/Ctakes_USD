package dashboard

import (
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Init() tea.Cmd {
	m.updateFileList()
	m.updateSystemInfo()
	m.updateTables()
	return tea.Batch(tickEvery(), m.spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateTables()
		m.updateViewportSize()
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-10)
			m.viewport.HighPerformanceRendering = false
			m.previewViewport = viewport.New(msg.Width/3, msg.Height-10)
			m.previewViewport.HighPerformanceRendering = false
			m.ready = true
		}

	case tickMsg:
		m.updateSystemInfo()
		m.lastUpdate = time.Time(msg)
		return m, tickEvery()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Tab):
			// Cycle through panels based on what's visible
			if m.showPreview {
				// Sidebar -> Main -> Preview -> Sidebar
				switch m.activePanel {
				case SidebarPanel:
					m.activePanel = MainPanel
				case MainPanel:
					m.activePanel = PreviewPanel
				case PreviewPanel:
					m.activePanel = SidebarPanel
				default:
					m.activePanel = SidebarPanel
				}
			} else {
				// Sidebar -> Main -> Sidebar (no preview)
				if m.activePanel == SidebarPanel {
					m.activePanel = MainPanel
				} else {
					m.activePanel = SidebarPanel
				}
			}

		case key.Matches(msg, m.keys.ShiftTab):
			// Reverse cycle
			if m.showPreview {
				switch m.activePanel {
				case SidebarPanel:
					m.activePanel = PreviewPanel
				case MainPanel:
					m.activePanel = SidebarPanel
				case PreviewPanel:
					m.activePanel = MainPanel
				default:
					m.activePanel = SidebarPanel
				}
			} else {
				if m.activePanel == SidebarPanel {
					m.activePanel = MainPanel
				} else {
					m.activePanel = SidebarPanel
				}
			}

		case key.Matches(msg, m.keys.Preview):
			m.showPreview = !m.showPreview
			if m.showPreview && m.fileTable.Cursor() < len(m.files) {
				m.loadFilePreview(m.files[m.fileTable.Cursor()])
			}
			m.updateTables()

		case key.Matches(msg, m.keys.Up):
			switch m.activePanel {
			case SidebarPanel:
				// Navigate menu items
				if m.cursor > 0 {
					m.cursor--
				}
			case MainPanel:
				// Don't update table here - handled in main update section
				// The table will be updated once at the end
			case SystemPanel:
				// Scroll system info
				m.viewport.LineUp(1)
			case PreviewPanel:
				// Scroll preview content
				m.previewViewport.LineUp(1)
			}

		case key.Matches(msg, m.keys.Down):
			switch m.activePanel {
			case SidebarPanel:
				// Navigate menu items
				if m.cursor < len(m.sidebarItems)-1 {
					m.cursor++
				}
			case MainPanel:
				// Don't update table here - handled in main update section
				// The table will be updated once at the end
			case SystemPanel:
				// Scroll system info
				m.viewport.LineDown(1)
			case PreviewPanel:
				// Scroll preview content
				m.previewViewport.LineDown(1)
			}

		case key.Matches(msg, m.keys.PageUp):
			switch m.activePanel {
			case MainPanel:
				// Don't update table here - handled in main update section
				// The table will be updated once at the end
			case SystemPanel:
				m.viewport.ViewUp()
			case PreviewPanel:
				m.previewViewport.ViewUp()
			}

		case key.Matches(msg, m.keys.PageDown):
			switch m.activePanel {
			case MainPanel:
				// Don't update table here - handled in main update section
				// The table will be updated once at the end
			case SystemPanel:
				m.viewport.ViewDown()
			case PreviewPanel:
				m.previewViewport.ViewDown()
			}

		case key.Matches(msg, m.keys.Enter):
			switch m.activePanel {
			case SidebarPanel:
				if m.cursor < len(m.sidebarItems) {
					// Switch to main panel when selecting a menu item
					m.activePanel = MainPanel
					cmd := m.handleMenuAction(m.sidebarItems[m.cursor].Action)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			case MainPanel:
				if m.cursor < len(m.sidebarItems) && m.sidebarItems[m.cursor].Action == "files" {
					if m.fileTable.Cursor() < len(m.files) {
						file := m.files[m.fileTable.Cursor()]
						if file.IsDir {
							m.currentPath = filepath.Join(m.currentPath, file.Name)
							m.updateFileList()
							m.updateTables()
						}
					}
				}
			}

		case key.Matches(msg, m.keys.Back):
			if m.activePanel == MainPanel && m.currentPath != "/" {
				m.currentPath = filepath.Dir(m.currentPath)
				m.updateFileList()
				m.updateTables()
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update file table only when it's the active panel
	if m.activePanel == MainPanel {
		oldCursor := m.fileTable.Cursor()
		var cmd tea.Cmd
		m.fileTable, cmd = m.fileTable.Update(msg)
		cmds = append(cmds, cmd)

		// Update preview if cursor changed
		newCursor := m.fileTable.Cursor()
		if m.showPreview && oldCursor != newCursor && newCursor >= 0 && newCursor < len(m.files) {
			m.loadFilePreview(m.files[newCursor])
		}
	}

	// Update viewports
	if m.activePanel == SystemPanel {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.showPreview && m.previewReady && m.activePanel == PreviewPanel {
		var cmd tea.Cmd
		m.previewViewport, cmd = m.previewViewport.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) handleMenuAction(action string) tea.Cmd {
	switch action {
	case "system", "files", "processes":
		// Just update the file list if needed
		if action == "files" {
			m.updateFileList()
			m.updateTables()
		}
	case "document_view", "analyze_view", "pipeline_view":
		return func() tea.Msg {
			return action
		}
	}
	return nil
}

func (m *Model) updateViewportSize() {
	// Update system viewport
	m.viewport.Width = m.width/2 - 4
	m.viewport.Height = m.height - 10

	// Update preview viewport
	if m.showPreview {
		m.previewViewport.Width = m.width/3 - 4
		m.previewViewport.Height = m.height - 10
	}
}
