package tools.stub;

import java.util.List;
import java.util.Map;
import java.util.Set;
import org.apache.ctakes.ytex.kernel.metric.ConceptSimilarityService.SimilarityMetricEnum;
import org.apache.ctakes.ytex.kernel.wsd.WordSenseDisambiguator;

public class StubWordSenseDisambiguator implements WordSenseDisambiguator {
    @Override
    public String disambiguate(List<Set<String>> contextConcepts, int targetIndex,
                               Set<String> targetConcepts, int windowSize,
                               SimilarityMetricEnum metric,
                               Map<String, Double> conceptFilter) {
        return null;
    }

    @Override
    public String disambiguate(List<Set<String>> contextConcepts, int targetIndex,
                               Set<String> targetConcepts, int windowSize,
                               SimilarityMetricEnum metric,
                               Map<String, Double> conceptFilter,
                               boolean useIC) {
        return null;
    }
}

