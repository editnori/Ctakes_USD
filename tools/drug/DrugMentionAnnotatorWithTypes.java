package tools.drug;

import org.apache.ctakes.drugner.ae.DrugMentionAnnotator;
import org.apache.uima.analysis_engine.AnalysisEngineDescription;
import org.apache.uima.fit.factory.AnalysisEngineFactory;
import org.apache.uima.resource.ResourceInitializationException;
import org.apache.uima.util.InvalidXMLException;

import java.io.IOException;

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

    private DrugMentionAnnotatorWithTypes() {
        // no instances
    }

    public static AnalysisEngineDescription createEngineDescription() throws ResourceInitializationException {
        try {
            return AnalysisEngineFactory.createEngineDescriptionFromPath(DESCRIPTOR_PATH);
        } catch (IOException | InvalidXMLException e) {
            throw new ResourceInitializationException(e);
        }
    }
}
