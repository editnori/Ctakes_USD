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
import org.apache.uima.UimaContext;
import org.apache.uima.analysis_component.JCasAnnotator_ImplBase;
import org.apache.uima.analysis_engine.AnalysisEngineProcessException;
import org.apache.uima.fit.descriptor.ConfigurationParameter;
import org.apache.uima.fit.util.JCasUtil;
import org.apache.uima.jcas.JCas;
import org.apache.uima.jcas.cas.FSArray;

import java.io.BufferedWriter;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.*;

/**
 * UIMA writer that emits a per-document normalized Clinical Concepts CSV
 * matching the columns used in the Excel report.
 *
 * Piper usage example:
 *   add tools.reporting.uima.ClinicalConceptCsvWriter SubDirectory=csv_table_concepts
 */
public class ClinicalConceptCsvWriter extends JCasAnnotator_ImplBase {

    public static final String PARAM_SUBDIR = "SubDirectory";
    @ConfigurationParameter(name = PARAM_SUBDIR, mandatory = false)
    private String subDir = "csv_table_concepts";

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
        String docId = getDocId(jCas);
        if (docId == null || docId.isEmpty()) docId = "doc";
        String text = jCas.getDocumentText();
        if (text == null) text = "";

        Path dir = Paths.get(outputBase, subDir);
        try { Files.createDirectories(dir); } catch (IOException e) { throw new AnalysisEngineProcessException(e); }
        Path out = dir.resolve(docId + ".CSV");

        // Build helper maps
        Map<IdentifiedAnnotation, String> docTimeRel = buildDocTimeRelMap(jCas);
        Map<IdentifiedAnnotation, Boolean> hasDegreeOf = buildDegreeOfMap(jCas);
        Map<IdentifiedAnnotation, String> locationText = buildLocationTextMap(jCas, text);
        CorefInfo coref = buildCoref(jCas, text);

        // Section map by span
        List<Segment> segments = new ArrayList<>(JCasUtil.select(jCas, Segment.class));

        try (BufferedWriter bw = Files.newBufferedWriter(out, StandardCharsets.UTF_8)) {
            // Header
            bw.write(String.join(",", Arrays.asList(
                    "Document","Begin","End","Text",
                    "Section","SmokingStatus",
                    "Semantic Group","Semantic Type","SemanticsFallback","CUI","TUI","PreferredText","PrefTextFallback","CodingScheme",
                    "CandidateCount","Candidates","Confidence","ConceptScore","Disambiguated",
                    "DocTimeRel","DegreeOf","LocationOfText","Coref","CorefChainId","CorefRepText",
                    "Polarity","Negated","Uncertain","Conditional","Generic","Subject","HistoryOf"
            )));
            bw.write("\n");

            for (IdentifiedAnnotation ia : JCasUtil.select(jCas, IdentifiedAnnotation.class)) {
                // Only output concept-bearing clinical concepts
                FSArray arr = ia.getOntologyConceptArr();
                int candCount = (arr == null) ? 0 : arr.size();
                String bestCui = ""; String bestTui = ""; String bestPref = ""; String bestScheme = ""; boolean disamb = false; double bestScore = 0.0;
                List<String> candStrs = new ArrayList<>();
                if (candCount > 0) {
                    for (int i=0;i<arr.size();i++) {
                        if (!(arr.get(i) instanceof UmlsConcept)) continue;
                        UmlsConcept c = (UmlsConcept) arr.get(i);
                        if (i == 0) { // best
                            bestCui = nvl(c.getCui()); bestTui = nvl(c.getTui()); bestPref = nvl(c.getPreferredText()); bestScheme = nvl(c.getCodingScheme()); disamb = c.getDisambiguated(); bestScore = c.getScore();
                        }
                        String cui = nvl(c.getCui()).isEmpty() ? (bestCui.isEmpty()?"?":bestCui) : c.getCui();
                        String tui = nvl(c.getTui()).isEmpty() ? (bestTui.isEmpty()?"?":bestTui) : c.getTui();
                        String pref = nvl(c.getPreferredText()); if (pref.isEmpty()) pref = bestPref.isEmpty()?safeText(text, ia.getBegin(), ia.getEnd()):bestPref;
                        candStrs.add(cui + ":" + tui + ":" + pref);
                    }
                }
                if (candCount <= 0 && bestCui.isEmpty()) continue; // skip non-concept mentions

                String section = findSection(segments, ia);
                String[] sem = semFromTui(bestTui);
                String sg = sem==null?"":sem[0]; String st = sem==null?"":sem[1]; boolean semFallback = (sem==null);
                boolean prefFallback = false;
                String prefOut = bestPref;
                if (prefOut == null || prefOut.isEmpty()) { prefOut = safeText(text, ia.getBegin(), ia.getEnd()); prefFallback = true; }

                String dtr = nvl(docTimeRel.get(ia));
                boolean deg = hasDegreeOf.getOrDefault(ia, false);
                String locTxt = nvl(locationText.get(ia));
                boolean isCoref = coref.chainIdByMention.containsKey(ia);
                String chainId = nvl(coref.chainIdByMention.get(ia));
                String chainRep = nvl(coref.repTextByChain.get(chainId));

                List<String> row = Arrays.asList(
                        docId,
                        String.valueOf(ia.getBegin()), String.valueOf(ia.getEnd()), csvEsc(safeText(text, ia.getBegin(), ia.getEnd())),
                        csvEsc(normalizeSection(section)), "",
                        csvEsc(sg), csvEsc(st), semFallback?"true":"", nvl(bestCui), nvl(bestTui), csvEsc(prefOut), prefFallback?"true":"", csvEsc(nvl(bestScheme)),
                        String.valueOf(candCount), csvEsc(String.join("; ", candStrs)), String.valueOf(ia.getConfidence()), String.valueOf(bestScore), String.valueOf(disamb),
                        csvEsc(dtr), String.valueOf(deg), csvEsc(locTxt), String.valueOf(isCoref), csvEsc(chainId), csvEsc(chainRep),
                        String.valueOf(ia.getPolarity()), String.valueOf(ia.getPolarity()<0), String.valueOf(ia.getUncertainty()!=0), String.valueOf(ia.getConditional()), String.valueOf(ia.getGeneric()), csvEsc(nvl(ia.getSubject())), String.valueOf(ia.getHistoryOf())
                );
                bw.write(String.join(",", row));
                bw.write("\n");
            }
        } catch (IOException e) {
            throw new AnalysisEngineProcessException(e);
        }
    }

    private String getDocId(JCas jCas) {
        for (DocumentID did : JCasUtil.select(jCas, DocumentID.class)) {
            String id = did.getDocumentID(); if (id != null && !id.isEmpty()) return id;
        }
        return "note";
    }

    private static String nvl(String s) { return (s==null)?"":s; }

    private static String safeText(String sofa, int b, int e) {
        if (sofa == null) return "";
        int bb = Math.max(0, Math.min(b, sofa.length()));
        int ee = Math.max(bb, Math.min(e, sofa.length()));
        return sofa.substring(bb, ee);
    }

    private static String csvEsc(String s) {
        if (s == null) return "";
        boolean need = s.contains(",") || s.contains("\"") || s.contains("\n") || s.contains("\r");
        if (!need) return s;
        String v = s.replace("\"", "\"\"");
        return "\"" + v + "\"";
    }

    private static String normalizeSection(String s) {
        if (s == null) return "";
        if ("SIMPLE_SEGMENT".equalsIgnoreCase(s)) return "S";
        return s;
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

    private static String[] semFromTui(String tui) {
        if (tui == null || tui.isEmpty()) return null;
        // Fallback mapping for common TUIs to (Semantic Group, Semantic Type label)
        // Minimal mapping; unknowns will be blank and SemanticsFallback=false won’t be set.
        Map<String,String[]> map = SEM_MAP;
        return map.get(tui.toUpperCase(Locale.ROOT));
    }

    private static final Map<String,String[]> SEM_MAP = buildSemMap();
    private static Map<String,String[]> buildSemMap() {
        Map<String,String[]> m = new HashMap<>();
        m.put("T047", new String[]{"DISO","Disease or Syndrome"});
        m.put("T033", new String[]{"FIND","Finding"});
        m.put("T184", new String[]{"FIND","Sign or Symptom"});
        m.put("T121", new String[]{"CHEM","Pharmacologic Substance"});
        m.put("T109", new String[]{"CHEM","Organic Chemical"});
        m.put("T200", new String[]{"CHEM","Clinical Drug"});
        m.put("T116", new String[]{"CHEM","Amino Acid, Peptide, or Protein"});
        m.put("T061", new String[]{"PROC","Therapeutic or Preventive Procedure"});
        m.put("T023", new String[]{"ANAT","Body Part, Organ, or Organ Component"});
        m.put("T046", new String[]{"DISO","Pathologic Function"});
        m.put("T191", new String[]{"ANAT","Neoplastic Process"});
        // Extend as needed
        return m;
    }

    // Build map from mention -> DocTimeRel value (only for EventMentions)
    private static Map<IdentifiedAnnotation,String> buildDocTimeRelMap(JCas jCas) {
        Map<IdentifiedAnnotation,String> map = new IdentityHashMap<>();
        for (EventMention evm : JCasUtil.select(jCas, EventMention.class)) {
            String dtr = evm.getEvent() != null && evm.getEvent().getProperties()!=null ? nvl(evm.getEvent().getProperties().getDocTimeRel()) : "";
            if (dtr == null) dtr = "";
            map.put(evm, dtr);
        }
        return map;
    }

    private static Map<IdentifiedAnnotation,Boolean> buildDegreeOfMap(JCas jCas) {
        Map<IdentifiedAnnotation,Boolean> map = new IdentityHashMap<>();
        for (DegreeOfTextRelation r : JCasUtil.select(jCas, DegreeOfTextRelation.class)) {
            IdentifiedAnnotation a1 = asMention(r.getArg1());
            IdentifiedAnnotation a2 = asMention(r.getArg2());
            if (a1 != null) map.put(a1, true);
            if (a2 != null) map.put(a2, true);
        }
        return map;
    }

    private static Map<IdentifiedAnnotation,String> buildLocationTextMap(JCas jCas, String sofa) {
        Map<IdentifiedAnnotation,String> map = new IdentityHashMap<>();
        for (LocationOfTextRelation r : JCasUtil.select(jCas, LocationOfTextRelation.class)) {
            IdentifiedAnnotation holder = asMention(r.getArg1()); // assume arg1 is the holder
            IdentifiedAnnotation partner = asMention(r.getArg2());
            if (holder != null && partner != null) map.put(holder, safeText(sofa, partner.getBegin(), partner.getEnd()));
        }
        return map;
    }

    private static IdentifiedAnnotation asMention(RelationArgument ra) {
        if (ra == null || ra.getArgument() == null) return null;
        if (ra.getArgument() instanceof IdentifiedAnnotation) return (IdentifiedAnnotation) ra.getArgument();
        return null;
    }

    private static class CorefInfo {
        Map<IdentifiedAnnotation,String> chainIdByMention = new IdentityHashMap<>();
        Map<String,String> repTextByChain = new HashMap<>();
    }

    private static CorefInfo buildCoref(JCas jCas, String sofa) {
        CorefInfo info = new CorefInfo();
        // Union-find by representative begin offset
        Map<IdentifiedAnnotation,IdentifiedAnnotation> parent = new IdentityHashMap<>();
        java.util.function.Function<IdentifiedAnnotation,IdentifiedAnnotation> find = new java.util.function.Function<IdentifiedAnnotation,IdentifiedAnnotation>(){
            public IdentifiedAnnotation apply(IdentifiedAnnotation x){
                IdentifiedAnnotation p = parent.getOrDefault(x, x);
                if (p != x) { p = this.apply(p); parent.put(x, p); } else parent.putIfAbsent(x, x);
                return parent.get(x);
            }
        };
        java.util.function.BiConsumer<IdentifiedAnnotation,IdentifiedAnnotation> union = (a,b) -> {
            IdentifiedAnnotation ra = find.apply(a); IdentifiedAnnotation rb = find.apply(b);
            if (ra != rb) parent.put(ra, rb);
        };

        Map<Markable,IdentifiedAnnotation> mentionByMark = new IdentityHashMap<>();
        for (IdentifiedAnnotation ia : JCasUtil.select(jCas, IdentifiedAnnotation.class)) {
            // Markables can be separate; we union later when linked by coref
        }
        // Build union via relations
        for (CoreferenceRelation cr : JCasUtil.select(jCas, CoreferenceRelation.class)) {
            IdentifiedAnnotation a1 = asMention(cr.getArg1());
            IdentifiedAnnotation a2 = asMention(cr.getArg2());
            if (a1 != null && a2 != null) union.accept(a1, a2);
        }
        // Assign chain ids and representative text
        Map<IdentifiedAnnotation,List<IdentifiedAnnotation>> chains = new IdentityHashMap<>();
        for (IdentifiedAnnotation ia : JCasUtil.select(jCas, IdentifiedAnnotation.class)) {
            IdentifiedAnnotation root = find.apply(ia);
            chains.computeIfAbsent(root, k -> new ArrayList<>()).add(ia);
        }
        int seq = 1;
        for (Map.Entry<IdentifiedAnnotation,List<IdentifiedAnnotation>> e : chains.entrySet()) {
            List<IdentifiedAnnotation> members = e.getValue();
            if (members.size() <= 1) continue; // singletons not considered coref chains
            // Choose representative: earliest begin
            members.sort(Comparator.comparingInt(IdentifiedAnnotation::getBegin));
            String chainId = String.valueOf(seq++);
            String rep = safeText(sofa, members.get(0).getBegin(), members.get(0).getEnd());
            for (IdentifiedAnnotation m : members) info.chainIdByMention.put(m, chainId);
            info.repTextByChain.put(chainId, rep);
        }
        return info;
    }
}
