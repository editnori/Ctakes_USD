package tools.relations;

import org.apache.ctakes.relationextractor.ae.RelationExtractorAnnotator.IdentifiedAnnotationPair;
import org.apache.uima.UimaContext;
import org.apache.uima.analysis_engine.AnalysisEngineProcessException;
import org.apache.uima.analysis_engine.AnalysisEngineDescription;
import org.apache.uima.fit.factory.AnalysisEngineFactory;
import org.apache.uima.jcas.JCas;
import org.apache.uima.jcas.tcas.Annotation;
import org.apache.uima.resource.ResourceInitializationException;

/**
 * Thread-safe wrapper for {@link FastLocationOfRelationExtractor} that keeps a shared delegate instance.
 */
public final class ThreadSafeFastLocationExtractor extends FastLocationOfRelationExtractor {

    private static final Object LOCK = new Object();
    private static final FastLocationOfRelationExtractor DELEGATE = new FastLocationOfRelationExtractor();
    private static volatile boolean initialized = false;

    @Override
    public void initialize(UimaContext context) throws ResourceInitializationException {
        if (!initialized) {
            synchronized (LOCK) {
                if (!initialized) {
                    DELEGATE.initialize(context);
                    initialized = true;
                }
            }
        }
    }

    @Override
    public void process(JCas jCas) throws AnalysisEngineProcessException {
        synchronized (LOCK) {
            DELEGATE.process(jCas);
        }
    }

    @Override
    public java.util.List<IdentifiedAnnotationPair> getCandidateRelationArgumentPairs(JCas jCas, Annotation covering) {
        return DELEGATE.getCandidateRelationArgumentPairs(jCas, covering);
    }

    @Override
    public void collectionProcessComplete() throws AnalysisEngineProcessException {
        synchronized (LOCK) {
            DELEGATE.collectionProcessComplete();
        }
    }

    public static AnalysisEngineDescription createAnnotatorDescription() throws ResourceInitializationException {
        return AnalysisEngineFactory.createEngineDescription(ThreadSafeFastLocationExtractor.class);
    }
}
