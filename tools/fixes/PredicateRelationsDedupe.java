package tools.fixes;

import org.apache.ctakes.typesystem.type.textsem.Predicate;
import org.apache.uima.analysis_component.JCasAnnotator_ImplBase;
import org.apache.uima.analysis_engine.AnalysisEngineProcessException;
import org.apache.uima.fit.util.JCasUtil;
import org.apache.uima.jcas.JCas;
import org.apache.uima.jcas.cas.FSArray;
import org.apache.uima.jcas.cas.TOP;

import java.util.LinkedHashSet;

/**
 * Prunes duplicate entries from Predicate.relations to avoid illegal
 * multi-references during XMI serialization. Keeps first occurrence order.
 */
public class PredicateRelationsDedupe extends JCasAnnotator_ImplBase {
    @Override
    public void process(JCas jCas) throws AnalysisEngineProcessException {
        for (Predicate p : JCasUtil.select(jCas, Predicate.class)) {
            FSArray rels = p.getRelations();
            if (rels == null || rels.size() <= 1) continue;
            LinkedHashSet<TOP> uniq = new LinkedHashSet<>();
            for (int i = 0; i < rels.size(); i++) {
                TOP fs = (TOP) rels.get(i);
                if (fs != null) uniq.add(fs);
            }
            if (uniq.size() != rels.size()) {
                FSArray cleaned = new FSArray(jCas, uniq.size());
                int idx = 0;
                for (TOP fs : uniq) cleaned.set(idx++, fs);
                cleaned.addToIndexes();
                p.setRelations(cleaned);
            }
        }
    }
}

