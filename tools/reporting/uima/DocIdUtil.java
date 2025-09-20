package tools.reporting.uima;

import org.apache.ctakes.typesystem.type.structured.DocumentID;
import org.apache.ctakes.typesystem.type.structured.SourceDocumentInformation;
import org.apache.uima.jcas.JCas;
import org.apache.uima.fit.util.JCasUtil;

/**
 * Utility helpers for resolving document identifiers in writers.
 */
public final class DocIdUtil {
    private DocIdUtil() {
    }

    public static String resolveDocId(JCas jCas) {
        for (DocumentID id : JCasUtil.select(jCas, DocumentID.class)) {
            String value = clean(id.getDocumentID());
            if (!value.isEmpty()) {
                return value;
            }
        }
        for (SourceDocumentInformation sdi : JCasUtil.select(jCas, SourceDocumentInformation.class)) {
            String value = clean(sdi.getUri());
            if (!value.isEmpty()) {
                return value;
            }
        }
        return "note";
    }

    private static String clean(String value) {
        if (value == null) {
            return "";
        }
        String trimmed = value.trim();
        if (trimmed.isEmpty()) {
            return "";
        }
        trimmed = trimmed.replace('\\', '/');
        int slash = trimmed.lastIndexOf('/');
        if (slash >= 0 && slash + 1 < trimmed.length()) {
            trimmed = trimmed.substring(slash + 1);
        }
        return trimmed;
    }
}
