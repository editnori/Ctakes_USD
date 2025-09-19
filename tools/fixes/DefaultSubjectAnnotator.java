package tools.fixes;

import org.apache.ctakes.typesystem.type.textsem.IdentifiedAnnotation;
import org.apache.uima.analysis_component.JCasAnnotator_ImplBase;
import org.apache.uima.analysis_engine.AnalysisEngineProcessException;
import org.apache.uima.fit.util.JCasUtil;
import org.apache.uima.jcas.JCas;

/**
 * Ensures IdentifiedAnnotation.subject is non-null to avoid downstream writer NPEs.
 * If subject is null or empty, sets it to "patient".
 */
public class DefaultSubjectAnnotator extends JCasAnnotator_ImplBase {
    @Override
    public void process(JCas jCas) throws AnalysisEngineProcessException {
        for (IdentifiedAnnotation ia : JCasUtil.select(jCas, IdentifiedAnnotation.class)) {
            String subj = ia.getSubject();
            if (subj == null || subj.isEmpty()) {
                ia.setSubject("patient");
            }
        }
    }
}

