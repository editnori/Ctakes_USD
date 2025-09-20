package tools.reporting.uima;

import org.apache.ctakes.typesystem.type.relation.CoreferenceRelation;
import org.apache.ctakes.typesystem.type.relation.DegreeOfTextRelation;
import org.apache.ctakes.typesystem.type.relation.LocationOfTextRelation;
import org.apache.ctakes.typesystem.type.relation.RelationArgument;
import org.apache.ctakes.typesystem.type.structured.DocumentID;
import org.apache.ctakes.typesystem.type.textsem.EventMention;
import org.apache.ctakes.typesystem.type.textsem.IdentifiedAnnotation;
import org.apache.ctakes.typesystem.type.textsem.Markable;
import org.apache.ctakes.typesystem.type.refsem.UmlsConcept;
import org.apache.ctakes.typesystem.type.textspan.Segment;
import org.apache.ctakes.typesystem.type.syntax.BaseToken;
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
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.IdentityHashMap;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Locale;
import java.util.Map;

/**
 * Writes a per-document CSV describing each disambiguated clinical concept.
 * Columns are grouped by subsystem using prefixes (core:, assertion:, temporal:, relations:, coref:, wsd:).
 */
public class ConceptCsvWriter extends JCasAnnotator_ImplBase {

    public static final String PARAM_SUBDIR = "SubDirectory";

    @ConfigurationParameter(name = PARAM_SUBDIR, mandatory = false)
    private String subDir = "concepts";

    private String outputBase;


    @Override
    public void initialize(UimaContext context) {
        Object od = context.getConfigParameterValue("OutputDirectory");
        if (od == null) od = System.getProperty("ctakes.output.dir");
        if (od == null) od = System.getProperty("OUTPUT_DIR");
        outputBase = (od == null) ? "." : od.toString();
        Object sd = context.getConfigParameterValue(PARAM_SUBDIR);
        if (sd != null && !sd.toString().trim().isEmpty()) {
            subDir = sd.toString().trim();
        }
    }

    @Override
    public void process(JCas jCas) throws AnalysisEngineProcessException {
        final String docId = getDocId(jCas);
        final String sofa = jCas.getDocumentText() == null ? "" : jCas.getDocumentText();
        final List<Segment> segments = new ArrayList<>(JCasUtil.select(jCas, Segment.class));

        Map<IdentifiedAnnotation, String> docTimeRelMap = buildDocTimeRelMap(jCas);
        Map<IdentifiedAnnotation, String> degreeMap = buildDegreeMap(jCas);
        Map<IdentifiedAnnotation, LocationInfo> locationMap = buildLocationInfo(jCas, sofa);
        CorefInfo coref = buildCoref(jCas, sofa);
        ColumnLayout layout = ColumnLayout.from(docTimeRelMap, degreeMap, locationMap, coref);

        List<Row> rows = new ArrayList<>();
        for (IdentifiedAnnotation ia : JCasUtil.select(jCas, IdentifiedAnnotation.class)) {
            UmlsConcept best = firstConcept(ia);
            String pos = collectPos(jCas, ia);
            rows.add(new Row(ia.getBegin(), ia.getEnd(),
                    buildRow(docId, sofa, segments, ia, best, docTimeRelMap, degreeMap, locationMap, coref, layout, pos)));
        }

        rows.sort((left, right) -> {
            int cmp = Integer.compare(left.begin, right.begin);
            if (cmp != 0) {
                return cmp;
            }
            return Integer.compare(left.end, right.end);
        });

        Path outDir = Paths.get(outputBase, subDir);
        try {
            Files.createDirectories(outDir);
            Path out = outDir.resolve(docId + ".csv");
            try (BufferedWriter bw = Files.newBufferedWriter(out, StandardCharsets.UTF_8,
                    StandardOpenOption.CREATE, StandardOpenOption.TRUNCATE_EXISTING)) {
                bw.write(String.join(",", layout.headers));
                bw.newLine();
                for (Row row : rows) {
                    bw.write(String.join(",", row.cells));
                    bw.newLine();
                }
            }
        } catch (IOException e) {
            throw new AnalysisEngineProcessException(e);
        }
    }

    private static List<String> buildRow(String docId,
                                         String sofa,
                                         List<Segment> segments,
                                         IdentifiedAnnotation ia,
                                         UmlsConcept concept,
                                         Map<IdentifiedAnnotation, String> docTimeRelMap,
                                         Map<IdentifiedAnnotation, String> degreeMap,
                                         Map<IdentifiedAnnotation, LocationInfo> locationMap,
                                         CorefInfo coref,
                                         ColumnLayout layout,
                                         String posTags) {
        String section = findSection(segments, ia);
        String cui = concept != null ? nvl(concept.getCui()) : "";
        String pref = concept != null ? nvl(concept.getPreferredText()) : "";
        String rxCui = collectRxnormCodes(ia);
        String tui = concept != null ? nvl(concept.getTui()) : "";
        String group = semanticGroupForTui(tui);
        String typeLabel = semanticTypeLabelForTui(tui);
        String polarity = ia.getPolarity() < 0 ? "NEG" : "POS";
        String uncertainty = binary(ia.getUncertainty() > 0);
        String conditional = binary(ia.getConditional());
        String generic = binary(ia.getGeneric());
        String subject = nvl(ia.getSubject());
        String docTimeRel = layout.includeTemporal ? nvl(docTimeRelMap.getOrDefault(ia, "")) : "";
        String degreeIndicator = layout.includeRelations ? nvl(degreeMap.getOrDefault(ia, "")) : "";
        String hasDegree = layout.includeRelations ? binary(!degreeIndicator.isEmpty()) : "";
        LocationInfo locInfo = layout.includeRelations ? locationMap.get(ia) : null;
        String locText = locInfo != null ? locInfo.text : "";
        String locCui = locInfo != null ? locInfo.cui : "";
        String corefId = layout.includeCoref ? nvl(coref.chainIdByMention.get(ia)) : "";
        String corefRep = "";
        if (layout.includeCoref && !corefId.isEmpty()) {
            corefRep = nvl(coref.repTextByChain.getOrDefault(corefId, ""));
        }
        String wsdDisambig = concept != null && concept.getDisambiguated() ? "Y" : binary(concept != null && concept.getScore() > 0);
        String wsdScore = concept != null ? formatConfidence((float) concept.getScore()) : "";

        List<String> cells = new ArrayList<>();
        cells.add(csv(docId));
        cells.add(Integer.toString(ia.getBegin()));
        cells.add(Integer.toString(ia.getEnd()));
        cells.add(csv(safeText(sofa, ia.getBegin(), ia.getEnd())));
        cells.add(csv(normalizeSection(section)));
        cells.add(csv(cui));
        cells.add(csv(rxCui));
        cells.add(csv(pref.isEmpty() ? safeText(sofa, ia.getBegin(), ia.getEnd()) : pref));
        cells.add(csv(tui));
        cells.add(csv(group));
        cells.add(csv(typeLabel));
        cells.add(csv(polarity));
        cells.add(csv(uncertainty));
        cells.add(csv(conditional));
        cells.add(csv(generic));
        cells.add(csv(subject.isEmpty() ? "patient" : subject));
        if (layout.includePos) {
            cells.add(csv(posTags));
        }
        if (layout.includeTemporal) {
            cells.add(csv(docTimeRel));
        }
        if (layout.includeRelations) {
            cells.add(csv(hasDegree));
            cells.add(csv(degreeIndicator));
            cells.add(csv(locText));
            cells.add(csv(locCui));
        }
        if (layout.includeCoref) {
            cells.add(csv(corefId));
            cells.add(csv(corefRep));
        }
        cells.add(csv(wsdDisambig));
        cells.add(csv(wsdScore));
        return cells;
    }

    private static UmlsConcept firstConcept(IdentifiedAnnotation ia) {
        if (ia == null || ia.getOntologyConceptArr() == null || ia.getOntologyConceptArr().size() == 0) {
            return null;
        }
        for (int i = 0; i < ia.getOntologyConceptArr().size(); i++) {
            if (ia.getOntologyConceptArr().get(i) instanceof UmlsConcept) {
                return (UmlsConcept) ia.getOntologyConceptArr().get(i);
            }
        }
        return null;
    }

    private static String collectRxnormCodes(IdentifiedAnnotation ia) {
        if (ia == null || ia.getOntologyConceptArr() == null) {
            return "";
        }
        LinkedHashSet<String> codes = new LinkedHashSet<>();
        for (int i = 0; i < ia.getOntologyConceptArr().size(); i++) {
            if (ia.getOntologyConceptArr().get(i) instanceof UmlsConcept) {
                UmlsConcept c = (UmlsConcept) ia.getOntologyConceptArr().get(i);
                String scheme = nvl(c.getCodingScheme()).toUpperCase(Locale.ROOT);
                if (!"RXNORM".equals(scheme)) {
                    continue;
                }
                String code = nvl(c.getCode());
                if (!code.isEmpty()) {
                    codes.add(code);
                }
            }
        }
        return String.join("|", codes);
    }

    private static Map<IdentifiedAnnotation, String> buildDocTimeRelMap(JCas jCas) {
        Map<IdentifiedAnnotation, String> map = new IdentityHashMap<>();
        for (EventMention evm : JCasUtil.select(jCas, EventMention.class)) {
            String dtr = "";
            if (evm.getEvent() != null && evm.getEvent().getProperties() != null) {
                dtr = nvl(evm.getEvent().getProperties().getDocTimeRel());
            }
            map.put(evm, dtr);
        }
        return map;
    }

    private static Map<IdentifiedAnnotation, String> buildDegreeMap(JCas jCas) {
        Map<IdentifiedAnnotation, String> map = new IdentityHashMap<>();
        for (DegreeOfTextRelation r : JCasUtil.select(jCas, DegreeOfTextRelation.class)) {
            IdentifiedAnnotation a1 = asMention(r.getArg1());
            IdentifiedAnnotation a2 = asMention(r.getArg2());
            if (a1 != null && a2 != null) {
            map.put(a1, safeText(jCas.getDocumentText(), a2.getBegin(), a2.getEnd()));
        }
        }
        return map;
    }

    private static Map<IdentifiedAnnotation, LocationInfo> buildLocationInfo(JCas jCas, String sofa) {
        Map<IdentifiedAnnotation, LocationInfo> map = new IdentityHashMap<>();
        for (LocationOfTextRelation r : JCasUtil.select(jCas, LocationOfTextRelation.class)) {
            IdentifiedAnnotation holder = asMention(r.getArg1());
            IdentifiedAnnotation partner = asMention(r.getArg2());
            if (holder != null && partner != null) {
                LocationInfo info = new LocationInfo();
                info.text = safeText(sofa, partner.getBegin(), partner.getEnd());
                UmlsConcept partnerConcept = firstConcept(partner);
                info.cui = partnerConcept != null ? nvl(partnerConcept.getCui()) : "";
                map.put(holder, info);
            }
        }
        return map;
    }

    private static CorefInfo buildCoref(JCas jCas, String sofa) {
        CorefInfo info = new CorefInfo();
        Map<IdentifiedAnnotation, IdentifiedAnnotation> parent = new IdentityHashMap<>();
        java.util.function.Function<IdentifiedAnnotation, IdentifiedAnnotation> find = new java.util.function.Function<IdentifiedAnnotation, IdentifiedAnnotation>() {
            @Override
            public IdentifiedAnnotation apply(IdentifiedAnnotation x) {
                IdentifiedAnnotation p = parent.getOrDefault(x, x);
                if (p != x) {
                    p = this.apply(p);
                    parent.put(x, p);
                } else {
                    parent.putIfAbsent(x, x);
                }
                return parent.get(x);
            }
        };
        java.util.function.BiConsumer<IdentifiedAnnotation, IdentifiedAnnotation> union = (a, b) -> {
            IdentifiedAnnotation ra = find.apply(a);
            IdentifiedAnnotation rb = find.apply(b);
            if (ra != rb) parent.put(ra, rb);
        };

        for (CoreferenceRelation cr : JCasUtil.select(jCas, CoreferenceRelation.class)) {
            IdentifiedAnnotation a1 = asMention(cr.getArg1());
            IdentifiedAnnotation a2 = asMention(cr.getArg2());
            if (a1 != null && a2 != null) union.accept(a1, a2);
        }

        Map<IdentifiedAnnotation, List<IdentifiedAnnotation>> chains = new IdentityHashMap<>();
        for (IdentifiedAnnotation ia : JCasUtil.select(jCas, IdentifiedAnnotation.class)) {
            IdentifiedAnnotation root = find.apply(ia);
            chains.computeIfAbsent(root, k -> new ArrayList<>()).add(ia);
        }
        int seq = 1;
        for (Map.Entry<IdentifiedAnnotation, List<IdentifiedAnnotation>> e : chains.entrySet()) {
            List<IdentifiedAnnotation> members = e.getValue();
            if (members.size() <= 1) continue;
            members.sort((a, b) -> Integer.compare(a.getBegin(), b.getBegin()));
            String chainId = Integer.toString(seq++);
            String rep = safeText(sofa, members.get(0).getBegin(), members.get(0).getEnd());
            for (IdentifiedAnnotation m : members) info.chainIdByMention.put(m, chainId);
            info.repTextByChain.put(chainId, rep);
        }
        return info;
    }

    private static IdentifiedAnnotation asMention(RelationArgument ra) {
        if (ra == null || ra.getArgument() == null) return null;
        if (ra.getArgument() instanceof IdentifiedAnnotation) return (IdentifiedAnnotation) ra.getArgument();
        return null;
    }

    private static String getDocId(JCas jCas) {
        for (DocumentID id : JCasUtil.select(jCas, DocumentID.class)) {
            String docId = id.getDocumentID();
            if (docId != null && !docId.trim().isEmpty()) {
                return docId.trim();
            }
        }
        return "UNKNOWN";
    }

    private static String safeText(String sofa, int begin, int end) {
        if (sofa == null || sofa.isEmpty()) return "";
        int b = Math.max(0, Math.min(begin, sofa.length()));
        int e = Math.max(b, Math.min(end, sofa.length()));
        return sofa.substring(b, e).replace('\n', ' ').replace('\r', ' ').trim();
    }

    private static String findSection(List<Segment> segments, IdentifiedAnnotation ia) {
        for (Segment s : segments) {
            if (ia.getBegin() >= s.getBegin() && ia.getEnd() <= s.getEnd()) {
                return nvl(s.getPreferredText());
            }
        }
        return "";
    }

    private static String normalizeSection(String section) {
        if (section == null) return "";
        String trimmed = section.trim();
        if (trimmed.isEmpty()) return "";
        return trimmed.replace(',', ';');
    }

    private static String csv(String value) {
        if (value == null) return "";
        String cleaned = value.replace("\"", "\"\"");
        if (cleaned.contains(",") || cleaned.contains("\n") || cleaned.contains("\r")) {
            return '"' + cleaned + '"';
        }
        return cleaned;
    }

    private static String nvl(String value) {
        return value == null ? "" : value;
    }

    private static String binary(boolean value) {
        return value ? "Y" : "N";
    }

    private static String formatConfidence(float confidence) {
        if (Float.isNaN(confidence)) return "";
        return String.format(Locale.ROOT, "%.3f", confidence);
    }


    private static final Map<String, String> TUI_TO_GROUP = buildGroupMap();
    private static final Map<String, String> TUI_TO_LABEL = buildLabelMap();

    static {
        SemGroupLoader.applyOverrides(TUI_TO_GROUP, TUI_TO_LABEL);
    }

    private static Map<String, String> buildGroupMap() {
        Map<String, String> m = new HashMap<>();
        putGroup(m, "T121", "CHEM");
        putGroup(m, "T200", "CHEM");
        putGroup(m, "T109", "CHEM");
        putGroup(m, "T103", "CHEM");
        putGroup(m, "T123", "CHEM");
        putGroup(m, "T023", "ANAT");
        putGroup(m, "T033", "FIND");
        putGroup(m, "T038", "DISO");
        putGroup(m, "T039", "DISO");
        putGroup(m, "T040", "DISO");
        putGroup(m, "T041", "DISO");
        putGroup(m, "T046", "DISO");
        putGroup(m, "T047", "DISO");
        putGroup(m, "T048", "DISO");
        putGroup(m, "T184", "FIND");
        putGroup(m, "T201", "CONC");
        putGroup(m, "T060", "PROC");
        putGroup(m, "T061", "PROC");
        putGroup(m, "T063", "PROC");
        putGroup(m, "T191", "ANAT");
        return m;
    }

    private static Map<String, String> buildLabelMap() {
        Map<String, String> m = new HashMap<>();
        putLabel(m, "T121", "Pharmacologic Substance");
        putLabel(m, "T200", "Clinical Drug");
        putLabel(m, "T109", "Organic Chemical");
        putLabel(m, "T103", "Chemical");
        putLabel(m, "T023", "Body Part, Organ, or Organ Component");
        putLabel(m, "T033", "Finding");
        putLabel(m, "T038", "Biologic Function");
        putLabel(m, "T039", "Physiologic Function");
        putLabel(m, "T040", "Organism Function");
        putLabel(m, "T041", "Mental Process");
        putLabel(m, "T046", "Pathologic Function");
        putLabel(m, "T047", "Disease or Syndrome");
        putLabel(m, "T048", "Mental or Behavioral Dysfunction");
        putLabel(m, "T184", "Sign or Symptom");
        putLabel(m, "T201", "Clinical Attribute");
        putLabel(m, "T060", "Diagnostic Procedure");
        putLabel(m, "T061", "Therapeutic or Preventive Procedure");
        putLabel(m, "T063", "Molecular Biology Research Technique");
        putLabel(m, "T191", "Neoplastic Process");
        return m;
    }

    private static void putGroup(Map<String, String> map, String tui, String group) {
        map.put(tui, group);
    }

    private static void putLabel(Map<String, String> map, String tui, String label) {
        map.put(tui, label);
    }

    private static String collectPos(JCas jCas, IdentifiedAnnotation ia) {
        StringBuilder sb = new StringBuilder();
        for (BaseToken token : JCasUtil.selectCovered(jCas, BaseToken.class, ia)) {
            String pos = nvl(token.getPartOfSpeech());
            if (!pos.isEmpty()) {
                if (sb.length() > 0) {
                    sb.append(' ');
                }
                sb.append(pos);
            }
        }
        return sb.toString();
    }

    private static class ColumnLayout {
        final boolean includeTemporal;
        final boolean includeRelations;
        final boolean includeCoref;
        final boolean includePos;
        final List<String> headers;

        private ColumnLayout(boolean includeTemporal, boolean includeRelations, boolean includeCoref, boolean includePos) {
            this.includeTemporal = includeTemporal;
            this.includeRelations = includeRelations;
            this.includeCoref = includeCoref;
            this.includePos = includePos;
            this.headers = buildHeaders();
        }

        static ColumnLayout from(Map<IdentifiedAnnotation, String> docTimeRelMap,
                                 Map<IdentifiedAnnotation, String> degreeMap,
                                 Map<IdentifiedAnnotation, LocationInfo> locationMap,
                                 CorefInfo coref) {
            boolean hasTemporal = docTimeRelMap.values().stream().anyMatch(v -> v != null && !v.trim().isEmpty());
            boolean hasDegree = degreeMap.values().stream().anyMatch(v -> v != null && !v.trim().isEmpty());
            boolean hasLocation = locationMap.values().stream().anyMatch(info -> info != null && (!info.text.isEmpty() || !info.cui.isEmpty()));
            boolean hasRelations = hasDegree || hasLocation;
            boolean hasCoref = !coref.chainIdByMention.isEmpty();
            return new ColumnLayout(hasTemporal, hasRelations, hasCoref, true);
        }

        private List<String> buildHeaders() {
            List<String> header = new ArrayList<>();
            header.addAll(Arrays.asList(
                    "core:Document",
                    "core:Begin",
                    "core:End",
                    "core:Text",
                    "core:Section",
                    "core:CUI",
                    "core:RxCUI",
                    "core:PreferredText",
                    "core:TUI",
                    "core:SemanticGroup",
                    "core:SemanticTypeLabel",
                    "assertion:Polarity",
                    "assertion:Uncertainty",
                    "assertion:Conditional",
                    "assertion:Generic",
                    "assertion:Subject"));
            if (includePos) {
                header.add("syntax:POS");
            }
            if (includeTemporal) {
                header.add("temporal:DocTimeRel");
            }
            if (includeRelations) {
                header.add("relations:HasDegree");
                header.add("relations:DegreeIndicator");
                header.add("relations:LocationText");
                header.add("relations:LocationCUI");
            }
            if (includeCoref) {
                header.add("coref:ChainId");
                header.add("coref:RepresentativeText");
            }
            header.add("wsd:Disambiguated");
            header.add("wsd:Score");
            return header;
        }
    }

    private static String semanticGroupForTui(String tui) {
        if (tui == null || tui.isEmpty()) return "";
        return nvl(TUI_TO_GROUP.get(tui.toUpperCase(Locale.ROOT)));
    }

    private static String semanticTypeLabelForTui(String tui) {
        if (tui == null || tui.isEmpty()) return "";
        return nvl(TUI_TO_LABEL.get(tui.toUpperCase(Locale.ROOT)));
    }

    private static class LocationInfo {
        String text = "";
        String cui = "";
    }

    private static class CorefInfo {
        Map<IdentifiedAnnotation, String> chainIdByMention = new IdentityHashMap<>();
        Map<String, String> repTextByChain = new HashMap<>();
    }
    private static final class Row {
        final int begin;
        final int end;
        final List<String> cells;

        Row(int begin, int end, List<String> cells) {
            this.begin = begin;
            this.end = end;
            this.cells = cells;
        }
    }

}







