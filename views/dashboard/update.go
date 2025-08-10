package dashboard

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ctakes-tui/ctakes-tui/internal/utils"
)

func (m Model) Init() tea.Cmd {
	// Don't call updateSystemInfo and updateTables here as they modify state
	// and we're using a value receiver. They will be called when the first
	// tick or window size message arrives.

	// Don't load files initially - wait until the file browser is selected
	return tea.Batch(tickEvery(), m.spinner.Tick)
}
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	// Track if we've already routed navigation for dictionary builder this tick
	routedDictNav := false
	// Track if we've already routed navigation for pipeline this tick
	routedPipelineNav := false

	switch msg := msg.(type) {
	case DirectoryLoadedMsg:
		// Handle async directory load completion
		// fmt.Printf("Update: Received DirectoryLoadedMsg for path: %s, files: %d\n", msg.Path, len(msg.Files))
		// Always clear the loading state, even for stale responses,
		// so we never leave the UI stuck showing a loading indicator.
		m.isLoadingDir = false
		if msg.RequestID == m.dirRequestID {
			m.HandleDirectoryLoaded(msg)
			return m, nil
		}
		// Stale response; ignore payload but ensure loading indicator is off
		return m, nil

	case DirectoryLoadingMsg:
		// Directory is being loaded
		m.isLoadingDir = true
		return m, nil

	case tea.WindowSizeMsg:
		// Store original terminal dimensions with safety margin
		// CRITICAL: Subtract 1 from dimensions to avoid terminal edge issues
		m.width = utils.Max(10, msg.Width-1)
		m.height = utils.Max(5, msg.Height-1)

		// Calculate safe dimensions accounting for borders and padding
		safeWidth := utils.Max(10, m.width-2)
		safeHeight := utils.Max(5, m.height-2)

		m.updateTables()
		m.updateViewportSize()
		if !m.ready {
			// Use safe dimensions for viewports with additional margin
			m.viewport = viewport.New(utils.Max(20, safeWidth-4), utils.Max(5, safeHeight-12))
			m.viewport.HighPerformanceRendering = false
			m.previewViewport = viewport.New(utils.Max(15, safeWidth/3-2), utils.Max(5, safeHeight-12))
			m.previewViewport.HighPerformanceRendering = false
			m.ready = true
		}

	case tickMsg:
		m.updateSystemInfo()
		m.lastUpdate = time.Time(msg)

		return m, tickEvery()

	case buildTickMsg:
		// Keep the UI refreshing during real builds and pipeline runs
		if m.dictBuilderState == DictStateBuilding || m.dictBuilderState == DictStateBuildingFullLogs {
			// Update elapsed time
			if !m.buildState.StartTime.IsZero() {
				m.buildState.ElapsedTime = time.Since(m.buildState.StartTime)
			}
			// Pull new log entries from the build logger and update UI state
			if m.buildLogger != nil {
				entries := m.buildLogger.GetEntries()
				// Always update viewport even if no new entries
				if m.lastLogIndex < len(entries) {
					// Append new lines and update progress/stage
					for i := m.lastLogIndex; i < len(entries); i++ {
						e := entries[i]
						ts := e.Timestamp.Format("15:04:05")
						progress := ""
						if e.Progress >= 0 {
							m.buildProgress = e.Progress
							m.buildState.Progress = e.Progress
							progress = fmt.Sprintf(" [%.1f%%]", e.Progress*100)
						}
						if e.Stage != "" {
							m.buildState.Stage = e.Stage
						}
						// Update current step with the message content
						if e.Message != "" {
							m.buildState.CurrentStep = e.Message
							// Extract processed items from messages like "Processed 12345 rows"
							if strings.Contains(e.Message, "Processed ") && strings.Contains(e.Message, " rows") {
								parts := strings.Fields(e.Message)
								for i, part := range parts {
									if part == "Processed" && i+1 < len(parts) {
										if num, err := strconv.Atoi(parts[i+1]); err == nil {
											m.buildState.ProcessedItems = num
										}
										break
									}
								}
							}
						}
						if e.Level.String() == "ERROR" {
							// Track error summary and set buildError once
							if m.buildError == nil {
								m.buildError = fmt.Errorf(e.Message)
							}
							m.buildState.Errors = append(m.buildState.Errors, e.Message)
						}
						line := fmt.Sprintf("[%s] %s %s: %s%s", ts, e.Level.String(), e.Stage, e.Message, progress)
						m.buildLogs = append(m.buildLogs, line)
					}
					m.lastLogIndex = len(entries)
					// Consider build complete when we reached 'done' stage with 100%
					if m.buildState.Stage == "done" && m.buildProgress >= 1.0 {
						m.buildState.IsComplete = true
					}
					// Clamp log size and refresh viewport
					if len(m.buildLogs) > 2000 {
						m.buildLogs = m.buildLogs[len(m.buildLogs)-2000:]
					}
					m.buildViewport.SetContent(strings.Join(m.buildLogs, "\n"))
					m.buildViewport.GotoBottom()
				}
			}
			// Continue polling
			return m, buildTickEvery()
		}
		// Pipeline run polling
		if m.pipelineState == PipelineRunning {
			if !m.buildState.StartTime.IsZero() {
				m.buildState.ElapsedTime = time.Since(m.buildState.StartTime)
			}
			if m.buildLogger != nil {
				entries := m.buildLogger.GetEntries()
				if m.lastLogIndex < len(entries) {
					for i := m.lastLogIndex; i < len(entries); i++ {
						e := entries[i]
						if e.Progress >= 0 {
							m.buildProgress = e.Progress
							m.buildState.Progress = e.Progress
						}
						if e.Stage != "" {
							m.buildState.Stage = e.Stage
						}
						m.buildLogs = append(m.buildLogs, fmt.Sprintf("[%s] %s %s: %s", e.Timestamp.Format("15:04:05"), e.Level.String(), e.Stage, e.Message))
					}
					m.lastLogIndex = len(entries)
					if m.buildState.Stage == "done" && m.buildProgress >= 1.0 {
						m.buildState.IsComplete = true
					}
					if len(m.buildLogs) > 2000 {
						m.buildLogs = m.buildLogs[len(m.buildLogs)-2000:]
					}
					m.buildViewport.SetContent(strings.Join(m.buildLogs, "\n"))
					m.buildViewport.GotoBottom()
				}
			}
			return m, buildTickEvery()
		}
		return m, nil

	case tea.KeyMsg:
		// Handle global navigation keys first
		switch {
		// Ctrl+C always quits the application
		case key.Matches(msg, m.keys.ForceQuit):
			return m, tea.Quit

		// Tab always cycles through panels
		case key.Matches(msg, m.keys.Tab):
			m.cycleActivePanel(true)
			return m, nil

		// Shift+Tab always cycles backwards through panels
		case key.Matches(msg, m.keys.ShiftTab):
			m.cycleActivePanel(false)
			return m, nil

		// ESC always goes back/cancels current operation
		case key.Matches(msg, m.keys.Back):
			return m.handleBackKey()

		// q goes back to previous menu (not quit app)
		case key.Matches(msg, m.keys.Quit):
			return m.handleQuitKey()

		// Preview toggle
		case key.Matches(msg, m.keys.Preview):
			if m.canShowPreview() {
				m.showPreview = !m.showPreview
				if m.showPreview && m.fileTable.Cursor() < len(m.files) {
					m.loadFilePreview(m.files[m.fileTable.Cursor()])
				}
				m.updateTables()
			}
			return m, nil
		}

		// Route to specific handlers based on active panel and context
		if m.activePanel == MainPanel {
			// Dictionary builder navigation
			if m.cursor < len(m.sidebarItems) &&
				m.sidebarItems[m.cursor].Action == "dictionary_builder_view" {
				newModel, cmd := m.HandleDictNavigation(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				m = newModel
				m.EnsureTableFocus()
				routedDictNav = true
				return m, tea.Batch(cmds...)
			}
			// Pipeline configuration navigation
			if m.cursor < len(m.sidebarItems) &&
				m.sidebarItems[m.cursor].Action == "pipeline" {
				newModel, cmd := m.HandlePipelineNavigation(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				m = newModel
				routedPipelineNav = true
				return m, tea.Batch(cmds...)
			}
			// File browser selection toggle
			if m.cursor < len(m.sidebarItems) && m.sidebarItems[m.cursor].Action == "files" {
				if key.Matches(msg, key.NewBinding(key.WithKeys(" "))) {
					m.toggleSelection()
					return m, nil
				}
			}
		}

		switch {

		case key.Matches(msg, m.keys.Up):
			switch m.activePanel {
			case SidebarPanel:
				// Navigate menu items
				if m.cursor > 0 {
					m.cursor--
				}
			case MainPanel:
				// The table will be updated once at the end
			case SystemPanel:
				// Navigate system panel
				if m.systemCursor > 0 {
					m.systemCursor--
				}
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
				// The table will be updated once at the end
			case SystemPanel:
				// Navigate system panel
				if m.systemCursor < len(m.processes)-1 {
					m.systemCursor++
				}
			case PreviewPanel:
				// Scroll preview content
				m.previewViewport.LineDown(1)
			}

		case key.Matches(msg, m.keys.PageUp):
			switch m.activePanel {
			case MainPanel:
				// Check if we're in file browser mode and handle pagination
				if m.cursor < len(m.sidebarItems) && m.sidebarItems[m.cursor].Action == "files" {
					cmd := m.handleFileBrowserPagination(false)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
				// The table will be updated once at the end
			case SystemPanel:
				m.viewport.ViewUp()
			case PreviewPanel:
				m.previewViewport.ViewUp()
			}

		case key.Matches(msg, m.keys.PageDown):
			switch m.activePanel {
			case MainPanel:
				// Check if we're in file browser mode and handle pagination
				if m.cursor < len(m.sidebarItems) && m.sidebarItems[m.cursor].Action == "files" {
					cmd := m.handleFileBrowserPagination(true)
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
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
					// Clear preview state to avoid spillover between sections
					m.previewReady = false
					m.previewContent = ""
					m.previewViewport.SetContent("")
					// Check if selecting pipeline configuration
					if m.sidebarItems[m.cursor].Action == "pipeline" {
						m.pipelineState = PipelineMainMenu
						if m.pipelineConfig.Name == "" {
							m.initPipelineConfig()
						}
						// Show preview for pipeline by default
						m.showPreview = true
					} else {
						cmd := m.handleMenuAction(m.sidebarItems[m.cursor].Action)
						if cmd != nil {
							cmds = append(cmds, cmd)
						}
					}
				}
			case MainPanel:
				if m.cursor < len(m.sidebarItems) {
					action := m.sidebarItems[m.cursor].Action
					switch action {
					case "pipeline":
						// Handle pipeline configuration actions
						return m.HandlePipelineNavigation(msg)
					case "dictionary_builder_view":
						// Handle dictionary builder actions based on state
						if m.dictBuilderState == DictStateEditingName {
							// Save the dictionary name
							m.dictConfig.Name = m.dictNameInput.Value()
							m.dictBuilderState = DictStateMainMenu
						} else if m.dictBuilderState == DictStateSelectingTUIs {
							// Confirm TUI selection and return to main menu
							m.dictBuilderState = DictStateMainMenu
						} else if m.dictBuilderState == DictStateSelectingVocabs {
							// Confirm vocabulary selection and return to main menu
							m.dictBuilderState = DictStateMainMenu
						} else if m.dictBuilderState == DictStateSelectUMLS {
							// In UMLS selection mode - handle directory navigation
							if m.fileTable.Cursor() < len(m.files) {
								file := m.files[m.fileTable.Cursor()]
								if file.IsDir {
									var targetPath string
									if file.Name == ".." {
										// Go up a directory
										targetPath = filepath.Dir(m.currentPath)
									} else {
										targetPath = filepath.Join(m.currentPath, file.Name)
									}
									rrfFiles := m.detectRRFFiles(targetPath)
									if len(rrfFiles) > 0 && file.Name != ".." {
										// Found RRF files - select this directory
										m.umlsPath = targetPath
										m.rrfFiles = rrfFiles
										m.dictBuilderState = DictStateMainMenu
									} else {
										// Navigate into directory or up
										m.currentPath = targetPath
										m.currentDirPage = 0
										cmd := m.updateFileList()
										if cmd != nil {
											cmds = append(cmds, cmd)
										}
									}
								}
							}
						}
					default:
						// Normal file browser: open directories/files
						cmds = append(cmds, m.handleFileAction())
					}
				}
			case PreviewPanel:
				// No special handling on Enter in preview
			}

			// Handle advanced configuration screen keys
		default:
			if m.activePanel == MainPanel {
				if m.cursor < len(m.sidebarItems) && m.sidebarItems[m.cursor].Action == "dictionary_builder_view" {
					// Advanced configuration states removed - handled in HandleDictNavigation
				}
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update file/dictionary tables when needed. If we already routed nav keys for
	// dictionary builder this tick, avoid double-processing.
	if m.activePanel == MainPanel && !routedDictNav && !routedPipelineNav {
		// Check if we're in dictionary builder mode
		if m.cursor < len(m.sidebarItems) &&
			m.sidebarItems[m.cursor].Action == "dictionary_builder_view" {

			if m.dictBuilderState == DictStateSelectUMLS {
				// Update file table for UMLS selection
				var cmd tea.Cmd
				m.fileTable, cmd = m.fileTable.Update(msg)
				cmds = append(cmds, cmd)

				// Don't update file preview in this mode
			} else if m.dictBuilderState == DictStateEditingName {
				// Update text input for dictionary name
				var cmd tea.Cmd
				m.dictNameInput, cmd = m.dictNameInput.Update(msg)
				cmds = append(cmds, cmd)
			} else if m.dictBuilderState == DictStateSelectingTUIs {
				// Don't pass space key to the table - handle it ourselves
				if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == " " {
					// Space key is handled above in the switch statement
					// Don't pass it to the table
				} else {
					// Update TUI table for other keys
					var cmd tea.Cmd
					m.tuiTable, cmd = m.tuiTable.Update(msg)
					cmds = append(cmds, cmd)
				}
			} else if m.dictBuilderState == DictStateSelectingVocabs {
				// Don't pass space key to the table - handle it ourselves
				if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == " " {
					// Space key is handled above in the switch statement
					// Don't pass it to the table
				} else {
					// Update vocabulary table for other keys
					var cmd tea.Cmd
					m.vocabTable, cmd = m.vocabTable.Update(msg)
					cmds = append(cmds, cmd)
				}
			} else if m.dictBuilderState == DictStateBuilding {
				// Update build viewport for scrolling
				var cmd tea.Cmd
				m.buildViewport, cmd = m.buildViewport.Update(msg)
				cmds = append(cmds, cmd)
			} else if m.dictBuilderState == DictStateViewingDictionaries {
				// Update dictionary viewer table
				var cmd tea.Cmd
				m.dictViewerTable, cmd = m.dictViewerTable.Update(msg)
				cmds = append(cmds, cmd)
			} else {
				// Update dictionary table for menu navigation
				var cmd tea.Cmd
				m.dictTable, cmd = m.dictTable.Update(msg)
				cmds = append(cmds, cmd)
			}
		} else {
			// Normal file browser mode
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
	}

	// Update viewports
	if m.pipelineState == PipelineRunning {
		var cmd tea.Cmd
		m.buildViewport, cmd = m.buildViewport.Update(msg)
		cmds = append(cmds, cmd)
	}
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

// (removed: unused routeDictBuilderNav helper)

func (m *Model) handleMenuAction(action string) tea.Cmd {
	switch action {
	case "system", "processes":
		// Just update the system info if needed
		return nil
	case "files":
		// Load files when file browser is selected
		return m.updateFileList()
	case "dictionary_builder_view":
		// Initialize dictionary builder with main menu
		m.dictBuilderState = DictStateMainMenu
		m.showPreview = true
		return nil
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

	// Update build logs viewport when in building/log views
	if m.dictBuilderState == DictStateBuilding || m.dictBuilderState == DictStateBuildingFullLogs || m.pipelineState == PipelineRunning {
		// Full logs live in the main panel; give a sensible width/height
		m.buildViewport.Width = (m.width*3)/5 - 6 // approximate main panel inner width
		if m.buildViewport.Width < 40 {
			m.buildViewport.Width = 40
		}
		m.buildViewport.Height = m.height - 12
		if m.buildViewport.Height < 5 {
			m.buildViewport.Height = 5
		}
	}
}

// cycleActivePanel cycles through available panels
func (m *Model) cycleActivePanel(forward bool) {
	if forward {
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
	} else {
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
	}
}

// handleBackKey handles ESC key consistently across all contexts
func (m *Model) handleBackKey() (Model, tea.Cmd) {
	switch m.activePanel {
	case MainPanel:
		// Check context-specific back actions
		if m.cursor < len(m.sidebarItems) {
			switch m.sidebarItems[m.cursor].Action {
			case "dictionary_builder_view":
				// Handle dictionary builder back navigation
				if m.dictBuilderState != DictStateMainMenu {
					m.dictBuilderState = DictStateMainMenu
					return *m, nil
				}
			case "pipeline":
				// Respect pipeline-specific ESC behavior instead of file browser default
				switch m.pipelineState {
				case PipelineSelectingInputDirs:
					m.pipelineConfig.InputDirs = m.collectSelectedInputDirs()
					m.pipelineState = PipelineMainMenu
					return *m, nil
				case PipelineSelectingOutputDir:
					// ESC chooses current path as output for quick UX
					m.pipelineConfig.OutputDir = m.currentPath
					m.pipelineState = PipelineMainMenu
					return *m, nil
				default:
					// From any pipeline sub-state, go back to pipeline main; if already at main, go back to sidebar
					if m.pipelineState != PipelineMainMenu {
						m.pipelineState = PipelineMainMenu
						return *m, nil
					}
					m.activePanel = SidebarPanel
					return *m, nil
				}
			default:
				// File browser back navigation
				if m.currentPath != "/" {
					m.currentPath = filepath.Dir(m.currentPath)
					cmd := m.updateFileList()
					return *m, cmd
				}
			}
		}
		// If no specific back action, go to sidebar
		m.activePanel = SidebarPanel
	case PreviewPanel, SystemPanel:
		// Go back to main panel
		m.activePanel = MainPanel
	case SidebarPanel:
		// Already at top level, do nothing
	}
	return *m, nil
}

// handleQuitKey handles 'q' key to go back to menu
func (m *Model) handleQuitKey() (Model, tea.Cmd) {
	// Always go back to sidebar menu
	m.activePanel = SidebarPanel
	return *m, nil
}

// canShowPreview determines if preview panel is available in current context
func (m *Model) canShowPreview() bool {
	if m.cursor >= len(m.sidebarItems) {
		return false
	}

	action := m.sidebarItems[m.cursor].Action
	return action == "dictionary_builder_view" ||
		action == "files"
}
