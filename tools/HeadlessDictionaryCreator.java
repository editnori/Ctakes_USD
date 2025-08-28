package tools;

import java.io.*;
import java.nio.charset.StandardCharsets;
import java.util.*;

/**
 * Headless scan that mimics cTAKES GUI DictionaryCreator discovery:
 * - Enumerates vocabularies (SAB) and languages (LAT) from MRCONSO.RRF
 * - Maps SAB names/versions from MRSAB.RRF (best effort)
 * - Counts CUIs per TUI from MRSTY.RRF
 * Prints progress similar to GUI and a summary at the end.
 * Emits two lines for consumption by scripts:
 *   DISCOVERED_SABS=A,B,C
 *   DISCOVERED_LANGUAGES=ENG,SPA,...
 */
public class HeadlessDictionaryCreator {
    public static void main(String[] args) throws Exception {
        if (args.length < 2 || !"-p".equals(args[0])) {
            System.err.println("Usage: java ... tools.HeadlessDictionaryCreator -p <builder.properties>");
            System.exit(1);
        }
        final String propsPath = args[1];
        final Properties props = new Properties();
        try (FileInputStream fis = new FileInputStream(propsPath)) {
            props.load(fis);
        }
        String umlsDir = require(props, "umls.dir");
        File umlsRoot = new File(umlsDir);
        if (!umlsRoot.isAbsolute()) {
            File repoBase = new File(propsPath).getAbsoluteFile().getParentFile();
            if (repoBase != null) repoBase = repoBase.getParentFile();
            if (repoBase != null) umlsRoot = new File(repoBase, umlsDir);
        }
        if (!umlsRoot.isAbsolute()) umlsRoot = umlsRoot.getAbsoluteFile();

        File meta = new File(umlsRoot, "META");
        File mrconso = new File(meta.exists() ? meta : umlsRoot, "MRCONSO.RRF");
        File mrsab = new File(meta.exists() ? meta : umlsRoot, "MRSAB.RRF");
        File mrsty = new File(meta.exists() ? meta : umlsRoot, "MRSTY.RRF");

        System.out.println("Scanning UMLS at: " + umlsRoot.getAbsolutePath());

        // MRCONSO scan
        Map<String, Long> sabCounts = new HashMap<>();
        Map<String, Long> latCounts = new HashMap<>();
        long lines = 0;
        try (BufferedReader br = reader(mrconso)) {
            String line;
            while ((line = br.readLine()) != null) {
                lines++;
                String[] f = split(line);
                if (f.length < 15) continue;
                String lat = f[1];
                String sab = f[11];
                if (lat != null && !lat.isEmpty()) latCounts.merge(lat, 1L, Long::sum);
                if (sab != null && !sab.isEmpty()) sabCounts.merge(sab, 1L, Long::sum);
                if (lines % 100000 == 0) {
                    System.out.println(String.format(Locale.ROOT, "File Line %,d  Vocabularies %d", lines, sabCounts.size()));
                }
            }
        }
        System.out.println(String.format(Locale.ROOT, "Parsed %d vocabulary types", sabCounts.size()));
        System.out.println(String.format(Locale.ROOT, "Parsed %d languages", latCounts.size()));

        // MRSAB map (best effort: RSAB code → VSAB long/version if available)
        Map<String, String> sabName = new HashMap<>();
        Map<String, String> sabVer = new HashMap<>();
        if (mrsab.exists()) {
            try (BufferedReader br = reader(mrsab)) {
                String line;
                while ((line = br.readLine()) != null) {
                    String[] f = split(line);
                    // Try sane defaults: RSAB (3) ~ index2, VSAB (4) ~ idx3, SVER maybe idx6
                    // Fields vary by release; guard with length checks
                    String rsab = val(f, 3);
                    String vsab = val(f, 4);
                    String sver = val(f, 6);
                    if (rsab != null && !rsab.isEmpty()) {
                        sabName.put(rsab, vsab != null ? vsab : rsab);
                        if (sver != null && !sver.isEmpty()) sabVer.put(rsab, sver);
                    }
                }
            }
            System.out.println(String.format(Locale.ROOT, "Parsed %d vocabulary names", sabName.size()));
        }

        // MRSTY scan
        Map<String, Set<String>> tuiCuis = new HashMap<>();
        long styLines = 0;
        if (mrsty.exists()) {
            try (BufferedReader br = reader(mrsty)) {
                String line;
                while ((line = br.readLine()) != null) {
                    styLines++;
                    String[] f = split(line);
                    if (f.length < 2) continue;
                    String cui = f[0];
                    String tui = f[1];
                    if (cui == null || cui.isEmpty() || tui == null || tui.isEmpty()) continue;
                    tuiCuis.computeIfAbsent(tui, k -> new HashSet<>()).add(cui);
                    if (styLines % 200000 == 0) {
                        // print a compact summary of top TUIs
                        System.out.println(progressTuiSummary(tuiCuis, styLines));
                    }
                }
            }
        }

        // Compose discovered sets
        List<String> sabList = new ArrayList<>(sabCounts.keySet());
        Collections.sort(sabList);
        List<String> latList = new ArrayList<>(latCounts.keySet());
        Collections.sort(latList);
        List<String> tuiList = new ArrayList<>(tuiCuis.keySet());
        Collections.sort(tuiList);

        System.out.println("DISCOVERED_SABS=" + String.join(",", sabList));
        System.out.println("DISCOVERED_LANGUAGES=" + String.join(",", latList));
        System.out.println("DISCOVERED_TUIS=" + String.join(",", tuiList));
    }

    private static String require(Properties p, String key) {
        String v = p.getProperty(key);
        if (v == null || v.trim().isEmpty()) throw new IllegalArgumentException("Missing property: " + key);
        return v.trim();
    }

    private static BufferedReader reader(File f) throws IOException {
        if (!f.isFile()) throw new FileNotFoundException("Missing: " + f);
        return new BufferedReader(new InputStreamReader(new FileInputStream(f), StandardCharsets.UTF_8), 1 << 20);
    }

    private static String[] split(String line) {
        return line.split("\\|", -1);
    }

    private static String val(String[] f, int idx1based) {
        int i = idx1based - 1;
        return (i >= 0 && i < f.length) ? f[i] : null;
    }

    private static String progressTuiSummary(Map<String, Set<String>> tuiCuis, long line) {
        // Build a compact summary like "File Line N Cuis: T047 123, T121 456, ..." (top 10 by count)
        List<Map.Entry<String, Integer>> list = new ArrayList<>();
        for (Map.Entry<String, Set<String>> e : tuiCuis.entrySet()) {
            list.add(new AbstractMap.SimpleEntry<>(e.getKey(), e.getValue().size()));
        }
        list.sort((a,b) -> Integer.compare(b.getValue(), a.getValue()));
        StringBuilder sb = new StringBuilder();
        sb.append(String.format(Locale.ROOT, "File Line %,d        Cuis:", line)).append(' ');
        int n = Math.min(10, list.size());
        for (int i=0;i<n;i++) {
            Map.Entry<String,Integer> e = list.get(i);
            sb.append(e.getKey()).append(' ').append(e.getValue());
            if (i < n-1) sb.append(", ");
        }
        return sb.toString();
    }
}

