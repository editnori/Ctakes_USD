package tools.reporting.uima;

import org.apache.ctakes.typesystem.type.refsem.UmlsConcept;
import org.apache.ctakes.typesystem.type.structured.DocumentID;
import org.apache.ctakes.typesystem.type.textsem.IdentifiedAnnotation;
import org.apache.ctakes.typesystem.type.textspan.Segment;
import org.apache.uima.UimaContext;
import org.apache.uima.analysis_component.JCasAnnotator_ImplBase;
import org.apache.uima.analysis_engine.AnalysisEngineProcessException;
import org.apache.uima.fit.descriptor.ConfigurationParameter;
import org.apache.uima.fit.util.JCasUtil;
import org.apache.uima.jcas.JCas;

import java.io.BufferedWriter;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.StandardOpenOption;
import java.util.*;

/**
 * Minimal RxNorm writer for Drug NER runs.
 * Writes one per-document CSV with only RxNorm-coded mentions and minimal columns.
 *
 * Columns:
 *   Document,Begin,End,Text,Section,RxCUI,RxNormName,TUI,SemanticGroup,SemanticTypeLabel
 *
 * Usage in Piper:
 *   add tools.reporting.uima.DrugRxNormCsvWriter SubDirectory=rxnorm_min
 */
public class DrugRxNormCsvWriter extends JCasAnnotator_ImplBase {

    public static final String PARAM_SUBDIR = "SubDirectory";
    @ConfigurationParameter(name = PARAM_SUBDIR, mandatory = false)
    private String subDir = "rxnorm_min";

    private String outputBase;

    @Override
    public void initialize(UimaContext context) {
        Object od = context.getConfigParameterValue("OutputDirectory");
        if (od == null) od = System.getProperty("ctakes.output.dir");
        if (od == null) od = System.getProperty("OUTPUT_DIR");
        outputBase = (od == null) ? "." : od.toString();
        Object sd = context.getConfigParameterValue(PARAM_SUBDIR);
        if (sd != null && !sd.toString().trim().isEmpty()) subDir = sd.toString().trim();
    }

    @Override
    public void process(JCas jCas) throws AnalysisEngineProcessException {
        final String docId = getDocId(jCas);
        final String text = jCas.getDocumentText() == null ? "" : jCas.getDocumentText();
        final List<Segment> segments = new ArrayList<>(JCasUtil.select(jCas, Segment.class));

        // Build rows
        List<String> rows = new ArrayList<>();
        for (IdentifiedAnnotation ia : JCasUtil.select(jCas, IdentifiedAnnotation.class)) {
            UmlsConcept bestRx = pickBestRxNorm(ia);
            if (bestRx == null) continue; // skip mentions without any usable ontology concept

            String section = findSection(segments, ia);
            String tui = nvl(bestRx.getTui());
            String rxCui = nvl(bestRx.getCui());
            String rxName = nvl(bestRx.getPreferredText());
            if (rxName.isEmpty()) rxName = safeText(text, ia.getBegin(), ia.getEnd());

            String group = groupForTui(tui); // CHEM for drug TUIs, blank otherwise
            String typeLabel = typeLabelForTui(tui);

            StringBuilder sb = new StringBuilder();
            sb.append(csv(docId)).append(',')
              .append(ia.getBegin()).append(',')
              .append(ia.getEnd()).append(',')
              .append(csv(safeText(text, ia.getBegin(), ia.getEnd()))).append(',')
              .append(csv(normalizeSection(section))).append(',')
              .append(csv(rxCui)).append(',')
              .append(csv(rxName)).append(',')
              .append(csv(tui)).append(',')
              .append(csv(group)).append(',')
              .append(csv(typeLabel));
            rows.add(sb.toString());
        }

        // Write if there is at least one RxNorm row
        try {
            Path outDir = Paths.get(outputBase, subDir);
            Files.createDirectories(outDir);
            Path out = outDir.resolve(docId + ".CSV");
            try (BufferedWriter bw = Files.newBufferedWriter(out, StandardCharsets.UTF_8,
                    StandardOpenOption.CREATE, StandardOpenOption.TRUNCATE_EXISTING)) {
                bw.write("Document,Begin,End,Text,Section,RxCUI,RxNormName,TUI,SemanticGroup,SemanticTypeLabel\n");
                for (String line : rows) bw.write(line + "\n");
            }
        } catch (IOException e) {
            throw new AnalysisEngineProcessException(e);
        }
    }

    private static UmlsConcept pickBestRxNorm(IdentifiedAnnotation ia) {
        if (ia == null || ia.getOntologyConceptArr() == null) return null;
        UmlsConcept bestRx = null;
        double bestRxScore = Double.NEGATIVE_INFINITY;
        UmlsConcept bestChemical = null;
        double bestChemicalScore = Double.NEGATIVE_INFINITY;
        UmlsConcept bestAny = null;
        double bestAnyScore = Double.NEGATIVE_INFINITY;
        for (int i = 0; i < ia.getOntologyConceptArr().size(); i++) {
            if (!(ia.getOntologyConceptArr().get(i) instanceof UmlsConcept)) continue;
            UmlsConcept c = (UmlsConcept) ia.getOntologyConceptArr().get(i);
            double sc = c.getScore();
            if (bestAny == null || sc > bestAnyScore) { bestAny = c; bestAnyScore = sc; }

            String scheme = nvl(c.getCodingScheme()).toUpperCase(Locale.ROOT);
            if ("RXNORM".equals(scheme)) {
                if (bestRx == null || sc > bestRxScore) { bestRx = c; bestRxScore = sc; }
                continue;
            }

            String tui = nvl(c.getTui());
            if (!tui.isEmpty() && "CHEM".equals(groupForTui(tui))) {
                if (bestChemical == null || sc > bestChemicalScore) { bestChemical = c; bestChemicalScore = sc; }
            }
        }
        if (bestRx != null) return bestRx;
        if (bestChemical != null) return bestChemical;
        return bestAny;
    }

    private static String getDocId(JCas jCas) {
        for (DocumentID did : JCasUtil.select(jCas, DocumentID.class)) {
            String id = did.getDocumentID(); if (id != null && !id.isEmpty()) return id;
        }
        return "note";
    }

    private static String findSection(List<Segment> segs, IdentifiedAnnotation ia) {
        for (Segment s : segs) {
            if (s.getBegin() <= ia.getBegin() && s.getEnd() >= ia.getEnd()) {
                String v = s.getPreferredText();
                if (v == null || v.isEmpty()) v = s.getId();
                return v==null?"":v;
            }
        }
        return "";
    }

    private static String normalizeSection(String s) {
        if (s == null) return "";
        if ("SIMPLE_SEGMENT".equalsIgnoreCase(s)) return "S";
        return s;
    }

    private static String safeText(String sofa, int b, int e) {
        if (sofa == null) return "";
        int bb = Math.max(0, Math.min(b, sofa.length()));
        int ee = Math.max(bb, Math.min(e, sofa.length()));
        return sofa.substring(bb, ee);
    }

    private static String csv(String s) {
        if (s == null) return "";
        boolean need = s.contains(",") || s.contains("\"") || s.contains("\n") || s.contains("\r");
        if (!need) return s;
        String v = s.replace("\"", "\"\"");
        return "\"" + v + "\"";
    }

    private static String nvl(String s) { return s == null ? "" : s; }

    // --- TUI -> Semantic Type label ---
    private static final Map<String,String> TUI_GROUP_MAP = buildGroupMap();
    private static final Map<String,String> TUI_LABEL_MAP = createTuiLabelMap();

    static {
        SemGroupLoader.applyOverrides(TUI_GROUP_MAP, TUI_LABEL_MAP);
    }

    private static Map<String,String> buildGroupMap() {
        Map<String,String> m = new HashMap<>();
        put(m, "T103", "CHEM");
        put(m, "T104", "CHEM");
        put(m, "T109", "CHEM");
        put(m, "T114", "CHEM");
        put(m, "T116", "CHEM");
        put(m, "T120", "CHEM");
        put(m, "T121", "CHEM");
        put(m, "T122", "CHEM");
        put(m, "T123", "CHEM");
        put(m, "T125", "CHEM");
        put(m, "T126", "CHEM");
        put(m, "T127", "CHEM");
        put(m, "T129", "CHEM");
        put(m, "T130", "CHEM");
        put(m, "T131", "CHEM");
        put(m, "T167", "CHEM");
        put(m, "T168", "CHEM");
        put(m, "T195", "CHEM");
        put(m, "T196", "CHEM");
        put(m, "T197", "CHEM");
        put(m, "T200", "CHEM");
        put(m, "T203", "CHEM");
        return m;
    }

    private static Map<String,String> createTuiLabelMap() {
        Map<String,String> m = new HashMap<>();
        // Selected set (can be extended). Values from user-provided list.
        put(m, "T001","Organism");
        put(m, "T002","Plant");
        put(m, "T004","Fungus");
        put(m, "T005","Virus");
        put(m, "T007","Bacterium");
        put(m, "T008","Animal");
        put(m, "T010","Vertebrate");
        put(m, "T011","Amphibian");
        put(m, "T012","Bird");
        put(m, "T013","Fish");
        put(m, "T014","Reptile");
        put(m, "T015","Mammal");
        put(m, "T016","Human");
        put(m, "T017","Anatomical Structure");
        put(m, "T018","Embryonic Structure");
        put(m, "T019","Congenital Abnormality");
        put(m, "T020","Acquired Abnormality");
        put(m, "T021","Fully Formed Anatomical Structure");
        put(m, "T022","Body System");
        put(m, "T023","Body Part, Organ, or Organ Component");
        put(m, "T024","Tissue");
        put(m, "T025","Cell");
        put(m, "T026","Cell Component");
        put(m, "T028","Gene or Genome");
        put(m, "T029","Body Location or Region");
        put(m, "T030","Body Space or Junction");
        put(m, "T031","Body Substance");
        put(m, "T032","Organism Attribute");
        put(m, "T033","Finding");
        put(m, "T034","Laboratory or Test Result");
        put(m, "T037","Injury or Poisoning");
        put(m, "T038","Biologic Function");
        put(m, "T039","Physiologic Function");
        put(m, "T040","Organism Function");
        put(m, "T041","Mental Process");
        put(m, "T042","Organ or Tissue Function");
        put(m, "T043","Cell Function");
        put(m, "T044","Molecular Function");
        put(m, "T045","Genetic Function");
        put(m, "T046","Pathologic Function");
        put(m, "T047","Disease or Syndrome");
        put(m, "T048","Mental or Behavioral Dysfunction");
        put(m, "T049","Cell or Molecular Dysfunction");
        put(m, "T050","Experimental Model of Disease");
        put(m, "T051","Event");
        put(m, "T052","Activity");
        put(m, "T053","Behavior");
        put(m, "T054","Social Behavior");
        put(m, "T055","Individual Behavior");
        put(m, "T056","Daily or Recreational Activity");
        put(m, "T057","Occupational Activity");
        put(m, "T058","Health Care Activity");
        put(m, "T059","Laboratory Procedure");
        put(m, "T060","Diagnostic Procedure");
        put(m, "T061","Therapeutic or Preventive Procedure");
        put(m, "T062","Research Activity");
        put(m, "T063","Molecular Biology Research Technique");
        put(m, "T064","Governmental or Regulatory Activity");
        put(m, "T065","Educational Activity");
        put(m, "T066","Machine Activity");
        put(m, "T067","Phenomenon or Process");
        put(m, "T068","Human-caused Phenomenon or Process");
        put(m, "T069","Environmental Effect of Humans");
        put(m, "T070","Natural Phenomenon or Process");
        put(m, "T071","Entity");
        put(m, "T072","Physical Object");
        put(m, "T073","Manufactured Object");
        put(m, "T074","Medical Device");
        put(m, "T075","Research Device");
        put(m, "T077","Conceptual Entity");
        put(m, "T078","Idea or Concept");
        put(m, "T079","Temporal Concept");
        put(m, "T080","Qualitative Concept");
        put(m, "T081","Quantitative Concept");
        put(m, "T082","Spatial Concept");
        put(m, "T083","Geographic Area");
        put(m, "T085","Molecular Sequence");
        put(m, "T086","Nucleotide Sequence");
        put(m, "T087","Amino Acid Sequence");
        put(m, "T088","Carbohydrate Sequence");
        put(m, "T089","Regulation or Law");
        put(m, "T090","Occupation or Discipline");
        put(m, "T091","Biomedical Occupation or Discipline");
        put(m, "T092","Organization");
        put(m, "T093","Health Care Related Organization");
        put(m, "T094","Professional Society");
        put(m, "T095","Self-help or Relief Organization");
        put(m, "T096","Group");
        put(m, "T097","Professional or Occupational Group");
        put(m, "T098","Population Group");
        put(m, "T099","Family Group");
        put(m, "T100","Age Group");
        put(m, "T101","Patient or Disabled Group");
        put(m, "T102","Group Attribute");
        put(m, "T103","Chemical");
        put(m, "T104","Chemical Viewed Structurally");
        put(m, "T109","Organic Chemical");
        put(m, "T114","Nucleic Acid, Nucleoside, or Nucleotide");
        put(m, "T116","Amino Acid, Peptide, or Protein");
        put(m, "T120","Chemical Viewed Functionally");
        put(m, "T121","Pharmacologic Substance");
        put(m, "T122","Biomedical or Dental Material");
        put(m, "T123","Biologically Active Substance");
        put(m, "T125","Hormone");
        put(m, "T126","Enzyme");
        put(m, "T127","Vitamin");
        put(m, "T129","Immunologic Factor");
        put(m, "T130","Indicator, Reagent, or Diagnostic Aid");
        put(m, "T131","Hazardous or Poisonous Substance");
        put(m, "T167","Substance");
        put(m, "T168","Food");
        put(m, "T169","Functional Concept");
        put(m, "T170","Intellectual Product");
        put(m, "T171","Language");
        put(m, "T184","Sign or Symptom");
        put(m, "T185","Classification");
        put(m, "T190","Anatomical Abnormality");
        put(m, "T191","Neoplastic Process");
        put(m, "T192","Receptor");
        put(m, "T194","Archaeon");
        put(m, "T195","Antibiotic");
        put(m, "T196","Element, Ion, or Isotope");
        put(m, "T197","Inorganic Chemical");
        put(m, "T200","Clinical Drug");
        put(m, "T201","Clinical Attribute");
        put(m, "T203","Drug Delivery Device");
        put(m, "T204","Eukaryote");
        return m;
    }

    private static void put(Map<String,String> m, String k, String v) { m.put(k, v); }

    private static String typeLabelForTui(String tui) {
        if (tui == null) return "";
        String lab = TUI_LABEL_MAP.get(tui.toUpperCase(Locale.ROOT));
        return lab == null ? "" : lab;
    }

    private static String groupForTui(String tui) {
        if (tui == null) return "";
        String key = tui.toUpperCase(Locale.ROOT);
        String group = TUI_GROUP_MAP.get(key);
        return group == null ? "" : group;
    }
}

