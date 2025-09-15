package tools.smoking;

import java.net.URL;

import org.apache.uima.UIMAFramework;
import org.apache.uima.analysis_engine.AnalysisEngineDescription;
import org.apache.uima.fit.component.JCasAnnotator_ImplBase;
import org.apache.uima.jcas.JCas;
import org.apache.uima.resource.ResourceInitializationException;
import org.apache.uima.resource.ResourceSpecifier;
import org.apache.uima.util.InvalidXMLException;
import org.apache.uima.util.XMLInputSource;

public class SmokingAggregateStep1 extends JCasAnnotator_ImplBase {
    private static final String RESOURCE =
            "ctakes-smoking-status/desc/analysis_engine/ProductionPostSentenceAggregate_step1.xml";

    public static AnalysisEngineDescription createAnnotatorDescription() throws ResourceInitializationException {
        try {
            URL url = SmokingAggregateStep1.class.getClassLoader().getResource(RESOURCE);
            if (url == null) {
                throw new ResourceInitializationException(new IllegalArgumentException("Resource not found: " + RESOURCE));
            }
            XMLInputSource xin = new XMLInputSource(url);
            ResourceSpecifier spec = UIMAFramework.getXMLParser().parseResourceSpecifier(xin);
            if (spec instanceof AnalysisEngineDescription) {
                return (AnalysisEngineDescription) spec;
            }
            throw new ResourceInitializationException(new InvalidXMLException(new Exception("Not an AE descriptor: " + RESOURCE)));
        } catch (Exception e) {
            if (e instanceof ResourceInitializationException) {
                throw (ResourceInitializationException) e;
            }
            throw new ResourceInitializationException(e);
        }
    }

    @Override
    public void process(JCas jCas) { /* never called; descriptor is used */ }
}
