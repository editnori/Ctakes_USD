package dashboard

import (
	"fmt"
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

	case buildTickMsg:
		// No simulated progress; just keep the UI refreshing during real builds.
		if m.dictBuilderState == DictStateBuilding {
			return m, buildTickEvery()
		}
		return m, nil

	case tea.KeyMsg:
		// Handle space key first for TUI/Vocab selection - before any other processing
		if msg.String() == " " && m.activePanel == MainPanel &&
			m.cursor < len(m.sidebarItems) &&
			m.sidebarItems[m.cursor].Action == "dictionary_builder_view" {

			if m.dictBuilderState == DictStateSelectingTUIs {
				// Toggle TUI selection
				m.updateTUITableSelection()
				m.initDictOptions() // Update preview
				return m, nil
			} else if m.dictBuilderState == DictStateSelectingVocabs {
				// Toggle vocabulary selection
				m.updateVocabTableSelection()
				m.initDictOptions() // Update preview
				return m, nil
			}
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			// Check if in dictionary builder special states - ESC to cancel
			if m.cursor < len(m.sidebarItems) &&
				m.sidebarItems[m.cursor].Action == "dictionary_builder_view" {

				if m.dictBuilderState == DictStateSelectUMLS ||
					m.dictBuilderState == DictStateEditingName ||
					m.dictBuilderState == DictStateSelectingTUIs ||
					m.dictBuilderState == DictStateSelectingVocabs ||
					m.dictBuilderState == DictStateViewingDictionaries ||
					m.dictBuilderState == DictStateMemoryConfig ||
					m.dictBuilderState == DictStateProcessingConfig ||
					m.dictBuilderState == DictStateFilterConfig ||
					m.dictBuilderState == DictStateOutputConfig ||
					m.dictBuilderState == DictStateRelationshipConfig {
					// Cancel and go back to menu
					m.dictBuilderState = DictStateConfiguring
					m.initDictOptions()
					// Use consistent table height calculation
					tableHeight := m.height - 6
					if tableHeight > 15 {
						tableHeight = 15
					}
					m.updateDictTable(m.width/2, tableHeight)
					// Ensure the table is focused
					m.dictTable.Focus()
					return m, nil
				} else if m.dictBuilderState == DictStateBuilding {
					// Cancel build and go back to menu
					m.dictBuilderState = DictStateConfiguring
					m.buildError = fmt.Errorf("Build cancelled by user")
					m.buildLogs = append(m.buildLogs, "", "=== Build Cancelled ===")
					m.initDictOptions()
					tableHeight := m.height - 6
					if tableHeight > 15 {
						tableHeight = 15
					}
					m.updateDictTable(m.width/2, tableHeight)
					m.dictTable.Focus()
					return m, nil
				}
			}
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
				if m.cursor < len(m.sidebarItems) {
					action := m.sidebarItems[m.cursor].Action
					if action == "files" {
						if m.fileTable.Cursor() < len(m.files) {
							file := m.files[m.fileTable.Cursor()]
							if file.IsDir {
								m.currentPath = filepath.Join(m.currentPath, file.Name)
								m.updateFileList()
								m.updateTables()
							}
						}
					} else if action == "dictionary_builder_view" {
						// Handle dictionary builder actions based on state
						if m.dictBuilderState == DictStateEditingName {
							// Save the dictionary name
							m.dictConfig.Name = m.dictNameInput.Value()
							m.dictBuilderState = DictStateConfiguring
							m.initDictOptions()
							tableHeight := m.height - 6
							if tableHeight > 15 {
								tableHeight = 15
							}
							m.updateDictTable(m.width/2, tableHeight)
							m.dictTable.Focus()
						} else if m.dictBuilderState == DictStateSelectingTUIs {
							// Confirm TUI selection and return to main menu
							m.dictBuilderState = DictStateConfiguring
							m.initDictOptions()
							tableHeight := m.height - 6
							if tableHeight > 15 {
								tableHeight = 15
							}
							m.updateDictTable(m.width/2, tableHeight)
							m.dictTable.Focus()
						} else if m.dictBuilderState == DictStateSelectingVocabs {
							// Confirm vocabulary selection and return to main menu
							m.dictBuilderState = DictStateConfiguring
							m.initDictOptions()
							tableHeight := m.height - 6
							if tableHeight > 15 {
								tableHeight = 15
							}
							m.updateDictTable(m.width/2, tableHeight)
							m.dictTable.Focus()
						} else if m.dictBuilderState == DictStateSelectUMLS {
							// In UMLS selection mode - handle directory navigation
							if m.fileTable.Cursor() < len(m.files) {
								file := m.files[m.fileTable.Cursor()]
								if file.IsDir {
									targetPath := filepath.Join(m.currentPath, file.Name)
									rrfFiles := m.detectRRFFiles(targetPath)
									if len(rrfFiles) > 0 {
										// Found RRF files - select this directory
										m.umlsPath = targetPath
										m.rrfFiles = rrfFiles
										m.dictBuilderState = DictStateConfiguring
										m.initDictOptions()
										// Use consistent table height calculation
										tableHeight := m.height - 6
										if tableHeight > 15 {
											tableHeight = 15
										}
										m.updateDictTable(m.width/2, tableHeight)
										// Ensure the table is focused after returning from UMLS selection
										m.dictTable.Focus()
									} else {
										// Navigate into directory
										m.currentPath = targetPath
										m.updateFileList()
										m.updateTables()
									}
								}
							}
						} else {
							// In main menu or config mode - handle menu actions
							if m.dictTable.Cursor() < len(m.dictOptions) {
								cmd := m.handleDictTableAction(m.dictTable.Cursor())
								if cmd != nil {
									cmds = append(cmds, cmd)
								}
							}
						}
					}
				}
			}

		case key.Matches(msg, m.keys.Back):
			if m.activePanel == MainPanel {
				// Check if in dictionary builder UMLS selection mode
				if m.cursor < len(m.sidebarItems) &&
					m.sidebarItems[m.cursor].Action == "dictionary_builder_view" &&
					m.dictBuilderState == DictStateSelectUMLS {
					// Navigate up in file browser
					if m.currentPath != "/" {
						m.currentPath = filepath.Dir(m.currentPath)
						m.updateFileList()
						m.updateTables()
					}
				} else if m.currentPath != "/" {
					// Normal file browser back navigation
					m.currentPath = filepath.Dir(m.currentPath)
					m.updateFileList()
					m.updateTables()
				}
			}

		// Handle advanced configuration screen keys
		default:
			if keyMsg, ok := msg.(tea.KeyMsg); ok && m.activePanel == MainPanel {
				if m.cursor < len(m.sidebarItems) && m.sidebarItems[m.cursor].Action == "dictionary_builder_view" {
					switch m.dictBuilderState {
					case DictStateConfiguringMemory:
						cmd := m.handleMemoryConfigKeys(keyMsg.String())
						if cmd != nil {
							cmds = append(cmds, cmd)
						}
					case DictStateConfiguringProcessing:
						cmd := m.handleProcessingConfigKeys(keyMsg.String())
						if cmd != nil {
							cmds = append(cmds, cmd)
						}
					case DictStateConfiguringFilters:
						cmd := m.handleFiltersConfigKeys(keyMsg.String())
						if cmd != nil {
							cmds = append(cmds, cmd)
						}
					case DictStateConfiguringOutputs:
						cmd := m.handleOutputsConfigKeys(keyMsg.String())
						if cmd != nil {
							cmds = append(cmds, cmd)
						}
					case DictStateConfiguringRelationships:
						cmd := m.handleRelationshipsConfigKeys(keyMsg.String())
						if cmd != nil {
							cmds = append(cmds, cmd)
						}
					// New interactive configuration states
					case DictStateMemoryConfig:
						cmd := m.handleInteractiveMemoryKeys(keyMsg.String())
						if cmd != nil {
							cmds = append(cmds, cmd)
						}
					case DictStateProcessingConfig:
						cmd := m.handleInteractiveProcessingKeys(keyMsg.String())
						if cmd != nil {
							cmds = append(cmds, cmd)
						}
					case DictStateFilterConfig:
						cmd := m.handleInteractiveFilterKeys(keyMsg.String())
						if cmd != nil {
							cmds = append(cmds, cmd)
						}
					case DictStateOutputConfig:
						cmd := m.handleInteractiveOutputKeys(keyMsg.String())
						if cmd != nil {
							cmds = append(cmds, cmd)
						}
					case DictStateRelationshipConfig:
						cmd := m.handleInteractiveRelationshipKeys(keyMsg.String())
						if cmd != nil {
							cmds = append(cmds, cmd)
						}
					}
				}
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update file table only when it's the active panel
	if m.activePanel == MainPanel {
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
	case "dictionary_builder_view":
    // Initialize dictionary builder with main menu (dashboard-integrated, pink theme)
    m.dictBuilderState = DictStateConfiguring
    m.showPreview = true
    m.initDictOptions()
    tableHeight := m.height - 6
    if tableHeight > 15 {
        tableHeight = 15
    }
    m.updateDictTable(m.width/2, tableHeight)
    m.dictTable.Focus()
    return nil
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

// Memory Configuration Key Handlers
func (m *Model) handleMemoryConfigKeys(key string) tea.Cmd {
	switch key {
	case "1":
		if m.dictConfig.InitialHeapMB > 512 {
			m.dictConfig.InitialHeapMB -= 256
			if m.dictConfig.InitialHeapMB < 512 {
				m.dictConfig.InitialHeapMB = 512
			}
		}
	case "2":
		if m.dictConfig.InitialHeapMB < 3072 {
			m.dictConfig.InitialHeapMB += 256
			if m.dictConfig.InitialHeapMB > 3072 {
				m.dictConfig.InitialHeapMB = 3072
			}
		}
	case "3":
		if m.dictConfig.MaxHeapMB > 512 {
			m.dictConfig.MaxHeapMB -= 256
			if m.dictConfig.MaxHeapMB < 512 {
				m.dictConfig.MaxHeapMB = 512
			}
		}
	case "4":
		if m.dictConfig.MaxHeapMB < 3072 {
			m.dictConfig.MaxHeapMB += 256
			if m.dictConfig.MaxHeapMB > 3072 {
				m.dictConfig.MaxHeapMB = 3072
			}
		}
	case "5":
		if m.dictConfig.StackSizeMB > 1 {
			m.dictConfig.StackSizeMB--
		}
	case "6":
		if m.dictConfig.StackSizeMB < 64 {
			m.dictConfig.StackSizeMB++
		}
	case "enter":
		m.dictBuilderState = DictStateConfiguring
		m.initDictOptions()
		tableHeight := m.height - 6
		if tableHeight > 15 {
			tableHeight = 15
		}
		m.updateDictTable(m.width/2, tableHeight)
	case "esc":
		m.dictBuilderState = DictStateConfiguring
		m.initDictOptions()
		tableHeight := m.height - 6
		if tableHeight > 15 {
			tableHeight = 15
		}
		m.updateDictTable(m.width/2, tableHeight)
	}
	return nil
}

// Processing Configuration Key Handlers
func (m *Model) handleProcessingConfigKeys(key string) tea.Cmd {
	switch key {
	case "1":
		if m.dictConfig.ThreadCount > 1 {
			m.dictConfig.ThreadCount--
		}
	case "2":
		if m.dictConfig.ThreadCount < 16 {
			m.dictConfig.ThreadCount++
		}
	case "3":
		if m.dictConfig.BatchSize > 100 {
			m.dictConfig.BatchSize -= 100
		}
	case "4":
		if m.dictConfig.BatchSize < 10000 {
			m.dictConfig.BatchSize += 100
		}
	case "5":
		if m.dictConfig.CacheSize > 64 {
			m.dictConfig.CacheSize -= 32
		}
	case "6":
		if m.dictConfig.CacheSize < 512 {
			m.dictConfig.CacheSize += 32
		}
	case "7":
		if m.dictConfig.MinWordLength > 1 {
			m.dictConfig.MinWordLength--
		}
	case "8":
		if m.dictConfig.MinWordLength < 10 {
			m.dictConfig.MinWordLength++
		}
	case "9":
		if m.dictConfig.MaxWordLength > 10 {
			m.dictConfig.MaxWordLength -= 10
		}
	case "0":
		if m.dictConfig.MaxWordLength < 256 {
			m.dictConfig.MaxWordLength += 10
		}
	case "p", "P":
		m.dictConfig.PreserveCase = !m.dictConfig.PreserveCase
	case "h", "H":
		m.dictConfig.HandlePunctuation = !m.dictConfig.HandlePunctuation
	case "enter":
		m.dictBuilderState = DictStateConfiguring
		m.initDictOptions()
		tableHeight := m.height - 6
		if tableHeight > 15 {
			tableHeight = 15
		}
		m.updateDictTable(m.width/2, tableHeight)
	case "esc":
		m.dictBuilderState = DictStateConfiguring
		m.initDictOptions()
		tableHeight := m.height - 6
		if tableHeight > 15 {
			tableHeight = 15
		}
		m.updateDictTable(m.width/2, tableHeight)
	}
	return nil
}

// Filter Configuration Key Handlers
func (m *Model) handleFiltersConfigKeys(key string) tea.Cmd {
	switch key {
	case "1":
		if m.dictConfig.MinTermLength > 1 {
			m.dictConfig.MinTermLength--
		}
	case "2":
		m.dictConfig.MinTermLength++
	case "3":
		if m.dictConfig.MaxTermLength > m.dictConfig.MinTermLength {
			m.dictConfig.MaxTermLength -= 5
		}
	case "4":
		m.dictConfig.MaxTermLength += 5
	case "5":
		if m.dictConfig.MinTokens > 1 {
			m.dictConfig.MinTokens--
		}
	case "6":
		m.dictConfig.MinTokens++
	case "7":
		if m.dictConfig.MaxTokens > m.dictConfig.MinTokens {
			m.dictConfig.MaxTokens--
		}
	case "8":
		m.dictConfig.MaxTokens++
	case "s", "S":
		m.dictConfig.ExcludeSuppressible = !m.dictConfig.ExcludeSuppressible
	case "o", "O":
		m.dictConfig.ExcludeObsolete = !m.dictConfig.ExcludeObsolete
	case "c", "C":
		m.dictConfig.CaseSensitive = !m.dictConfig.CaseSensitive
	case "n", "N":
		m.dictConfig.UseNormalization = !m.dictConfig.UseNormalization
	case "r", "R":
		m.dictConfig.UseMRRANK = !m.dictConfig.UseMRRANK
	case "d", "D":
		m.dictConfig.Deduplicate = !m.dictConfig.Deduplicate
	case "p", "P":
		m.dictConfig.PreferredOnly = !m.dictConfig.PreferredOnly
	case "t", "T":
		m.dictConfig.StripPunctuation = !m.dictConfig.StripPunctuation
	case "w", "W":
		m.dictConfig.CollapseWhitespace = !m.dictConfig.CollapseWhitespace
	case "m", "M":
		m.dictConfig.ExcludeNumericOnly = !m.dictConfig.ExcludeNumericOnly
	case "u", "U":
		m.dictConfig.ExcludePunctOnly = !m.dictConfig.ExcludePunctOnly
	case "enter":
		m.dictBuilderState = DictStateConfiguring
		m.initDictOptions()
		tableHeight := m.height - 6
		if tableHeight > 15 {
			tableHeight = 15
		}
		m.updateDictTable(m.width/2, tableHeight)
	case "esc":
		m.dictBuilderState = DictStateConfiguring
		m.initDictOptions()
		tableHeight := m.height - 6
		if tableHeight > 15 {
			tableHeight = 15
		}
		m.updateDictTable(m.width/2, tableHeight)
	}
	return nil
}

// Output Configuration Key Handlers
func (m *Model) handleOutputsConfigKeys(key string) tea.Cmd {
	switch key {
	case "b", "B":
		m.dictConfig.EmitBSV = !m.dictConfig.EmitBSV
	case "h", "H":
		m.dictConfig.BuildHSQLDB = !m.dictConfig.BuildHSQLDB
	case "l", "L":
		m.dictConfig.BuildLucene = !m.dictConfig.BuildLucene
	case "r", "R":
		m.dictConfig.UseRareWords = !m.dictConfig.UseRareWords
	case "t", "T":
		m.dictConfig.EmitTSV = !m.dictConfig.EmitTSV
	case "j", "J":
		m.dictConfig.EmitJSONL = !m.dictConfig.EmitJSONL
	case "d", "D":
		m.dictConfig.EmitDescriptor = !m.dictConfig.EmitDescriptor
	case "p", "P":
		m.dictConfig.EmitPipeline = !m.dictConfig.EmitPipeline
	case "m", "M":
		m.dictConfig.EmitManifest = !m.dictConfig.EmitManifest
	case "enter":
		m.dictBuilderState = DictStateConfiguring
		m.initDictOptions()
		tableHeight := m.height - 6
		if tableHeight > 15 {
			tableHeight = 15
		}
		m.updateDictTable(m.width/2, tableHeight)
	case "esc":
		m.dictBuilderState = DictStateConfiguring
		m.initDictOptions()
		tableHeight := m.height - 6
		if tableHeight > 15 {
			tableHeight = 15
		}
		m.updateDictTable(m.width/2, tableHeight)
	}
	return nil
}

// Relationships Configuration Key Handlers
func (m *Model) handleRelationshipsConfigKeys(key string) tea.Cmd {
	switch key {
	case "e", "E":
		m.dictConfig.EnableRelationships = !m.dictConfig.EnableRelationships
		if m.dictConfig.EnableRelationships && len(m.dictConfig.RelationshipTypes) == 0 {
			// Set default relationship types when first enabled
			m.dictConfig.RelationshipTypes = []string{"PAR", "CHD", "RB", "RN", "SY"}
		}
	case "1":
		if m.dictConfig.EnableRelationships && m.dictConfig.RelationshipDepth > 0 {
			m.dictConfig.RelationshipDepth--
		}
	case "2":
		if m.dictConfig.EnableRelationships && m.dictConfig.RelationshipDepth < 5 {
			m.dictConfig.RelationshipDepth++
		}
	case "t", "T":
		// For now, just toggle common relationship types
		if m.dictConfig.EnableRelationships {
			if len(m.dictConfig.RelationshipTypes) == 0 {
				m.dictConfig.RelationshipTypes = []string{"PAR", "CHD", "RB", "RN", "SY"}
			} else {
				m.dictConfig.RelationshipTypes = []string{}
			}
		}
	case "enter":
		m.dictBuilderState = DictStateConfiguring
		m.initDictOptions()
		tableHeight := m.height - 6
		if tableHeight > 15 {
			tableHeight = 15
		}
		m.updateDictTable(m.width/2, tableHeight)
	case "esc":
		m.dictBuilderState = DictStateConfiguring
		m.initDictOptions()
		tableHeight := m.height - 6
		if tableHeight > 15 {
			tableHeight = 15
		}
		m.updateDictTable(m.width/2, tableHeight)
	}
	return nil
}
