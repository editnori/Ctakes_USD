package dashboard

import (
	tea "github.com/charmbracelet/bubbletea"
)

// New interactive configuration key handlers with enhanced controls

// Interactive Memory Configuration Key Handler
func (m *Model) handleInteractiveMemoryKeys(key string) tea.Cmd {
	switch key {
	case "up":
		// Increase initial heap
		if m.dictConfig.InitialHeapMB < 3072 {
			m.dictConfig.InitialHeapMB += 256
			if m.dictConfig.InitialHeapMB > 3072 {
				m.dictConfig.InitialHeapMB = 3072
			}
		}
	case "down":
		// Decrease initial heap
		if m.dictConfig.InitialHeapMB > 512 {
			m.dictConfig.InitialHeapMB -= 256
			if m.dictConfig.InitialHeapMB < 512 {
				m.dictConfig.InitialHeapMB = 512
			}
		}
	case "1":
		// Large UMLS preset
		m.dictConfig.InitialHeapMB = 2048
		m.dictConfig.MaxHeapMB = 3072
		m.dictConfig.StackSizeMB = 16
	case "2":
		// Medium build preset
		m.dictConfig.InitialHeapMB = 1024
		m.dictConfig.MaxHeapMB = 2048
		m.dictConfig.StackSizeMB = 8
	case "3":
		// Small build preset
		m.dictConfig.InitialHeapMB = 512
		m.dictConfig.MaxHeapMB = 1024
		m.dictConfig.StackSizeMB = 4
	case "enter":
		return m.returnToMainMenu()
	case "esc":
		return m.returnToMainMenu()
	}
	return nil
}

// Interactive Processing Configuration Key Handler
func (m *Model) handleInteractiveProcessingKeys(key string) tea.Cmd {
	switch key {
	case "up":
		// Increase thread count
		if m.dictConfig.ThreadCount < 16 {
			m.dictConfig.ThreadCount++
		}
	case "down":
		// Decrease thread count
		if m.dictConfig.ThreadCount > 1 {
			m.dictConfig.ThreadCount--
		}
	case "p", "P":
		m.dictConfig.PreserveCase = !m.dictConfig.PreserveCase
	case "h", "H":
		m.dictConfig.HandlePunctuation = !m.dictConfig.HandlePunctuation
	case "1":
		// High performance preset
		m.dictConfig.ThreadCount = 8
		m.dictConfig.BatchSize = 2000
		m.dictConfig.CacheSize = 256
	case "2":
		// Balanced preset
		m.dictConfig.ThreadCount = 4
		m.dictConfig.BatchSize = 1000
		m.dictConfig.CacheSize = 128
	case "3":
		// Memory conservative preset
		m.dictConfig.ThreadCount = 2
		m.dictConfig.BatchSize = 500
		m.dictConfig.CacheSize = 64
	case "enter":
		return m.returnToMainMenu()
	case "esc":
		return m.returnToMainMenu()
	}
	return nil
}

// Interactive Filter Configuration Key Handler
func (m *Model) handleInteractiveFilterKeys(key string) tea.Cmd {
	switch key {
	case "s", "S":
		m.dictConfig.ExcludeSuppressible = !m.dictConfig.ExcludeSuppressible
	case "o", "O":
		m.dictConfig.ExcludeObsolete = !m.dictConfig.ExcludeObsolete
	case "r", "R":
		m.dictConfig.PreferredOnly = !m.dictConfig.PreferredOnly
	case "m", "M":
		m.dictConfig.UseMRRANK = !m.dictConfig.UseMRRANK
	case "c", "C":
		m.dictConfig.CaseSensitive = !m.dictConfig.CaseSensitive
	case "n", "N":
		m.dictConfig.UseNormalization = !m.dictConfig.UseNormalization
	case "d", "D":
		m.dictConfig.Deduplicate = !m.dictConfig.Deduplicate
	case "t", "T":
		m.dictConfig.StripPunctuation = !m.dictConfig.StripPunctuation
	case "w", "W":
		m.dictConfig.CollapseWhitespace = !m.dictConfig.CollapseWhitespace
	case "1":
		m.dictConfig.ExcludeNumericOnly = !m.dictConfig.ExcludeNumericOnly
	case "2":
		m.dictConfig.ExcludePunctOnly = !m.dictConfig.ExcludePunctOnly
	case "up":
		// Increase min term length
		m.dictConfig.MinTermLength++
	case "down":
		// Decrease min term length
		if m.dictConfig.MinTermLength > 1 {
			m.dictConfig.MinTermLength--
		}
	case "enter":
		return m.returnToMainMenu()
	case "esc":
		return m.returnToMainMenu()
	}
	return nil
}

// Interactive Output Configuration Key Handler
func (m *Model) handleInteractiveOutputKeys(key string) tea.Cmd {
	switch key {
	case "b", "B":
		// BSV is required, so don't allow disabling
		if !m.dictConfig.EmitBSV {
			m.dictConfig.EmitBSV = true
		}
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
	case "1":
		// Clinical preset - enable databases and pipeline files
		m.dictConfig.BuildHSQLDB = true
		m.dictConfig.BuildLucene = true
		m.dictConfig.UseRareWords = true
		m.dictConfig.EmitDescriptor = true
		m.dictConfig.EmitPipeline = true
		m.dictConfig.EmitManifest = true
	case "2":
		// Minimal preset - only BSV
		m.dictConfig.BuildHSQLDB = false
		m.dictConfig.BuildLucene = false
		m.dictConfig.UseRareWords = false
		m.dictConfig.EmitTSV = false
		m.dictConfig.EmitJSONL = false
		m.dictConfig.EmitDescriptor = false
		m.dictConfig.EmitPipeline = false
		m.dictConfig.EmitManifest = false
	case "enter":
		return m.returnToMainMenu()
	case "esc":
		return m.returnToMainMenu()
	}
	return nil
}

// Interactive Relationship Configuration Key Handler
func (m *Model) handleInteractiveRelationshipKeys(key string) tea.Cmd {
	switch key {
	case "e", "E":
		m.dictConfig.EnableRelationships = !m.dictConfig.EnableRelationships
		if m.dictConfig.EnableRelationships && len(m.dictConfig.RelationshipTypes) == 0 {
			// Set default relationship types when first enabled
			m.dictConfig.RelationshipTypes = []string{"PAR", "CHD", "RB", "RN", "SY"}
		}
	case "up":
		if m.dictConfig.EnableRelationships && m.dictConfig.RelationshipDepth < 5 {
			m.dictConfig.RelationshipDepth++
		}
	case "down":
		if m.dictConfig.EnableRelationships && m.dictConfig.RelationshipDepth > 0 {
			m.dictConfig.RelationshipDepth--
		}
	case "1", "2", "3", "4", "5", "6":
		if m.dictConfig.EnableRelationships {
			// Toggle specific relationship type
			commonTypes := []string{"PAR", "CHD", "RB", "RN", "SY", "isa"}
			typeIndex := int(key[0] - '1')
			if typeIndex < len(commonTypes) {
				relType := commonTypes[typeIndex]
				m.toggleRelationshipType(relType)
			}
		}
	case "a", "A":
		if m.dictConfig.EnableRelationships {
			// Select all common types
			m.dictConfig.RelationshipTypes = []string{"PAR", "CHD", "RB", "RN", "SY", "isa", "part_of"}
		}
	case "c", "C":
		if m.dictConfig.EnableRelationships {
			// Clear all selections
			m.dictConfig.RelationshipTypes = []string{}
		}
	case "enter":
		return m.returnToMainMenu()
	case "esc":
		return m.returnToMainMenu()
	}
	return nil
}

// Helper function to return to main menu
func (m *Model) returnToMainMenu() tea.Cmd {
	m.dictBuilderState = DictStateConfiguring
	m.initDictOptions()
	tableHeight := m.height - 6
	if tableHeight > 15 {
		tableHeight = 15
	}
	m.updateDictTable(m.width/2, tableHeight)
	m.dictTable.Focus()
	return nil
}

// Helper function to toggle relationship type selection
func (m *Model) toggleRelationshipType(relType string) {
	// Check if type is already selected
	for i, selectedType := range m.dictConfig.RelationshipTypes {
		if selectedType == relType {
			// Remove it
			m.dictConfig.RelationshipTypes = append(
				m.dictConfig.RelationshipTypes[:i],
				m.dictConfig.RelationshipTypes[i+1:]...,
			)
			return
		}
	}
	// Add it
	m.dictConfig.RelationshipTypes = append(m.dictConfig.RelationshipTypes, relType)
}

// Handle directory loaded messages
func (m *Model) HandleDirectoryLoaded(msg DirectoryLoadedMsg) tea.Cmd {
	if msg.Error != nil {
		m.err = msg.Error
		return nil
	}

	m.files = msg.Files
	m.currentPath = msg.Path
	m.isLoadingDir = false
	// Refresh tables so the UI updates immediately after async load
	m.updateTables()
	return nil
}

// Handle dictionary navigation
func (m *Model) HandleDictNavigation(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch m.dictBuilderState {
	case DictStateSelectUMLS:
		// Delegate to UMLS selector handler (defined in dict_umls.go)
		return m.handleUMLSKeys(msg)
	case DictStateEditingName:
		return m.handleDictNameInputKeys(msg)
	case DictStateSelectingTUIs:
		// Delegate to TUI selector handler (defined in dict_tui.go)
		return m.handleTUIKeys(msg)
	case DictStateSelectingVocabs:
		return m.handleVocabSelectionKeys(msg)
	case DictStateViewingDictionaries:
		// Delegate to dictionary viewer handlers
		return m.handleViewerKeys(msg)
	case DictStateBuilding:
		return m.handleBuildKeys(msg)
	case DictStateBuildingFullLogs:
		return m.handleFullLogKeys(msg)
	case DictStateCasedConfig:
		return m.handleCasedKeys(msg)
	case DictStateMemoryConfig, DictStateProcessingConfig, DictStateFilterConfig,
		DictStateOutputConfig, DictStateRelationshipConfig:
		// These advanced screens are handled in update.go via direct key paths.
		// Fall back to no-op here.
		return *m, nil
	default:
		// Main menu
		return m.handleMenuKeys(msg)
	}
}

// Ensure table focus
func (m *Model) EnsureTableFocus() {
	switch m.dictBuilderState {
	case DictStateSelectUMLS:
		m.fileTable.Focus()
	case DictStateSelectingTUIs:
		m.tuiTable.Focus()
	case DictStateSelectingVocabs:
		m.vocabTable.Focus()
	default:
		m.dictTable.Focus()
	}
}

// Placeholder methods for dictionary navigation handlers
// Name input handler: allow editing and commit/cancel
func (m *Model) handleDictNameInputKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.dictConfig.Name = m.dictNameInput.Value()
		m.dictBuilderState = DictStateMainMenu
		return *m, nil
	case "esc":
		m.dictBuilderState = DictStateMainMenu
		return *m, nil
	}
	var cmd tea.Cmd
	m.dictNameInput, cmd = m.dictNameInput.Update(msg)
	return *m, cmd
}

// Vocabulary selection handler: toggle with space, navigate with arrows, confirm/cancel
func (m *Model) handleVocabSelectionKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k", "down", "j":
		var cmd tea.Cmd
		m.vocabTable, cmd = m.vocabTable.Update(msg)
		return *m, cmd
	case " ", "space":
		m.updateVocabTableSelection()
		return *m, nil
	case "enter":
		m.dictBuilderState = DictStateMainMenu
		return *m, nil
	case "esc":
		m.dictBuilderState = DictStateMainMenu
		return *m, nil
	}
	return *m, nil
}
