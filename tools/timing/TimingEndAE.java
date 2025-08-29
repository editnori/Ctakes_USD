package tools.timing;

import org.apache.uima.analysis_component.JCasAnnotator_ImplBase;
import org.apache.uima.analysis_engine.AnalysisEngineProcessException;
import org.apache.uima.jcas.JCas;

import java.io.BufferedWriter;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.StandardOpenOption;

/**
 * Minimal timing annotator to mark a per-document end. Place at the very end of the pipeline
 * (after writers) so the duration covers writing time.
 *
 * Emits to stdout (per CAS):
 *   [timing] END\t<docKey>\t<startMillis>\t<endMillis>\t<durationMillis>
 *
 * Optional config parameter: TimingFile (absolute path) — if set, append TSV rows:
 *   docKey\tstartMillis\tendMillis\tdurationMillis
 */
public class TimingEndAE extends JCasAnnotator_ImplBase {
    public static final String PARAM_TIMING_FILE = "TimingFile";
    private volatile Path timingPath;
    private final Object fileLock = new Object();

    @Override
    public void process(JCas jCas) throws AnalysisEngineProcessException {
        if (timingPath == null) {
            Object v = getContext().getConfigParameterValue(PARAM_TIMING_FILE);
            if (v instanceof String && !((String) v).trim().isEmpty()) {
                timingPath = Paths.get(((String) v).trim());
            }
        }
        String key = TimingUtil.docKey(jCas);
        long end = System.currentTimeMillis();
        Long start = TimingStartAE.STARTS.getOrDefault(key, null);
        long dur = (start == null) ? -1L : Math.max(0L, end - start);
        System.out.println("[timing] END\t" + key + "\t" + (start == null ? "" : start) + "\t" + end + "\t" + dur);
        if (timingPath != null) {
            String row = key + "\t" + (start == null ? "" : start) + "\t" + end + "\t" + dur + "\n";
            synchronized (fileLock) {
                try (BufferedWriter bw = Files.newBufferedWriter(timingPath,
                        StandardCharsets.UTF_8,
                        StandardOpenOption.CREATE, StandardOpenOption.APPEND)) {
                    bw.write(row);
                } catch (IOException ignore) {}
            }
        }
    }
}

