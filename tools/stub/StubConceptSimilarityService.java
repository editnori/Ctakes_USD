package tools.stub;

import java.util.BitSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import org.apache.ctakes.ytex.kernel.metric.*;
import org.apache.ctakes.ytex.kernel.model.ConceptGraph;

public class StubConceptSimilarityService implements ConceptSimilarityService {
    @Override public String getConceptGraphName() { return "stub"; }
    @Override public int lcs(String a, String b, List<LCSPath> paths) { return 0; }
    @Override public ConceptGraph getConceptGraph() { return null; }
    @Override public Map<String, BitSet> getCuiTuiMap() { return java.util.Collections.emptyMap(); }
    @Override public List<String> getTuiList() { return java.util.Collections.emptyList(); }
    @Override public double loadConceptFilter(String s, int i, Map<String, Double> map) { return 0.0; }
    @Override public int getLCS(String a, String b, Set<String> filter, List<LCSPath> paths) { return 0; }
    @Override public Object[] getBestLCS(Set<String> set, boolean b, Map<String, Double> map) { return null; }
    @Override public double getIC(String s, boolean b) { return 0.0; }
    @Override public ConceptPairSimilarity similarity(List<ConceptSimilarityService.SimilarityMetricEnum> metrics, String a, String b, Map<String, Double> map, boolean b2) { return null; }
    @Override public List<ConceptPairSimilarity> similarity(List<ConceptPair> pairs, List<ConceptSimilarityService.SimilarityMetricEnum> metrics, Map<String, Double> map, boolean b) { return java.util.Collections.emptyList(); }
    @Override public int getDepth(String s) { return 0; }
}

