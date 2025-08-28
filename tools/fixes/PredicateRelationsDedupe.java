package tools.fixes;

import org.apache.ctakes.typesystem.type.textsem.Predicate;
import org.apache.uima.analysis_component.JCasAnnotator_ImplBase;
import org.apache.uima.analysis_engine.AnalysisEngineProcessException;
import org.apache.uima.fit.util.JCasUtil;
import org.apache.uima.jcas.JCas;
import org.apache.uima.jcas.cas.FSArray;
import org.apache.uima.jcas.cas.FSList;
import org.apache.uima.jcas.cas.NonEmptyFSList;
import org.apache.uima.jcas.cas.EmptyFSList;
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
            // cTAKES 6 uses FSList<SemanticRoleRelation> for Predicate.relations.
            // Older builds may use FSArray. Handle both defensively.
            FSList list = p.getRelations();
            int listSize = sizeOf(list);
            if (list == null || listSize <= 1) continue;
            LinkedHashSet<TOP> uniq = toSet(list);
            if (uniq.size() != listSize) {
                FSList cleaned = fromSet(jCas, uniq);
                p.setRelations(cleaned);
            }
        }
    }

    private static int sizeOf(FSList list) {
        int n = 0; FSList cur = list;
        while (cur != null && cur instanceof NonEmptyFSList) {
            n++; cur = (FSList) ((NonEmptyFSList) cur).getTail();
        }
        return n;
    }

    private static LinkedHashSet<TOP> toSet(FSList list) {
        LinkedHashSet<TOP> s = new LinkedHashSet<>();
        FSList cur = list;
        while (cur != null && cur instanceof NonEmptyFSList) {
            TOP head = ((NonEmptyFSList) cur).getHead();
            if (head != null) s.add(head);
            cur = (FSList) ((NonEmptyFSList) cur).getTail();
        }
        return s;
    }

    private static FSList fromSet(JCas jCas, LinkedHashSet<TOP> s) {
        FSList out = new EmptyFSList(jCas);
        TOP[] items = s.toArray(new TOP[0]);
        for (int i = items.length - 1; i >= 0; i--) {
            NonEmptyFSList nel = new NonEmptyFSList(jCas);
            nel.setHead(items[i]);
            nel.setTail(out);
            out = nel;
        }
        return out;
    }
}
