package tools.smoking;

import org.apache.uima.analysis_engine.AnalysisEngineDescription;
import org.apache.uima.fit.component.JCasAnnotator_ImplBase;
import org.apache.uima.jcas.JCas;
import org.apache.uima.resource.ResourceInitializationException;

public class NegationAnnotatorAE extends JCasAnnotator_ImplBase {
    public static AnalysisEngineDescription createAnnotatorDescription() throws ResourceInitializationException {
        return DescriptorLoader.load("ctakes-ne-contexts/desc/NegationAnnotator.xml");
    }
    @Override public void process(JCas jCas) { }
}

