package tools.timing;

import org.apache.uima.analysis_component.JCasAnnotator_ImplBase;
import org.apache.uima.analysis_engine.AnalysisEngineProcessException;
import org.apache.uima.jcas.JCas;
import org.apache.uima.jcas.tcas.Annotation;

import java.util.concurrent.ConcurrentHashMap;

/**
 * Minimal timing annotator to mark a per-document start. Place near the top of the pipeline.
 * Logs a stable line and records the start in a static map used by TimingEndAE.
 *
 * Emits to stdout (per CAS):
 *   [timing] START\t<docKey>\t<startMillis>
 */
public class TimingStartAE extends JCasAnnotator_ImplBase {
    static final ConcurrentHashMap<String, Long> STARTS = new ConcurrentHashMap<>();

    @Override
    public void process(JCas jCas) throws AnalysisEngineProcessException {
        String key = TimingUtil.docKey(jCas);
        long now = System.currentTimeMillis();
        STARTS.put(key, now);
        System.out.println("[timing] START\t" + key + "\t" + now);
    }
}

