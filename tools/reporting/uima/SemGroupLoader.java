package tools.reporting.uima;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Set;
import java.util.regex.Pattern;

/**
 * Loads semantic group/label overrides from SemGroups.txt when available.
 * The file can be referenced via the ctakes.semGroups.path system property, the
 * CTAKES_SEMGROUPS_PATH environment variable, or by dropping SemGroups.txt in
 * the working directory (or resources/).
 */
final class SemGroupLoader {

    private static final Pattern TUI_PATTERN = Pattern.compile("^T\\\d{3}$", Pattern.CASE_INSENSITIVE);

    private static Map<String, String> cachedGroups;
    private static Map<String, String> cachedLabels;
    private static boolean attempted;
    private static String loadedSource;

    private SemGroupLoader() {
    }

    static synchronized void applyOverrides(Map<String, String> groupMap, Map<String, String> labelMap) {
        if (groupMap == null || labelMap == null) {
            return;
        }
        if (!attempted) {
            attempted = true;
            Map<String, String> groups = new HashMap<>();
            Map<String, String> labels = new HashMap<>();
            loadedSource = loadMappings(groups, labels);
            if (loadedSource != null) {
                cachedGroups = groups;
                cachedLabels = labels;
                System.err.println("[SemGroupLoader] Loaded semantic groups from " + loadedSource + '.');
            } else {
                cachedGroups = null;
                cachedLabels = null;
                System.err.println("[SemGroupLoader] SemGroups.txt not found; using built-in semantic groups.");
            }
        }
        if (cachedGroups != null) {
            groupMap.putAll(cachedGroups);
        }
        if (cachedLabels != null) {
            labelMap.putAll(cachedLabels);
        }
    }

    private static String loadMappings(Map<String, String> groupMap, Map<String, String> labelMap) {
        List<Path> candidates = buildCandidates();
        for (Path candidate : candidates) {
            if (candidate == null) {
                continue;
            }
            try {
                if (Files.isRegularFile(candidate)) {
                    int loaded = apply(candidate, groupMap, labelMap);
                    if (loaded > 0) {
                        return candidate.toAbsolutePath().toString();
                    }
                }
            } catch (IOException e) {
                System.err.println("[SemGroupLoader] Failed to read " + candidate + ": " + e.getMessage());
            }
        }
        try (InputStream in = SemGroupLoader.class.getClassLoader().getResourceAsStream("SemGroups.txt")) {
            if (in != null) {
                try (BufferedReader reader = new BufferedReader(new InputStreamReader(in, StandardCharsets.UTF_8))) {
                    int loaded = apply(reader, groupMap, labelMap);
                    if (loaded > 0) {
                        return "classpath:SemGroups.txt";
                    }
                }
            }
        } catch (IOException e) {
            System.err.println("[SemGroupLoader] Failed to load SemGroups.txt from classpath: " + e.getMessage());
        }
        return null;
    }

    private static int apply(Path path, Map<String, String> groupMap, Map<String, String> labelMap) throws IOException {
        try (BufferedReader reader = Files.newBufferedReader(path, StandardCharsets.UTF_8)) {
            return apply(reader, groupMap, labelMap);
        }
    }

    private static int apply(BufferedReader reader, Map<String, String> groupMap, Map<String, String> labelMap) throws IOException {
        int count = 0;
        String line;
        while ((line = reader.readLine()) != null) {
            String trimmed = line.trim();
            if (trimmed.isEmpty() || trimmed.startsWith("#") || trimmed.startsWith("//")) {
                continue;
            }
            String[] rawParts = trimmed.split("\\|");
            List<String> parts = new ArrayList<>(rawParts.length);
            for (String raw : rawParts) {
                String p = raw.trim();
                if (!p.isEmpty()) {
                    parts.add(p);
                }
            }
            if (parts.size() < 2) {
                continue;
            }
            int tuiIndex = -1;
            for (int i = 0; i < parts.size(); i++) {
                if (TUI_PATTERN.matcher(parts.get(i)).matches()) {
                    tuiIndex = i;
                    break;
                }
            }
            if (tuiIndex < 0) {
                continue;
            }
            String tui = parts.get(tuiIndex).toUpperCase(Locale.ROOT);
            String label = parts.get(parts.size() - 1);
            String group = "";
            if (parts.size() >= 3) {
                for (int i = 0; i < parts.size(); i++) {
                    if (i == tuiIndex || i == parts.size() - 1) {
                        continue;
                    }
                    group = parts.get(i);
                    break;
                }
            } else if (parts.size() == 2 && tuiIndex == 0) {
                group = parts.get(1);
            }
            if (!group.isEmpty()) {
                groupMap.put(tui, group.toUpperCase(Locale.ROOT));
            }
            if (!label.isEmpty()) {
                labelMap.put(tui, label);
            }
            count++;
        }
        return count;
    }

    private static List<Path> buildCandidates() {
        Set<Path> candidates = new LinkedHashSet<>();
        String prop = System.getProperty("ctakes.semGroups.path");
        if (prop != null && !prop.trim().isEmpty()) {
            try {
                candidates.add(Paths.get(prop.trim()));
            } catch (Exception ignored) {
            }
        }
        String env = System.getenv("CTAKES_SEMGROUPS_PATH");
        if (env != null && !env.trim().isEmpty()) {
            try {
                candidates.add(Paths.get(env.trim()));
            } catch (Exception ignored) {
            }
        }
        String userDir = System.getProperty("user.dir");
        if (userDir != null && !userDir.trim().isEmpty()) {
            Path base = Paths.get(userDir.trim());
            candidates.add(base.resolve("SemGroups.txt"));
            candidates.add(base.resolve("resources").resolve("SemGroups.txt"));
        }
        candidates.add(Paths.get("SemGroups.txt"));
        candidates.add(Paths.get("resources", "SemGroups.txt"));
        return new ArrayList<>(candidates);
    }
}