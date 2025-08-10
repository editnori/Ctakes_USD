package dictionary

// Vocabulary - UMLS source
type Vocabulary struct {
	Code        string
	Name        string
	Description string
	Category    string
}

// GetVocabularies - all UMLS sources
func GetVocabularies() []Vocabulary {
	return []Vocabulary{
		// Core Clinical
		{Code: "SNOMEDCT_US", Name: "SNOMED CT US Edition", Description: "Systematized Nomenclature of Medicine Clinical Terms", Category: "Clinical"},
		{Code: "ICD10CM", Name: "ICD-10-CM", Description: "International Classification of Diseases, 10th Rev, Clinical Modification", Category: "Diagnosis"},
		{Code: "ICD10PCS", Name: "ICD-10-PCS", Description: "ICD-10 Procedure Coding System", Category: "Procedure"},
		{Code: "ICD9CM", Name: "ICD-9-CM", Description: "International Classification of Diseases, 9th Rev, Clinical Modification", Category: "Diagnosis"},

		// Medications
		{Code: "RXNORM", Name: "RxNorm", Description: "Normalized names for clinical drugs", Category: "Medication"},
		{Code: "NDDF", Name: "National Drug Data File", Description: "First DataBank drug database", Category: "Medication"},
		{Code: "NDFRT", Name: "National Drug File Reference Terminology", Description: "VA drug classification", Category: "Medication"},
		{Code: "VANDF", Name: "VA National Drug File", Description: "Veterans Administration drug file", Category: "Medication"},
		{Code: "DRUGBANK", Name: "DrugBank", Description: "Comprehensive drug database", Category: "Medication"},
		{Code: "ATC", Name: "ATC", Description: "Anatomical Therapeutic Chemical Classification", Category: "Medication"},
		{Code: "GS", Name: "Gold Standard", Description: "Gold Standard Drug Database", Category: "Medication"},
		{Code: "MMSL", Name: "Multum MediSource", Description: "Multum drug database", Category: "Medication"},
		{Code: "MMX", Name: "Micromedex", Description: "Micromedex drug database", Category: "Medication"},

		// Laboratory
		{Code: "LNC", Name: "LOINC", Description: "Logical Observation Identifiers Names and Codes", Category: "Laboratory"},
		{Code: "LOINC", Name: "LOINC (alternate)", Description: "Laboratory and clinical observations", Category: "Laboratory"},

		// Procedures
		{Code: "CPT", Name: "CPT", Description: "Current Procedural Terminology", Category: "Procedure"},
		{Code: "HCPCS", Name: "HCPCS", Description: "Healthcare Common Procedure Coding System", Category: "Procedure"},
		{Code: "HCPT", Name: "HCPT", Description: "Healthcare Procedure Coding System", Category: "Procedure"},

		// Specialized
		{Code: "NCI", Name: "NCI Thesaurus", Description: "National Cancer Institute Thesaurus", Category: "Oncology"},
		{Code: "PDQ", Name: "PDQ", Description: "Physician Data Query cancer information", Category: "Oncology"},
		{Code: "ICPC", Name: "ICPC", Description: "International Classification of Primary Care", Category: "Primary Care"},
		{Code: "MEDCIN", Name: "MEDCIN", Description: "Clinical terminology for EHR documentation", Category: "Clinical"},
		{Code: "DSM-5", Name: "DSM-5", Description: "Diagnostic and Statistical Manual of Mental Disorders", Category: "Psychiatry"},
		{Code: "ICNP", Name: "ICNP", Description: "International Classification for Nursing Practice", Category: "Nursing"},
		{Code: "NANDA-I", Name: "NANDA-I", Description: "NANDA International nursing diagnoses", Category: "Nursing"},
		{Code: "NIC", Name: "NIC", Description: "Nursing Interventions Classification", Category: "Nursing"},
		{Code: "NOC", Name: "NOC", Description: "Nursing Outcomes Classification", Category: "Nursing"},

		// Anatomy
		{Code: "FMA", Name: "FMA", Description: "Foundational Model of Anatomy", Category: "Anatomy"},
		{Code: "UWDA", Name: "UWDA", Description: "University of Washington Digital Anatomist", Category: "Anatomy"},
		{Code: "RADLEX", Name: "RadLex", Description: "Radiology Lexicon", Category: "Radiology"},

		// Genetics
		{Code: "HGNC", Name: "HGNC", Description: "HUGO Gene Nomenclature Committee", Category: "Genetics"},
		{Code: "OMIM", Name: "OMIM", Description: "Online Mendelian Inheritance in Man", Category: "Genetics"},
		{Code: "HPO", Name: "HPO", Description: "Human Phenotype Ontology", Category: "Genetics"},
		{Code: "GO", Name: "GO", Description: "Gene Ontology", Category: "Genetics"},

		// Devices
		{Code: "UMD", Name: "UMD", Description: "Universal Medical Device Nomenclature System", Category: "Device"},
		{Code: "MDC", Name: "MDC", Description: "Medical Device Classification", Category: "Device"},
		{Code: "GMDN", Name: "GMDN", Description: "Global Medical Device Nomenclature", Category: "Device"},

		// Administrative
		{Code: "SOP", Name: "SOP", Description: "Source of Payment Typology", Category: "Administrative"},
		{Code: "CDT", Name: "CDT", Description: "Current Dental Terminology", Category: "Dental"},

		// Research
		{Code: "MSH", Name: "MeSH", Description: "Medical Subject Headings", Category: "Literature"},
		{Code: "PSY", Name: "PSY", Description: "Psychological Index Terms", Category: "Psychology"},
		{Code: "AOD", Name: "AOD", Description: "Alcohol and Other Drug Thesaurus", Category: "Substance"},

		// Consumer
		{Code: "MEDLINEPLUS", Name: "MedlinePlus", Description: "Consumer health vocabulary", Category: "Consumer"},
		{Code: "CHV", Name: "CHV", Description: "Consumer Health Vocabulary", Category: "Consumer"},

		// Other Standards
		{Code: "HL7V3.0", Name: "HL7 v3", Description: "Health Level Seven vocabulary", Category: "Standard"},
		{Code: "ISO639", Name: "ISO 639", Description: "Language codes", Category: "Standard"},
		{Code: "MTHICD9", Name: "Metathesaurus ICD9", Description: "Metathesaurus version of ICD9", Category: "Diagnosis"},
		{Code: "MTHICPC2ICD10", Name: "ICPC2-ICD10", Description: "ICPC2 to ICD10 mapping", Category: "Mapping"},
		{Code: "WHO", Name: "WHO", Description: "World Health Organization terminology", Category: "International"},
		{Code: "ICPC2P", Name: "ICPC-2 Plus", Description: "International Classification of Primary Care v2 Plus", Category: "Primary Care"},
	}
}

// Language - UMLS supported
type Language struct {
	Code string
	Name string
}

// GetLanguages - all supported
func GetLanguages() []Language {
	return []Language{
		{Code: "ENG", Name: "English"},
		{Code: "SPA", Name: "Spanish"},
		{Code: "FRE", Name: "French"},
		{Code: "GER", Name: "German"},
		{Code: "ITA", Name: "Italian"},
		{Code: "POR", Name: "Portuguese"},
		{Code: "DUT", Name: "Dutch"},
		{Code: "RUS", Name: "Russian"},
		{Code: "SWE", Name: "Swedish"},
		{Code: "DAN", Name: "Danish"},
		{Code: "NOR", Name: "Norwegian"},
		{Code: "FIN", Name: "Finnish"},
		{Code: "CZE", Name: "Czech"},
		{Code: "HUN", Name: "Hungarian"},
		{Code: "POL", Name: "Polish"},
		{Code: "TUR", Name: "Turkish"},
		{Code: "HEB", Name: "Hebrew"},
		{Code: "JPN", Name: "Japanese"},
		{Code: "CHI", Name: "Chinese"},
		{Code: "KOR", Name: "Korean"},
		{Code: "LAV", Name: "Latvian"},
		{Code: "GRE", Name: "Greek"},
		{Code: "SCR", Name: "Croatian"},
		{Code: "EST", Name: "Estonian"},
		{Code: "BAQ", Name: "Basque"},
		{Code: "UKR", Name: "Ukrainian"},
		{Code: "ARA", Name: "Arabic"},
		{Code: "HIN", Name: "Hindi"},
		{Code: "THA", Name: "Thai"},
	}
}

// TermType - UMLS term type
type TermType struct {
	Code        string
	Name        string
	Description string
}

// GetTermTypes - common types
func GetTermTypes() []TermType {
	return []TermType{
		// Preferred
		{Code: "PT", Name: "Preferred Term", Description: "Preferred form of term"},
		{Code: "PN", Name: "Metathesaurus Preferred Name", Description: "Preferred name in Metathesaurus"},
		{Code: "HT", Name: "Hierarchical Term", Description: "Hierarchical descriptor"},
		{Code: "MTH_PT", Name: "Metathesaurus Preferred Term", Description: "MTH preferred term"},

		// Synonyms
		{Code: "SY", Name: "Synonym", Description: "Designated synonym"},
		{Code: "SYN", Name: "Designated Synonym", Description: "Source asserted synonym"},
		{Code: "SYGB", Name: "British Synonym", Description: "British spelling variant"},
		{Code: "SS", Name: "Synthesized Synonym", Description: "Synthesized synonym"},
		{Code: "XM", Name: "Cross-mapping", Description: "Cross-reference mapping"},

		// Abbreviations
		{Code: "AB", Name: "Abbreviation", Description: "Abbreviated form"},
		{Code: "ACR", Name: "Acronym", Description: "Acronym form"},
		{Code: "AC", Name: "Acronym/Abbreviation", Description: "Either acronym or abbreviation"},
		{Code: "AA", Name: "Attribute Type Abbreviation", Description: "Abbreviated attribute"},
		{Code: "AUN", Name: "Authority Name", Description: "Authority form of name"},

		// Entry
		{Code: "ET", Name: "Entry Term", Description: "Entry vocabulary term"},
		{Code: "EP", Name: "Entry Term, Print", Description: "Print entry term"},
		{Code: "EQ", Name: "Equivalent Name", Description: "Equivalent term"},
		{Code: "ES", Name: "Entry Term, Short Form", Description: "Short form entry"},

		// Full Names
		{Code: "FN", Name: "Full Form", Description: "Full form of descriptor"},
		{Code: "FSY", Name: "Foreign Synonym", Description: "Foreign language synonym"},
		{Code: "FFN", Name: "Full Form of Descriptor", Description: "Complete descriptor"},

		// Scientific
		{Code: "SN", Name: "Scientific Name", Description: "Scientific nomenclature"},
		{Code: "SCN", Name: "Scientific Name", Description: "Scientific term"},
		{Code: "USN", Name: "Unique Scientific Name", Description: "Unique scientific identifier"},

		// Brand/Generic
		{Code: "BN", Name: "Brand Name", Description: "Brand/Trade name"},
		{Code: "GN", Name: "Generic Name", Description: "Generic drug name"},
		{Code: "IN", Name: "Ingredient Name", Description: "Active ingredient"},
		{Code: "PIN", Name: "Preferred Ingredient Name", Description: "Preferred ingredient"},
		{Code: "MIN", Name: "Multiple Ingredient Name", Description: "Multiple ingredients"},

		// Clinical
		{Code: "CD", Name: "Clinical Drug", Description: "Clinical drug name"},
		{Code: "BD", Name: "Brand Drug", Description: "Branded drug"},
		{Code: "DP", Name: "Drug Product", Description: "Drug product name"},
		{Code: "DFG", Name: "Dose Form Group", Description: "Dosage form group"},
		{Code: "DF", Name: "Dose Form", Description: "Dosage form"},

		// Hierarchical
		{Code: "HX", Name: "Hierarchical Context", Description: "Hierarchical context term"},
		{Code: "HS", Name: "Hierarchical Synonym", Description: "Hierarchical synonym"},
		{Code: "HTN", Name: "Hierarchical Term, Narrow", Description: "Narrower hierarchical term"},
		{Code: "HTX", Name: "Hierarchical Term, Expanded", Description: "Expanded hierarchical term"},

		// Other
		{Code: "LC", Name: "Long Common Name", Description: "Long common name"},
		{Code: "LN", Name: "LOINC Name", Description: "LOINC official name"},
		{Code: "MH", Name: "Main Heading", Description: "Main subject heading"},
		{Code: "NM", Name: "Name", Description: "Name of substance"},
		{Code: "OA", Name: "Obsolete Abbreviation", Description: "Obsolete abbreviated form"},
		{Code: "OM", Name: "Obsolete Modifier", Description: "Obsolete modifier term"},
		{Code: "OS", Name: "Obsolete Synonym", Description: "Obsolete synonym"},
		{Code: "PM", Name: "Preferred Modifier", Description: "Preferred modifier term"},
		{Code: "TQ", Name: "Topical Qualifier", Description: "Topical qualifier term"},
		{Code: "TX", Name: "Text", Description: "Text form"},
		{Code: "TMSY", Name: "Tall Man Synonym", Description: "Tall man lettering form"},

		// Variants
		{Code: "LA", Name: "Language Variant", Description: "Language-specific variant"},
		{Code: "LC", Name: "Long Common Name", Description: "Long form common name"},
		{Code: "LO", Name: "Local Term", Description: "Local/regional term"},
		{Code: "LPDN", Name: "Lexical Variant", Description: "Lexical variant form"},

		// Codes
		{Code: "CE", Name: "Code Entry", Description: "Code-based entry"},
		{Code: "CI", Name: "Code Indexed", Description: "Indexed code"},
		{Code: "CN", Name: "Code Name", Description: "Code name form"},
		{Code: "CS", Name: "Code Short", Description: "Short code form"},
	}
}

// GetCommonTermTypes - commonly used
func GetCommonTermTypes() []string {
	return []string{"PT", "SY", "AB", "ACR", "ET", "FN", "BN", "GN"}
}

// GetClinicalVocabularies - clinical text sources
func GetClinicalVocabularies() []string {
	return []string{"SNOMEDCT_US", "RXNORM", "ICD10CM", "LOINC", "CPT"}
}

// GetMedicationVocabularies - medication sources
func GetMedicationVocabularies() []string {
	return []string{"RXNORM", "NDDF", "NDFRT", "VANDF", "DRUGBANK", "ATC"}
}

// GetRadiologyVocabularies - radiology sources
func GetRadiologyVocabularies() []string {
	return []string{"RADLEX", "SNOMEDCT_US", "LOINC"}
}
