package tools.timing;

import org.apache.ctakes.typesystem.type.structured.DocumentID;
import org.apache.uima.examples.SourceDocumentInformation;
import org.apache.uima.jcas.JCas;

import java.net.URI;
import java.nio.file.Paths;

final class TimingUtil {
    private TimingUtil() {}

    static String docKey(JCas jCas) {
        try {
            // Try cTAKES DocumentID
            org.apache.uima.cas.FSIterator<?> it = jCas.getAnnotationIndex(DocumentID.type).iterator();
            if (it.hasNext()) {
                Object o = it.next();
                if (o instanceof DocumentID) {
                    String id = ((DocumentID) o).getDocumentID();
                    if (id != null && !id.isEmpty()) return stripTxt(extless(id));
                }
            }
        } catch (Throwable ignore) {}
        try {
            // Try UIMA example SourceDocumentInformation
            org.apache.uima.cas.FSIterator<?> it2 = jCas.getAnnotationIndex(SourceDocumentInformation.type).iterator();
            if (it2.hasNext()) {
                Object o2 = it2.next();
                if (o2 instanceof SourceDocumentInformation) {
                    String uri = ((SourceDocumentInformation) o2).getUri();
                    if (uri != null && !uri.isEmpty()) {
                        try { return stripTxt(extless(Paths.get(URI.create(uri)).getFileName().toString())); } catch (Exception ignore) {}
                        try { return stripTxt(extless(Paths.get(uri).getFileName().toString())); } catch (Exception ignore) {}
                    }
                }
            }
        } catch (Throwable ignore) {}
        // Fallback: synthetic ID based on hash
        String txt = jCas.getDocumentText();
        int h = (txt==null?0:txt.length()) ^ System.identityHashCode(jCas);
        return "DOC_" + Integer.toHexString(h);
    }

    private static String extless(String s) {
        int i = s.lastIndexOf('.');
        return i > 0 ? s.substring(0, i) : s;
    }
    private static String stripTxt(String s) {
        return s.endsWith(".txt") ? s.substring(0, s.length()-4) : s;
    }
}
