package tools.smoking;

import org.apache.uima.analysis_engine.AnalysisEngineDescription;
import org.apache.uima.fit.component.JCasAnnotator_ImplBase;
import org.apache.uima.jcas.JCas;
import org.apache.uima.resource.ResourceInitializationException;

/**
 * Wrapper for the Apache cTAKES SentenceAdjuster.
 * The "Wow! some sentence is null" errors come from the underlying implementation
 * when tokens aren't properly covered by sentences.
 * 
 * This wrapper loads the original descriptor and delegates all processing to it.
 * To fix the errors, ensure ArtificialSentenceAE runs before this.
 */
public class SentenceAdjusterAE extends JCasAnnotator_ImplBase {
    
    @Override 
    public void process(JCas jCas) { 
        // Processing is handled by the descriptor loaded in createAnnotatorDescription()
        // This empty method is just to satisfy the JCasAnnotator_ImplBase contract
    }
    
    public static AnalysisEngineDescription createAnnotatorDescription() throws ResourceInitializationException {
        return DescriptorLoader.load("org/apache/ctakes/smoking/status/analysis_engine/SentenceAdjuster.xml");
    }
}