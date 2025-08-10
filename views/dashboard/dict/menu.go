package dict

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
)

// Main menu items
type MenuItem struct {
	Label       string
	Description string
	Action      func(*DictController) tea.Cmd
	Enabled     bool
}

// RenderMainMenu renders the main dictionary menu
func RenderMainMenu(dc *DictController, width, height int) string {
	lines := []string{}

	// Header
	header := theme.RenderHeader("Dictionary Builder", width)
	lines = append(lines, header)
	lines = append(lines, theme.RenderDivider(width))

	// Menu items
	menuItems := getMainMenuItems()

	for i, item := range menuItems {
		focused := i == dc.cursor
		icon := theme.RenderIcon("inactive")
		if focused {
			icon = theme.RenderIcon("active")
		}

		content := fmt.Sprintf(" %s  %s", icon, item.Label)
		if item.Description != "" {
			// Add description on same line if there's room
			maxDescWidth := width - len(content) - 5
			if len(item.Description) <= maxDescWidth {
				content += theme.RenderTextDim(" - " + item.Description)
			}
		}

		if focused {
			line := theme.RenderSelection(content, width)
			lines = append(lines, line)
		} else if !item.Enabled {
			line := theme.RenderTextDim(content)
			lines = append(lines, line)
		} else {
			line := theme.RenderText(content)
			lines = append(lines, line)
		}
	}

	// Status section
	lines = append(lines, "")
	lines = append(lines, theme.RenderTextBold("Current Configuration:"))
	lines = append(lines, renderConfigStatus(dc))

	// Help
	lines = append(lines, "")
	helpItems := []string{"[Enter] Select", "[Q] Quit", "[Tab] Switch Panel"}
	help := theme.RenderHelpBar(helpItems, width)
	lines = append(lines, help)

	return strings.Join(lines, "\n")
}

func getMainMenuItems() []MenuItem {
	return []MenuItem{
		{
			Label:       "Select UMLS Data",
			Description: "Choose UMLS metathesaurus files",
			Enabled:     true,
		},
		{
			Label:       "Dictionary Name",
			Description: "Set name and description",
			Enabled:     true,
		},
		{
			Label:       "Semantic Types (TUIs)",
			Description: "Choose medical concept types",
			Enabled:     true,
		},
		{
			Label:       "Vocabularies",
			Description: "Select source vocabularies",
			Enabled:     true,
		},
		{
			Label:       "Memory Settings",
			Description: "Configure memory usage",
			Enabled:     true,
		},
		{
			Label:       "Processing Settings",
			Description: "Set threading and batch options",
			Enabled:     true,
		},
		{
			Label:       "Filter Settings",
			Description: "Configure term filtering",
			Enabled:     true,
		},
		{
			Label:       "Output Settings",
			Description: "Choose output format and options",
			Enabled:     true,
		},
		{
			Label:       "Relationship Settings",
			Description: "Configure concept relationships",
			Enabled:     true,
		},
		{
			Label:       "Build Dictionary",
			Description: "Start dictionary creation",
			Enabled:     true,
		},
		{
			Label:       "View Built Dictionaries",
			Description: "Browse existing dictionaries",
			Enabled:     true,
		},
	}
}

func renderConfigStatus(dc *DictController) string {
	config := dc.config
	status := []string{}

	// UMLS Path
	umlsStatus := "Not selected"
	if config.UMLSPath != "" {
		umlsStatus = theme.RenderStatus("success", config.UMLSPath)
	} else {
		umlsStatus = theme.RenderStatus("warning", "Not selected")
	}
	status = append(status, "  UMLS Data: "+umlsStatus)

	// Name
	nameStatus := "Not set"
	if config.Name != "" {
		nameStatus = theme.RenderStatus("success", config.Name)
	} else {
		nameStatus = theme.RenderStatus("warning", "Not set")
	}
	status = append(status, "  Dictionary Name: "+nameStatus)

	// TUIs
	tuiStatus := fmt.Sprintf("%d selected", len(config.SelectedTUIs))
	if len(config.SelectedTUIs) == 0 {
		tuiStatus = theme.RenderStatus("warning", "None selected")
	} else {
		tuiStatus = theme.RenderStatus("success", tuiStatus)
	}
	status = append(status, "  Semantic Types: "+tuiStatus)

	// Vocabularies
	vocabStatus := fmt.Sprintf("%d selected", len(config.SelectedVocabs))
	if len(config.SelectedVocabs) == 0 {
		vocabStatus = theme.RenderStatus("warning", "None selected")
	} else {
		vocabStatus = theme.RenderStatus("success", vocabStatus)
	}
	status = append(status, "  Vocabularies: "+vocabStatus)

	// Ready to build?
	readyToBuild := config.UMLSPath != "" && config.Name != "" &&
		len(config.SelectedTUIs) > 0 && len(config.SelectedVocabs) > 0

	if readyToBuild {
		status = append(status, "")
		status = append(status, theme.RenderStatus("success", "Ready to build!"))
	} else {
		status = append(status, "")
		status = append(status, theme.RenderStatus("warning", "Complete configuration to build"))
	}

	return strings.Join(status, "\n")
}

// HandleMainMenuUpdate handles key events for the main menu
func HandleMainMenuUpdate(dc *DictController, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if dc.cursor > 0 {
				dc.cursor--
			}
		case "down", "j":
			menuItems := getMainMenuItems()
			if dc.cursor < len(menuItems)-1 {
				dc.cursor++
			}
		case "enter":
			return handleMenuSelection(dc)
		case "q", "esc":
			return tea.Quit
		}
	}
	return nil
}

func handleMenuSelection(dc *DictController) tea.Cmd {
	switch dc.cursor {
	case 0: // Select UMLS Data
		dc.SetState(DictStateSelectUMLS)
	case 1: // Dictionary Name
		dc.SetState(DictStateEditingName)
	case 2: // Semantic Types (TUIs)
		dc.SetState(DictStateSelectingTUIs)
	case 3: // Vocabularies
		dc.SetState(DictStateSelectingVocabs)
	case 4: // Memory Settings
		dc.SetState(DictStateMemoryConfig)
	case 5: // Processing Settings
		dc.SetState(DictStateProcessingConfig)
	case 6: // Filter Settings
		dc.SetState(DictStateFilterConfig)
	case 7: // Output Settings
		dc.SetState(DictStateOutputConfig)
	case 8: // Relationship Settings
		dc.SetState(DictStateRelationshipConfig)
	case 9: // Build Dictionary
		return startBuildProcess(dc)
	case 10: // View Built Dictionaries
		dc.SetState(DictStateViewingDictionaries)
	}
	return nil
}

func startBuildProcess(dc *DictController) tea.Cmd {
	// Validate configuration first
	config := dc.config
	if config.UMLSPath == "" || config.Name == "" ||
		len(config.SelectedTUIs) == 0 || len(config.SelectedVocabs) == 0 {
		// Show error message or stay on menu
		return nil
	}

	dc.SetState(DictStateBuilding)
	dc.building = true
	dc.buildProgress = 0
	dc.buildLogs = []string{"Starting dictionary build..."}

	// Return command to start actual build process
	return startDictionaryBuildCmd(dc)
}

// Build command placeholder - will be implemented in build.go
func startDictionaryBuildCmd(dc *DictController) tea.Cmd {
	return func() tea.Msg {
		// This will be implemented in the build module
		return BuildStartedMsg{}
	}
}

// Build messages
type BuildStartedMsg struct{}
type BuildProgressMsg struct {
	Progress float64
	Message  string
}
type BuildCompletedMsg struct {
	Success bool
	Error   error
}

// Preset configurations
func ApplyClinicalPreset(dc *DictController) {
	config := dc.config
	config.Name = "Clinical Terms"
	config.Description = "Common clinical and medical terms"
	config.SelectedTUIs = []string{
		"T047", // Disease or Syndrome
		"T184", // Sign or Symptom
		"T061", // Therapeutic or Preventive Procedure
		"T060", // Diagnostic Procedure
		"T121", // Pharmacologic Substance
		"T200", // Clinical Drug
	}
	config.SelectedVocabs = []string{"SNOMEDCT_US", "MSH", "ICD10CM", "ICD10PCS", "RXNORM"}
}

func ApplyMedicationPreset(dc *DictController) {
	config := dc.config
	config.Name = "Medications"
	config.Description = "Drug and medication terms"
	config.SelectedTUIs = []string{
		"T121", // Pharmacologic Substance
		"T200", // Clinical Drug
		"T195", // Antibiotic
		"T109", // Organic Chemical
	}
	config.SelectedVocabs = []string{"RXNORM", "MSH", "SNOMEDCT_US"}
}

func ApplyAnatomyPreset(dc *DictController) {
	config := dc.config
	config.Name = "Anatomy"
	config.Description = "Anatomical and body part terms"
	config.SelectedTUIs = []string{
		"T017", // Anatomical Structure
		"T029", // Body Location or Region
		"T023", // Body Part, Organ, or Organ Component
		"T030", // Body Space or Junction
	}
	config.SelectedVocabs = []string{"SNOMEDCT_US", "MSH", "FMA"}
}
