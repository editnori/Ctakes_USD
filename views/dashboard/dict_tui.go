package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ctakes-tui/ctakes-tui/internal/dictionary"
	"github.com/ctakes-tui/ctakes-tui/internal/theme"
)

// remove removes an element from a string slice
func remove(slice []string, element string) []string {
	result := make([]string, 0)
	for _, v := range slice {
		if v != element {
			result = append(result, v)
		}
	}
	return result
}

// Semantic Types (TUI) selector - Using proper UMLS semantic types from internal package
func (m *Model) initTUIList() {
	// Get all semantic types from the dictionary package
	semanticTypes := dictionary.GetAllSemanticTypes()

	// Convert to the format expected by the UI
	m.tuiList = make([]string, 0, len(semanticTypes))
	for _, st := range semanticTypes {
		tuiItem := fmt.Sprintf("%s %s", st.Code, st.Name)
		m.tuiList = append(m.tuiList, tuiItem)
	}
}

// GetClinicalTUIPreset loads clinical TUIs preset
func (m *Model) loadClinicalTUIPreset() {
	m.dictConfig.TUIs = dictionary.GetClinicalTUIs()
	sort.Strings(m.dictConfig.TUIs)
}

// GetMedicationTUIPreset loads medication TUIs preset
func (m *Model) loadMedicationTUIPreset() {
	m.dictConfig.TUIs = dictionary.GetMedicationTUIs()
	sort.Strings(m.dictConfig.TUIs)
}

// GetRadiologyTUIPreset loads radiology TUIs preset
func (m *Model) loadRadiologyTUIPreset() {
	m.dictConfig.TUIs = dictionary.GetRadiologyTUIs()
	sort.Strings(m.dictConfig.TUIs)
}

// GetMinimalTUIPreset loads minimal TUIs preset
func (m *Model) loadMinimalTUIPreset() {
	m.dictConfig.TUIs = dictionary.GetMinimalTUIs()
	sort.Strings(m.dictConfig.TUIs)
}

// GetProcedureTUIPreset loads procedure TUIs preset
func (m *Model) loadProcedureTUIPreset() {
	m.dictConfig.TUIs = dictionary.GetProcedureTUIs()
	sort.Strings(m.dictConfig.TUIs)
}

// GetDiagnosisTUIPreset loads diagnosis TUIs preset
func (m *Model) loadDiagnosisTUIPreset() {
	m.dictConfig.TUIs = dictionary.GetDiagnosisTUIs()
	sort.Strings(m.dictConfig.TUIs)
}

// GetLaboratoryTUIPreset loads laboratory TUIs preset
func (m *Model) loadLaboratoryTUIPreset() {
	m.dictConfig.TUIs = dictionary.GetLaboratoryTUIs()
	sort.Strings(m.dictConfig.TUIs)
}

// initVocabList initializes the vocabulary list from UMLS using RRF parser
func (m *Model) initVocabList() {
	// If we have a UMLS path and RRF files, use the parser to get vocabularies
	if m.umlsPath != "" && len(m.rrfFiles) > 0 {
		m.loadVocabulariesFromUMLS()
	} else {
		// Fallback to hardcoded common vocabularies
		m.vocabList = []string{
			"SNOMEDCT_US SNOMED CT US Edition",
			"RXNORM RxNorm",
			"ICD10CM ICD-10-CM",
			"ICD10PCS ICD-10-PCS",
			"ICD9CM ICD-9-CM",
			"CPT Current Procedural Terminology",
			"HCPCS Healthcare Common Procedure Coding System",
			"LOINC Logical Observation Identifiers Names and Codes",
			"NDFRT National Drug File - Reference Terminology",
			"VANDF Veterans Health Administration National Drug File",
			"DRUGBANK DrugBank",
			"NDDF First DataBank NDDF Plus",
			"MMSL Multum MediSource Lexicon",
			"MSH Medical Subject Headings",
			"NCI NCI Thesaurus",
			"OMIM Online Mendelian Inheritance in Man",
			"PDQ Physician Data Query",
			"MEDLINEPLUS MedlinePlus Health Topics",
			"AOD Alcohol and Other Drug Thesaurus",
			"DSM5 Diagnostic and Statistical Manual of Mental Disorders, 5th Edition",
			"GO Gene Ontology",
			"HPO Human Phenotype Ontology",
			"RADLEX RadLex",
			"ICPC2P International Classification of Primary Care, 2nd Edition",
			"ICF International Classification of Functioning, Disability and Health",
			"WHO-ART World Health Organization Adverse Reaction Terminology",
			"MEDDRA Medical Dictionary for Regulatory Activities",
			"ICNP International Classification for Nursing Practice",
			"CCC Clinical Care Classification",
			"NOC Nursing Outcomes Classification",
			"NIC Nursing Interventions Classification",
			"NANDA NANDA International",
			"CDT Current Dental Terminology",
			"SNODENT Systematized Nomenclature of Dentistry",
			"MTHMST Metathesaurus Minimal Spanning Tree",
			"ALT Alternative Billing Codes",
		}
	}
}

// loadVocabulariesFromUMLS loads available vocabularies from UMLS using RRF parser
func (m *Model) loadVocabulariesFromUMLS() {
	// Determine the correct META path
	metaPath := m.umlsPath
	if _, err := os.Stat(filepath.Join(m.umlsPath, "META")); err == nil {
		metaPath = filepath.Join(m.umlsPath, "META")
	}

	parser, err := dictionary.NewRRFParser(metaPath)
	if err != nil {
		// Fallback to default list if parser fails
		m.initVocabList()
		return
	}

	// Create a map to collect unique vocabularies
	vocabMap := make(map[string]string)

	// Parse MRSAB.RRF to get vocabulary information
	err = parser.ParseMRSAB(func(vocab dictionary.VocabularyInfo) error {
		// Only include current versions that are in the current subset
		if vocab.CURVER == "Y" && vocab.SABIN == "Y" {
			key := vocab.RSAB // Root source abbreviation
			name := vocab.SON // Source official name
			if key != "" && name != "" {
				vocabMap[key] = fmt.Sprintf("%s %s", key, name)
			}
		}
		return nil
	})

	if err != nil {
		// Fallback on error
		m.initVocabList()
		return
	}

	// Convert map to sorted slice
	m.vocabList = make([]string, 0, len(vocabMap))
	for _, vocabStr := range vocabMap {
		m.vocabList = append(m.vocabList, vocabStr)
	}
	sort.Strings(m.vocabList)
}

func (m *Model) renderTUISelector(width, height int) string {
	// Use consistent selection header
	header := theme.RenderSelectionHeader(
		"Semantic Types (TUIs)",
		len(m.dictConfig.TUIs),
		len(m.tuiList),
		width,
	)
	headerLines := strings.Count(header, "\n") + 1

	// Calculate visible window
	footerLines := 3 // Footer with help text
	visibleHeight := height - headerLines - footerLines
	if visibleHeight < 5 {
		visibleHeight = 5
	}

	// Calculate scroll position
	startIdx := 0
	if m.tuiCursor >= visibleHeight {
		startIdx = m.tuiCursor - visibleHeight + 1
	}
	endIdx := startIdx + visibleHeight
	if endIdx > len(m.tuiList) {
		endIdx = len(m.tuiList)
	}

	// Build content lines
	lines := []string{header}

	// Render visible items with full-width row highlighting
	for i := startIdx; i < endIdx; i++ {
		item := m.tuiList[i]
		code := strings.Fields(item)[0]
		isChecked := contains(m.dictConfig.TUIs, code)
		isFocused := i == m.tuiCursor

		// Use consistent checkbox rendering
		line := theme.RenderCheckboxItem(item, isChecked, width, isFocused)
		lines = append(lines, line)
	}

	// Add scroll indicator if needed
	if len(m.tuiList) > visibleHeight {
		scrollInfo := theme.RenderScrollIndicator(startIdx, endIdx, len(m.tuiList), width)
		lines = append(lines, scrollInfo)
	}

	// Add footer with consistent help styling and preset shortcuts
	footer := theme.RenderSelectionFooter(width)
	presetHelp := theme.SubtitleStyle.Render("Presets: 1=Clinical | 2=Medication | 3=Radiology | 4=Minimal | 5=Procedure | 6=Diagnosis | 7=Laboratory")
	lines = append(lines, footer, presetHelp)

	// Return properly formatted selection view
	return strings.Join(lines, "\n")
}

func (m *Model) handleTUIKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.tuiCursor > 0 {
			m.tuiCursor--
		}
	case "down", "j":
		if m.tuiCursor < len(m.tuiList)-1 {
			m.tuiCursor++
		}
	case " ", "space":
		if m.tuiCursor < len(m.tuiList) {
			code := strings.Fields(m.tuiList[m.tuiCursor])[0]
			if contains(m.dictConfig.TUIs, code) {
				m.dictConfig.TUIs = remove(m.dictConfig.TUIs, code)
			} else {
				m.dictConfig.TUIs = append(m.dictConfig.TUIs, code)
				sort.Strings(m.dictConfig.TUIs)
			}
		}
	case "a", "A":
		// Select all
		m.dictConfig.TUIs = []string{}
		for _, item := range m.tuiList {
			code := strings.Fields(item)[0]
			m.dictConfig.TUIs = append(m.dictConfig.TUIs, code)
		}
		sort.Strings(m.dictConfig.TUIs)
	case "c", "C":
		// Clear all
		m.dictConfig.TUIs = []string{}
	case "1":
		// Load clinical preset
		m.loadClinicalTUIPreset()
	case "2":
		// Load medication preset
		m.loadMedicationTUIPreset()
	case "3":
		// Load radiology preset
		m.loadRadiologyTUIPreset()
	case "4":
		// Load minimal preset
		m.loadMinimalTUIPreset()
	case "5":
		// Load procedure preset
		m.loadProcedureTUIPreset()
	case "6":
		// Load diagnosis preset
		m.loadDiagnosisTUIPreset()
	case "7":
		// Load laboratory preset
		m.loadLaboratoryTUIPreset()
	case "enter":
		m.dictBuilderState = DictStateMainMenu
	case "esc":
		m.dictBuilderState = DictStateMainMenu
	case "q", "Q":
		// Quit dictionary builder - return to main dashboard
		m.activePanel = SidebarPanel
		m.cursor = 0
	case "p", "P":
		// Switch to pipeline view
		m.activePanel = SidebarPanel
		m.cursor = 5 // Pipeline is item 5 in the sidebar (0-indexed)
	}
	return *m, nil
}
