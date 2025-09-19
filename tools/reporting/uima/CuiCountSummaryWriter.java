package tools.reporting.uima;

import org.apache.ctakes.typesystem.type.refsem.UmlsConcept;
import org.apache.ctakes.typesystem.type.structured.DocumentID;
import org.apache.ctakes.typesystem.type.textsem.IdentifiedAnnotation;
import org.apache.uima.UimaContext;
import org.apache.uima.analysis_engine.AnalysisEngineProcessException;
import org.apache.uima.fit.descriptor.ConfigurationParameter;
import org.apache.uima.fit.util.JCasUtil;
import org.apache.uima.jcas.JCas;
import org.apache.uima.resource.ResourceInitializationException;

import java.io.BufferedWriter;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.StandardOpenOption;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.TreeMap;

/**
 * Writes a per-document bar-separated summary of disambiguated CUIs and their affirmed/negated counts.
 * Normalizes CUIs that appear with leading polarity markers (e.g. "-C1234567") so the polarity is tracked in,
 * dedicated columns instead of the identifier itself.
 */
public class CuiCountSummaryWriter extends org.apache.uima.analysis_component.JCasAnnotator_ImplBase {

    public static final String PARAM_SUBDIR = "SubDirectory";

    @ConfigurationParameter(name = PARAM_SUBDIR, mandatory = false)
    private String subDir = "cui_counts";

    private String outputBase;

    @Override
    public void initialize(UimaContext context) throws ResourceInitializationException {
        super.initialize(context);
        Object od = context.getConfigParameterValue("OutputDirectory");
        if (od == null) {
            od = System.getProperty("ctakes.output.dir");
        }
        if (od == null) {
            od = System.getProperty("OUTPUT_DIR");
        }
        outputBase = (od == null) ? "." : od.toString();
        Object sd = context.getConfigParameterValue(PARAM_SUBDIR);
        if (sd != null && !sd.toString().trim().isEmpty()) {
            subDir = sd.toString().trim();
        }
    }

    @Override
    public void process(JCas jCas) throws AnalysisEngineProcessException {
        Map<String, Counts> countsByCui = new TreeMap<>();
        for (IdentifiedAnnotation annotation : JCasUtil.select(jCas, IdentifiedAnnotation.class)) {
            UmlsConcept concept = firstConcept(annotation);
            if (concept == null) {
                continue;
            }
            String normalized = normalizeCui(concept.getCui());
            if (normalized.isEmpty()) {
                continue;
            }
            boolean negated = annotation.getPolarity() == -1;
            countsByCui.computeIfAbsent(normalized, key -> new Counts()).increment(negated);
        }

        List<Map.Entry<String, Counts>> ordered = new ArrayList<>(countsByCui.entrySet());
        ordered.sort(Comparator
                .comparingInt((Map.Entry<String, Counts> e) -> e.getValue().total()).reversed()
                .thenComparing(Map.Entry::getKey));

        Path outDir = Paths.get(outputBase, subDir);
        try {
            Files.createDirectories(outDir);
        } catch (IOException e) {
            throw new AnalysisEngineProcessException(e);
        }
        Path outFile = outDir.resolve(getDocId(jCas) + ".bsv");
        try (BufferedWriter writer = Files.newBufferedWriter(outFile, StandardCharsets.UTF_8,
                StandardOpenOption.CREATE, StandardOpenOption.TRUNCATE_EXISTING)) {
            writer.write("CUI|Affirmed|Negated");
            writer.newLine();
            for (Map.Entry<String, Counts> entry : ordered) {
                Counts counts = entry.getValue();
                writer.write(entry.getKey());
                writer.write('|');
                writer.write(Integer.toString(counts.affirmed));
                writer.write('|');
                writer.write(Integer.toString(counts.negated));
                writer.newLine();
            }
        } catch (IOException e) {
            throw new AnalysisEngineProcessException(e);
        }
    }

    private static UmlsConcept firstConcept(IdentifiedAnnotation annotation) {
        if (annotation == null || annotation.getOntologyConceptArr() == null
                || annotation.getOntologyConceptArr().size() == 0) {
            return null;
        }
        for (int i = 0; i < annotation.getOntologyConceptArr().size(); i++) {
            if (annotation.getOntologyConceptArr().get(i) instanceof UmlsConcept) {
                return (UmlsConcept) annotation.getOntologyConceptArr().get(i);
            }
        }
        return null;
    }

    private static String normalizeCui(String raw) {
        if (raw == null) {
            return "";
        }
        String trimmed = raw.trim();
        while (!trimmed.isEmpty() && (trimmed.charAt(0) == '-' || trimmed.charAt(0) == '+')) {
            trimmed = trimmed.substring(1);
        }
        return trimmed.toUpperCase(Locale.ROOT);
    }

    private static String getDocId(JCas jCas) {
        for (DocumentID id : JCasUtil.select(jCas, DocumentID.class)) {
            String value = id.getDocumentID();
            if (value != null) {
                String trimmed = value.trim();
                if (!trimmed.isEmpty()) {
                    return trimmed;
                }
            }
        }
        return "UNKNOWN";
    }

    private static final class Counts {
        int affirmed;
        int negated;

        void increment(boolean isNegated) {
            if (isNegated) {
                negated++;
            } else {
                affirmed++;
            }
        }

        int total() {
            return affirmed + negated;
        }
    }
}

