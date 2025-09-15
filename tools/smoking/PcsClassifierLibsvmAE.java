package tools.smoking;

import org.apache.uima.analysis_engine.AnalysisEngineDescription;
import org.apache.uima.fit.component.JCasAnnotator_ImplBase;
import org.apache.uima.jcas.JCas;
import org.apache.uima.resource.ResourceInitializationException;

public class PcsClassifierLibsvmAE extends JCasAnnotator_ImplBase {
    public static AnalysisEngineDescription createAnnotatorDescription() throws ResourceInitializationException {
        return DescriptorLoader.load("ctakes-smoking-status/desc/analysis_engine/PcsClassifierAnnotator_libsvm.xml");
    }
    @Override public void process(JCas jCas) { }
}
