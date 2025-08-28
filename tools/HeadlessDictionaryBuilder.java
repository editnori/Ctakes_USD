package tools;

import org.apache.ctakes.core.util.annotation.SemanticTui;

import java.io.File;
import java.io.FileInputStream;
import java.io.IOException;
import java.util.*;

/**
 * Headless wrapper around cTAKES GUI DictionaryBuilder.
 * Loads a simple .properties file and invokes DictionaryBuilder.buildDictionary.
 *
 * Required properties:
 *  - umls.dir: path to UMLS RRF directory
 *  - dictionary.name: name/id for the dictionary (used for XML and DB name)
 * Optional properties:
 *  - languages: comma-separated list (default: ENG)
 *  - vocabularies: SAB list (e.g., SNOMEDCT_US,RXNORM,LOINC,...). If omitted, all sources used.
 *  - term.types: TTY list (e.g., PT,SY,BN,IN,...). If omitted, all types used.
 *  - semantic.types: TUI list (e.g., T047,T184). If omitted, all TUIs used.
 *
 * Environment:
 *  - CTAKES_HOME must point to the cTAKES installation root.
 */
public class HeadlessDictionaryBuilder {

    private static void usage() {
        System.err.println("Usage: java ... tools.HeadlessDictionaryBuilder -p <builder.properties>");
    }

    public static void main(String[] args) throws Exception {
        if (args.length < 2 || !"-p".equals(args[0])) {
            usage();
            System.exit(1);
        }

        final String propsPath = args[1];
        final Properties props = new Properties();
        try (FileInputStream fis = new FileInputStream(propsPath)) {
            props.load(fis);
        } catch (IOException e) {
            System.err.println("Failed to read properties: " + propsPath + ": " + e.getMessage());
            System.exit(1);
            return;
        }

        final String ctakesHome = Optional.ofNullable(System.getenv("CTAKES_HOME"))
                .orElseThrow(() -> new IllegalStateException("CTAKES_HOME not set"));

        final String umlsDir = require(props, "umls.dir");
        final String dictName = require(props, "dictionary.name");

        final List<String> languages = splitList(props.getProperty("languages", "ENG"));
        final List<String> sources = normalizeSab(splitList(props.getProperty("vocabularies", "")));
        final List<String> termTypes = splitList(props.getProperty("term.types", ""));
        final List<SemanticTui> tuis = parseTuis(props.getProperty("semantic.types", ""));

        ensureResourcesDir(ctakesHome);

        // Resolve umls.dir relative to the repo root (parent of the properties file directory)
        final File propsFile = new File(propsPath).getAbsoluteFile();
        final File propsParent = propsFile.getParentFile();
        final File propsBase = propsParent != null ? propsParent.getParentFile() : null;
        final String repoBaseProp = System.getProperty("repo.base", "");
        File umlsRoot = new File(umlsDir);
        if (!umlsRoot.isAbsolute()) {
            if (!repoBaseProp.isEmpty()) {
                umlsRoot = new File(new File(repoBaseProp), umlsDir);
            } else if (propsBase != null) {
                umlsRoot = new File(propsBase, umlsDir);
            }
            if (!umlsRoot.isAbsolute()) {
                umlsRoot = umlsRoot.getAbsoluteFile();
            }
        }

        System.out.println("Building dictionary");
        System.out.println("  CTAKES_HOME:     " + ctakesHome);
        System.out.println("  UMLS_DIR:        " + umlsDir);
        System.out.println("  UMLS_DIR (abs):  " + umlsRoot.getAbsolutePath());
        System.out.println("  DICT_NAME:       " + dictName);
        System.out.println("  LANGUAGES:       " + String.join(",", languages));
        System.out.println("  VOCABULARIES:    " + (sources.isEmpty() ? "<ALL>" : String.join(",", sources)));
        System.out.println("  TERM.TYPES (TTY):" + (termTypes.isEmpty() ? "<ALL>" : String.join(",", termTypes)));
        System.out.println("  SEMANTIC.TYPES:  " + (tuis.isEmpty() ? "<ALL>" : tuis.toString()));

        // Reflectively invoke the package-private DictionaryBuilder.buildDictionary(...)
        final Class<?> dbClass = Class.forName("org.apache.ctakes.gui.dictionary.DictionaryBuilder");
        final java.lang.reflect.Method build = dbClass.getDeclaredMethod(
                "buildDictionary",
                String.class, String.class, String.class,
                java.util.Collection.class, java.util.Collection.class, java.util.Collection.class,
                java.util.Collection.class
        );
        build.setAccessible(true);
        // IMPORTANT: Parameter order expected by DictionaryBuilder is
        // (umlsDir, ctakesHome, dictName, languages, vocabularies(SAB), term.types(TTY), tuis)
        boolean ok = (Boolean) build.invoke(null,
                umlsRoot.getAbsolutePath(),
                new File(ctakesHome).getAbsolutePath(),
                dictName,
                languages,
                sources,
                termTypes,
                tuis
        );

        if (!ok) {
            System.err.println("Dictionary build failed.");
            System.exit(2);
        } else {
            // Where the dictionary XML lands per DictionaryBuilder: under CTAKES_HOME/resources/.../fast/<name>.xml
            final File xml = new File(new File(ctakesHome, "resources/org/apache/ctakes/dictionary/lookup/fast"), dictName + ".xml");
            System.out.println("Dictionary XML:    " + xml.getAbsolutePath());
            System.out.println("Done.");
        }
    }

    private static List<String> splitList(String csv) {
        if (csv == null || csv.trim().isEmpty()) return Collections.emptyList();
        final String[] parts = csv.split(",");
        final List<String> list = new ArrayList<>(parts.length);
        for (String p : parts) {
            final String s = p.trim();
            if (!s.isEmpty()) list.add(s);
        }
        return list;
    }

    private static List<SemanticTui> parseTuis(String csv) {
        if (csv == null || csv.trim().isEmpty()) return Collections.emptyList();
        final List<SemanticTui> list = new ArrayList<>();
        for (String p : csv.split(",")) {
            final String code = p.trim();
            if (code.isEmpty()) continue;
            list.add(SemanticTui.getTui(code));
        }
        return list;
    }

    private static List<String> normalizeSab(List<String> sabList) {
        if (sabList.isEmpty()) return sabList;
        final Map<String,String> map = new HashMap<>();
        map.put("LOINC", "LNC");
        map.put("MEDDRA", "MDR");
        map.put("NCIT", "NCI");
        map.put("NCI_THESAURUS", "NCI");
        // common pass-throughs
        final Set<String> passthru = new HashSet<>(Arrays.asList(
                "SNOMEDCT_US","RXNORM","ICD10CM","ICD10PCS","CPT","HCPCS",
                "RADLEX","HPO","ATC","ICF","MSH","CHV","MTH","LNC","MDR","NCI"
        ));
        final List<String> out = new ArrayList<>(sabList.size());
        for (String s : sabList) {
            final String up = s.trim().toUpperCase(Locale.ROOT);
            final String norm = map.getOrDefault(up, up);
            out.add(norm);
            if (!passthru.contains(norm)) {
                System.out.println("  [info] SAB normalized: " + up + " -> " + norm);
            }
        }
        return out;
    }

    private static String require(Properties p, String key) {
        final String v = p.getProperty(key);
        if (v == null || v.trim().isEmpty()) {
            throw new IllegalArgumentException("Missing required property: " + key);
        }
        return v.trim();
    }

    private static void ensureResourcesDir(String ctakesHome) {
        final File dir = new File(ctakesHome, "resources/org/apache/ctakes/dictionary/lookup/fast");
        if (!dir.isDirectory()) {
            boolean ok = dir.mkdirs();
            if (!ok && !dir.isDirectory()) {
                throw new RuntimeException("Failed to create resources dir: " + dir);
            }
        }
    }
}
