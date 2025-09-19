package tools.wsd;

import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Locale;
import java.util.Set;
import java.util.regex.Pattern;

import org.apache.ctakes.typesystem.type.refsem.OntologyConcept;
import org.apache.ctakes.typesystem.type.refsem.UmlsConcept;
import org.apache.ctakes.typesystem.type.textsem.IdentifiedAnnotation;
import org.apache.ctakes.typesystem.type.textspan.Sentence;
import org.apache.uima.UimaContext;
import org.apache.uima.analysis_component.JCasAnnotator_ImplBase;
import org.apache.uima.analysis_engine.AnalysisEngineProcessException;
import org.apache.uima.fit.descriptor.ConfigurationParameter;
import org.apache.uima.fit.factory.ConfigurationParameterInitializer;
import org.apache.uima.fit.util.JCasUtil;
import org.apache.uima.jcas.JCas;
import org.apache.uima.jcas.cas.FSArray;
import org.apache.uima.resource.ResourceInitializationException;

/**
 * Simple, local word-sense disambiguator for cTAKES mentions.
 * - Picks exactly one best OntologyConcept per IdentifiedAnnotation.
 * - Scores candidates by overlap between context tokens (covering sentence)
 *   and candidate preferred text (normalized); tie-breakers favour stronger
 *   context matches, then mention similarity, then label length.
 * - Does not require YTEX or any external DB.
 */
public class SimpleWsdDisambiguatorAnnotator extends JCasAnnotator_ImplBase {
    public static final String PARAM_KEEP_ALL = "KeepAllCandidates";
    public static final String PARAM_MOVE_BEST_FIRST = "MoveBestFirst";
    public static final String PARAM_MARK_DISAMBIGUATED = "MarkDisambiguated";
    public static final String PARAM_MIN_TOKEN_LEN = "MinTokenLen"; // int, default 1
    public static final String PARAM_FILTER_SINGLE_CHAR_STOPS = "FilterSingleCharStops"; // boolean, default true

    @ConfigurationParameter(name = PARAM_KEEP_ALL, mandatory = false)
    private boolean keepAll = true;

    @ConfigurationParameter(name = PARAM_MOVE_BEST_FIRST, mandatory = false)
    private boolean moveBestFirst = true;

    @ConfigurationParameter(name = PARAM_MARK_DISAMBIGUATED, mandatory = false)
    private boolean markDisambiguated = true;

    @ConfigurationParameter(name = PARAM_MIN_TOKEN_LEN, mandatory = false)
    private int minTokenLen = 1;

    @ConfigurationParameter(name = PARAM_FILTER_SINGLE_CHAR_STOPS, mandatory = false)
    private boolean filterSingleCharStops = true;

    private static final Pattern SPLIT = Pattern.compile("[^a-z0-9]+");

    @Override
    public void initialize(UimaContext context) throws ResourceInitializationException {
        super.initialize(context);
        ConfigurationParameterInitializer.initialize(this, context);
        keepAll = coerceBoolean(context.getConfigParameterValue(PARAM_KEEP_ALL), keepAll);
        moveBestFirst = coerceBoolean(context.getConfigParameterValue(PARAM_MOVE_BEST_FIRST), moveBestFirst);
        markDisambiguated = coerceBoolean(context.getConfigParameterValue(PARAM_MARK_DISAMBIGUATED), markDisambiguated);
        filterSingleCharStops = coerceBoolean(context.getConfigParameterValue(PARAM_FILTER_SINGLE_CHAR_STOPS), filterSingleCharStops);
        minTokenLen = coerceInt(context.getConfigParameterValue(PARAM_MIN_TOKEN_LEN), minTokenLen);
        if (minTokenLen < 1) {
            minTokenLen = 1;
        } else if (minTokenLen > 10) {
            minTokenLen = 10;
        }
    }

    @Override
    public void process(JCas jCas) throws AnalysisEngineProcessException {
        for (IdentifiedAnnotation ia : JCasUtil.select(jCas, IdentifiedAnnotation.class)) {
            FSArray ocArr = ia.getOntologyConceptArr();
            if (ocArr == null) {
                continue;
            }
            int n = ocArr.size();
            if (n <= 0) {
                continue;
            }

            String contextText = contextTextForMention(jCas, ia);
            Set<String> contextTokens = toTokenSet(contextText);
            Set<String> mentionTokens = toTokenSet(ia.getCoveredText());
            if (contextTokens.isEmpty() && !mentionTokens.isEmpty()) {
                contextTokens = mentionTokens;
            }

            if (n == 1) {
                OntologyConcept oc = (OntologyConcept) ocArr.get(0);
                CandidateMetrics metrics = computeMetrics(oc, ia, contextTokens, mentionTokens);
                applyBest(ia, oc, metrics);
                continue;
            }

            int bestIdx = 0;
            double bestNorm = -1.0;
            double bestRaw = -1.0;
            double bestMentionAlign = -1.0;
            int bestLabelLen = -1;
            List<CandidateMetrics> metricsCache = new ArrayList<>(n);
            for (int i = 0; i < n; i++) {
                OntologyConcept oc = (OntologyConcept) ocArr.get(i);
                CandidateMetrics metrics = computeMetrics(oc, ia, contextTokens, mentionTokens);
                metricsCache.add(metrics);

                int cmp = Double.compare(metrics.normalizedContext, bestNorm);
                if (cmp > 0 || (cmp == 0 && betterTie(metrics, bestRaw, bestMentionAlign, bestLabelLen))) {
                    bestIdx = i;
                    bestNorm = metrics.normalizedContext;
                    bestRaw = metrics.contextOverlap;
                    bestMentionAlign = metrics.mentionAlignment;
                    bestLabelLen = metrics.labelLength;
                }
            }

            OntologyConcept best = (OntologyConcept) ocArr.get(bestIdx);
            CandidateMetrics bestMetrics = metricsCache.get(bestIdx);
            applyBest(ia, best, bestMetrics);

            if (keepAll) {
                FSArray kept = new FSArray(jCas, n);
                if (moveBestFirst) {
                    kept.set(0, best);
                    int k = 1;
                    for (int i = 0; i < n; i++) {
                        if (i == bestIdx) {
                            continue;
                        }
                        kept.set(k++, (OntologyConcept) ocArr.get(i));
                    }
                } else {
                    for (int i = 0; i < n; i++) {
                        kept.set(i, (OntologyConcept) ocArr.get(i));
                    }
                }
                ia.setOntologyConceptArr(kept);
            } else {
                FSArray one = new FSArray(jCas, 1);
                one.set(0, best);
                ia.setOntologyConceptArr(one);
            }
        }
    }

    private boolean betterTie(CandidateMetrics candidate,
                              double bestRaw,
                              double bestMentionAlign,
                              int bestLabelLen) {
        int cmp = Double.compare(candidate.contextOverlap, bestRaw);
        if (cmp > 0) {
            return true;
        }
        if (cmp < 0) {
            return false;
        }
        cmp = Double.compare(candidate.mentionAlignment, bestMentionAlign);
        if (cmp > 0) {
            return true;
        }
        if (cmp < 0) {
            return false;
        }
        return candidate.labelLength > bestLabelLen;
    }

    private CandidateMetrics computeMetrics(OntologyConcept oc,
                                            IdentifiedAnnotation ia,
                                            Set<String> contextTokens,
                                            Set<String> mentionTokens) {
        String label = preferredLabel(oc, ia);
        Set<String> labelTokens = toTokenSet(label);
        Set<String> evalTokens = labelTokens.isEmpty() ? mentionTokens : labelTokens;
        double contextOverlap = overlapScore(contextTokens, evalTokens);
        double normalizedContext = evalTokens.isEmpty() ? 0.0 : contextOverlap / Math.max(1.0, (double) evalTokens.size());
        double mentionAlignment = 0.0;
        if (!labelTokens.isEmpty() && !mentionTokens.isEmpty()) {
            double overlap = overlapScore(labelTokens, mentionTokens);
            mentionAlignment = overlap / Math.max(1.0, (double) labelTokens.size());
        }
        return new CandidateMetrics(normalizedContext,
                contextOverlap,
                mentionAlignment,
                label != null ? label.length() : 0);
    }

    private void applyBest(IdentifiedAnnotation ia, OntologyConcept concept, CandidateMetrics metrics) {
        if (markDisambiguated && concept instanceof UmlsConcept) {
            UmlsConcept umls = (UmlsConcept) concept;
            umls.setDisambiguated(true);
            try {
                umls.setScore(metrics.normalizedContext);
            } catch (Throwable t) {
                // optional setter may not exist on older type system definitions
            }
        }
        try {
            ia.setConfidence((float) metrics.normalizedContext);
        } catch (Throwable t) {
            // ignore optional setter issues
        }
    }

    private String contextTextForMention(JCas jCas, IdentifiedAnnotation ia) {
        List<Sentence> covering = JCasUtil.selectCovering(jCas, Sentence.class, ia.getBegin(), ia.getEnd());
        if (!covering.isEmpty()) {
            return covering.get(0).getCoveredText();
        }
        return ia.getCoveredText();
    }

    private String preferredLabel(OntologyConcept oc, IdentifiedAnnotation ia) {
        if (oc instanceof UmlsConcept) {
            UmlsConcept umls = (UmlsConcept) oc;
            String label = umls.getPreferredText();
            if (label != null && !label.isEmpty()) {
                return label;
            }
        }
        if (oc.getCode() != null && !oc.getCode().isEmpty()) {
            return oc.getCode();
        }
        return ia.getCoveredText();
    }

    private Set<String> toTokenSet(String text) {
        Set<String> tokens = new HashSet<>();
        if (text == null) {
            return tokens;
        }
        String[] parts = SPLIT.split(text.toLowerCase(Locale.ROOT));
        for (String p : parts) {
            if (p.isEmpty()) {
                continue;
            }
            if (p.length() < minTokenLen) {
                continue;
            }
            if (filterSingleCharStops && p.length() == 1 && ("a".equals(p) || "i".equals(p))) {
                continue;
            }
            tokens.add(p);
        }
        return tokens;
    }

    private boolean coerceBoolean(Object value, boolean fallback) {
        if (value == null) {
            return fallback;
        }
        if (value instanceof Boolean) {
            return (Boolean) value;
        }
        if (value instanceof String) {
            String v = ((String) value).trim();
            if (!v.isEmpty()) {
                return Boolean.parseBoolean(v);
            }
        }
        return fallback;
    }

    private int coerceInt(Object value, int fallback) {
        if (value == null) {
            return fallback;
        }
        if (value instanceof Integer) {
            return (Integer) value;
        }
        if (value instanceof Number) {
            return ((Number) value).intValue();
        }
        if (value instanceof String) {
            String v = ((String) value).trim();
            if (!v.isEmpty()) {
                try {
                    return Integer.parseInt(v);
                } catch (NumberFormatException ignored) {
                    // fall through to fallback
                }
            }
        }
        return fallback;
    }

    private static double overlapScore(Set<String> a, Set<String> b) {
        if (a.isEmpty() || b.isEmpty()) {
            return 0.0;
        }
        Set<String> smaller = a.size() <= b.size() ? a : b;
        Set<String> larger = smaller == a ? b : a;
        double inter = 0.0;
        for (String token : smaller) {
            if (larger.contains(token)) {
                inter++;
            }
        }
        return inter;
    }

    private static final class CandidateMetrics {
        final double normalizedContext;
        final double contextOverlap;
        final double mentionAlignment;
        final int labelLength;

        CandidateMetrics(double normalizedContext,
                         double contextOverlap,
                         double mentionAlignment,
                         int labelLength) {
            this.normalizedContext = normalizedContext;
            this.contextOverlap = contextOverlap;
            this.mentionAlignment = mentionAlignment;
            this.labelLength = labelLength;
        }
    }
}


