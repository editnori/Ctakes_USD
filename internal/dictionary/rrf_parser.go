package dictionary

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RRFParser handles parsing of UMLS RRF files
type RRFParser struct {
	MetaPath string
	Files    map[string]string
}

// ConceptInfo represents a UMLS concept from MRCONSO.RRF
type ConceptInfo struct {
	CUI      string // Concept Unique Identifier
	LAT      string // Language
	TS       string // Term status
	LUI      string // Lexical Unique Identifier
	STT      string // String type
	SUI      string // String Unique Identifier
	ISPREF   string // Preferred term indicator
	AUI      string // Atom Unique Identifier
	SAUI     string // Source atom identifier
	SCUI     string // Source concept identifier
	SDUI     string // Source descriptor identifier
	SAB      string // Source abbreviation (vocabulary)
	TTY      string // Term type
	CODE     string // Code in source vocabulary
	STR      string // String/term text
	SRL      string // Source restriction level
	SUPPRESS string // Suppression status
	CVF      string // Content view flag
}

// SemanticTypeInfo represents semantic type info from MRSTY.RRF
type SemanticTypeInfo struct {
	CUI  string // Concept Unique Identifier
	TUI  string // Type Unique Identifier
	STN  string // Semantic type tree number
	STY  string // Semantic type name
	ATUI string // Attribute type unique identifier
	CVF  string // Content view flag
}

// VocabularyInfo represents vocabulary source info from MRSAB.RRF
type VocabularyInfo struct {
	VCUI   string // Root source concept identifier
	RCUI   string // Versioned source concept identifier
	VSAB   string // Versioned source abbreviation
	RSAB   string // Root source abbreviation
	SON    string // Source official name
	SF     string // Source family
	SVER   string // Source version
	VSTART string // Valid start date
	VEND   string // Valid end date
	IMETA  string // Insert meta version
	RMETA  string // Remove meta version
	SLC    string // Source license contact
	SCC    string // Source content contact
	SRL    string // Source restriction level
	TFR    string // Term frequency
	CFR    string // CUI frequency
	CXTY   string // Context type
	TTYL   string // Term type list
	ATNL   string // Attribute name list
	LAT    string // Language
	CENC   string // Character encoding
	CURVER string // Current version flag
	SABIN  string // Source in current subset
	SSN    string // Source short name
	SCIT   string // Source citation
}

// NewRRFParser creates a new RRF parser for the given UMLS META directory
func NewRRFParser(metaPath string) (*RRFParser, error) {
	// Verify META directory exists
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("META directory not found: %s", metaPath)
	}

	parser := &RRFParser{
		MetaPath: metaPath,
		Files:    make(map[string]string),
	}

	// Map common RRF files
	rrfFiles := []string{
		"MRCONSO.RRF",
		"MRSTY.RRF",
		"MRSAB.RRF",
		"MRREL.RRF",
		"MRSAT.RRF",
		"MRDEF.RRF",
		"MRRANK.RRF",
		"MRMAP.RRF",
		"MRHIER.RRF",
	}

	for _, file := range rrfFiles {
		path := filepath.Join(metaPath, file)
		if _, err := os.Stat(path); err == nil {
			parser.Files[file] = path
		}
	}

	// Check for required files
	if _, ok := parser.Files["MRCONSO.RRF"]; !ok {
		return nil, fmt.Errorf("required file MRCONSO.RRF not found in %s", metaPath)
	}
	if _, ok := parser.Files["MRSTY.RRF"]; !ok {
		return nil, fmt.Errorf("required file MRSTY.RRF not found in %s", metaPath)
	}

	return parser, nil
}

// GetAvailableFiles returns a list of available RRF files
func (p *RRFParser) GetAvailableFiles() []string {
	files := make([]string, 0, len(p.Files))
	for file := range p.Files {
		files = append(files, file)
	}
	return files
}

// ParseMRCONSO parses MRCONSO.RRF file with optional filters
func (p *RRFParser) ParseMRCONSO(filters MRCONSOFilters, callback func(concept ConceptInfo) error) error {
	path, ok := p.Files["MRCONSO.RRF"]
	if !ok {
		return fmt.Errorf("MRCONSO.RRF not found")
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open MRCONSO.RRF: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Set larger buffer for potentially long lines
	buf := make([]byte, 0, 1024*1024) // 1MB buffer
	scanner.Buffer(buf, 1024*1024*10) // Max 10MB line

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse pipe-delimited fields
		fields := strings.Split(line, "|")
		if len(fields) < 18 {
			// Skip malformed lines
			continue
		}

		concept := ConceptInfo{
			CUI:      fields[0],
			LAT:      fields[1],
			TS:       fields[2],
			LUI:      fields[3],
			STT:      fields[4],
			SUI:      fields[5],
			ISPREF:   fields[6],
			AUI:      fields[7],
			SAUI:     getField(fields, 8),
			SCUI:     getField(fields, 9),
			SDUI:     getField(fields, 10),
			SAB:      fields[11],
			TTY:      fields[12],
			CODE:     fields[13],
			STR:      fields[14],
			SRL:      fields[15],
			SUPPRESS: fields[16],
			CVF:      getField(fields, 17),
		}

		// Apply filters
		if !filters.Accept(concept) {
			continue
		}

		// Call callback
		if err := callback(concept); err != nil {
			return fmt.Errorf("callback error at line %d: %w", lineNum, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading MRCONSO.RRF: %w", err)
	}

	return nil
}

// ParseMRSTY parses MRSTY.RRF file with optional TUI filters
func (p *RRFParser) ParseMRSTY(tuis []string, callback func(semType SemanticTypeInfo) error) error {
	path, ok := p.Files["MRSTY.RRF"]
	if !ok {
		return fmt.Errorf("MRSTY.RRF not found")
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open MRSTY.RRF: %w", err)
	}
	defer file.Close()

	// Create TUI filter map for fast lookup
	tuiFilter := make(map[string]bool)
	for _, tui := range tuis {
		tuiFilter[tui] = true
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if line == "" {
			continue
		}

		fields := strings.Split(line, "|")
		if len(fields) < 6 {
			continue
		}

		semType := SemanticTypeInfo{
			CUI:  fields[0],
			TUI:  fields[1],
			STN:  fields[2],
			STY:  fields[3],
			ATUI: getField(fields, 4),
			CVF:  getField(fields, 5),
		}

		// Apply TUI filter if specified
		if len(tuiFilter) > 0 && !tuiFilter[semType.TUI] {
			continue
		}

		if err := callback(semType); err != nil {
			return fmt.Errorf("callback error at line %d: %w", lineNum, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading MRSTY.RRF: %w", err)
	}

	return nil
}

// ParseMRSAB parses MRSAB.RRF file to get vocabulary information
func (p *RRFParser) ParseMRSAB(callback func(vocab VocabularyInfo) error) error {
	path, ok := p.Files["MRSAB.RRF"]
	if !ok {
		return fmt.Errorf("MRSAB.RRF not found")
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open MRSAB.RRF: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Set larger buffer for potentially long lines
	buf := make([]byte, 0, 1024*1024) // 1MB buffer
	scanner.Buffer(buf, 1024*1024*10) // Max 10MB line

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if line == "" {
			continue
		}

		fields := strings.Split(line, "|")
		if len(fields) < 25 {
			continue
		}

		vocab := VocabularyInfo{
			VCUI:   getField(fields, 0),
			RCUI:   getField(fields, 1),
			VSAB:   fields[2],
			RSAB:   fields[3],
			SON:    fields[4],
			SF:     fields[5],
			SVER:   getField(fields, 6),
			VSTART: getField(fields, 7),
			VEND:   getField(fields, 8),
			IMETA:  getField(fields, 9),
			RMETA:  getField(fields, 10),
			SLC:    getField(fields, 11),
			SCC:    getField(fields, 12),
			SRL:    fields[13],
			TFR:    getField(fields, 14),
			CFR:    getField(fields, 15),
			CXTY:   getField(fields, 16),
			TTYL:   getField(fields, 17),
			ATNL:   getField(fields, 18),
			LAT:    getField(fields, 19),
			CENC:   fields[20],
			CURVER: fields[21],
			SABIN:  fields[22],
			SSN:    fields[23],
			SCIT:   fields[24],
		}

		if err := callback(vocab); err != nil {
			return fmt.Errorf("callback error at line %d: %w", lineNum, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading MRSAB.RRF: %w", err)
	}

	return nil
}

// MRCONSOFilters defines filters for MRCONSO parsing
type MRCONSOFilters struct {
	Languages     []string // Filter by language (e.g., "ENG")
	Vocabularies  []string // Filter by source vocabulary (SAB field)
	TermTypes     []string // Filter by term type (TTY field)
	NoSuppress    bool     // Exclude suppressed terms
	PreferredOnly bool     // Only preferred terms
}

// Accept checks if a concept passes the filters
func (f *MRCONSOFilters) Accept(concept ConceptInfo) bool {
	// Check language filter
	if len(f.Languages) > 0 && !contains(f.Languages, concept.LAT) {
		return false
	}

	// Check vocabulary filter
	if len(f.Vocabularies) > 0 && !contains(f.Vocabularies, concept.SAB) {
		return false
	}

	// Check term type filter
	if len(f.TermTypes) > 0 && !contains(f.TermTypes, concept.TTY) {
		return false
	}

	// Check suppression filter
	if f.NoSuppress && concept.SUPPRESS != "N" {
		return false
	}

	// Check preferred term filter
	if f.PreferredOnly && concept.ISPREF != "Y" {
		return false
	}

	return true
}

// GetStatistics returns statistics about the UMLS data
func (p *RRFParser) GetStatistics() (*UMLSStatistics, error) {
	stats := &UMLSStatistics{
		Vocabularies: make(map[string]int),
		Languages:    make(map[string]int),
		TermTypes:    make(map[string]int),
		TUIs:         make(map[string]int),
	}

	// Count concepts in MRCONSO
	if err := p.ParseMRCONSO(MRCONSOFilters{}, func(c ConceptInfo) error {
		stats.TotalConcepts++
		stats.Vocabularies[c.SAB]++
		stats.Languages[c.LAT]++
		stats.TermTypes[c.TTY]++
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to parse MRCONSO: %w", err)
	}

	// Count semantic types in MRSTY
	if err := p.ParseMRSTY(nil, func(s SemanticTypeInfo) error {
		stats.TotalSemanticTypes++
		stats.TUIs[s.TUI]++
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to parse MRSTY: %w", err)
	}

	return stats, nil
}

// UMLSStatistics holds statistics about UMLS data
type UMLSStatistics struct {
	TotalConcepts      int
	TotalSemanticTypes int
	Vocabularies       map[string]int
	Languages          map[string]int
	TermTypes          map[string]int
	TUIs               map[string]int
}

// Helper functions

func getField(fields []string, index int) string {
	if index < len(fields) {
		return fields[index]
	}
	return ""
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
