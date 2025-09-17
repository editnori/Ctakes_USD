package tools.drug;

import org.apache.ctakes.drugner.ae.DrugMentionAnnotator;
import org.apache.uima.UimaContext;
import org.apache.uima.analysis_engine.AnalysisEngineDescription;
import org.apache.uima.analysis_engine.AnalysisEngineProcessException;
import org.apache.uima.cas.Type;
import org.apache.uima.cas.impl.TypeImpl;
import org.apache.uima.fit.factory.AnalysisEngineFactory;
import org.apache.uima.jcas.JCas;
import org.apache.uima.resource.ResourceInitializationException;
import org.apache.uima.util.InvalidXMLException;

import java.io.IOException;
import java.lang.reflect.Field;
import java.util.Objects;

/**
 * Thin wrapper around ctakes-drug-ner's DrugMentionAnnotator that reuses the upstream descriptor
 * (patched via resources_override) so the pipeline can load it through Piper. The stock
 * DrugMentionAnnotator class does not expose a static createEngineDescription method, which causes
 * Piper to fail when resolving the component. Providing one here keeps all original behaviour while
 * ensuring the descriptor's TypeSystem imports remain intact.
 */
public class DrugMentionAnnotatorWithTypes extends DrugMentionAnnotator {

    private static final String DESCRIPTOR_PATH =
            "ctakes-drug-ner/desc/analysis_engine/DrugMentionAnnotator.xml";
    private static final String DEFAULT_SENTENCE_TYPE =
            "org.apache.ctakes.typesystem.type.textspan.Sentence";

    private static final Field ANNOTATION_TYPE_FIELD;
    private static final Field BOUNDARY_TYPE_FIELD;

    static {
        try {
            ANNOTATION_TYPE_FIELD = DrugMentionAnnotator.class.getDeclaredField("iAnnotationType");
            ANNOTATION_TYPE_FIELD.setAccessible(true);
            BOUNDARY_TYPE_FIELD = DrugMentionAnnotator.class.getDeclaredField("iBoundaryAnnType");
            BOUNDARY_TYPE_FIELD.setAccessible(true);
        } catch (NoSuchFieldException e) {
            throw new ExceptionInInitializerError(e);
        }
    }

    private String distanceAnnotationName;
    private String boundaryAnnotationName;

    public DrugMentionAnnotatorWithTypes() {
        // default constructor required for UIMA reflection
    }

    public static AnalysisEngineDescription createEngineDescription() throws ResourceInitializationException {
        try {
            return AnalysisEngineFactory.createEngineDescriptionFromPath(DESCRIPTOR_PATH);
        } catch (IOException | InvalidXMLException e) {
            throw new ResourceInitializationException(e);
        }
    }

    @Override
    public void initialize(UimaContext context) throws ResourceInitializationException {
        distanceAnnotationName = (String) context.getConfigParameterValue(DISTANCE_ANN_TYPE);
        boundaryAnnotationName = (String) context.getConfigParameterValue(BOUNDARY_ANN_TYPE);
        super.initialize(context);
    }

    @Override
    public void process(JCas jcas) throws AnalysisEngineProcessException {
        ensureTypeIds(jcas);
        super.process(jcas);
    }

    private void ensureTypeIds(JCas jcas) throws AnalysisEngineProcessException {
        try {
            if (ANNOTATION_TYPE_FIELD.getInt(this) == NO_ANNOTATION_TYPE_SPECIFIED) {
                ANNOTATION_TYPE_FIELD.setInt(this, resolveTypeCode(jcas, distanceAnnotationName));
            }
            if (BOUNDARY_TYPE_FIELD.getInt(this) == NO_ANNOTATION_TYPE_SPECIFIED) {
                BOUNDARY_TYPE_FIELD.setInt(this, resolveTypeCode(jcas, boundaryAnnotationName));
            }
        } catch (IllegalAccessException e) {
            throw new AnalysisEngineProcessException(e);
        }
    }

    private int resolveTypeCode(JCas jcas, String configuredName) throws AnalysisEngineProcessException {
        String candidate = Objects.requireNonNullElse(configuredName, DEFAULT_SENTENCE_TYPE);
        Type resolved = jcas.getTypeSystem().getType(candidate);
        if (resolved == null && !DEFAULT_SENTENCE_TYPE.equals(candidate)) {
            resolved = jcas.getTypeSystem().getType(DEFAULT_SENTENCE_TYPE);
        }
        if (resolved == null) {
            throw new AnalysisEngineProcessException(new IllegalStateException(
                    "Unable to resolve type for " + candidate));
        }
        return ((TypeImpl) resolved).getCode();
    }
}
