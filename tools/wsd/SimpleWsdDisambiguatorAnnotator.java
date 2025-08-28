package tools.wsd;

import java.util.*;
import java.util.regex.Pattern;

import org.apache.ctakes.typesystem.type.textsem.IdentifiedAnnotation;
import org.apache.ctakes.typesystem.type.refsem.OntologyConcept;
import org.apache.ctakes.typesystem.type.refsem.UmlsConcept;
import org.apache.ctakes.typesystem.type.textspan.Sentence;
import org.apache.uima.analysis_component.JCasAnnotator_ImplBase;
import org.apache.uima.analysis_engine.AnalysisEngineProcessException;
import org.apache.uima.fit.util.JCasUtil;
import org.apache.uima.jcas.JCas;
import org.apache.uima.jcas.cas.FSArray;

/**
 * Simple, local word-sense disambiguator for cTAKES mentions.
 * - Picks exactly one best OntologyConcept per IdentifiedAnnotation.
 * - Scores candidates by overlap between context tokens (covering sentence)
 *   and candidate preferred text; tiebreak by preferred text length.
 * - Does not require YTEX or any external DB.
 */
public class SimpleWsdDisambiguatorAnnotator extends JCasAnnotator_ImplBase {
    public static final String PARAM_KEEP_ALL = "KeepAllCandidates";
    public static final String PARAM_MOVE_BEST_FIRST = "MoveBestFirst";
    public static final String PARAM_MARK_DISAMBIGUATED = "MarkDisambiguated";
    public static final String PARAM_MIN_TOKEN_LEN = "MinTokenLen"; // int, default 1
    public static final String PARAM_FILTER_SINGLE_CHAR_STOPS = "FilterSingleCharStops"; // boolean, default true

    private boolean keepAll = false;
    private boolean moveBestFirst = true;
    private boolean markDisambiguated = true;
    private int minTokenLen = 1;
    private boolean filterSingleCharStops = true;

    @Override
    public void initialize(org.apache.uima.UimaContext context) throws org.apache.uima.resource.ResourceInitializationException {
        super.initialize(context);
        Object v;
        v = context.getConfigParameterValue(PARAM_KEEP_ALL);
        if (v instanceof Boolean) keepAll = (Boolean) v;
        v = context.getConfigParameterValue(PARAM_MOVE_BEST_FIRST);
        if (v instanceof Boolean) moveBestFirst = (Boolean) v;
        v = context.getConfigParameterValue(PARAM_MARK_DISAMBIGUATED);
        if (v instanceof Boolean) markDisambiguated = (Boolean) v;
        v = context.getConfigParameterValue(PARAM_MIN_TOKEN_LEN);
        if (v instanceof Integer) { int ml = (Integer) v; if (ml < 1) ml = 1; if (ml > 10) ml = 10; minTokenLen = ml; }
        v = context.getConfigParameterValue(PARAM_FILTER_SINGLE_CHAR_STOPS);
        if (v instanceof Boolean) filterSingleCharStops = (Boolean) v;
    }
    private static final Pattern SPLIT = Pattern.compile("[^a-z0-9]+");

    @Override
    public void process(JCas jCas) throws AnalysisEngineProcessException {
        for (IdentifiedAnnotation ia : JCasUtil.select(jCas, IdentifiedAnnotation.class)) {
            FSArray ocArr = ia.getOntologyConceptArr();
            if (ocArr == null) continue;
            int n = ocArr.size();

            // Build context from covering sentence (fallback to mention text)
            String context;
            List<Sentence> covering = JCasUtil.selectCovering(jCas, Sentence.class, ia.getBegin(), ia.getEnd());
            if (!covering.isEmpty()) context = covering.get(0).getCoveredText();
            else context = ia.getCoveredText();
            Set<String> ctx = toTokenSet(context);

            if (n == 1) {
                // Single-candidate: compute Confidence/Score the same way (normalized overlap)
                OntologyConcept oc = (OntologyConcept) ocArr.get(0);
                String label = null;
                if (oc instanceof UmlsConcept) label = ((UmlsConcept) oc).getPreferredText();
                if (label == null || label.isEmpty()) label = oc.getCode() != null ? oc.getCode() : ia.getCoveredText();
                Set<String> cand = toTokenSet(label);
                if (cand.isEmpty()) cand = toTokenSet(ia.getCoveredText());
                double inter = overlapScore(ctx, cand);
                double scoreNorm = cand.isEmpty() ? 0.0 : inter / Math.max(1.0, (double)cand.size());
                // Set features
                if (markDisambiguated && oc instanceof UmlsConcept) {
                    ((UmlsConcept) oc).setDisambiguated(true);
                    try { ((UmlsConcept) oc).setScore(scoreNorm); } catch (Throwable t) { /* ignore */ }
                }
                try { ia.setConfidence((float)scoreNorm); } catch (Throwable t) { /* ignore */ }
                // Leave the single candidate in place
                continue;
            }
            if (n <= 0) continue;

            // Multi-candidate case: rank and select best
            int bestIdx = 0; double bestScore = -1.0; int bestLen = -1; double bestScoreNorm = 0.0;
            for (int i = 0; i < n; i++) {
                OntologyConcept oc = (OntologyConcept) ocArr.get(i);
                String label = null;
                if (oc instanceof UmlsConcept) label = ((UmlsConcept) oc).getPreferredText();
                if (label == null || label.isEmpty()) label = oc.getCode() != null ? oc.getCode() : ia.getCoveredText();
                Set<String> cand = toTokenSet(label);
                if (cand.isEmpty()) cand = toTokenSet(ia.getCoveredText());
                double inter = overlapScore(ctx, cand);
                double score = inter; // raw overlap for ranking
                double scoreNorm = cand.isEmpty() ? 0.0 : inter / Math.max(1.0, (double)cand.size());
                int labLen = label != null ? label.length() : 0;
                if (score > bestScore || (score == bestScore && labLen > bestLen)) {
                    bestScore = score; bestLen = labLen; bestIdx = i; bestScoreNorm = scoreNorm;
                }
            }

            OntologyConcept best = (OntologyConcept) ocArr.get(bestIdx);
            if (markDisambiguated && best instanceof UmlsConcept) {
                ((UmlsConcept) best).setDisambiguated(true);
                try { ((UmlsConcept) best).setScore(bestScoreNorm); } catch (Throwable t) { /* ignore */ }
            }
            try { ia.setConfidence((float)bestScoreNorm); } catch (Throwable t) { /* ignore */ }

            if (keepAll) {
                FSArray kept = new FSArray(jCas, n);
                if (moveBestFirst) {
                    kept.set(0, best);
                    int k = 1;
                    for (int i = 0; i < n; i++) {
                        if (i == bestIdx) continue;
                        kept.set(k++, (OntologyConcept) ocArr.get(i));
                    }
                } else {
                    for (int i = 0; i < n; i++) kept.set(i, (OntologyConcept) ocArr.get(i));
                }
                ia.setOntologyConceptArr(kept);
            } else {
                FSArray one = new FSArray(jCas, 1);
                one.set(0, best);
                ia.setOntologyConceptArr(one);
            }
        }
    }

    private Set<String> toTokenSet(String text) {
        Set<String> s = new HashSet<>();
        if (text == null) return s;
        String[] parts = SPLIT.split(text.toLowerCase(Locale.ROOT));
        for (String p : parts) {
            if (p.isEmpty()) continue;
            if (p.length() < minTokenLen) continue;
            if (filterSingleCharStops && p.length() == 1 && (p.equals("a") || p.equals("i"))) continue;
            s.add(p);
        }
        return s;
    }

    private static double overlapScore(Set<String> a, Set<String> b) {
        if (a.isEmpty() || b.isEmpty()) return 0.0;
        int inter = 0;
        // iterate smaller set
        if (a.size() > b.size()) {
            Set<String> t = a; a = b; b = t;
        }
        for (String x : a) if (b.contains(x)) inter++;
        return inter;
    }
}
