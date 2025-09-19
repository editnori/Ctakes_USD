package tools.relations;

import java.util.ArrayList;
import java.util.Collection;
import java.util.List;

import org.apache.ctakes.relationextractor.ae.LocationOfRelationExtractorAnnotator;
import org.apache.ctakes.relationextractor.ae.RelationExtractorAnnotator.IdentifiedAnnotationPair;
import org.apache.ctakes.typesystem.type.syntax.BaseToken;
import org.apache.ctakes.typesystem.type.textsem.IdentifiedAnnotation;
import org.apache.uima.UimaContext;
import org.apache.uima.fit.util.JCasUtil;
import org.apache.uima.jcas.JCas;
import org.apache.uima.jcas.tcas.Annotation;
import org.apache.uima.resource.ResourceInitializationException;

/**
 * Drops relation candidates whose mentions are far apart to keep the location extractor light-weight.
 */
public class FastLocationOfRelationExtractor extends LocationOfRelationExtractorAnnotator {

    private static final int DEFAULT_MAX_TOKEN_DISTANCE = 30;
    private static final int MAX_TOKEN_DISTANCE = Integer.getInteger(
            "ctakes.relations.max_token_distance", DEFAULT_MAX_TOKEN_DISTANCE);

    @Override
    public void initialize(UimaContext context) throws ResourceInitializationException {
        super.initialize(context);
    }

    @Override
    public List<IdentifiedAnnotationPair> getCandidateRelationArgumentPairs(JCas jCas, Annotation covering) {
        List<IdentifiedAnnotationPair> original = super.getCandidateRelationArgumentPairs(jCas, covering);
        if (MAX_TOKEN_DISTANCE <= 0 || original.isEmpty()) {
            return original;
        }
        List<IdentifiedAnnotationPair> filtered = new ArrayList<>(original.size());
        for (IdentifiedAnnotationPair pair : original) {
            if (withinTokenDistance(jCas, pair.getArg1(), pair.getArg2(), MAX_TOKEN_DISTANCE)) {
                filtered.add(pair);
            }
        }
        return filtered;
    }

    private static boolean withinTokenDistance(JCas jCas, IdentifiedAnnotation left, IdentifiedAnnotation right,
            int maxTokens) {
        if (maxTokens <= 0) {
            return true;
        }
        IdentifiedAnnotation first = left.getBegin() <= right.getBegin() ? left : right;
        IdentifiedAnnotation second = first == left ? right : left;
        int count = tokenCount(jCas, first.getBegin(), first.getEnd());
        if (first.getEnd() <= second.getBegin()) {
            count += JCasUtil.selectBetween(jCas, BaseToken.class, first, second).size();
        }
        count += tokenCount(jCas, second.getBegin(), second.getEnd());
        if (first.getEnd() > second.getBegin()) {
            // Overlapping mentions require counting the union of their token spans only once.
            int overlapBegin = second.getBegin();
            int overlapEnd = Math.min(first.getEnd(), second.getEnd());
            count -= tokenCount(jCas, overlapBegin, overlapEnd);
        }
        return count <= maxTokens;
    }

    private static int tokenCount(JCas jCas, int begin, int end) {
        if (end <= begin) {
            return 0;
        }
        Collection<BaseToken> tokens = JCasUtil.selectCovered(jCas, BaseToken.class, begin, end);
        if (!tokens.isEmpty()) {
            return tokens.size();
        }
        int chars = end - begin;
        return Math.max(1, chars / 5);
    }
}
