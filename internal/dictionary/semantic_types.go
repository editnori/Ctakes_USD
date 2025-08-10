package dictionary

// SemanticType represents a UMLS semantic type
type SemanticType struct {
	Code  string
	Name  string
	Group string
}

// GetAllSemanticTypes returns all 134 UMLS semantic types
func GetAllSemanticTypes() []SemanticType {
	return []SemanticType{
		// Organisms
		{Code: "T001", Name: "Organism", Group: "ENTITY"},
		{Code: "T002", Name: "Plant", Group: "ENTITY"},
		{Code: "T003", Name: "Alga", Group: "ENTITY"},
		{Code: "T004", Name: "Fungus", Group: "ENTITY"},
		{Code: "T005", Name: "Virus", Group: "DISORDER"},
		{Code: "T006", Name: "Rickettsia or Chlamydia", Group: "DISORDER"},
		{Code: "T007", Name: "Bacterium", Group: "ENTITY"},
		{Code: "T008", Name: "Animal", Group: "ENTITY"},
		{Code: "T009", Name: "Invertebrate", Group: "ENTITY"},
		{Code: "T010", Name: "Vertebrate", Group: "ENTITY"},
		{Code: "T011", Name: "Amphibian", Group: "ENTITY"},
		{Code: "T012", Name: "Bird", Group: "ENTITY"},
		{Code: "T013", Name: "Fish", Group: "ENTITY"},
		{Code: "T014", Name: "Reptile", Group: "ENTITY"},
		{Code: "T015", Name: "Mammal", Group: "ENTITY"},
		{Code: "T016", Name: "Human", Group: "SUBJECT"},

		// Anatomy
		{Code: "T017", Name: "Anatomical Structure", Group: "ANATOMY"},
		{Code: "T018", Name: "Embryonic Structure", Group: "ANATOMY"},
		{Code: "T019", Name: "Congenital Abnormality", Group: "DISORDER"},
		{Code: "T020", Name: "Acquired Abnormality", Group: "DISORDER"},
		{Code: "T021", Name: "Fully Formed Anatomical Structure", Group: "ANATOMY"},
		{Code: "T022", Name: "Body System", Group: "ANATOMY"},
		{Code: "T023", Name: "Body Part, Organ, or Organ Component", Group: "ANATOMY"},
		{Code: "T024", Name: "Tissue", Group: "ANATOMY"},
		{Code: "T025", Name: "Cell", Group: "ANATOMY"},
		{Code: "T026", Name: "Cell Component", Group: "ANATOMY"},
		{Code: "T028", Name: "Gene or Genome", Group: "FINDING"},
		{Code: "T029", Name: "Body Location or Region", Group: "ANATOMY"},
		{Code: "T030", Name: "Body Space or Junction", Group: "ANATOMY"},
		{Code: "T031", Name: "Body Substance", Group: "FINDING"},
		{Code: "T032", Name: "Organism Attribute", Group: "SUBJECT"},

		// Findings and Disorders
		{Code: "T033", Name: "Finding", Group: "FINDING"},
		{Code: "T034", Name: "Laboratory or Test Result", Group: "LAB"},
		{Code: "T037", Name: "Injury or Poisoning", Group: "DISORDER"},
		{Code: "T038", Name: "Biologic Function", Group: "PHENOMENON"},
		{Code: "T039", Name: "Physiologic Function", Group: "FINDING"},
		{Code: "T040", Name: "Organism Function", Group: "FINDING"},
		{Code: "T041", Name: "Mental Process", Group: "FINDING"},
		{Code: "T042", Name: "Organ or Tissue Function", Group: "FINDING"},
		{Code: "T043", Name: "Cell Function", Group: "FINDING"},
		{Code: "T044", Name: "Molecular Function", Group: "FINDING"},
		{Code: "T045", Name: "Genetic Function", Group: "FINDING"},
		{Code: "T046", Name: "Pathologic Function", Group: "FINDING"},
		{Code: "T047", Name: "Disease or Syndrome", Group: "DISORDER"},
		{Code: "T048", Name: "Mental or Behavioral Dysfunction", Group: "DISORDER"},
		{Code: "T049", Name: "Cell or Molecular Dysfunction", Group: "DISORDER"},
		{Code: "T050", Name: "Experimental Model of Disease", Group: "DISORDER"},

		// Events and Activities
		{Code: "T051", Name: "Event", Group: "EVENT"},
		{Code: "T052", Name: "Activity", Group: "EVENT"},
		{Code: "T053", Name: "Behavior", Group: "FINDING"},
		{Code: "T054", Name: "Social Behavior", Group: "FINDING"},
		{Code: "T055", Name: "Individual Behavior", Group: "FINDING"},
		{Code: "T056", Name: "Daily or Recreational Activity", Group: "FINDING"},
		{Code: "T057", Name: "Occupational Activity", Group: "EVENT"},
		{Code: "T058", Name: "Health Care Activity", Group: "PROCEDURE"},
		{Code: "T059", Name: "Laboratory Procedure", Group: "PROCEDURE"},
		{Code: "T060", Name: "Diagnostic Procedure", Group: "PROCEDURE"},
		{Code: "T061", Name: "Therapeutic or Preventive Procedure", Group: "PROCEDURE"},
		{Code: "T062", Name: "Research Activity", Group: "PROCEDURE"},
		{Code: "T063", Name: "Molecular Biology Research Technique", Group: "PROCEDURE"},
		{Code: "T064", Name: "Governmental or Regulatory Activity", Group: "EVENT"},
		{Code: "T065", Name: "Educational Activity", Group: "PROCEDURE"},
		{Code: "T066", Name: "Machine Activity", Group: "PROCEDURE"},

		// Phenomena
		{Code: "T067", Name: "Phenomenon or Process", Group: "PHENOMENON"},
		{Code: "T068", Name: "Human-caused Phenomenon or Process", Group: "PHENOMENON"},
		{Code: "T069", Name: "Environmental Effect of Humans", Group: "PHENOMENON"},
		{Code: "T070", Name: "Natural Phenomenon or Process", Group: "PHENOMENON"},

		// Entities and Objects
		{Code: "T071", Name: "Entity", Group: "ENTITY"},
		{Code: "T072", Name: "Physical Object", Group: "ENTITY"},
		{Code: "T073", Name: "Manufactured Object", Group: "DEVICE"},
		{Code: "T074", Name: "Medical Device", Group: "DEVICE"},
		{Code: "T075", Name: "Research Device", Group: "DEVICE"},
		{Code: "T077", Name: "Conceptual Entity", Group: "FINDING"},
		{Code: "T078", Name: "Idea or Concept", Group: "FINDING"},
		{Code: "T079", Name: "Temporal Concept", Group: "TIME"},
		{Code: "T080", Name: "Qualitative Concept", Group: "MODIFIER"},
		{Code: "T081", Name: "Quantitative Concept", Group: "LAB_MODIFIER"},
		{Code: "T082", Name: "Spatial Concept", Group: "MODIFIER"},
		{Code: "T083", Name: "Geographic Area", Group: "ENTITY"},
		{Code: "T085", Name: "Molecular Sequence", Group: "FINDING"},
		{Code: "T086", Name: "Nucleotide Sequence", Group: "FINDING"},
		{Code: "T087", Name: "Amino Acid Sequence", Group: "DRUG"},
		{Code: "T088", Name: "Carbohydrate Sequence", Group: "DRUG"},
		{Code: "T089", Name: "Regulation or Law", Group: "ENTITY"},

		// Organizations and Groups
		{Code: "T090", Name: "Occupation or Discipline", Group: "SUBJECT"},
		{Code: "T091", Name: "Biomedical Occupation or Discipline", Group: "TITLE"},
		{Code: "T092", Name: "Organization", Group: "ENTITY"},
		{Code: "T093", Name: "Health Care Related Organization", Group: "ENTITY"},
		{Code: "T094", Name: "Professional Society", Group: "ENTITY"},
		{Code: "T095", Name: "Self-help or Relief Organization", Group: "ENTITY"},
		{Code: "T096", Name: "Group", Group: "SUBJECT"},
		{Code: "T097", Name: "Professional or Occupational Group", Group: "SUBJECT"},
		{Code: "T098", Name: "Population Group", Group: "SUBJECT"},
		{Code: "T099", Name: "Family Group", Group: "SUBJECT"},
		{Code: "T100", Name: "Age Group", Group: "SUBJECT"},
		{Code: "T101", Name: "Patient or Disabled Group", Group: "SUBJECT"},
		{Code: "T102", Name: "Group Attribute", Group: "SUBJECT"},

		// Chemicals and Drugs
		{Code: "T103", Name: "Chemical", Group: "DRUG"},
		{Code: "T104", Name: "Chemical Viewed Structurally", Group: "DRUG"},
		{Code: "T109", Name: "Organic Chemical", Group: "DRUG"},
		{Code: "T110", Name: "Steroid", Group: "DRUG"},
		{Code: "T111", Name: "Eicosanoid", Group: "ENTITY"},
		{Code: "T114", Name: "Nucleic Acid, Nucleoside, or Nucleotide", Group: "DRUG"},
		{Code: "T115", Name: "Organophosphorous Compound", Group: "DRUG"},
		{Code: "T116", Name: "Amino Acid, Peptide, or Protein", Group: "DRUG"},
		{Code: "T118", Name: "Carbohydrate", Group: "DRUG"},
		{Code: "T119", Name: "Lipid", Group: "DRUG"},
		{Code: "T120", Name: "Chemical Viewed Functionally", Group: "DRUG"},
		{Code: "T121", Name: "Pharmacologic Substance", Group: "DRUG"},
		{Code: "T122", Name: "Biomedical or Dental Material", Group: "DRUG"},
		{Code: "T123", Name: "Biologically Active Substance", Group: "DRUG"},
		{Code: "T124", Name: "Neuroreactive Substance or Biogenic Amine", Group: "DRUG"},
		{Code: "T125", Name: "Hormone", Group: "DRUG"},
		{Code: "T126", Name: "Enzyme", Group: "DRUG"},
		{Code: "T127", Name: "Vitamin", Group: "DRUG"},
		{Code: "T129", Name: "Immunologic Factor", Group: "DRUG"},
		{Code: "T130", Name: "Indicator, Reagent, or Diagnostic Aid", Group: "DRUG"},
		{Code: "T131", Name: "Hazardous or Poisonous Substance", Group: "DRUG"},

		// Additional Types
		{Code: "T167", Name: "Substance", Group: "DRUG"},
		{Code: "T168", Name: "Food", Group: "DRUG"},
		{Code: "T169", Name: "Functional Concept", Group: "FINDING"},
		{Code: "T170", Name: "Intellectual Product", Group: "FINDING"},
		{Code: "T171", Name: "Language", Group: "ENTITY"},
		{Code: "T184", Name: "Sign or Symptom", Group: "FINDING"},
		{Code: "T185", Name: "Classification", Group: "FINDING"},
		{Code: "T190", Name: "Anatomical Abnormality", Group: "DISORDER"},
		{Code: "T191", Name: "Neoplastic Process", Group: "DISORDER"},
		{Code: "T192", Name: "Receptor", Group: "FINDING"},
		{Code: "T194", Name: "Archaeon", Group: "ENTITY"},
		{Code: "T195", Name: "Antibiotic", Group: "DRUG"},
		{Code: "T196", Name: "Element, Ion, or Isotope", Group: "DRUG"},
		{Code: "T197", Name: "Inorganic Chemical", Group: "DRUG"},
		{Code: "T200", Name: "Clinical Drug", Group: "DRUG"},
		{Code: "T201", Name: "Clinical Attribute", Group: "CLINICAL_ATTRIBUTE"},
		{Code: "T203", Name: "Drug Delivery Device", Group: "DEVICE"},
		{Code: "T204", Name: "Eukaryote", Group: "ENTITY"},
	}
}

// SemanticGroup represents a group of related semantic types
type SemanticGroup struct {
	Name  string
	Code  string
	Types []SemanticType
}

// GetSemanticGroups returns semantic types organized by groups
func GetSemanticGroups() []SemanticGroup {
	allTypes := GetAllSemanticTypes()

	// Group types by their semantic group
	groupMap := make(map[string][]SemanticType)
	for _, t := range allTypes {
		groupMap[t.Group] = append(groupMap[t.Group], t)
	}

	// Create organized groups for UI display
	return []SemanticGroup{
		{
			Name:  "Anatomy",
			Code:  "ANATOMY",
			Types: groupMap["ANATOMY"],
		},
		{
			Name:  "Chemicals & Drugs",
			Code:  "DRUG",
			Types: groupMap["DRUG"],
		},
		{
			Name:  "Disorders",
			Code:  "DISORDER",
			Types: groupMap["DISORDER"],
		},
		{
			Name:  "Procedures",
			Code:  "PROCEDURE",
			Types: groupMap["PROCEDURE"],
		},
		{
			Name:  "Findings",
			Code:  "FINDING",
			Types: groupMap["FINDING"],
		},
		{
			Name:  "Devices",
			Code:  "DEVICE",
			Types: groupMap["DEVICE"],
		},
		{
			Name:  "Laboratory",
			Code:  "LAB",
			Types: groupMap["LAB"],
		},
		{
			Name:  "Subjects & Groups",
			Code:  "SUBJECT",
			Types: groupMap["SUBJECT"],
		},
		{
			Name:  "Events",
			Code:  "EVENT",
			Types: groupMap["EVENT"],
		},
		{
			Name:  "Entities",
			Code:  "ENTITY",
			Types: groupMap["ENTITY"],
		},
		{
			Name:  "Phenomena",
			Code:  "PHENOMENON",
			Types: groupMap["PHENOMENON"],
		},
		{
			Name:  "Modifiers",
			Code:  "MODIFIER",
			Types: append(groupMap["MODIFIER"], groupMap["LAB_MODIFIER"]...),
		},
		{
			Name:  "Time",
			Code:  "TIME",
			Types: groupMap["TIME"],
		},
		{
			Name:  "Clinical Attributes",
			Code:  "CLINICAL_ATTRIBUTE",
			Types: groupMap["CLINICAL_ATTRIBUTE"],
		},
		{
			Name:  "Titles",
			Code:  "TITLE",
			Types: groupMap["TITLE"],
		},
	}
}

// GetClinicalTUIs returns TUIs commonly used for clinical text
func GetClinicalTUIs() []string {
	return []string{
		"T017", // Anatomical Structure
		"T023", // Body Part, Organ, or Organ Component
		"T029", // Body Location or Region
		"T033", // Finding
		"T034", // Laboratory or Test Result
		"T047", // Disease or Syndrome
		"T048", // Mental or Behavioral Dysfunction
		"T059", // Laboratory Procedure
		"T060", // Diagnostic Procedure
		"T061", // Therapeutic or Preventive Procedure
		"T121", // Pharmacologic Substance
		"T184", // Sign or Symptom
		"T191", // Neoplastic Process
		"T200", // Clinical Drug
	}
}

// GetMedicationTUIs returns TUIs for medication extraction
func GetMedicationTUIs() []string {
	return []string{
		"T103", // Chemical
		"T109", // Organic Chemical
		"T110", // Steroid
		"T114", // Nucleic Acid, Nucleoside, or Nucleotide
		"T116", // Amino Acid, Peptide, or Protein
		"T121", // Pharmacologic Substance
		"T122", // Biomedical or Dental Material
		"T125", // Hormone
		"T126", // Enzyme
		"T127", // Vitamin
		"T129", // Immunologic Factor
		"T195", // Antibiotic
		"T200", // Clinical Drug
	}
}

// GetRadiologyTUIs returns TUIs for radiology reports
func GetRadiologyTUIs() []string {
	return []string{
		"T017", // Anatomical Structure
		"T023", // Body Part, Organ, or Organ Component
		"T029", // Body Location or Region
		"T030", // Body Space or Junction
		"T033", // Finding
		"T046", // Pathologic Function
		"T060", // Diagnostic Procedure
		"T190", // Anatomical Abnormality
		"T191", // Neoplastic Process
	}
}

// GetMinimalTUIs returns minimal TUIs for fast processing
func GetMinimalTUIs() []string {
	return []string{
		"T047", // Disease or Syndrome
		"T184", // Sign or Symptom
		"T200", // Clinical Drug
	}
}

// GetProcedureTUIs returns TUIs for procedure extraction
func GetProcedureTUIs() []string {
	return []string{
		"T061", // Therapeutic or Preventive Procedure
		"T060", // Diagnostic Procedure
		"T059", // Laboratory Procedure
		"T058", // Health Care Activity
		"T033", // Finding
	}
}

// GetDiagnosisTUIs returns TUIs for diagnosis and disease extraction
func GetDiagnosisTUIs() []string {
	return []string{
		"T047", // Disease or Syndrome
		"T048", // Mental or Behavioral Dysfunction
		"T191", // Neoplastic Process
		"T190", // Anatomical Abnormality
		"T019", // Congenital Abnormality
		"T037", // Injury or Poisoning
	}
}

// GetLaboratoryTUIs returns TUIs for laboratory and test extraction
func GetLaboratoryTUIs() []string {
	return []string{
		"T059", // Laboratory Procedure
		"T034", // Laboratory or Test Result
		"T081", // Quantitative Concept
		"T033", // Finding
		"T201", // Clinical Attribute
	}
}
