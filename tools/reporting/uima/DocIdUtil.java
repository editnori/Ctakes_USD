package tools.reporting.uima;

import java.lang.reflect.Method;

import org.apache.ctakes.typesystem.type.structured.DocumentID;
import org.apache.uima.fit.util.JCasUtil;
import org.apache.uima.jcas.JCas;
import org.apache.uima.jcas.cas.TOP;

/**
 * Utility helpers for resolving document identifiers in writers.
 */
public final class DocIdUtil {
    private DocIdUtil() {
    }

    public static String resolveDocId(JCas jCas) {
        String fromDocument = fromDocumentAnnotation(jCas);
        if (!fromDocument.isEmpty()) {
            return fromDocument;
        }
        for (DocumentID id : JCasUtil.select(jCas, DocumentID.class)) {
            String value = clean(id.getDocumentID());
            if (!value.isEmpty()) {
                return value;
            }
        }
        return "note";
    }

    private static String fromDocumentAnnotation(JCas jCas) {
        TOP annotation = jCas.getDocumentAnnotationFs();
        if (annotation == null) {
            return "";
        }
        String[] candidates = {
                invokeString(annotation, "getDocumentID"),
                invokeString(annotation, "getSourceUri"),
                invokeString(annotation, "getSourceUriString"),
        };
        for (String candidate : candidates) {
            String cleaned = clean(candidate);
            if (!cleaned.isEmpty()) {
                return cleaned;
            }
        }
        return "";
    }

    private static String invokeString(Object target, String methodName) {
        if (target == null) {
            return "";
        }
        try {
            Method method = target.getClass().getMethod(methodName);
            Object value = method.invoke(target);
            return value instanceof String ? (String) value : "";
        } catch (Exception ignore) {
            return "";
        }
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
