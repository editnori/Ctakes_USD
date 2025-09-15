package tools.timing;

import org.apache.uima.analysis_component.JCasAnnotator_ImplBase;
import org.apache.uima.analysis_engine.AnalysisEngineProcessException;
import org.apache.uima.fit.descriptor.ConfigurationParameter;
import org.apache.uima.jcas.JCas;

import java.io.BufferedWriter;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.StandardOpenOption;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicInteger;

/**
 * Lightweight progress logger AE.
 * Logs to stdout lines like:
 *   [progress] <label> k/N (p%) doc=<docKey>
 * Optionally appends TSV lines to ProgressFile:
 *   k\tN\tpercent\tlabel\tdocKey\n
 */
public class ProgressLoggerAE extends JCasAnnotator_ImplBase {
    public static final String PARAM_TOTAL_DOCS = "TotalDocs";
    public static final String PARAM_LABEL = "Label";
    public static final String PARAM_EVERY_N = "EveryN";
    public static final String PARAM_PROGRESS_FILE = "ProgressFile";

    @ConfigurationParameter(name = PARAM_TOTAL_DOCS, mandatory = false)
    private Integer totalDocs = 0;

    @ConfigurationParameter(name = PARAM_LABEL, mandatory = false)
    private String label = "";

    @ConfigurationParameter(name = PARAM_EVERY_N, mandatory = false)
    private Integer everyN = 10;

    @ConfigurationParameter(name = PARAM_PROGRESS_FILE, mandatory = false)
    private String progressFile;

    private static final Map<String, AtomicInteger> COUNTS = new ConcurrentHashMap<>();

    @Override
    public void process(JCas jCas) throws AnalysisEngineProcessException {
        String key = TimingUtil.docKey(jCas);
        String lab = label == null ? "" : label;
        int tot = (totalDocs == null || totalDocs < 0) ? 0 : totalDocs;
        int step = (everyN == null || everyN < 1) ? 10 : everyN;

        AtomicInteger ai = COUNTS.computeIfAbsent(lab, k -> new AtomicInteger(0));
        int k = ai.incrementAndGet();
        int denom = (tot <= 0) ? Math.max(k, 1) : tot;
        int pct = (int)Math.round(100.0 * k / denom);

        if (k == 1 || k % step == 0 || (tot > 0 && k >= tot)) {
            System.out.println("[progress] " + lab + " " + k + "/" + denom + " (" + pct + "%) doc=" + key);
            if (progressFile != null && !progressFile.trim().isEmpty()) {
                Path p = Paths.get(progressFile);
                String row = k + "\t" + denom + "\t" + pct + "\t" + lab + "\t" + key + "\n";
                try (BufferedWriter bw = Files.newBufferedWriter(p, StandardCharsets.UTF_8,
                        StandardOpenOption.CREATE, StandardOpenOption.APPEND)) {
                    bw.write(row);
                } catch (IOException ignore) {}
            }
        }
    }
}

