package tools.reporting;

import java.io.*;
import java.nio.charset.StandardCharsets;
import java.nio.file.*;
import java.util.*;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * Build a single XLSX workbook consolidating cTAKES run outputs.
 * Note: This tool now emits XLSX only; the legacy Excel 2003 XML writer
 * has been removed to reduce confusion and maintenance overhead.
 * Sheets:
 * - RunInfo: parsed from run log (build version, times, counts), piper file, dict xml
 * - Mentions: aggregated BSV tables from bsv_table/ (adds Document column)
 * - CuiCounts: aggregated CUI counts
 * - CuiList: aggregated CUI list
 * - Tokens: aggregated token BSV
 * - Modules: parsed from piper file (load/add lines)
 * - SheetGuide: describes columns and their module origins
 *
 * Usage:
 *   java tools.reporting.ExcelXmlReport -o <output_dir> -w <workbook.xlsx> [-l <run.log>] [-p <pipeline.piper>] [-d <dict.xml>] [-M <mode>]
 *
 * Modes:
 *   - summary: fast, avoids XMI parse; uses counts and lightweight stats
 *   - full:    full aggregation, parses XMI for Clinical Concepts sheet
 *   - csv:     aggregates Clinical Concepts from per-document CSVs (e.g., csv_table_concepts), no XMI parse
 */
public class ExcelXmlReport {
    private static final String NS = "urn:schemas-microsoft-com:office:spreadsheet";
    // Header fill color for XLSX styles (ARGB). Defaults to light gray; can be overridden per pipeline.
    private static String HEADER_FILL_ARGB = "FFEFEFEF";
    private static final Map<String,String> PIPELINE_COLOR_ARGB;
    static {
        Map<String,String> m = new LinkedHashMap<>();
        // Legacy-aligned palette (ARGB: FF + RRGGBB)
        m.put("S_core", "FF00A3E0");
        m.put("S_core_rel", "FF4F81BD");
        m.put("S_core_temp", "FF9BBB59");
        m.put("S_core_temp_coref", "FF2C3E50");
        m.put("S_core_temp_coref_smoke", "FF8AB4F8");
        m.put("D_core_rel", "FFC0504D");
        m.put("D_core_temp", "FF8064A2");
        m.put("D_core_temp_coref", "FF4BACC6");
        m.put("D_core_temp_coref_smoke", "FF2E86C1");
        m.put("WSD_Compare", "FF8E44AD");
        m.put("TsSectionedTemporalCoref", "FFE07A00");
        PIPELINE_COLOR_ARGB = Collections.unmodifiableMap(m);
    }

    public static void main(String[] args) throws Exception {
        Map<String, String> cli = parseArgs(args);
        if (!cli.containsKey("-o") || !cli.containsKey("-w")) {
            System.err.println("Usage: java tools.reporting.ExcelXmlReport -o <output_dir> -w <workbook.xlsx> [-l <run.log>] [-p <pipeline.piper>] [-d <dict.xml>] [-M <mode>]");
            System.exit(2);
        }
        Path outDir = Paths.get(cli.get("-o")).toAbsolutePath().normalize();
        Path workbook = Paths.get(cli.get("-w")).toAbsolutePath().normalize();
        Path runLog = cli.containsKey("-l") ? Paths.get(cli.get("-l")) : findRunLog(outDir);
        Path piper = cli.containsKey("-p") ? Paths.get(cli.get("-p")) : findPiperFromLog(runLog);
        Path dictXml = cli.containsKey("-d") ? Paths.get(cli.get("-d")) : findDictXmlInDir(outDir);
        String mode = cli.containsKey("-M") ? cli.get("-M") : (cli.containsKey("--mode") ? cli.get("--mode") : "full");

        System.out.println("[report] Building workbook");
        System.out.println("[report]   outDir   = " + outDir);
        System.out.println("[report]   workbook = " + workbook);
        if (piper != null) System.out.println("[report]   piper    = " + piper);
        if (runLog != null) System.out.println("[report]   runLog   = " + runLog);
        if (dictXml != null) System.out.println("[report]   dictXml  = " + dictXml);
        System.out.println("[report]   mode     = " + mode);

        // Build sheets
        // Special case: If this looks like a compare parent dir with multiple subruns, build summary sheets and exit early.
        List<List<String>> pipelines = buildPipelinesSummaryIfAny(outDir);
        if (pipelines != null && pipelines.size() > 1) {
            LinkedHashMap<String, List<List<String>>> sheets = new LinkedHashMap<>();
            sheets.put("Pipelines Summary", pipelines);
            // Aggregate processing metrics across subruns
            List<List<String>> procAgg = buildProcessingMetricsAggregateForParent(outDir);
            if (procAgg != null && procAgg.size() > 1) sheets.put("Processing Metrics (Aggregate)", procAgg);
            // Add a clinician-friendly summary sheet
            List<List<String>> clinician = buildClinicianSummaryIfAny(outDir);
            if (clinician != null && clinician.size() > 1) sheets.put("Clinician Summary", clinician);
            Path parent = workbook.getParent();
            if (parent != null) Files.createDirectories(parent);
            // Force .xlsx
            if (!workbook.toString().toLowerCase(java.util.Locale.ROOT).endsWith(".xlsx")) {
                workbook = java.nio.file.Paths.get(workbook.toString() + ".xlsx");
            }
            writeWorkbookXlsx(sheets, workbook);
            System.out.println("Wrote workbook: " + workbook);
            return;
        }

        // Set header color per pipeline for single-run workbooks
        String pipelineKey = detectPipelineKeyFromOutDir(outDir);
        if (pipelineKey != null && PIPELINE_COLOR_ARGB.containsKey(pipelineKey)) {
            HEADER_FILL_ARGB = PIPELINE_COLOR_ARGB.get(pipelineKey);
        } else {
            HEADER_FILL_ARGB = "FFEFEFEF";
        }

        List<List<String>> runInfo = buildRunInfo(runLog, piper, dictXml, outDir);
        List<List<String>> modules = parsePiperModules(piper);
        List<List<String>> guide = buildSheetGuide();
        List<List<String>> pipelineMap = buildPipelineMap(modules, guide);
        List<List<String>> procMetrics = buildProcessingMetrics(runLog, outDir);
        LinkedHashMap<String, List<List<String>>> sheets = new LinkedHashMap<>();

        if ("summary".equalsIgnoreCase(mode)) {
            // Fast path: avoid parsing XMI or large BSVs
            List<List<String>> cuiCounts = aggregateCuiCounts(outDir);
            // If CuiCounts are missing/empty, fall back to CSV aggregation for a useful Overview
            List<List<String>> summary;
            if (cuiCounts == null || cuiCounts.size() <= 1) {
                List<List<String>> mentionsCsv = aggregateMentionsFromCsv(outDir);
                summary = buildSummary(outDir, mentionsCsv, null, runInfo, modules, runLog);
            } else {
                summary = buildSummaryFast(outDir, runInfo, modules, runLog, cuiCounts);
            }
            List<List<String>> cuiTotals = aggregateCuiTotals(cuiCounts);
            sheets.put("Overview", summary);
            sheets.put("Pipeline Map", pipelineMap);
            sheets.put("Processing Metrics", procMetrics);
            sheets.put("CuiCounts", cuiCounts);
            sheets.put("Tokens", aggregateTokens(outDir));
            sheets.put("CuiList", cuiTotals);
        } else if ("csv".equalsIgnoreCase(mode)) {
            // Aggregate Clinical Concepts from per-document CSVs (no XMI parse)
            List<List<String>> mentionsCsv = aggregateMentionsFromCsv(outDir);
            List<List<String>> summary = buildSummary(outDir, mentionsCsv, null, runInfo, modules, runLog);
            List<List<String>> cuiCounts = aggregateCuiCounts(outDir);
            List<List<String>> cuiTotals = aggregateCuiTotals(cuiCounts);
            sheets.put("Overview", summary);
            sheets.put("Pipeline Map", pipelineMap);
            sheets.put("Processing Metrics", procMetrics);
            sheets.put("Clinical Concepts", mentionsCsv);
            sheets.put("CuiCounts", cuiCounts);
            sheets.put("Tokens", aggregateTokens(outDir));
            sheets.put("CuiList", cuiTotals);
        } else {
            // Full report with clinical concepts sheet
            List<List<String>> mentionsFull = aggregateMentionsFull(outDir.resolve("xmi"), outDir.resolve("bsv_table"));
            List<List<String>> summary = buildSummary(outDir, mentionsFull, null, runInfo, modules, runLog);
            List<List<String>> cuiCounts = aggregateCuiCounts(outDir);
            List<List<String>> cuiTotals = aggregateCuiTotals(cuiCounts);
            sheets.put("Overview", summary);
            sheets.put("Pipeline Map", pipelineMap);
            sheets.put("Processing Metrics", procMetrics);
            sheets.put("Clinical Concepts", mentionsFull);
            sheets.put("CuiCounts", cuiCounts);
            sheets.put("Tokens", aggregateTokens(outDir));
            sheets.put("CuiList", cuiTotals);
        }

        Path parent2 = workbook.getParent();
        if (parent2 != null) Files.createDirectories(parent2);
        // Force .xlsx
        if (!workbook.toString().toLowerCase(java.util.Locale.ROOT).endsWith(".xlsx")) {
            workbook = java.nio.file.Paths.get(workbook.toString() + ".xlsx");
        }
        writeWorkbookXlsx(sheets, workbook);
        System.out.println("[report] Wrote workbook: " + workbook);
    }

    private static String detectPipelineKeyFromOutDir(Path outDir) {
        if (outDir == null) return null;
        String name = outDir.getFileName() != null ? outDir.getFileName().toString() : outDir.toString();
        int idx = name.indexOf("_mimic");
        String key = idx > 0 ? name.substring(0, idx) : name;
        for (String k : PIPELINE_COLOR_ARGB.keySet()) {
            if (key.equals(k)) return k;
        }
        return key;
    }

    // =============== Aggregate Clinical Concepts from per-document CSVs ===============
    private static List<List<String>> aggregateMentionsFromCsv(Path outDir) throws IOException {
        List<List<String>> rows = new ArrayList<>();
        // Header must match aggregateMentionsFull output exactly
        List<String> header = Arrays.asList(
                "Document","Begin","End","Text",
                "Section","SmokingStatus",
                "Semantic Group","Semantic Type","SemanticsFallback","CUI","TUI","PreferredText","PrefTextFallback","CodingScheme",
                "CandidateCount","Candidates","Confidence","ConceptScore","Disambiguated",
                "DocTimeRel","DegreeOf","LocationOfText","Coref","CorefChainId","CorefRepText",
                "Polarity","Negated","Uncertain","Conditional","Generic","Subject","HistoryOf"
        );
        rows.add(header);

        // Candidate directories: prefer only csv_table_concepts at top-level and in shards.
        // We intentionally avoid csv_table to prevent mixing column schemas that inflate doc counts.
        List<Path> csvDirs = new ArrayList<>();
        Path c1 = outDir.resolve("csv_table_concepts");
        if (Files.isDirectory(c1)) csvDirs.add(c1);
        try (DirectoryStream<Path> shards = Files.newDirectoryStream(outDir, p -> Files.isDirectory(p) && p.getFileName().toString().startsWith("shard_"))) {
            for (Path sh : shards) {
                Path sc1 = sh.resolve("csv_table_concepts");
                if (Files.isDirectory(sc1)) csvDirs.add(sc1);
            }
        } catch (IOException ignore) {}

        int fileCount = 0;
        for (Path dir : csvDirs) {
            try (DirectoryStream<Path> ds = Files.newDirectoryStream(dir, p -> {
                String n = p.getFileName().toString().toLowerCase(java.util.Locale.ROOT);
                return n.endsWith(".csv");
            })) {
                for (Path p : ds) {
                    fileCount++;
                    if (fileCount % 1000 == 0) System.out.println("[report]   aggregated " + fileCount + " per-doc CSVs from " + dir.getFileName());
                    List<String> lines = Files.readAllLines(p, StandardCharsets.UTF_8);
                    if (lines.isEmpty()) continue;
                    // Read header and require an exact match to the expected columns to avoid pulling in csv_table rows
                    String h = lines.get(0);
                    // Simple CSV parse for header to count columns
                    List<String> cols = parseCsvLine(h);
                    int expected = header.size();
                    boolean headerMatches = cols.size() == expected;
                    if (headerMatches) {
                        for (int i=0;i<expected;i++) {
                            if (!header.get(i).equalsIgnoreCase(cols.get(i))) { headerMatches = false; break; }
                        }
                    }
                    if (!headerMatches) {
                        // Skip files that are not our per-doc Clinical Concepts CSVs
                        continue;
                    }
                    for (int i=1;i<lines.size();i++) {
                        String line = lines.get(i);
                        if (line == null) continue;
                        line = line.trim();
                        if (line.isEmpty()) continue;
                        List<String> cells = parseCsvLine(line);
                        // Normalize to expected columns (pad/truncate)
                        if (cells.size() < expected) {
                            List<String> pads = new ArrayList<>(expected - cells.size());
                            for (int k=0;k<expected - cells.size();k++) pads.add("");
                            List<String> fixed = new ArrayList<>(cells);
                            fixed.addAll(pads);
                            rows.add(fixed);
                        } else if (cells.size() > expected) {
                            rows.add(new ArrayList<>(cells.subList(0, expected)));
                        } else {
                            rows.add(cells);
                        }
                    }
                }
            }
        }
        if (fileCount > 0) System.out.println("[report]   aggregated total per-doc CSVs: " + fileCount);
        if (rows.size() == 1) rows.add(Arrays.asList("No data found"));
        return rows;
    }

    // Minimal CSV line parser supporting quotes and escaped quotes ("")
    private static List<String> parseCsvLine(String line) {
        List<String> out = new ArrayList<>();
        if (line == null) return out;
        StringBuilder sb = new StringBuilder();
        boolean inQuotes = false;
        for (int i=0;i<line.length();i++) {
            char c = line.charAt(i);
            if (inQuotes) {
                if (c == '"') {
                    if (i+1 < line.length() && line.charAt(i+1) == '"') { // escaped quote
                        sb.append('"'); i++;
                    } else {
                        inQuotes = false;
                    }
                } else {
                    sb.append(c);
                }
            } else {
                if (c == '"') {
                    inQuotes = true;
                } else if (c == ',') {
                    out.add(sb.toString()); sb.setLength(0);
                } else {
                    sb.append(c);
                }
            }
        }
        out.add(sb.toString());
        return out;
    }
    private static SubdirMetrics aggregateMetricsFromPipelinesSummary(org.w3c.dom.Element ws, String NS) {
        org.w3c.dom.NodeList tables = ws.getElementsByTagNameNS(NS, "Table");
        if (tables.getLength() == 0) return null;
        org.w3c.dom.Element table = (org.w3c.dom.Element) tables.item(0);
        org.w3c.dom.NodeList rows = table.getElementsByTagNameNS(NS, "Row");
        if (rows.getLength() < 2) return null;
        java.util.List<String> header = new java.util.ArrayList<>();
        java.util.Map<String,Integer> colIdx = new java.util.HashMap<>();
        org.w3c.dom.Element hr = (org.w3c.dom.Element) rows.item(0);
        java.util.List<String> hcells = extractRowCells(hr, NS);
        for (int i=0;i<hcells.size();i++) { header.add(hcells.get(i)); colIdx.put(hcells.get(i), i); }
        int iDocs = colIdx.getOrDefault("Documents", -1);
        int iConcepts = colIdx.getOrDefault("Clinical Concepts", -1);
        SubdirMetrics m = new SubdirMetrics();
        long docs = 0; long mentions = 0;
        for (int r=1;r<rows.getLength();r++) {
            java.util.List<String> cells = extractRowCells((org.w3c.dom.Element) rows.item(r), NS);
            if (cells.isEmpty()) continue;
            if (iDocs>=0 && iDocs<cells.size()) try { docs += Long.parseLong(nvl(cells.get(iDocs)).replace(",","")); } catch (Exception ignore) {}
            if (iConcepts>=0 && iConcepts<cells.size()) try { mentions += Long.parseLong(nvl(cells.get(iConcepts)).replace(",","")); } catch (Exception ignore) {}
        }
        m.docCount = (int)docs;
        m.mentionCount = (int)mentions;
        return m;
    }

    private static Map<String,String> parseArgs(String[] args) {
        Map<String,String> m = new HashMap<>();
        for (int i=0;i<args.length-1;i+=2) {
            m.put(args[i], args[i+1]);
        }
        return m;
    }

    private static Path findRunLog(Path outDir) {
        if (outDir == null) return null;
        Path logs = outDir.resolve("logs");
        if (Files.isDirectory(logs)) {
            try {
                Optional<Path> latest = Files.list(logs)
                        .filter(p -> p.getFileName().toString().endsWith(".log"))
                        .max(Comparator.comparingLong(p -> p.toFile().lastModified()));
                if (latest.isPresent()) return latest.get();
            } catch (IOException ignored) {}
        }
        // compare runner writes run.log in run dir
        Path runLog = outDir.resolve("run.log");
        if (Files.isRegularFile(runLog)) return runLog;
        return null;
    }

    // =============== Processing Metrics from log and outputs ===============
    private static List<List<String>> buildProcessingMetrics(Path runLog, Path outDir) throws IOException {
        List<List<String>> rows = new ArrayList<>();
        // Legend
        rows.add(Arrays.asList("Legend","","","",""));
        rows.add(Arrays.asList("Column","Meaning","","",""));
        rows.add(Arrays.asList("Init Count","Thread-level initializations (multi-thread pipelines init per thread)","","",""));
        rows.add(Arrays.asList("Process Count","Per-note processing events (often ≈ documents processed)","","",""));
        rows.add(Arrays.asList("Files Written","Outputs the writer produced (XMI, tables, tokens, counts)","","",""));
        rows.add(Arrays.asList("","","","",""));
        // AE activity table
        rows.add(Arrays.asList("Phase","AE/Writer","Init Count","Process Count","Files Written"));

        Map<String,int[]> counts = new LinkedHashMap<>(); // key -> [init, process, files]
        if (runLog != null && Files.isRegularFile(runLog)) {
            List<String> lines = Files.readAllLines(runLog, StandardCharsets.UTF_8);
            for (String line : lines) {
                String s = line;
                int idx = s.indexOf(" - ");
                if (idx > 0) {
                    // Try to grab logger/AE before hyphen
                    String left = s.substring(0, idx);
                    String name = left.replaceFirst("^.* INFO ", "").trim();
                    if (name.isEmpty()) continue;
                    if (isNoiseLogger(name)) continue;
                    int[] arr = counts.computeIfAbsent(name, k -> new int[3]);
                    String rest = s.substring(idx+3).toLowerCase(Locale.ROOT);
                    if (rest.contains("initializing")) arr[0]++;
                    if (rest.contains("process(jcas)") || rest.startsWith("processing") || rest.contains("starting processing") || rest.contains("finished processing")) arr[1]++;
                    if (rest.startsWith("writing ")) arr[2]++;
                }
            }
        }
        // Supplement with file counts from outputs (XMI/tables)
        Map<String,Integer> fileTotals = new LinkedHashMap<>();
        fileTotals.put("FileTreeXmiWriter", countFiles(outDir.resolve("xmi"), ".xmi"));
        fileTotals.put("SemanticTableFileWriter", countFiles(outDir.resolve("bsv_table"), ".BSV") + countFiles(outDir.resolve("csv_table"), ".CSV") + countFiles(outDir.resolve("html_table"), ".HTML"));
        fileTotals.put("CuiListFileWriter", countFiles(outDir.resolve("cui_list"), ".bsv"));
        fileTotals.put("CuiCountFileWriter", countFiles(outDir.resolve("cui_count"), ".bsv") + countFiles(outDir, ".cuicount.bsv"));
        fileTotals.put("TokenTableFileWriter", countFiles(outDir.resolve("bsv_tokens"), ".BSV"));

        // Render rows: use friendly labels where possible
        for (Map.Entry<String,int[]> e : counts.entrySet()) {
            String key = e.getKey();
            int[] c = e.getValue();
            int files = c[2] + fileTotals.getOrDefault(key, 0);
            String label = friendlyAeLabel(key);
            if (label.isEmpty()) label = key;
            String phase = phaseForAeLabel(key, label);
            rows.add(Arrays.asList(phase, label + " (" + key + ")", String.valueOf(c[0]), String.valueOf(c[1]), String.valueOf(files)));
        }
        // Add any writers not seen in log
        for (Map.Entry<String,Integer> e : fileTotals.entrySet()) {
            String key = e.getKey();
            boolean present = false;
            for (List<String> r : rows) if (r.size()>1 && r.get(1).contains("("+key+")")) { present = true; break; }
            if (!present && e.getValue() > 0) {
                String label = friendlyAeLabel(key);
                if (label.isEmpty()) label = key;
                String phase = phaseForAeLabel(key, label);
                rows.add(Arrays.asList(phase, label + " (" + key + ")", "0", "0", String.valueOf(e.getValue())));
            }
        }

        // Doc timings section
        List<DocTiming> dts = parseDocTimings(runLog);
        rows.add(Arrays.asList("","","","",""));
        rows.add(Arrays.asList("Doc Timings","","","",""));
        rows.add(Arrays.asList("Document","Start","End","Duration (s)",""));
        java.text.SimpleDateFormat outFmt = new java.text.SimpleDateFormat("yyyy-MM-dd HH:mm:ss");
        for (DocTiming dt : dts) {
            double sec = Math.max(0, dt.endMs - dt.startMs) / 1000.0;
            String secStr = sec < 1.0 ? "< 1.00" : String.format(Locale.ROOT, "%.2f", sec);
            rows.add(Arrays.asList(dt.doc, outFmt.format(new java.util.Date(dt.startMs)), outFmt.format(new java.util.Date(dt.endMs)), secStr, ""));
        }

        if (rows.size() == 1) rows.add(Arrays.asList("No data found"));
        return rows;
    }

    private static boolean isNoiseLogger(String name) {
        String n = name == null ? "" : name.toLowerCase(Locale.ROOT);
        return n.contains("xmicasserializer") || n.contains("serviceproxypool") || n.contains("piperfilereader") || n.contains("dictionarydescriptorparser") || n.contains("jdbcconnectionfactory") || n.contains("jdbcrareworddictionary") || n.contains("jdbcconceptfactory") || n.contains("filetreereader") || n.contains("abstracttablefilewriter") || n.contains("cleartkanalysisengine");
    }

    private static int countFiles(Path dir, String suffix) {
        int n = 0;
        if (dir == null) return 0;
        try {
            if (Files.isDirectory(dir)) {
                try (DirectoryStream<Path> ds = Files.newDirectoryStream(dir)) {
                    for (Path p : ds) {
                        String name = p.getFileName().toString();
                        if (name.toLowerCase(Locale.ROOT).endsWith(suffix.toLowerCase(Locale.ROOT))) n++;
                    }
                }
            } else if (Files.isRegularFile(dir)) {
                // If a file path passed accidentally
                String name = dir.getFileName().toString();
                if (name.toLowerCase(Locale.ROOT).endsWith(suffix.toLowerCase(Locale.ROOT))) n = 1;
            } else {
                // Also scan run dir for cuicount pattern when suffix is ".cuicount.bsv"
                if (suffix.equalsIgnoreCase(".cuicount.bsv") && dir != null && Files.isDirectory(dir)) {
                    try (DirectoryStream<Path> ds = Files.newDirectoryStream(dir)) {
                        for (Path p : ds) if (p.getFileName().toString().toLowerCase(Locale.ROOT).endsWith(".cuicount.bsv")) n++;
                    }
                }
            }
        } catch (IOException ignored) {}
        return n;
    }

    // =============== Combined Pipelines Summary (for compare runs) ===============
    private static List<List<String>> buildPipelinesSummaryIfAny(Path outDir) {
        List<List<String>> rows = new ArrayList<>();
        rows.add(Arrays.asList(
                "Pipeline Dir","Documents","Average Seconds per Document","Init Time (s)","Process Time (s)","Total Time (s)","Timed Docs","Timing Coverage (%)",
                "Clinical Concepts","Avg Confidence","% Concepts with DocTimeRel",
                "Relations per Doc","Coref Markables per Doc",
                "Distinct CUIs","Disambig Rate","Avg Candidate Count","Sec per 100 Concepts",
                "Score (general)","Recommended",
                "Recommended (Speed)","Recommended (Temporal)","Recommended (Relations)",
                "Reason","Run Log"
        ));
        if (outDir == null || !Files.isDirectory(outDir)) return rows;
        class R { Path dir; String name; SubdirMetrics m; double avgSec; String runLog; }
        List<R> all = new ArrayList<>();
        boolean hasShardsDir = false;
        try (DirectoryStream<Path> ds = Files.newDirectoryStream(outDir)) {
            for (Path sub : ds) {
                if (!Files.isDirectory(sub)) continue;
                String subName = sub.getFileName().toString();
                // Ignore per-run shard directories; they are not separate pipelines
                if (subName.equalsIgnoreCase("shards") || subName.startsWith("shard_")) { hasShardsDir = true; continue; }
                // Ignore standard output directories of a single run
                String low = subName.toLowerCase(java.util.Locale.ROOT);
                if (low.equals("xmi") || low.equals("bsv_table") || low.equals("csv_table") || low.equals("csv_table_concepts") || low.equals("html_table") ||
                    low.equals("cui_list") || low.equals("cui_count") || low.equals("bsv_tokens") || low.equals("logs") ||
                    low.startsWith("pending_")) {
                    continue;
                }
                Path runLog = sub.resolve("run.log");
                if (!Files.isRegularFile(runLog)) {
                    Path logs = sub.resolve("logs");
                    if (Files.isDirectory(logs)) {
                        Path latest = null; long lm = Long.MIN_VALUE;
                        try (DirectoryStream<Path> lds = Files.newDirectoryStream(logs, p -> p.getFileName().toString().endsWith(".log"))) {
                            for (Path p : lds) { long m = p.toFile().lastModified(); if (m > lm) { lm = m; latest = p; } }
                        }
                        if (latest != null) runLog = latest; else runLog = null;
                    } else {
                        // No run log at this level (e.g., sharded runs); still include this subdir
                        runLog = null;
                    }
                }
                SubdirMetrics m = computeMetricsFromReportIfAny(sub);
                if (m == null) m = computeSubdirMetrics(sub);
                String avg = (runLog!=null) ? parseValueFromLog(runLog, "Average Seconds per Document") : null;
                if (avg == null || avg.isEmpty() || "".equals(avg) || "0.00".equals(avg)) {
                    java.util.List<DocTiming> dts = parseDocTimings(runLog);
                    if (!dts.isEmpty()) { long sum=0; for (DocTiming dt : dts) sum += Math.max(0, dt.endMs - dt.startMs);
                        avg = String.format(java.util.Locale.ROOT, "%.2f", (sum/1000.0)/dts.size()); }
                }
                if (avg == null || avg.isEmpty() || "0.00".equals(avg)) {
                    // Final fallback for sharded logs: derive from earliest Processing Start Time and latest Processing End Time across shards
                    Double wnd = computeAvgFromProcessingWindow(runLog, m.docCount);
                    if (wnd != null && wnd > 0) avg = String.format(java.util.Locale.ROOT, "%.2f", wnd);
                }
                if (avg == null || avg.isEmpty() || "0.00".equals(avg)) {
                    // Fallback: derive from a per-run workbook (e.g., summary of shards)
                    Double derived = computeAvgSecFromSummary(sub);
                    if (derived != null && derived > 0) avg = String.format(java.util.Locale.ROOT, "%.2f", derived);
                }
                R r = new R(); r.dir=sub; r.name=subName; r.m=m; r.avgSec=parseDouble(avg); r.runLog=(runLog==null?"":runLog.toString());
                all.add(r);
            }
        } catch (IOException ignore) {}
        // If no qualifying subruns found, do not treat this as a compare parent
        if (all.isEmpty()) return rows;

        // Prepare derived metrics for scoring
        List<Double> conceptsPerDoc = new ArrayList<>();
        List<Double> distinctPerDoc = new ArrayList<>();
        List<Double> avgConf = new ArrayList<>();
        List<Double> disambRate = new ArrayList<>();
        List<Double> docTimeRelPct = new ArrayList<>();
        List<Double> relPerDoc = new ArrayList<>();
        List<Double> corefPerDoc = new ArrayList<>();
        List<Double> secPerDoc = new ArrayList<>();
        List<Double> avgCand = new ArrayList<>();
        List<Double> secPer100 = new ArrayList<>();

        for (R r : all) {
            SubdirMetrics m = r.m; double docs = Math.max(1, m.docCount);
            double mentions = Math.max(1, m.mentionCount);
            double cpd = m.mentionCount / docs;
            // Use global distinct CUI count (not averaged per doc) for ranking and display
            double dpd = m.distinctCuiCount;
            double conf = m.avgConfidence;
            double disr = m.mentionCount>0 ? (m.disambTrueCount/(double)m.mentionCount) : 0.0;
            double dtrp = m.mentionCount>0 ? (100.0*m.docTimeRelCount/m.mentionCount) : 0.0;
            double rpd = m.docCount>0 ? (m.relationCount/(double)m.docCount) : 0.0;
            double cored = m.docCount>0 ? (m.markableCount/(double)m.docCount) : 0.0;
            double spd = r.avgSec;
            double ac = m.mentionCount>0 ? (m.candidateCountSum/(double)m.mentionCount) : 0.0;
            double s100 = cpd>0 ? (spd / cpd * 100.0) : 0.0;
            conceptsPerDoc.add(cpd); distinctPerDoc.add(dpd); avgConf.add(conf); disambRate.add(disr);
            docTimeRelPct.add(dtrp); relPerDoc.add(rpd); corefPerDoc.add(cored); secPerDoc.add(spd); avgCand.add(ac); secPer100.add(s100);
        }

        // Helper to normalize
        java.util.function.BiFunction<List<Double>,Boolean,List<Double>> norm = (vals, invert) -> {
            double min = Double.POSITIVE_INFINITY, max = Double.NEGATIVE_INFINITY;
            for (double v : vals) { if (v<min) min=v; if (v>max) max=v; }
            double range = (max-min);
            List<Double> out = new ArrayList<>(vals.size());
            for (double v : vals) {
                double nv = (range==0)? 1.0 : ((v-min)/range);
                if (invert) nv = 1.0 - nv;
                out.add(Math.max(0.0, Math.min(1.0, nv)));
            }
            return out;
        };

        List<Double> n_cpd = norm.apply(conceptsPerDoc, false);
        List<Double> n_dpd = norm.apply(distinctPerDoc, false);
        List<Double> n_conf = norm.apply(avgConf, false);
        List<Double> n_disr = norm.apply(disambRate, false);
        List<Double> n_dtrp = norm.apply(docTimeRelPct, false);
        List<Double> n_rpd = norm.apply(relPerDoc, false);
        List<Double> n_coref = norm.apply(corefPerDoc, false);
        List<Double> n_spd = norm.apply(secPerDoc, true);   // lower is better
        List<Double> n_ac = norm.apply(avgCand, true);      // lower is better

        // Weights (general preset); renormalize to sum=1
        double[] w = new double[]{0.15,0.10,0.15,0.10,0.15,0.10,0.05,0.15,0.05};
        double wsum=0; for (double x : w) wsum+=x; for (int i=0;i<w.length;i++) w[i]/=wsum;

        // Rank helper for reasons
        java.util.function.Function<List<Double>,int[]> ranksDesc = vals -> {
            int n=vals.size(); int[] rank=new int[n];
            java.util.List<Integer> idx=new java.util.ArrayList<>(); for(int i=0;i<n;i++) idx.add(i);
            idx.sort((a,b)->Double.compare(vals.get(b), vals.get(a)));
            for (int rnk=0;rnk<n;rnk++) rank[idx.get(rnk)] = rnk+1;
            return rank;
        };
        int[] rk_dtrp = ranksDesc.apply(docTimeRelPct);
        int[] rk_spd = ranksDesc.apply(n_spd); // already inverted; higher is better
        int[] rk_cpd = ranksDesc.apply(conceptsPerDoc);
        int[] rk_conf = ranksDesc.apply(avgConf);

        // Compute global bests for recommendations
        double[] scores = new double[all.size()];
        int bestGeneralIdx = 0; double bestGeneralScore = -1;
        int bestSpeedIdx = 0; double bestSpeedVal = Double.POSITIVE_INFINITY; // lower is better
        int bestTemporalIdx = 0; double bestTemporalVal = Double.NEGATIVE_INFINITY;
        int bestRelIdx = 0; double bestRelVal = Double.NEGATIVE_INFINITY;
        for (int i=0;i<all.size();i++) {
            double sc = w[0]*n_cpd.get(i) + w[1]*n_dpd.get(i) + w[2]*n_conf.get(i) + w[3]*n_disr.get(i) + w[4]*n_dtrp.get(i) + w[5]*n_rpd.get(i) + w[6]*n_coref.get(i) + w[7]*n_spd.get(i) + w[8]*n_ac.get(i);
            scores[i] = sc; if (sc > bestGeneralScore) { bestGeneralScore = sc; bestGeneralIdx = i; }
            double spd = secPerDoc.get(i); if (spd < bestSpeedVal) { bestSpeedVal = spd; bestSpeedIdx = i; }
            double tmp = docTimeRelPct.get(i); if (tmp > bestTemporalVal) { bestTemporalVal = tmp; bestTemporalIdx = i; }
            double relv = relPerDoc.get(i); if (relv > bestRelVal) { bestRelVal = relv; bestRelIdx = i; }
        }

        // Compose rows with score and lightweight recommendation
        for (int i=0;i<all.size();i++) {
            R r = all.get(i); SubdirMetrics m = r.m; double docs = Math.max(1, m.docCount);
            double score = scores[i];
            String rec = ""; String reason = "";
            if (i==bestGeneralIdx) {
                rec = "Yes";
                List<String> rs = new ArrayList<>();
                if (rk_dtrp[i]==1) rs.add("best DocTimeRel%");
                if (rk_spd[i]==1) rs.add("fastest");
                if (rk_cpd[i]==1) rs.add("most concepts/doc");
                if (rk_conf[i]==1) rs.add("highest confidence");
                reason = String.join(", ", rs);
            }
            String recSpeed = (i==bestSpeedIdx) ? "Yes" : "";
            String recTemporal = (i==bestTemporalIdx) ? "Yes" : "";
            String recRelations = (i==bestRelIdx) ? "Yes" : "";
            String docsStr = String.valueOf(m.docCount);
            String avgSecStr = String.format(java.util.Locale.ROOT, "%.2f", r.avgSec);
            String initSecStr = ""; String procSecStr = ""; String totalSecStr = "";
            int timedDocs = 0; String coverageStr = "";
            try {
                java.nio.file.Path rlog = (r.runLog==null || r.runLog.isEmpty()) ? null : java.nio.file.Paths.get(r.runLog);
                if (rlog != null && java.nio.file.Files.isRegularFile(rlog)) {
                    initSecStr = secondsFromElapsed(parseValueFromLog(rlog, "Initialization Time Elapsed"));
                    procSecStr = secondsFromElapsed(parseValueFromLog(rlog, "Processing Time Elapsed"));
                    totalSecStr = secondsFromElapsed(parseValueFromLog(rlog, "Total Run Time Elapsed"));
                    java.util.List<DocTiming> dtsR = parseDocTimings(rlog);
                    timedDocs = dtsR.size();
                    // If document count is 0 but we have timed docs, use timed docs count
                    if (m.docCount == 0 && timedDocs > 0) {
                        m.docCount = timedDocs;
                    }
                    if (m.docCount > 0) coverageStr = String.format(java.util.Locale.ROOT, "%.1f%%", (100.0 * timedDocs) / m.docCount);
                    // Fallbacks from processing window if any are empty
                    ProcWindow win = computeProcessingWindow(rlog);
                    if (win != null) {
                        if ((procSecStr==null || procSecStr.isEmpty()) && win.earliestStart!=null && win.latestEnd!=null && win.latestEnd > win.earliestStart) {
                            procSecStr = String.valueOf((win.latestEnd - win.earliestStart)/1000L);
                        }
                        if ((initSecStr==null || initSecStr.isEmpty()) && win.runStart!=null && win.earliestStart!=null && win.earliestStart > win.runStart) {
                            initSecStr = String.valueOf((win.earliestStart - win.runStart)/1000L);
                        }
                        if ((totalSecStr==null || totalSecStr.isEmpty()) && win.runStart!=null && win.latestEnd!=null && win.latestEnd > win.runStart) {
                            totalSecStr = String.valueOf((win.latestEnd - win.runStart)/1000L);
                        }
                    }
                }
            } catch (Exception ignore) {}
            String confStr = m.mentionCount>0?String.format(java.util.Locale.ROOT, "%.3f", m.avgConfidence):"";
            String dtrStr = m.mentionCount>0?String.format(java.util.Locale.ROOT, "%.1f%%", 100.0*m.docTimeRelCount/m.mentionCount):"";
            String rpdStr = m.docCount>0?String.format(java.util.Locale.ROOT, "%.2f", (double)m.relationCount/m.docCount):"";
            String corefStr = m.docCount>0?String.format(java.util.Locale.ROOT, "%.2f", (double)m.markableCount/m.docCount):"";
            String dpdStr = String.valueOf(m.distinctCuiCount);
            String disrStr = m.mentionCount>0?String.format(java.util.Locale.ROOT, "%.2f", (double)m.disambTrueCount/m.mentionCount):"";
            String acStr = m.mentionCount>0?String.format(java.util.Locale.ROOT, "%.2f", (double)m.candidateCountSum/m.mentionCount):"";
            double cpd = m.docCount>0 ? (double)m.mentionCount/m.docCount : 0.0;
            String s100Str = cpd>0?String.format(java.util.Locale.ROOT, "%.2f", r.avgSec/cpd*100.0):"";
            String scoreStr = String.format(java.util.Locale.ROOT, "%.3f", score);
            rows.add(Arrays.asList(
                    r.name,
                    docsStr, avgSecStr, initSecStr, procSecStr, totalSecStr, String.valueOf(timedDocs), coverageStr,
                    String.valueOf(m.mentionCount),
                    confStr, dtrStr,
                    rpdStr, corefStr,
                    dpdStr, disrStr, acStr, s100Str,
                    scoreStr, rec,
                    recSpeed, recTemporal, recRelations,
                    reason, r.runLog
            ));
            // Accumulate for parent-level CSV
            try { 
                addPipelineTimingRow(outDir, r.name, m.docCount, timedDocs, avgSecStr, initSecStr, procSecStr, totalSecStr, r.runLog);
                // Also consolidate individual timing CSVs into parent timing_csv directory
                Path runDir = r.runLog != null && !r.runLog.isEmpty() ? 
                    java.nio.file.Paths.get(r.runLog).getParent() : null;
                if (runDir != null) consolidateTimingCsv(outDir, runDir);
            } catch (Exception ignore) {}
        }
        return rows;
    }

    // Parse elapsed strings like "1 minutes, 38 seconds" or "4 minutes, 16 seconds" into total seconds
    private static String secondsFromElapsed(String s) {
        if (s == null) return "";
        long total = 0L;
        try {
            java.util.regex.Matcher m = java.util.regex.Pattern
                    .compile("(\\d+)\\s*(hour|hours|minute|minutes|second|seconds)", java.util.regex.Pattern.CASE_INSENSITIVE)
                    .matcher(s);
            while (m.find()) {
                long v = Long.parseLong(m.group(1));
                String unit = m.group(2).toLowerCase(java.util.Locale.ROOT);
                if (unit.startsWith("hour")) total += v * 3600L;
                else if (unit.startsWith("minute")) total += v * 60L;
                else total += v;
            }
        } catch (Exception ignore) {}
        return total > 0 ? String.valueOf(total) : "";
    }

    // =============== Clinician Summary (parent compare) ===============
    private static List<List<String>> buildClinicianSummaryIfAny(Path outDir) {
        List<List<String>> rows = new ArrayList<>();
        rows.add(Arrays.asList("Pipeline","What It Adds","Best For","Speed","Pros","Tradeoffs","Why choose this"));
        if (outDir == null || !Files.isDirectory(outDir)) return rows;

        class R { String name; SubdirMetrics m; double avgSec; }
        List<R> all = new ArrayList<>();
        try (DirectoryStream<Path> ds = Files.newDirectoryStream(outDir)) {
            for (Path sub : ds) {
                if (!Files.isDirectory(sub)) continue;
                Path runLog = sub.resolve("run.log");
                if (!Files.isRegularFile(runLog)) {
                    Path logs = sub.resolve("logs");
                    if (Files.isDirectory(logs)) {
                        Path latest = null; long lm = Long.MIN_VALUE;
                        try (DirectoryStream<Path> lds = Files.newDirectoryStream(logs, p -> p.getFileName().toString().endsWith(".log"))) {
                            for (Path p : lds) { long m = p.toFile().lastModified(); if (m > lm) { lm = m; latest = p; } }
                        }
                        if (latest != null) runLog = latest; else continue;
                    } else continue;
                }
                SubdirMetrics m = computeMetricsFromReportIfAny(sub);
                if (m == null) m = computeSubdirMetrics(sub);
                String avg = parseValueFromLog(runLog, "Average Seconds per Document");
                double avgSec = parseDouble(avg);
                if (avgSec <= 0) {
                    List<DocTiming> dts = parseDocTimings(runLog);
                    if (!dts.isEmpty()) {
                        long sum=0; for (DocTiming dt : dts) sum += Math.max(0, dt.endMs - dt.startMs);
                        avgSec = (sum/1000.0)/dts.size();
                    }
                }
                R r = new R(); r.name=sub.getFileName().toString(); r.m=m; r.avgSec = avgSec;
                all.add(r);
            }
        } catch (IOException ignore) {}
        if (all.isEmpty()) return rows;

        // Derive simple clinician-facing flags
        for (R r : all) {
            String name = r.name;
            SubdirMetrics m = r.m; double docs = Math.max(1, m.docCount);
            boolean addsTemporal = (m.mentionCount>0 && m.docTimeRelCount>0) || name.toLowerCase(java.util.Locale.ROOT).contains("temp");
            boolean addsRelations = (m.relationCount > 0) || name.toLowerCase(java.util.Locale.ROOT).contains("rel");
            boolean addsCoref = (m.markableCount > 0) || name.toLowerCase(java.util.Locale.ROOT).contains("coref");
            boolean addsSmoking = m.hasSmoking || name.toLowerCase(java.util.Locale.ROOT).contains("smoke");

            String whatAdds;
            List<String> adds = new ArrayList<>();
            if (addsTemporal) adds.add("Time tags");
            if (addsRelations) adds.add("Relationships");
            if (addsCoref) adds.add("Coreference");
            if (addsSmoking) adds.add("Smoking status");
            whatAdds = adds.isEmpty()?"Core concepts only":String.join(", ", adds);

            String bestFor;
            if (addsTemporal && !addsRelations && !addsCoref && !addsSmoking) bestFor = "Timing (before/after/overlap)";
            else if (addsRelations && !addsTemporal && !addsCoref && !addsSmoking) bestFor = "Finding relationships (e.g., located in)";
            else if (addsCoref && !addsTemporal && !addsRelations && !addsSmoking) bestFor = "Linking repeated mentions";
            else if (addsSmoking && !addsTemporal && !addsRelations && !addsCoref) bestFor = "Patient smoking status";
            else if (addsTemporal && addsCoref && !addsRelations) bestFor = "Timing plus linking mentions";
            else if (addsRelations && addsTemporal) bestFor = "Timing and relationships";
            else bestFor = "Fast general tagging";

            String speed;
            if (r.avgSec <= 2.0) speed = "Fast";
            else if (r.avgSec <= 8.0) speed = "Moderate";
            else speed = "Slow";

            List<String> pros = new ArrayList<>();
            if ("Fast".equals(speed)) pros.add("Fastest");
            if (addsTemporal) pros.add("Adds time info");
            if (addsRelations) pros.add("Adds relationships");
            if (addsCoref) pros.add("Links same-thing mentions");
            String prosStr = String.join(", ", pros);

            List<String> cons = new ArrayList<>();
            if ("Slow".equals(speed)) cons.add("Slower");
            if (!addsTemporal) cons.add("No time info");
            if (!addsRelations) cons.add("No relationships");
            if (!addsCoref) cons.add("No linking of mentions");
            String consStr = String.join(", ", cons);

            String why = "Choose when you need: ";
            if (addsTemporal || addsRelations || addsCoref) {
                why += whatAdds.toLowerCase(java.util.Locale.ROOT);
                if ("Slow".equals(speed)) why += "; accept slower speed";
            } else {
                why += "speed and basic concepts";
            }

            rows.add(Arrays.asList(name, whatAdds, bestFor, speed, prosStr, consStr, why));
        }
        return rows;
    }
    // Try to compute summary metrics by reading per-pipeline report (XML 2003 or XLSX)
    private static SubdirMetrics computeMetricsFromReportIfAny(Path subDir) {
        Path report = subDir.resolve("report.xml");
        if (!Files.isRegularFile(report)) {
            // Try to locate a per-pipeline workbook generated by build_xlsx_report.sh
            // Prefer XML 2003; else fall back to XLSX. Skip dictionary xml and parent compare.
            Path newestXml = null; long lmXml = Long.MIN_VALUE;
            try (DirectoryStream<Path> ds = Files.newDirectoryStream(subDir, p -> {
                String n = p.getFileName().toString().toLowerCase(java.util.Locale.ROOT);
                if (!n.endsWith(".xml")) return false;
                if (n.contains("fullclinical") && n.contains("_local")) return false; // dictionary xml
                if (n.startsWith("ctakes-report-compare")) return false; // parent summary
                return true;
            })) {
                for (Path p : ds) { long m = p.toFile().lastModified(); if (m > lmXml) { lmXml = m; newestXml = p; } }
            } catch (IOException ignore) {}
            if (newestXml != null) {
                report = newestXml;
            } else {
                // Try XLSX
                Path newestXlsx = null; long lmXlsx = Long.MIN_VALUE;
                try (DirectoryStream<Path> ds = Files.newDirectoryStream(subDir, p -> {
                    String n = p.getFileName().toString().toLowerCase(java.util.Locale.ROOT);
                    if (!n.endsWith(".xlsx")) return false;
                    if (n.startsWith("ctakes-report-compare")) return false; // parent summary
                    return true;
                })) {
                    for (Path p : ds) { long m = p.toFile().lastModified(); if (m > lmXlsx) { lmXlsx = m; newestXlsx = p; } }
                } catch (IOException ignore) {}
                if (newestXlsx != null) {
                    SubdirMetrics mx = computeMetricsFromXlsx(newestXlsx);
                    if (mx != null) return mx; else return null;
                }
                return null;
            }
        }
        try {
            javax.xml.parsers.DocumentBuilderFactory dbf = javax.xml.parsers.DocumentBuilderFactory.newInstance();
            dbf.setNamespaceAware(true);
            javax.xml.parsers.DocumentBuilder db = dbf.newDocumentBuilder();
            org.w3c.dom.Document doc = db.parse(report.toFile());
            org.w3c.dom.Element root = doc.getDocumentElement();
            String NS = "urn:schemas-microsoft-com:office:spreadsheet";
            org.w3c.dom.NodeList sheets = root.getElementsByTagNameNS(NS, "Worksheet");
            org.w3c.dom.Element target = null;
            for (int i=0;i<sheets.getLength();i++) {
                org.w3c.dom.Element ws = (org.w3c.dom.Element) sheets.item(i);
                String name = ws.getAttributeNS(NS, "Name");
                if (name == null || name.isEmpty()) name = ws.getAttribute("ss:Name");
                if ("Clinical Concepts".equalsIgnoreCase(name)) { target = ws; break; }
            }
            if (target == null) {
                // Fallback for sharded run workbook: aggregate from Pipelines Summary sheet
                for (int i=0;i<sheets.getLength();i++) {
                    org.w3c.dom.Element ws = (org.w3c.dom.Element) sheets.item(i);
                    String n2 = ws.getAttributeNS(NS, "Name");
                    if (n2 == null || n2.isEmpty()) n2 = ws.getAttribute("ss:Name");
                    if ("Pipelines Summary".equalsIgnoreCase(n2)) {
                        return aggregateMetricsFromPipelinesSummary(ws, NS);
                    }
                }
                return null;
            }
            org.w3c.dom.NodeList tables = target.getElementsByTagNameNS(NS, "Table");
            if (tables.getLength() == 0) return null;
            org.w3c.dom.Element table = (org.w3c.dom.Element) tables.item(0);
            org.w3c.dom.NodeList rows = table.getElementsByTagNameNS(NS, "Row");
            if (rows.getLength() < 2) return null;
            java.util.List<String> header = new java.util.ArrayList<>();
            java.util.Map<String,Integer> colIdx = new java.util.HashMap<>();
            // Parse header
            org.w3c.dom.Element hr = (org.w3c.dom.Element) rows.item(0);
            java.util.List<String> hcells = extractRowCells(hr, NS);
            for (int i=0;i<hcells.size();i++) { header.add(hcells.get(i)); colIdx.put(hcells.get(i), i); }
            int iDoc = colIdx.getOrDefault("Document", -1);
            int iConf = colIdx.getOrDefault("Confidence", -1);
            int iDtr = colIdx.getOrDefault("DocTimeRel", -1);
            int iDeg = colIdx.getOrDefault("DegreeOf", -1);
            int iLoc = colIdx.getOrDefault("LocationOfText", -1);
            int iCoref = colIdx.getOrDefault("Coref", -1);
            int iCui = colIdx.getOrDefault("CUI", -1);
            int iDis = colIdx.getOrDefault("Disambiguated", -1);
            int iCand = colIdx.getOrDefault("CandidateCount", -1);
            if (iDoc < 0) return null;
            java.util.Set<String> docs = new java.util.HashSet<>();
            int mentionCount = 0, dtrCount = 0, relCount = 0, corefCount = 0;
            double confSum = 0.0; int confN = 0;
            java.util.Set<String> distinctCuis = new java.util.HashSet<>();
            int disambTrue = 0; long candSum = 0;
            for (int r=1;r<rows.getLength();r++) {
                org.w3c.dom.Element rr = (org.w3c.dom.Element) rows.item(r);
                java.util.List<String> cells = extractRowCells(rr, NS);
                if (cells.isEmpty()) continue;
                String docId = iDoc<cells.size()? cells.get(iDoc):"";
                if (!docId.isEmpty()) docs.add(docId);
                mentionCount++;
                if (iConf>=0 && iConf<cells.size()) {
                    try { confSum += Double.parseDouble(cells.get(iConf)); confN++; } catch (Exception ignore) {}
                }
                if (iDtr>=0 && iDtr<cells.size() && !nvl(cells.get(iDtr)).isEmpty()) dtrCount++;
                if (iDeg>=0 && iDeg<cells.size() && "true".equalsIgnoreCase(cells.get(iDeg))) relCount++;
                if (iLoc>=0 && iLoc<cells.size() && !nvl(cells.get(iLoc)).isEmpty()) relCount++;
                if (iCoref>=0 && iCoref<cells.size() && "true".equalsIgnoreCase(cells.get(iCoref))) corefCount++;
                if (iCui>=0 && iCui<cells.size()) { String cui = cells.get(iCui); if (cui!=null && !cui.isEmpty()) distinctCuis.add(cui); }
                if (iDis>=0 && iDis<cells.size()) { String d = cells.get(iDis); if ("true".equalsIgnoreCase(d)) disambTrue++; }
                if (iCand>=0 && iCand<cells.size()) {
                    try { candSum += Long.parseLong(cells.get(iCand).trim()); } catch (Exception ignore) {}
                }
            }
            SubdirMetrics m = new SubdirMetrics();
            m.docCount = docs.size();
            m.mentionCount = mentionCount;
            m.avgConfidence = confN>0? confSum/confN : 0.0;
            m.docTimeRelCount = dtrCount;
            m.relationCount = relCount;
            m.markableCount = corefCount;
            m.distinctCuiCount = distinctCuis.size();
            m.disambTrueCount = disambTrue;
            m.candidateCountSum = candSum;
            return m;
        } catch (Exception e) {
            return null;
        }
    }
    // Minimal XLSX metrics reader: reads either Clinical Concepts or Overview sheet
    private static SubdirMetrics computeMetricsFromXlsx(Path xlsx) {
        try (java.util.zip.ZipFile zip = new java.util.zip.ZipFile(xlsx.toFile())) {
            // Map rId -> sheet target path, and rId -> name
            Map<String,String> idToTarget = new LinkedHashMap<>();
            Map<String,String> idToName = new LinkedHashMap<>();
            try (InputStream rels = zip.getInputStream(new java.util.zip.ZipEntry("xl/_rels/workbook.xml.rels"))) {
                javax.xml.parsers.DocumentBuilderFactory dbf = javax.xml.parsers.DocumentBuilderFactory.newInstance();
                javax.xml.parsers.DocumentBuilder db = dbf.newDocumentBuilder();
                org.w3c.dom.Document d = db.parse(rels);
                org.w3c.dom.NodeList rs = d.getElementsByTagName("Relationship");
                for (int i=0;i<rs.getLength();i++) {
                    org.w3c.dom.Element e = (org.w3c.dom.Element) rs.item(i);
                    String id = e.getAttribute("Id");
                    String target = e.getAttribute("Target");
                    if (id!=null && !id.isEmpty() && target!=null && !target.isEmpty()) idToTarget.put(id, target);
                }
            }
            try (InputStream wb = zip.getInputStream(new java.util.zip.ZipEntry("xl/workbook.xml"))) {
                javax.xml.parsers.DocumentBuilderFactory dbf = javax.xml.parsers.DocumentBuilderFactory.newInstance();
                javax.xml.parsers.DocumentBuilder db = dbf.newDocumentBuilder();
                org.w3c.dom.Document d = db.parse(wb);
                org.w3c.dom.NodeList sheets = d.getElementsByTagName("sheet");
                for (int i=0;i<sheets.getLength();i++) {
                    org.w3c.dom.Element e = (org.w3c.dom.Element) sheets.item(i);
                    String name = e.getAttribute("name");
                    String rid = e.getAttributeNS("http://schemas.openxmlformats.org/officeDocument/2006/relationships", "id");
                    if (rid==null || rid.isEmpty()) rid = e.getAttribute("r:id");
                    if (rid!=null && !rid.isEmpty()) idToName.put(rid, name);
                }
            }
            // Find desired sheet
            String clinicalTarget = null, overviewTarget = null;
            for (Map.Entry<String,String> e : idToName.entrySet()) {
                String name = e.getValue()==null?"":e.getValue();
                String target = idToTarget.get(e.getKey());
                if (name.equalsIgnoreCase("Clinical Concepts")) clinicalTarget = target;
                else if (name.equalsIgnoreCase("Overview")) overviewTarget = target;
            }
            if (clinicalTarget != null) {
                List<List<String>> rows = readXlsxSheetRows(zip, "xl/" + clinicalTarget);
                if (rows.size() > 1) {
                    // Header indices
                    List<String> h = rows.get(0);
                    Map<String,Integer> idx = new HashMap<>();
                    for (int i=0;i<h.size();i++) idx.put(h.get(i), i);
                    int iDoc = idx.getOrDefault("Document", -1);
                    int iConf = idx.getOrDefault("Confidence", -1);
                    int iDtr = idx.getOrDefault("DocTimeRel", -1);
                    int iDeg = idx.getOrDefault("DegreeOf", -1);
                    int iLoc = idx.getOrDefault("LocationOfText", -1);
                    int iCoref = idx.getOrDefault("Coref", -1);
                    int iCui = idx.getOrDefault("CUI", -1);
                    int iDis = idx.getOrDefault("Disambiguated", -1);
                    int iCand = idx.getOrDefault("CandidateCount", -1);
                    Set<String> docs = new LinkedHashSet<>();
                    int mentionCount = 0, dtr=0, rel=0, coref=0, dis=0; double confSum=0.0; int confN=0; Set<String> dcuis=new LinkedHashSet<>(); long candSum=0;
                    for (int r=1;r<rows.size();r++) {
                        List<String> row = rows.get(r); if (row==null || row.isEmpty()) continue;
                        String doc = (iDoc>=0 && iDoc<row.size()) ? row.get(iDoc) : ""; if (doc!=null && !doc.isEmpty()) docs.add(doc);
                        mentionCount++;
                        if (iConf>=0 && iConf<row.size()) try { confSum += Double.parseDouble(nvl(row.get(iConf))); confN++; } catch (Exception ignore) {}
                        if (iDtr>=0 && iDtr<row.size() && !nvl(row.get(iDtr)).isEmpty()) dtr++;
                        if (iDeg>=0 && iDeg<row.size() && "true".equalsIgnoreCase(nvl(row.get(iDeg)))) rel++;
                        if (iLoc>=0 && iLoc<row.size() && !nvl(row.get(iLoc)).isEmpty()) rel++;
                        if (iCoref>=0 && iCoref<row.size() && "true".equalsIgnoreCase(nvl(row.get(iCoref)))) coref++;
                        if (iCui>=0 && iCui<row.size()) { String cui = nvl(row.get(iCui)); if (!cui.isEmpty()) dcuis.add(cui); }
                        if (iDis>=0 && iDis<row.size() && "true".equalsIgnoreCase(nvl(row.get(iDis)))) dis++;
                        if (iCand>=0 && iCand<row.size()) { try { candSum += Long.parseLong(nvl(row.get(iCand))); } catch (Exception ignore) {} }
                    }
                    SubdirMetrics m = new SubdirMetrics();
                    m.docCount = docs.size();
                    m.mentionCount = mentionCount;
                    m.avgConfidence = confN>0? confSum/confN : 0.0;
                    m.docTimeRelCount = dtr;
                    m.relationCount = rel;
                    m.markableCount = coref;
                    m.distinctCuiCount = dcuis.size();
                    m.disambTrueCount = dis;
                    m.candidateCountSum = candSum;
                    return m;
                }
            }
            if (overviewTarget != null) {
                List<List<String>> rows = readXlsxSheetRows(zip, "xl/" + overviewTarget);
                SubdirMetrics m = new SubdirMetrics();
                for (List<String> r : rows) {
                    if (r.size() < 2) continue;
                    String k = nvl(r.get(0)); String v = nvl(r.get(1));
                    if (k.equalsIgnoreCase("Documents Found") || k.equalsIgnoreCase("Documents Found (XMI)") || k.equalsIgnoreCase("Documents Processed")) {
                        try { m.docCount = Integer.parseInt(v.replace(",","")); } catch (Exception ignore) {}
                    } else if (k.equalsIgnoreCase("Clinical Concepts Total")) {
                        try { m.mentionCount = Integer.parseInt(v.replace(",","")); } catch (Exception ignore) {}
                    } else if (k.equalsIgnoreCase("Distinct CUIs")) {
                        try { m.distinctCuiCount = Integer.parseInt(v.replace(",","")); } catch (Exception ignore) {}
                    } else if (k.equalsIgnoreCase("Concepts With DocTimeRel")) {
                        try { m.docTimeRelCount = Integer.parseInt(v.replace(",","")); } catch (Exception ignore) {}
                    } else if (k.equalsIgnoreCase("Concepts With DegreeOf")) {
                        try { m.relationCount += Integer.parseInt(v.replace(",","")); } catch (Exception ignore) {}
                    } else if (k.equalsIgnoreCase("Concepts In Coref")) {
                        try { m.markableCount = Integer.parseInt(v.replace(",","")); } catch (Exception ignore) {}
                    }
                }
                return m;
            }
        } catch (Exception ignore) {}
        return null;
    }
    private static List<List<String>> readXlsxSheetRows(java.util.zip.ZipFile zip, String entryName) throws IOException {
        java.util.List<java.util.List<String>> out = new java.util.ArrayList<>();
        java.util.zip.ZipEntry e = zip.getEntry(entryName);
        if (e == null) return out;
        try (InputStream in = zip.getInputStream(e)) {
            javax.xml.parsers.DocumentBuilderFactory dbf = javax.xml.parsers.DocumentBuilderFactory.newInstance();
            javax.xml.parsers.DocumentBuilder db = dbf.newDocumentBuilder();
            org.w3c.dom.Document d = db.parse(in);
            org.w3c.dom.NodeList rows = d.getElementsByTagName("row");
            for (int i=0;i<rows.getLength();i++) {
                org.w3c.dom.Element r = (org.w3c.dom.Element) rows.item(i);
                org.w3c.dom.NodeList cs = r.getElementsByTagName("c");
                List<String> row = new ArrayList<>();
                int colPos = 0;
                for (int j=0;j<cs.getLength();j++) {
                    org.w3c.dom.Element c = (org.w3c.dom.Element) cs.item(j);
                    String ref = c.getAttribute("r");
                    int idx = excelColIndex(ref);
                    while (colPos < idx) { row.add(""); colPos++; }
                    String t = c.getAttribute("t");
                    String val = "";
                    if ("inlineStr".equals(t)) {
                        org.w3c.dom.NodeList is = c.getElementsByTagName("is");
                        if (is.getLength()>0) {
                            org.w3c.dom.Element isEl = (org.w3c.dom.Element) is.item(0);
                            org.w3c.dom.NodeList ts = isEl.getElementsByTagName("t");
                            if (ts.getLength()>0) val = ts.item(0).getTextContent();
                        }
                    } else {
                        org.w3c.dom.NodeList vs = c.getElementsByTagName("v");
                        if (vs.getLength()>0) val = vs.item(0).getTextContent();
                    }
                    row.add(val==null?"":val);
                    colPos++;
                }
                out.add(row);
            }
        } catch (Exception ex) {
            // ignore
        }
        return out;
    }
    private static int excelColIndex(String cellRef) {
        if (cellRef == null || cellRef.isEmpty()) return 0;
        int i = 0; while (i < cellRef.length() && Character.isLetter(cellRef.charAt(i))) i++;
        String col = cellRef.substring(0, i).toUpperCase(java.util.Locale.ROOT);
        int idx = 0; for (int k=0;k<col.length();k++) idx = idx*26 + (col.charAt(k)-'A'+1);
        return Math.max(0, idx-1);
    }
    private static java.util.List<String> extractRowCells(org.w3c.dom.Element row, String NS) {
        java.util.List<String> out = new java.util.ArrayList<>();
        org.w3c.dom.NodeList cells = row.getElementsByTagNameNS(NS, "Cell");
        for (int i=0;i<cells.getLength();i++) {
            org.w3c.dom.Element c = (org.w3c.dom.Element) cells.item(i);
            org.w3c.dom.NodeList datas = c.getElementsByTagNameNS(NS, "Data");
            String v = "";
            if (datas.getLength()>0) v = datas.item(0).getTextContent();
            out.add(v==null?"":v);
        }
        return out;
    }
    private static Double computeAvgSecFromSummary(Path subDir) {
        if (subDir == null || !Files.isDirectory(subDir)) return null;
        // Locate per-run workbook xml
        Path report = null;
        try (DirectoryStream<Path> ds = Files.newDirectoryStream(subDir, p -> {
            String n = p.getFileName().toString().toLowerCase(java.util.Locale.ROOT);
            if (!n.endsWith(".xml")) return false;
            if (n.contains("fullclinical") && n.contains("_local")) return false;
            if (n.startsWith("ctakes-report-compare")) return false;
            return true;
        })) {
            Path newest = null; long lm = Long.MIN_VALUE;
            for (Path p : ds) { long m = p.toFile().lastModified(); if (m > lm) { lm = m; newest = p; } }
            report = newest;
        } catch (IOException ignore) {}
        if (report == null || !Files.isRegularFile(report)) return null;
        try {
            javax.xml.parsers.DocumentBuilderFactory dbf = javax.xml.parsers.DocumentBuilderFactory.newInstance();
            dbf.setNamespaceAware(true);
            javax.xml.parsers.DocumentBuilder db = dbf.newDocumentBuilder();
            org.w3c.dom.Document doc = db.parse(report.toFile());
            org.w3c.dom.Element root = doc.getDocumentElement();
            String NS = "urn:schemas-microsoft-com:office:spreadsheet";
            org.w3c.dom.NodeList sheets = root.getElementsByTagNameNS(NS, "Worksheet");
            org.w3c.dom.Element target = null;
            for (int i=0;i<sheets.getLength();i++) {
                org.w3c.dom.Element ws = (org.w3c.dom.Element) sheets.item(i);
                String n2 = ws.getAttributeNS(NS, "Name");
                if (n2 == null || n2.isEmpty()) n2 = ws.getAttribute("ss:Name");
                if ("Pipelines Summary".equalsIgnoreCase(n2)) { target = ws; break; }
            }
            if (target == null) return null;
            org.w3c.dom.NodeList tables = target.getElementsByTagNameNS(NS, "Table");
            if (tables.getLength() == 0) return null;
            org.w3c.dom.Element table = (org.w3c.dom.Element) tables.item(0);
            org.w3c.dom.NodeList rows = table.getElementsByTagNameNS(NS, "Row");
            if (rows.getLength() < 2) return null;
            java.util.List<String> header = new java.util.ArrayList<>();
            java.util.Map<String,Integer> colIdx = new java.util.HashMap<>();
            org.w3c.dom.Element hr = (org.w3c.dom.Element) rows.item(0);
            java.util.List<String> hcells = extractRowCells(hr, NS);
            for (int i=0;i<hcells.size();i++) { header.add(hcells.get(i)); colIdx.put(hcells.get(i), i); }
            int iDocs = colIdx.getOrDefault("Documents", -1);
            int iAvg = colIdx.getOrDefault("Average Seconds per Document", -1);
            if (iDocs < 0 || iAvg < 0) return null;
            long totalDocs = 0; double weighted = 0.0;
            for (int r=1;r<rows.getLength();r++) {
                java.util.List<String> cells = extractRowCells((org.w3c.dom.Element) rows.item(r), NS);
                if (cells.isEmpty()) continue;
                long docs = 0; double avg = 0.0;
                try { docs = Long.parseLong(nvl(cells.get(iDocs)).replace(",","")); } catch (Exception ignore) {}
                try { avg = Double.parseDouble(nvl(cells.get(iAvg)).replace(",","")); } catch (Exception ignore) {}
                if (docs > 0 && avg > 0) { totalDocs += docs; weighted += docs * avg; }
            }
            if (totalDocs > 0) return weighted / totalDocs;
        } catch (Exception ignore) {}
        return null;
    }
    private static class SubdirMetrics {
        int docCount;                 // number of XMI documents
        int mentionCount;             // total clinical concept mentions
        double avgConfidence;         // average confidence across mentions
        int docTimeRelCount;          // mentions with DocTimeRel
        int relationCount;            // DegreeOf + LocationOf counts
        int markableCount;            // coref markables (proxy for chains)
        int distinctCuiCount;         // distinct CUIs across all mentions
        int disambTrueCount;          // mentions with Disambiguated=true
        long candidateCountSum;       // sum of CandidateCount across mentions
        boolean hasSmoking;           // any smoking status annotation present
    }
    private static SubdirMetrics computeSubdirMetrics(Path subDir) {
        SubdirMetrics m = new SubdirMetrics();
        Path xmiDir = subDir.resolve("xmi");
        java.util.List<Path> xmiDirs = new java.util.ArrayList<>();
        if (Files.isDirectory(xmiDir)) {
            xmiDirs.add(xmiDir);
        } else {
            // Sharded run: collect shard_*/xmi dirs
            try (DirectoryStream<Path> shards = Files.newDirectoryStream(subDir, p -> Files.isDirectory(p) && p.getFileName().toString().startsWith("shard_"))) {
                for (Path sh : shards) {
                    Path sx = sh.resolve("xmi");
                    if (Files.isDirectory(sx)) xmiDirs.add(sx);
                }
            } catch (IOException ignore) {}
            if (xmiDirs.isEmpty()) return m;
        }
        int parsed = 0;
        for (Path xd : xmiDirs) {
            try (DirectoryStream<Path> ds = Files.newDirectoryStream(xd, p -> p.toString().endsWith(".xmi") || p.toString().endsWith(".XMI"))) {
                for (Path p : ds) {
                parsed++;
                if (parsed % 1000 == 0) System.out.println("[report]   parsed " + parsed + " XMI files (metrics)");
                m.docCount++;
                try {
                    javax.xml.parsers.DocumentBuilderFactory dbf = javax.xml.parsers.DocumentBuilderFactory.newInstance();
                    dbf.setNamespaceAware(true);
                    javax.xml.parsers.DocumentBuilder db = dbf.newDocumentBuilder();
                    org.w3c.dom.Document doc = db.parse(p.toFile());
                    org.w3c.dom.Element root = doc.getDocumentElement();
                    // Mentions (count only concept-bearing mentions)
                    org.w3c.dom.NodeList all = root.getChildNodes();
                    int mentions = 0; double confSum = 0.0; int confN=0; int docTimeRel=0;
                    java.util.Set<String> distinctCuis = new java.util.HashSet<>();
                    for (int i=0;i<all.getLength();i++) {
                        org.w3c.dom.Node n = all.item(i);
                        if (n.getNodeType()!=org.w3c.dom.Node.ELEMENT_NODE) continue;
                        org.w3c.dom.Element e=(org.w3c.dom.Element)n;
                        String ns = e.getNamespaceURI(); String name = e.getLocalName();
                        if (ns!=null && ns.endsWith("/textsem.ecore") && name!=null && name.endsWith("Mention")) {
                            String oc = e.getAttribute("ontologyConceptArr");
                            if (oc==null || oc.isEmpty()) continue; // skip non-concept mentions
                            mentions++;
                            try { String c = e.getAttribute("confidence"); if (!c.isEmpty()) { confSum+=Double.parseDouble(c); confN++; } } catch (Exception ignore) {}
                            String ev = e.getAttribute("event"); if (!ev.isEmpty()) docTimeRel++;
                            // crude best-concept CUI extraction: if single id, try to find matching UmlsConcept by direct id
                            // (full candidate stats handled by report-based path)
                            String bestId = oc.indexOf(' ')>=0 ? oc.substring(0, oc.indexOf(' ')).trim() : oc.trim();
                            if (!bestId.isEmpty()) {
                                try {
                                    org.w3c.dom.NodeList umls = root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/refsem.ecore", "UmlsConcept");
                                    for (int ui=0; ui<umls.getLength(); ui++) {
                                        org.w3c.dom.Element ue = (org.w3c.dom.Element) umls.item(ui);
                                        if (bestId.equals(ue.getAttribute("xmi:id"))) {
                                            String cui = ue.getAttribute("cui");
                                            if (cui!=null && !cui.isEmpty()) distinctCuis.add(cui);
                                            break;
                                        }
                                    }
                                } catch (Exception ignore) {}
                            }
                        }
                    }
                    m.mentionCount += mentions;
                    if (confN>0) m.avgConfidence += confSum/confN; // avg per doc, then average across docs
                    // Relations and coref
                    m.relationCount += root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/relation.ecore", "DegreeOfTextRelation").getLength();
                    m.relationCount += root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/relation.ecore", "LocationOfTextRelation").getLength();
                    m.markableCount += root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/textsem.ecore", "Markable").getLength();
                    m.docTimeRelCount += docTimeRel;
                    // Smoking status presence (either namespace variant)
                    try {
                        int s1 = root.getElementsByTagNameNS("http:///org/apache/ctakes/smokingstatus/type.ecore", "*").getLength();
                        int s2 = root.getElementsByTagNameNS("http:///org/apache/ctakes/smokingstatus/i2b2/type.ecore", "*").getLength();
                        int s3 = root.getElementsByTagNameNS("http:///org/apache/ctakes/smokingstatus/type/libsvm.ecore", "*").getLength();
                        if (s1 + s2 + s3 > 0) m.hasSmoking = true;
                    } catch (Exception ignore) {}
                    m.distinctCuiCount += distinctCuis.size();
                } catch (Exception ignore) {}
            }
            } catch (IOException ignore) {}
        }
        if (parsed > 0) System.out.println("[report]   parsed total " + parsed + " XMI files (metrics)");
        if (m.docCount>0) m.avgConfidence = m.avgConfidence / m.docCount;
        return m;
    }
    private static String parseValueFromLog(Path runLog, String key) {
        // For "Documents Processed", return the maximum value found (not the last occurrence)
        // For other keys, return the last occurrence (final summary)
        String last = "";
        int maxDocs = 0;
        boolean isDocumentCount = key.equalsIgnoreCase("Documents Processed");
        try (java.io.BufferedReader br = java.nio.file.Files.newBufferedReader(runLog, java.nio.charset.StandardCharsets.UTF_8)) {
            String line; while ((line = br.readLine()) != null) {
                int i = line.indexOf(':');
                if (i > 0) {
                    String k = line.substring(0,i).trim();
                    if (k.equalsIgnoreCase(key)) {
                        String val = line.substring(i+1).trim();
                        if (isDocumentCount) {
                            try {
                                int docCount = Integer.parseInt(val.replace(",", ""));
                                if (docCount > maxDocs) {
                                    maxDocs = docCount;
                                    last = val;
                                }
                            } catch (Exception ignore) {
                                last = val;
                            }
                        } else {
                            last = val;
                        }
                    }
                }
            }
        } catch (IOException ignore) {}
        return last;
    }

    private static class ProcWindow { Long runStart; Long earliestStart; Long latestEnd; }
    private static ProcWindow computeProcessingWindow(Path runLog) {
        if (runLog == null || !java.nio.file.Files.isRegularFile(runLog)) return null;
        ProcWindow w = new ProcWindow();
        java.text.SimpleDateFormat fmt = new java.text.SimpleDateFormat("EEE MMM dd HH:mm:ss z yyyy", java.util.Locale.ENGLISH);
        java.text.SimpleDateFormat fmtNoTz = new java.text.SimpleDateFormat("EEE MMM dd HH:mm:ss yyyy", java.util.Locale.ENGLISH);
        try (java.io.BufferedReader br = java.nio.file.Files.newBufferedReader(runLog, java.nio.charset.StandardCharsets.UTF_8)) {
            String line; while ((line = br.readLine()) != null) {
                int i = line.indexOf(':'); if (i <= 0) continue;
                String k = line.substring(0,i).trim(); String v = line.substring(i+1).trim();
                if (k.equalsIgnoreCase("Run Start Time")) {
                    Long t = parseDateLenient(v, fmt, fmtNoTz); if (t != null) w.runStart = t;
                } else if (k.equalsIgnoreCase("Processing Start Time")) {
                    Long t = parseDateLenient(v, fmt, fmtNoTz); if (t != null && (w.earliestStart==null || t < w.earliestStart)) w.earliestStart = t;
                } else if (k.equalsIgnoreCase("Processing End Time")) {
                    Long t = parseDateLenient(v, fmt, fmtNoTz); if (t != null && (w.latestEnd==null || t > w.latestEnd)) w.latestEnd = t;
                }
            }
        } catch (Exception ignore) {}
        if (w.runStart==null && w.earliestStart==null && w.latestEnd==null) return null;
        return w;
    }

    // Compute average seconds per doc using run-level Processing Start/End across shards.
    // We scan all occurrences and take earliest start and latest end, then divide window by docCount.
    private static Double computeAvgFromProcessingWindow(Path runLog, int docCount) {
        if (runLog == null || !java.nio.file.Files.isRegularFile(runLog) || docCount <= 0) return null;
        java.text.SimpleDateFormat fmt = new java.text.SimpleDateFormat("EEE MMM dd HH:mm:ss z yyyy", java.util.Locale.ENGLISH);
        java.text.SimpleDateFormat fmtNoTz = new java.text.SimpleDateFormat("EEE MMM dd HH:mm:ss yyyy", java.util.Locale.ENGLISH);
        long earliestStart = Long.MAX_VALUE, latestEnd = Long.MIN_VALUE;
        try (java.io.BufferedReader br = java.nio.file.Files.newBufferedReader(runLog, java.nio.charset.StandardCharsets.UTF_8)) {
            String line;
            while ((line = br.readLine()) != null) {
                int i = line.indexOf(':');
                if (i <= 0) continue;
                String k = line.substring(0,i).trim();
                if (!k.equalsIgnoreCase("Processing Start Time") && !k.equalsIgnoreCase("Processing End Time")) continue;
                String v = line.substring(i+1).trim();
                Long ts = parseDateLenient(v, fmt, fmtNoTz);
                if (ts == null) continue;
                if (k.equalsIgnoreCase("Processing Start Time")) {
                    if (ts < earliestStart) earliestStart = ts;
                } else {
                    if (ts > latestEnd) latestEnd = ts;
                }
            }
        } catch (Exception ignore) {}
        if (earliestStart == Long.MAX_VALUE || latestEnd == Long.MIN_VALUE || latestEnd <= earliestStart) return null;
        double totalSec = (latestEnd - earliestStart) / 1000.0;
        if (totalSec <= 0) return null;
        return totalSec / Math.max(1, docCount);
    }

    private static Long parseDateLenient(String s, java.text.SimpleDateFormat fmt, java.text.SimpleDateFormat fmtNoTz) {
        if (s == null || s.isEmpty()) return null;
        try { return fmt.parse(s).getTime(); } catch (Exception ignore) {}
        try { return fmtNoTz.parse(s).getTime(); } catch (Exception ignore) {}
        return null;
    }

    private static Path findPiperFromLog(Path log) {
        if (log == null) return null;
        Pattern p = Pattern.compile("Loading Piper File (.*?\\.piper)");
        try (BufferedReader br = Files.newBufferedReader(log, StandardCharsets.UTF_8)) {
            String line; while ((line = br.readLine()) != null) {
                Matcher m = p.matcher(line);
                if (m.find()) {
                    String s = m.group(1).trim();
                    Path pp = Paths.get(s);
                    if (pp.toFile().exists()) return pp.toAbsolutePath().normalize();
                }
            }
        } catch (IOException ignored) {}
        return null;
    }

    private static Path findDictXmlInDir(Path outDir) {
        if (outDir == null) return null;
        try {
            Optional<Path> any = Files.list(outDir)
                    .filter(p -> p.getFileName().toString().endsWith(".xml"))
                    .findFirst();
            return any.orElse(null);
        } catch (IOException ignored) {}
        return null;
    }

    private static List<List<String>> buildRunInfo(Path log, Path piper, Path dictXml, Path outDir) {
        List<List<String>> rows = new ArrayList<>();
        rows.add(Arrays.asList("Key","Value"));
        Map<String,String> kv = new LinkedHashMap<>();
        if (log != null) {
            Map<String,String> parsed = parseKeyValuesFromLog(log);
            kv.putAll(parsed);
            kv.put("Run Log", log.toString());
        }
        if (piper != null) kv.put("Pipeline", piper.toString());
        if (dictXml != null) kv.put("Dictionary XML", dictXml.toString());
        if (outDir != null) kv.put("Output Dir", outDir.toString());
        // Add actual document count from XMI directory; if it's greater than
        // the parsed "Documents Processed" from log, prefer the XMI count.
        try {
            int xmiDocs = countXmiDocs(outDir.resolve("xmi"));
            String logged = kv.getOrDefault("Documents Processed", "");
            int loggedInt = 0; try { loggedInt = Integer.parseInt(logged.trim()); } catch (Exception ignore) {}
            if (xmiDocs > 0 && xmiDocs > loggedInt) {
                kv.put("Documents Processed", String.valueOf(xmiDocs));
            }
            kv.put("Documents Found (XMI)", String.valueOf(xmiDocs));
        } catch (Exception ignored) {}
        for (Map.Entry<String,String> e : kv.entrySet()) {
            rows.add(Arrays.asList(e.getKey(), e.getValue()));
        }
        return rows;
    }

    private static Map<String,String> parseKeyValuesFromLog(Path log) {
        Map<String,String> kv = new LinkedHashMap<>();
        Pattern colonLine = Pattern.compile("^(Build Version|Build Date|Run Start Time|Processing Start Time|Processing End Time|Initialization Time Elapsed|Processing Time Elapsed|Total Run Time Elapsed|Documents Processed|Average Seconds per Document)\s*:\s*(.*)$");
        try (BufferedReader br = Files.newBufferedReader(log, StandardCharsets.UTF_8)) {
            String line; while ((line = br.readLine()) != null) {
                Matcher m = colonLine.matcher(line);
                if (m.find()) {
                    kv.put(m.group(1).trim(), m.group(2).trim());
                }
            }
        } catch (IOException ignored) {}
        return kv;
    }

    private static List<List<String>> aggregateBsvWithDoc(Path dir, String[] expectedHeader) throws IOException {
        List<List<String>> rows = new ArrayList<>();
        List<String> header = new ArrayList<>();
        header.add("Document");
        header.addAll(Arrays.asList(expectedHeader));
        rows.add(header);
        if (!Files.isDirectory(dir)) return rows;
        int fileCount = 0;
        try (DirectoryStream<Path> ds = Files.newDirectoryStream(dir, path -> {
            String n = path.toString();
            return n.endsWith(".BSV") || n.endsWith(".bsv");
        })) {
            for (Path p : ds) {
                fileCount++;
                if (fileCount % 1000 == 0) System.out.println("[report]   aggregated " + fileCount + " files from " + dir.getFileName());
                List<String> lines = Files.readAllLines(p, StandardCharsets.UTF_8);
                if (lines.isEmpty()) continue;
                for (int i=1;i<lines.size();i++) { // skip header
                    String line = lines.get(i).trim();
                    if (line.isEmpty()) continue;
                    String[] cells = Arrays.stream(line.split("\\|", -1)).map(String::trim).toArray(String[]::new);
                    List<String> row = new ArrayList<>();
                    row.add(baseName(p));
                    row.addAll(Arrays.asList(cells));
                    rows.add(row);
                }
            }
        }
        if (fileCount > 0) System.out.println("[report]   aggregated total " + fileCount + " files from " + dir.getFileName());
        if (rows.size() == 1) rows.add(Arrays.asList("No data found"));
        return rows;
    }

    private static List<List<String>> aggregateCuiCounts(Path outDir) throws IOException {
        List<List<String>> rows = new ArrayList<>();
        rows.add(Arrays.asList("Document","CUI","Negated","Count"));
        // Prefer cui_count subdir; fallback to any *.cuicount.bsv in outDir
        Path cc = outDir.resolve("cui_count");
        List<Path> files = new ArrayList<>();
        if (Files.isDirectory(cc)) {
            try (DirectoryStream<Path> ds = Files.newDirectoryStream(cc, path -> {
                String n = path.toString().toLowerCase(java.util.Locale.ROOT);
                return n.endsWith(".bsv") || n.endsWith(".cuicount") || n.endsWith(".cuicount.bsv");
            })) {
                for (Path p : ds) files.add(p);
            }
        }
        // Also gather from shard_*/cui_count
        try (DirectoryStream<Path> ds = Files.newDirectoryStream(outDir, p -> Files.isDirectory(p) && p.getFileName().toString().startsWith("shard_"))) {
            for (Path sh : ds) {
                Path scc = sh.resolve("cui_count");
                if (!Files.isDirectory(scc)) continue;
                try (DirectoryStream<Path> ds2 = Files.newDirectoryStream(scc, path -> {
                    String n = path.toString().toLowerCase(java.util.Locale.ROOT);
                    return n.endsWith(".bsv") || n.endsWith(".cuicount") || n.endsWith(".cuicount.bsv");
                })) {
                    for (Path p : ds2) files.add(p);
                } catch (IOException ignore) {}
            }
        } catch (IOException ignore) {}
        // Fallback: any *.cuicount.bsv at current level
        if (files.isEmpty()) {
            try (DirectoryStream<Path> ds = Files.newDirectoryStream(outDir, path -> path.getFileName().toString().toLowerCase(java.util.Locale.ROOT).contains("cuicount") && path.toString().toLowerCase(java.util.Locale.ROOT).endsWith(".bsv"))) {
                for (Path p : ds) files.add(p);
            }
        }
        // If we found explicit cui_count files, aggregate them and return
        if (!files.isEmpty()) {
            for (Path p : files) {
                List<String> lines = Files.readAllLines(p, StandardCharsets.UTF_8);
                for (String line : lines) {
                    if (line.trim().isEmpty()) continue;
                    String[] parts = line.split("\\|", -1);
                    if (parts.length >= 2) {
                        String rawCui = parts[0].trim();
                        boolean neg = rawCui.startsWith("-");
                        String cui = neg ? rawCui.substring(1) : rawCui;
                        rows.add(Arrays.asList(baseName(p), cui, String.valueOf(neg), parts[1].trim()));
                    }
                }
            }
            return rows;
        }
        // Fallback: derive per-document CUI counts from csv_table_concepts when cui_count is absent
        Map<String, Map<String, long[]>> perDoc = new LinkedHashMap<>(); // doc -> (cui -> [affirmed, negated])
        List<Path> csvDirs = new ArrayList<>();
        Path c1 = outDir.resolve("csv_table_concepts");
        if (Files.isDirectory(c1)) csvDirs.add(c1);
        try (DirectoryStream<Path> shards = Files.newDirectoryStream(outDir, p -> Files.isDirectory(p) && p.getFileName().toString().startsWith("shard_"))) {
            for (Path sh : shards) {
                Path sc1 = sh.resolve("csv_table_concepts");
                if (Files.isDirectory(sc1)) csvDirs.add(sc1);
            }
        } catch (IOException ignore) {}
        for (Path dir : csvDirs) {
            try (DirectoryStream<Path> ds = Files.newDirectoryStream(dir, p -> p.toString().toLowerCase(java.util.Locale.ROOT).endsWith(".csv"))) {
                for (Path p : ds) {
                    String doc = baseName(p);
                    List<String> lines = Files.readAllLines(p, StandardCharsets.UTF_8);
                    if (lines.isEmpty()) continue;
                    List<String> header = parseCsvLine(lines.get(0));
                    int idxCui = indexOf(header, "CUI");
                    int idxNeg = indexOf(header, "Negated");
                    int idxPol = indexOf(header, "Polarity");
                    if (idxCui < 0) continue;
                    Map<String,long[]> map = perDoc.computeIfAbsent(doc, k -> new LinkedHashMap<>());
                    for (int i=1;i<lines.size();i++) {
                        String line = lines.get(i);
                        if (line == null) continue; line = line.trim(); if (line.isEmpty()) continue;
                        List<String> cols = parseCsvLine(line);
                        if (idxCui >= cols.size()) continue;
                        String cui = nvl(cols.get(idxCui)).trim();
                        if (cui.isEmpty() || cui.equalsIgnoreCase("null")) continue;
                        boolean neg = false;
                        if (idxNeg >= 0 && idxNeg < cols.size()) {
                            String nv = nvl(cols.get(idxNeg));
                            neg = "true".equalsIgnoreCase(nv) || "1".equals(nv);
                        } else if (idxPol >= 0 && idxPol < cols.size()) {
                            String pv = nvl(cols.get(idxPol));
                            try { neg = Integer.parseInt(pv) < 0; } catch (Exception ignore2) {}
                        }
                        long[] arr = map.computeIfAbsent(cui, k -> new long[2]);
                        if (neg) arr[1]++; else arr[0]++;
                    }
                }
            }
        }
        // Emit rows per document
        for (Map.Entry<String, Map<String,long[]>> e : perDoc.entrySet()) {
            String doc = e.getKey();
            for (Map.Entry<String,long[]> c : e.getValue().entrySet()) {
                String cui = c.getKey(); long[] arr = c.getValue();
                long affirmed = arr[0]; long negated = arr[1];
                if (affirmed > 0) rows.add(Arrays.asList(doc, cui, "false", String.valueOf(affirmed)));
                if (negated > 0) rows.add(Arrays.asList(doc, cui, "true", String.valueOf(negated)));
            }
        }
        return rows;
    }

    // Consolidated unique CUI list with counts (derived from CuiCounts sheet)
    private static List<List<String>> aggregateCuiTotals(List<List<String>> cuiCounts) {
        List<List<String>> rows = new ArrayList<>();
        rows.add(Arrays.asList("CUI","Total Count","Affirmed Count","Negated Count"));
        if (cuiCounts == null || cuiCounts.size() <= 1) return rows;
        Map<String,long[]> map = new LinkedHashMap<>(); // CUI -> [total, affirmed, negated]
        for (int i=1;i<cuiCounts.size();i++) {
            List<String> r = cuiCounts.get(i);
            if (r.size() < 4) continue;
            String cui = nvl(r.get(1));
            String negStr = nvl(r.get(2));
            String cntStr = nvl(r.get(3));
            if (cui.isEmpty()) continue;
            long cnt = 0; try { cnt = Long.parseLong(cntStr); } catch (Exception ignore) {}
            boolean neg = "true".equalsIgnoreCase(negStr) || "1".equals(negStr) || negStr.startsWith("-");
            long[] a = map.computeIfAbsent(cui, k -> new long[3]);
            a[0] += cnt; // total
            if (neg) a[2] += cnt; else a[1] += cnt; // negated vs affirmed
        }
        // Emit sorted by total desc
        List<Map.Entry<String,long[]>> es = new ArrayList<>(map.entrySet());
        es.sort((a,b)->Long.compare(b.getValue()[0], a.getValue()[0]));
        for (Map.Entry<String,long[]> e : es) {
            long[] a = e.getValue();
            rows.add(Arrays.asList(e.getKey(), String.valueOf(a[0]), String.valueOf(a[1]), String.valueOf(a[2])));
        }
        return rows;
    }

    // =============== Aggregate Tokens from bsv_tokens ===============
    private static List<List<String>> aggregateTokens(Path outDir) throws IOException {
        List<List<String>> rows = new ArrayList<>();
        rows.add(Arrays.asList("Document","Token","Begin","End"));
        if (outDir == null || !Files.isDirectory(outDir)) return rows;
        List<Path> tokenDirs = new ArrayList<>();
        Path t1 = outDir.resolve("bsv_tokens");
        if (Files.isDirectory(t1)) tokenDirs.add(t1);
        try (DirectoryStream<Path> shards = Files.newDirectoryStream(outDir, p -> Files.isDirectory(p) && p.getFileName().toString().startsWith("shard_"))) {
            for (Path sh : shards) {
                Path td = sh.resolve("bsv_tokens");
                if (Files.isDirectory(td)) tokenDirs.add(td);
            }
        } catch (IOException ignore) {}
        int files = 0;
        for (Path dir : tokenDirs) {
            try (DirectoryStream<Path> ds = Files.newDirectoryStream(dir, p -> p.toString().toLowerCase(java.util.Locale.ROOT).endsWith(".bsv"))) {
                for (Path p : ds) {
                    files++;
                    List<String> lines = Files.readAllLines(p, StandardCharsets.UTF_8);
                    if (lines.isEmpty()) continue;
                    String header = lines.get(0);
                    String[] h = header.split("\\|", -1);
                    int idxTok = -1, idxSpan = -1;
                    for (int i=0;i<h.length;i++) {
                        String col = h[i].trim();
                        if (col.equalsIgnoreCase("Token Text")) idxTok = i;
                        else if (col.equalsIgnoreCase("Text Span")) idxSpan = i;
                    }
                    for (int i=1;i<lines.size();i++) {
                        String line = lines.get(i).trim();
                        if (line.isEmpty()) continue;
                        String[] cells = line.split("\\|", -1);
                        String tok = (idxTok>=0 && idxTok<cells.length) ? cells[idxTok].trim() : (cells.length>0?cells[0].trim():"");
                        String span = (idxSpan>=0 && idxSpan<cells.length) ? cells[idxSpan].trim() : (cells.length>1?cells[1].trim():"");
                        String begin = ""; String end = "";
                        String[] be = span.split(",");
                        if (be.length==2) { begin = be[0].trim(); end = be[1].trim(); }
                        rows.add(Arrays.asList(baseName(p), tok, begin, end));
                    }
                }
            }
        }
        if (rows.size()==1) rows.add(Arrays.asList("No data found"));
        return rows;
    }

    // =============== Minimal XLSX writer (with styled header + freeze top row) ===============
    private static void writeWorkbookXlsx(LinkedHashMap<String, List<List<String>>> sheets, Path out) throws IOException {
        Files.createDirectories(out.getParent());
        try (java.util.zip.ZipOutputStream zos = new java.util.zip.ZipOutputStream(Files.newOutputStream(out))) {
            // [Content_Types].xml
            putEntry(zos, "[Content_Types].xml", contentTypesXml(sheets.size()));
            // _rels/.rels
            putEntry(zos, "_rels/.rels", relsRelsXml());
            // xl/workbook.xml
            List<String> names = new ArrayList<>(sheets.keySet());
            putEntry(zos, "xl/workbook.xml", workbookXml(names));
            // xl/_rels/workbook.xml.rels
            putEntry(zos, "xl/_rels/workbook.xml.rels", workbookRelsXml(sheets.size()));
            // xl/styles.xml (basic style: normal + header bold w/ light gray fill)
            putEntry(zos, "xl/styles.xml", stylesXml());
            // xl/worksheets/sheetN.xml
            int idx = 1;
            for (List<List<String>> data : sheets.values()) {
                putEntry(zos, "xl/worksheets/sheet"+idx+".xml", sheetXml(data));
                idx++;
            }
        }
    }
    private static void putEntry(java.util.zip.ZipOutputStream zos, String name, String xml) throws IOException {
        java.util.zip.ZipEntry e = new java.util.zip.ZipEntry(name);
        zos.putNextEntry(e);
        byte[] b = xml.getBytes(StandardCharsets.UTF_8);
        zos.write(b, 0, b.length);
        zos.closeEntry();
    }
    private static String contentTypesXml(int sheetCount) {
        StringBuilder sb = new StringBuilder();
        sb.append("<?xml version=\"1.0\" encoding=\"UTF-8\"?>");
        sb.append("<Types xmlns=\"http://schemas.openxmlformats.org/package/2006/content-types\">");
        sb.append("<Default Extension=\"rels\" ContentType=\"application/vnd.openxmlformats-package.relationships+xml\"/>");
        sb.append("<Default Extension=\"xml\" ContentType=\"application/xml\"/>");
        sb.append("<Override PartName=\"/xl/workbook.xml\" ContentType=\"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml\"/>");
        sb.append("<Override PartName=\"/xl/styles.xml\" ContentType=\"application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml\"/>");
        for (int i=1;i<=sheetCount;i++) {
            sb.append("<Override PartName=\"/xl/worksheets/sheet"+i+".xml\" ContentType=\"application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml\"/>");
        }
        sb.append("</Types>");
        return sb.toString();
    }
    private static String relsRelsXml() {
        return "<?xml version=\"1.0\" encoding=\"UTF-8\"?><Relationships xmlns=\"http://schemas.openxmlformats.org/package/2006/relationships\"><Relationship Id=\"rId1\" Type=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument\" Target=\"xl/workbook.xml\"/></Relationships>";
    }
    private static String workbookXml(List<String> names) {
        StringBuilder sb = new StringBuilder();
        sb.append("<?xml version=\"1.0\" encoding=\"UTF-8\"?>");
        sb.append("<workbook xmlns=\"http://schemas.openxmlformats.org/spreadsheetml/2006/main\" xmlns:r=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships\">");
        sb.append("<sheets>");
        for (int i=0;i<names.size();i++) {
            String n = sanitizeSheetName(names.get(i));
            sb.append("<sheet name=\""+xmlEscape(n)+"\" sheetId=\""+(i+1)+"\" r:id=\"rId"+(i+1)+"\"/>");
        }
        sb.append("</sheets></workbook>");
        return sb.toString();
    }
    private static String workbookRelsXml(int sheetCount) {
        StringBuilder sb = new StringBuilder();
        sb.append("<?xml version=\"1.0\" encoding=\"UTF-8\"?>");
        sb.append("<Relationships xmlns=\"http://schemas.openxmlformats.org/package/2006/relationships\">");
        for (int i=1;i<=sheetCount;i++) {
            sb.append("<Relationship Id=\"rId"+i+"\" Type=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet\" Target=\"worksheets/sheet"+i+".xml\"/>");
        }
        // styles relationship as the last one
        sb.append("<Relationship Id=\"rId"+(sheetCount+1)+"\" Type=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles\" Target=\"styles.xml\"/>");
        sb.append("</Relationships>");
        return sb.toString();
    }
    private static String sheetXml(List<List<String>> data) {
        StringBuilder sb = new StringBuilder();
        sb.append("<?xml version=\"1.0\" encoding=\"UTF-8\"?>");
        sb.append("<worksheet xmlns=\"http://schemas.openxmlformats.org/spreadsheetml/2006/main\">");
        // Freeze top row
        sb.append("<sheetViews><sheetView workbookViewId=\"0\"><pane ySplit=\"1\" topLeftCell=\"A2\" activePane=\"bottomLeft\" state=\"frozen\"/></sheetView></sheetViews>");
        // Approximate auto-fit: compute widths based on max string length per column
        int cols = 0;
        if (data != null) { for (List<String> row : data) cols = Math.max(cols, row.size()); }
        if (cols > 0) {
            int[] maxLen = new int[cols];
            for (int i=0;i<cols;i++) maxLen[i] = 0;
            for (List<String> row : data) {
                for (int c=0;c<row.size();c++) {
                    String v = row.get(c);
                    int len = (v==null) ? 0 : v.length();
                    if (len > maxLen[c]) maxLen[c] = len;
                }
            }
            sb.append("<cols>");
            for (int c=0;c<cols;c++) {
                int w = Math.max(10, Math.min(80, (int)Math.round(maxLen[c]*1.1) + 2));
                sb.append("<col min=\""+(c+1)+"\" max=\""+(c+1)+"\" width=\""+w+"\" customWidth=\"1\"/>");
            }
            sb.append("</cols>");
        }
        sb.append("<sheetData>");
        int headerCols = 0;
        if (data != null) {
            for (int r=0;r<data.size();r++) {
                List<String> row = data.get(r);
                if (r == 0) headerCols = row.size();
                sb.append("<row r=\""+(r+1)+"\">");
                for (int c=0;c<row.size();c++) {
                    String v = row.get(c);
                    String cellRef = colRef(c+1) + (r+1);
                    boolean isHeader = (r == 0);
                    String styleAttr = isHeader ? " s=\"1\"" : ""; // 0=normal, 1=header
                    if (isNumeric(v)) {
                        sb.append("<c r=\""+cellRef+"\" t=\"n\""+styleAttr+"><v>"+xmlEscape(v)+"</v></c>");
                    } else {
                        sb.append("<c r=\""+cellRef+"\" t=\"inlineStr\""+styleAttr+"><is><t>"+xmlEscape(v)+"</t></is></c>");
                    }
                }
                sb.append("</row>");
            }
        }
        sb.append("</sheetData>");
        // AutoFilter on header row if present
        if (headerCols > 0) {
            String lastCol = colRef(headerCols);
            sb.append("<autoFilter ref=\"A1:"+lastCol+"1\"/>");
        }
        sb.append("</worksheet>");
        return sb.toString();
    }
    private static String stylesXml() {
        // Minimal styles: 2 fonts (normal, bold), 2 fills (none, light gray), 2 cellXfs (normal idx0, header idx1)
        StringBuilder sb = new StringBuilder();
        sb.append("<?xml version=\"1.0\" encoding=\"UTF-8\"?>");
        sb.append("<styleSheet xmlns=\"http://schemas.openxmlformats.org/spreadsheetml/2006/main\">");
        sb.append("<fonts count=\"2\">");
        sb.append("<font><sz val=\"11\"/><name val=\"Calibri\"/></font>");
        sb.append("<font><b/><sz val=\"11\"/><name val=\"Calibri\"/></font>");
        sb.append("</fonts>");
        sb.append("<fills count=\"2\">");
        sb.append("<fill><patternFill patternType=\"none\"/></fill>");
        // Header fill (ARGB): dynamic per pipeline if set, else light gray
        String fill = (HEADER_FILL_ARGB==null||HEADER_FILL_ARGB.trim().isEmpty()) ? "FFEFEFEF" : HEADER_FILL_ARGB;
        sb.append("<fill><patternFill patternType=\"solid\"><fgColor rgb=\""+fill+"\"/><bgColor indexed=\"64\"/></patternFill></fill>");
        sb.append("</fills>");
        sb.append("<borders count=\"1\"><border/></borders>");
        sb.append("<cellStyleXfs count=\"1\"><xf numFmtId=\"0\" fontId=\"0\" fillId=\"0\" borderId=\"0\"/></cellStyleXfs>");
        sb.append("<cellXfs count=\"2\">");
        sb.append("<xf numFmtId=\"0\" fontId=\"0\" fillId=\"0\" borderId=\"0\" xfId=\"0\"/>");
        sb.append("<xf numFmtId=\"0\" fontId=\"1\" fillId=\"1\" borderId=\"0\" xfId=\"0\" applyFont=\"1\" applyFill=\"1\"/>");
        sb.append("</cellXfs>");
        sb.append("</styleSheet>");
        return sb.toString();
    }

    // Build aggregated processing metrics across immediate subruns under a compare parent dir
    private static List<List<String>> buildProcessingMetricsAggregateForParent(Path outDir) throws IOException {
        List<List<String>> rows = new ArrayList<>();
        rows.add(Arrays.asList("Phase","AE/Writer","Init Count (sum)","Process Count (sum)","Files Written (sum)"));
        if (outDir == null || !Files.isDirectory(outDir)) return rows;
        Map<String,int[]> agg = new LinkedHashMap<>(); // key label -> [init, process, files]
        try (DirectoryStream<Path> ds = Files.newDirectoryStream(outDir)) {
            for (Path sub : ds) {
                if (!Files.isDirectory(sub)) continue;
                String subName = sub.getFileName().toString();
                String low = subName.toLowerCase(java.util.Locale.ROOT);
                if (low.equals("xmi") || low.equals("bsv_table") || low.equals("csv_table") || low.equals("csv_table_concepts") || low.equals("html_table") ||
                        low.equals("cui_list") || low.equals("cui_count") || low.equals("bsv_tokens") || low.equals("logs") || low.startsWith("pending_") || low.startsWith("shard_")) {
                    continue;
                }
                Path runLog = sub.resolve("run.log");
                if (!Files.isRegularFile(runLog)) {
                    Path logs = sub.resolve("logs");
                    if (Files.isDirectory(logs)) {
                        Path latest = null; long lm = Long.MIN_VALUE;
                        try (DirectoryStream<Path> lds = Files.newDirectoryStream(logs, p -> p.getFileName().toString().endsWith(".log"))) {
                            for (Path p : lds) { long m = p.toFile().lastModified(); if (m > lm) { lm = m; latest = p; } }
                        }
                        if (latest != null) runLog = latest; else runLog = null;
                    } else runLog = null;
                }
                Map<String,int[]> counts = parseAeCountsFromRun(runLog, sub);
                for (Map.Entry<String,int[]> e : counts.entrySet()) {
                    int[] dest = agg.computeIfAbsent(e.getKey(), k -> new int[3]);
                    int[] src = e.getValue();
                    dest[0] += src[0]; dest[1] += src[1]; dest[2] += src[2];
                }
            }
        } catch (IOException ignore) {}
        for (Map.Entry<String,int[]> e : agg.entrySet()) {
            String key = e.getKey();
            int[] c = e.getValue();
            String label = friendlyAeLabel(key);
            String phase = phaseForAeLabel(key, label);
            rows.add(Arrays.asList(phase, label + " ("+key+")", String.valueOf(c[0]), String.valueOf(c[1]), String.valueOf(c[2])));
        }
        if (rows.size() == 1) rows.add(Arrays.asList("No data found"));
        return rows;
    }

    private static Map<String,int[]> parseAeCountsFromRun(Path runLog, Path outDir) throws IOException {
        Map<String,int[]> counts = new LinkedHashMap<>();
        if (runLog != null && Files.isRegularFile(runLog)) {
            List<String> lines = Files.readAllLines(runLog, StandardCharsets.UTF_8);
            for (String line : lines) {
                int idx = line.indexOf(" - ");
                if (idx > 0) {
                    String left = line.substring(0, idx);
                    String name = left.replaceFirst("^.* INFO ", "").trim();
                    if (name.isEmpty()) continue;
                    int[] arr = counts.computeIfAbsent(name, k -> new int[3]);
                    String rest = line.substring(idx+3).toLowerCase(java.util.Locale.ROOT);
                    if (rest.contains("initializing")) arr[0]++;
                    if (rest.contains("process(jcas)") || rest.startsWith("processing") || rest.contains("starting processing") || rest.contains("finished processing")) arr[1]++;
                }
            }
        }
        // Approximate files written by counting files in output folders
        Map<String,Integer> fileTotals = new LinkedHashMap<>();
        fileTotals.put("FileTreeXmiWriter", countFiles(outDir.resolve("xmi"), ".xmi"));
        fileTotals.put("SemanticTableFileWriter", countFiles(outDir.resolve("bsv_table"), ".BSV") + countFiles(outDir.resolve("csv_table"), ".CSV") + countFiles(outDir.resolve("html_table"), ".HTML"));
        fileTotals.put("CuiListFileWriter", countFiles(outDir.resolve("cui_list"), ".bsv"));
        fileTotals.put("CuiCountFileWriter", countFiles(outDir.resolve("cui_count"), ".bsv") + countFiles(outDir, ".cuicount.bsv"));
        fileTotals.put("TokenTableFileWriter", countFiles(outDir.resolve("bsv_tokens"), ".BSV"));
        for (Map.Entry<String,Integer> e : fileTotals.entrySet()) {
            int[] arr = counts.computeIfAbsent(e.getKey(), k -> new int[3]);
            arr[2] += e.getValue();
        }
        return counts;
    }

    // (use existing countFiles overload earlier in class)
    private static String colRef(int idx) {
        StringBuilder sb = new StringBuilder();
        while (idx > 0) { idx--; sb.insert(0, (char)('A' + (idx % 26))); idx /= 26; }
        return sb.toString();
    }
    private static String sanitizeSheetName(String s) {
        if (s == null) return "Sheet";
        String n = s.replaceAll("[\\/:*\\[\\]?]", " ").replace(']', ' ');
        if (n.length() > 31) n = n.substring(0,31);
        if (n.trim().isEmpty()) n = "Sheet";
        return n;
    }
    private static boolean isNumeric(String s) {
        if (s == null) return false; s = s.trim();
        if (s.isEmpty()) return false;
        // simple heuristic: digits, optional leading -, optional dot
        return s.matches("-?\\d+(\\.\\d+)?");
    }
    private static String xmlEscape(String s) {
        if (s == null) return "";
        return s.replace("&","&amp;").replace("<","&lt;").replace(">","&gt;").replace("\"","&quot;");
    }

    private static List<List<String>> parsePiperModules(Path piper) throws IOException {
        List<List<String>> rows = new ArrayList<>();
        rows.add(Arrays.asList("Order","Phase","Step","AE","Label","Line"));
        if (piper == null || !Files.isRegularFile(piper)) return rows;
        List<String> lines = Files.readAllLines(piper, StandardCharsets.UTF_8);
        int order = 0;
        for (String line : lines) {
            String t = line.trim();
            if (t.isEmpty() || t.startsWith("//")) continue;
            if (t.startsWith("load ") || t.startsWith("add ") || t.startsWith("addDescription ") || t.startsWith("addLast ")) {
                String step = t.split("\\s+",2)[0];
                order++;
                String[] parts = t.split("\\s+");
                String ae = parts.length >= 2 ? parts[1] : "";
                String[] norm = normalizeAeAndLabel(step, ae, t);
                String aeNorm = norm[0];
                String label = norm[1];
                String phase = phaseForAeLabel(aeNorm, label);
                rows.add(Arrays.asList(String.valueOf(order), phase, step, aeNorm, label, t));
            }
        }
        return rows;
    }

    private static String friendlyAeLabel(String ae) {
        if (ae == null) return "";
        switch (ae) {
            case "TsDefaultTokenizerPipeline": return "Tokenizer (default)";
            case "TsFullTokenizerPipeline": return "Tokenizer (full/sectioned)";
            case "ContextDependentTokenizerAnnotator": return "Context-Dependent Tokenizer";
            case "POSTagger": return "POS Tagger";
            case "TsChunkerSubPipe": return "Chunker";
            case "TsDictionarySubPipe": return "Dictionary Lookup";
            case "tools.wsd.SimpleWsdDisambiguatorAnnotator": return "WSD (Simple)";
            case "TsAttributeCleartkSubPipe": return "Assertion";
            case "TsTemporalSubPipe": return "Temporal";
            case "TsRelationSubPipe": return "Relation Extraction";
            case "TsCorefSubPipe": return "Coreference";
            case "FileTreeXmiWriter": return "XMI Writer";
            case "SemanticTableFileWriter": return "Semantic Table Writer";
            case "CuiListFileWriter": return "CUI List Writer";
            case "CuiCountFileWriter": return "CUI Count Writer";
            case "TokenTableFileWriter": return "Token Table Writer";
            case "EventTimeAnaforaWriter": return "Anafora Temporal Writer";
            case "util.log.FinishedLogger": return "Finished Logger";
            default:
                if (ae.toLowerCase(Locale.ROOT).contains("writer")) return "Writer";
                if (ae.toLowerCase(Locale.ROOT).contains("temporal")) return "Temporal";
                if (ae.toLowerCase(Locale.ROOT).contains("coref")) return "Coreference";
                if (ae.toLowerCase(Locale.ROOT).contains("relation")) return "Relation Extraction";
                return "";
        }
    }

    private static String[] normalizeAeAndLabel(String step, String ae, String line) {
        String aeOut = ae;
        String label = friendlyAeLabel(ae);
        // Normalize includes/paths like ../../pipelines/includes/Writers_Xmi_Table.piper
        if (ae != null && (ae.contains("/") || ae.contains("\\"))) {
            String base = ae;
            int slash = Math.max(ae.lastIndexOf('/'), ae.lastIndexOf('\\'));
            if (slash >= 0) base = ae.substring(slash+1);
            if (base.endsWith(".piper")) base = base.substring(0, base.length()-6);
            aeOut = base;
            if (base.toLowerCase(Locale.ROOT).contains("writer")) label = "Writers Include";
            else label = "Include: " + base;
        }
        // DefaultSubjectAnnotator missing label
        if ((label == null || label.isEmpty()) && ae != null && ae.endsWith("DefaultSubjectAnnotator")) label = "Default Subject";
        return new String[]{aeOut, label==null?"":label};
    }

    private static String phaseForAeLabel(String ae, String label) {
        String key = (ae==null?"":ae).toLowerCase(Locale.ROOT) + " " + (label==null?"":label).toLowerCase(Locale.ROOT);
        if (key.contains("tokenizer") || key.contains("postagger") || key.contains("chunk")) return "Tokenization";
        if (key.contains("dictionary")) return "Dictionary";
        if (key.contains("wsd")) return "WSD";
        if (key.contains("assertion") || key.contains("subject")) return "Assertion";
        if (key.contains("writer") || key.contains("include")) return "Writers";
        if (key.contains("temporal")) return "Temporal";
        if (key.contains("relation")) return "Relations";
        if (key.contains("coref")) return "Coreference";
        return "Core";
    }

    private static List<List<String>> buildSheetGuide() {
        List<List<String>> rows = new ArrayList<>();
        rows.add(Arrays.asList("Sheet","Column","Meaning","Module/Source"));
        // Clinical Concepts (consolidated)
        rows.add(Arrays.asList("Clinical Concepts","Document/Begin/End","Document id and character offsets","Tokenizer + cTAKES mention"));
        rows.add(Arrays.asList("Clinical Concepts","Section","Detected section (or SIMPLE_SEGMENT)","Sectionizer (if present)"));
        rows.add(Arrays.asList("Clinical Concepts","Semantic Group","UMLS semantic group of best concept","Dictionary + WSD (TUI fallback if missing)"));
        rows.add(Arrays.asList("Clinical Concepts","Semantic Type","UMLS semantic type (TUI label)","Dictionary + WSD (TUI fallback if missing)"));
        rows.add(Arrays.asList("Clinical Concepts","Type","Mention type (SignSymptom, Medication, etc.)","Core pipeline"));
        rows.add(Arrays.asList("Clinical Concepts","Polarity/Confidence/Negated/Uncertain/Conditional/Generic/Subject/HistoryOf","Assertion and confidence","Assertion modules / WSD"));
        rows.add(Arrays.asList("Clinical Concepts","CandidateCount/Disambiguated","OntologyConcept candidates retained and chosen flag","Dictionary + WSD"));
        rows.add(Arrays.asList("Clinical Concepts","CUI/TUI/PreferredText/CodingScheme/ConceptScore","Chosen concept details and score","Dictionary + WSD (PreferredText falls back to mention text if missing)"));
        rows.add(Arrays.asList("Clinical Concepts","Candidates","All candidate concepts (CUI:TUI:PreferredText; …)","Dictionary + WSD"));
        rows.add(Arrays.asList("Clinical Concepts","SemanticsFallback","true when Semantic Group/Type came from TUI fallback","Dictionary + WSD"));
        rows.add(Arrays.asList("Clinical Concepts","PrefTextFallback","true when PreferredText came from mention text fallback","Dictionary + WSD"));
        rows.add(Arrays.asList("Clinical Concepts","DocTimeRel","Event temporal relation to document time","Temporal (TsTemporalSubPipe)"));
        rows.add(Arrays.asList("Clinical Concepts","DegreeOf","Degree-of relation present for the concept","Relations (TsRelationSubPipe)"));
        rows.add(Arrays.asList("Clinical Concepts","LocationOfText","Location-of partner text (first partner)","Relations (TsRelationSubPipe)"));
        rows.add(Arrays.asList("Clinical Concepts","Coref","Mention participates in a coreference chain","Coreference (TsCorefSubPipe)"));
        rows.add(Arrays.asList("Clinical Concepts","Text","Covered mention text","Tokenizer + cTAKES mention"));
        // CuiCounts
        rows.add(Arrays.asList("CuiCounts","CUI","UMLS concept identifier","Dictionary + WSD"));
        rows.add(Arrays.asList("CuiCounts","Negated","Derived from leading '-' in count files","CuiCountFileWriter"));
        rows.add(Arrays.asList("CuiCounts","Count","Occurrences in document","CuiCountFileWriter"));
        // CuiList
        rows.add(Arrays.asList("CuiList","Columns","Basic concept listing per mention","CuiListFileWriter"));
        // Tokens
        rows.add(Arrays.asList("Tokens","Token Text/Text Span","Token text and offsets","Tokenizer"));
        // RunInfo
        rows.add(Arrays.asList("Overview","Build/Time fields","Pipeline timing summary","cTAKES log"));
        rows.add(Arrays.asList("Overview","Pipeline/Dictionary XML","Configuration used","Runner + piper file"));
        return rows;
    }

    // =============== Pipeline Map (Pipeline + Column Mapping) ===============
    private static List<List<String>> buildPipelineMap(List<List<String>> modules, List<List<String>> guide) {
        List<List<String>> rows = new ArrayList<>();
        // Header row
        rows.add(Arrays.asList("Pipeline Map","","","",""));
        // Pipeline section
        rows.add(Arrays.asList("Pipeline (Order/Phase/Step/AE/Label)","","","",""));
        rows.add(Arrays.asList("Order","Phase","Step","AE","Label"));
        if (modules != null && modules.size() > 1) {
            for (int i = 1; i < modules.size(); i++) {
                List<String> m = modules.get(i);
                // modules: Order | Phase | Step | AE | Label | (Line omitted for clinician clarity)
                String order = m.size()>0? m.get(0):"";
                String phase = m.size()>1? m.get(1):"";
                String step  = m.size()>2? m.get(2):"";
                String ae    = m.size()>3? m.get(3):"";
                String label = m.size()>4? m.get(4):"";
                rows.add(Arrays.asList(order, phase, step, ae, label));
            }
        }
        rows.add(Arrays.asList("","","","",""));
        // Embedded Interpretation Guide just below pipeline order
        rows.add(Arrays.asList("Interpretation Guide","","","",""));
        rows.add(Arrays.asList("AE","What it does","Example","Columns in report",""));
        try {
            List<List<String>> guideRowsTop = buildClinicianGuide(null, modules);
            for (int i = 1; i < guideRowsTop.size(); i++) {
                List<String> r = guideRowsTop.get(i);
                String ae = r.size()>0 ? r.get(0) : "";
                String what = r.size()>1 ? r.get(1) : "";
                String ex = r.size()>2 ? r.get(2) : "";
                String cols = r.size()>3 ? r.get(3) : "";
                rows.add(Arrays.asList(ae, what, ex, cols, ""));
            }
        } catch (Exception ignore) {}
        rows.add(Arrays.asList("","","","",""));
        // Column mapping section for Clinical Concepts
        rows.add(Arrays.asList("Clinical Concepts Column Mapping","","","",""));
        rows.add(Arrays.asList("Column","Source Color","Meaning","Module/Source",""));
        if (guide != null && guide.size() > 1) {
            for (int i = 1; i < guide.size(); i++) {
                List<String> g = guide.get(i);
                if (g.size() < 4) continue;
                String sheet = g.get(0);
                if (!"Clinical Concepts".equalsIgnoreCase(sheet) && !"Mentions".equalsIgnoreCase(sheet)) continue;
                String col = g.get(1), meaning = g.get(2), source = g.get(3);
                String color = sourceToColor(source);
                rows.add(Arrays.asList(col, color, meaning, source, ""));
            }
        }
        rows.add(Arrays.asList("","","","",""));
        // Color legend
        rows.add(Arrays.asList("Color Legend","","","",""));
        rows.add(Arrays.asList("Color","Columns/Scope","AE/Module","",""));
        rows.add(Arrays.asList("Dictionary (Yellow)","CUI/TUI/PreferredText/CodingScheme/Semantic Group/Type","TsDictionarySubPipe","",""));
        rows.add(Arrays.asList("WSD (Light Blue)","Confidence/ConceptScore/Disambiguated/CandidateCount/Candidates","tools.wsd.SimpleWsdDisambiguatorAnnotator","",""));
        rows.add(Arrays.asList("Assertion (Light Red)","Polarity/Negated/Uncertain/Conditional/Generic/Subject/HistoryOf","TsAttributeCleartkSubPipe","",""));
        rows.add(Arrays.asList("Temporal (Orange)","DocTimeRel","TsTemporalSubPipe","",""));
        rows.add(Arrays.asList("Relations (Teal)","DegreeOf/LocationOfText","TsRelationSubPipe","",""));
        rows.add(Arrays.asList("Coref (Gray)","Coref","TsCorefSubPipe","",""));
        rows.add(Arrays.asList("Token/Span (Light Green)","Document/Begin/End/Text","TsDefaultTokenizerPipeline (+ ContextDependent)","",""));
        rows.add(Arrays.asList("Meta/Section (Violet)","Section","Sectionizer (if present)","",""));

        rows.add(Arrays.asList("","","","",""));
        // Assertion meanings
        rows.add(Arrays.asList("Assertion Meanings","","","",""));
        rows.add(Arrays.asList("Field","Meaning","Example","",""));
        rows.add(Arrays.asList("Polarity","-1 negated, 1 affirmed","\"no chest pain\" → -1","",""));
        rows.add(Arrays.asList("Negated","true if Polarity < 0","\"no fever\" → true","",""));
        rows.add(Arrays.asList("Uncertain","hedged/uncertain","\"possible pneumonia\" → true","",""));
        rows.add(Arrays.asList("Conditional","conditional clause","\"if pain worsens\" → true","",""));
        rows.add(Arrays.asList("Generic","general, not patient-specific","\"aspirin can cause bleeding\" → true","",""));
        rows.add(Arrays.asList("Subject","who it refers to","\"family history of diabetes\" → family_member","",""));
        rows.add(Arrays.asList("HistoryOf","1 = history context","\"history of MI\" → 1","",""));

        rows.add(Arrays.asList("","","","",""));
        // WSD scoring explainer
        rows.add(Arrays.asList("WSD Scoring","","","",""));
        rows.add(Arrays.asList("Step","Explanation","","",""));
        rows.add(Arrays.asList("Candidates","Dictionary finds all CUI/TUI/PreferredText options","","",""));
        rows.add(Arrays.asList("Context","Covering sentence tokens (fallback: mention text)","","",""));
        rows.add(Arrays.asList("Tokenization","Lowercase; split on non-alphanumerics; tokens len ≥ 1; filters single-letter stop words (a,i)","","",""));
        rows.add(Arrays.asList("Score","|context∩candidate| / max(1, |candidate|)","","",""));
        rows.add(Arrays.asList("Confidence","Set equal to Score (mention-level)","","",""));
        rows.add(Arrays.asList("Tie-break","Longer PreferredText wins; else first seen","","",""));
        rows.add(Arrays.asList("Output","Chosen CUI/TUI/PreferredText/CodingScheme; Disambiguated=true","","",""));
        rows.add(Arrays.asList("","","","",""));
        // Examples table header (avoid duplicate section header + table header)
        rows.add(Arrays.asList("Example","Explanation","","",""));
        rows.add(Arrays.asList("A","note1: ‘aspirin’; both candidates match; Score=1.0; tie→T109 chosen","","",""));
        rows.add(Arrays.asList("B","note2: ‘chest pain’; ‘chest pain’=1.0 & ‘pain’=1.0; tie→longer label wins","","",""));

        return rows;
    }

    private static String sourceToColor(String source) {
        if (source == null) return "";
        String s = source.toLowerCase(Locale.ROOT);
        if (s.contains("dictionary")) return "Dictionary (Yellow)";
        if (s.contains("wsd")) return "WSD (Light Blue)";
        if (s.contains("assertion")) return "Assertion (Light Red)";
        if (s.contains("temporal")) return "Temporal (Orange)";
        if (s.contains("relation")) return "Relations (Teal)";
        if (s.contains("coref")) return "Coref (Gray)";
        if (s.contains("token") || s.contains("tokenizer")) return "Token/Span (Light Green)";
        if (s.contains("section")) return "Meta/Section (Violet)";
        return "";
    }

    private static String baseName(Path p) {
        String n = p.getFileName().toString();
        int dot = n.indexOf('.');
        return dot>0 ? n.substring(0, dot) : n;
    }

    // NOTE: Legacy Excel 2003 XML writer removed. Codebase now emits XLSX only.

    // Build map: docId -> ("begin,end" -> [section, semGroup, semType])
    private static Map<String, Map<String, String[]>> buildBsvSpanMap(Path bsvDir) throws IOException {
        Map<String, Map<String, String[]>> byDoc = new HashMap<>();
        if (bsvDir == null || !Files.isDirectory(bsvDir)) return byDoc;
        try (DirectoryStream<Path> ds = Files.newDirectoryStream(bsvDir, p -> p.toString().endsWith(".BSV"))) {
            for (Path p : ds) {
                List<String> lines = Files.readAllLines(p, StandardCharsets.UTF_8);
                if (lines.isEmpty()) continue;
                String base = baseName(p);
                String doc = base.endsWith("_table") ? base.substring(0, base.length()-6) : base;
                String header = lines.get(0);
                String[] h = header.split("\\|", -1);
                int idxSpan=-1, idxSection=-1, idxSg=-1, idxSt=-1;
                for (int i=0;i<h.length;i++) {
                    String col = h[i].trim();
                    if (col.equalsIgnoreCase("Span")) idxSpan = i;
                    else if (col.equalsIgnoreCase("Section")) idxSection = i;
                    else if (col.equalsIgnoreCase("Semantic Group")) idxSg = i;
                    else if (col.equalsIgnoreCase("Semantic Type")) idxSt = i;
                }
                if (idxSpan < 0) continue;
                Map<String, String[]> bySpan = byDoc.computeIfAbsent(doc, k->new HashMap<>());
                for (int i=1;i<lines.size();i++) {
                    String line = lines.get(i).trim();
                    if (line.isEmpty()) continue;
                    String[] cells = line.split("\\|", -1);
                    if (cells.length <= idxSpan) continue;
                    String span = cells[idxSpan].trim();
                    String section = (idxSection>=0 && idxSection<cells.length) ? cells[idxSection].trim() : "";
                    String sg = (idxSg>=0 && idxSg<cells.length) ? cells[idxSg].trim() : "";
                    String st = (idxSt>=0 && idxSt<cells.length) ? cells[idxSt].trim() : "";
                    // normalize span to 'begin,end'
                    String key = span.replaceAll("\\s+", "");
                    bySpan.putIfAbsent(key, new String[]{section, sg, st});
                }
            }
        }
        return byDoc;
    }

    // =============== MentionsDetails from XMI ===============
    private static List<List<String>> aggregateMentionsDetails(Path xmiDir) throws Exception {
        List<List<String>> rows = new ArrayList<>();
        rows.add(Arrays.asList("Document","Begin","End","Type","Polarity","Confidence","Negated","Uncertain","Conditional","Generic","Subject","HistoryOf","CandidateCount","Disambiguated","CUI","TUI","PreferredText","CodingScheme","ConceptScore","Candidates","Text"));
        if (xmiDir == null || !Files.isDirectory(xmiDir)) return rows;
        try (DirectoryStream<Path> ds = Files.newDirectoryStream(xmiDir, p -> p.toString().endsWith(".xmi") || p.toString().endsWith(".XMI"))) {
            for (Path p : ds) {
                MentionsFromXmi m = parseXmiMentions(p);
                for (MentionRow r : m.rows) {
                    rows.add(Arrays.asList(m.docId,
                            String.valueOf(r.begin), String.valueOf(r.end), r.type,
                            String.valueOf(r.polarity), String.valueOf(r.confidence), String.valueOf(r.negated), String.valueOf(r.uncertain), String.valueOf(r.conditional), String.valueOf(r.generic), r.subject, String.valueOf(r.historyOf),
                            String.valueOf(r.candidateCount), String.valueOf(r.disambiguated),
                            nvl(r.cui), nvl(r.tui), nvl(r.pref), nvl(r.scheme), String.valueOf(r.conceptScore), nvl(r.candidatesJoined), nvl(r.text)));
                }
            }
        }
        return rows;
    }

    // =============== MentionsFull: XMI details + Section/Semantics from BSV join ===============
    private static List<List<String>> aggregateMentionsFull(Path xmiDir, Path bsvDir) throws Exception {
        List<List<String>> rows = new ArrayList<>();
        // Grouped and ordered by color coding:
        // Green (Token): Document, Begin, End, Text
        // Violet (Meta): Section
        // Yellow (Dictionary): Semantic Group, Semantic Type, CUI, TUI, PreferredText, CodingScheme
        // Blue (WSD): CandidateCount, Candidates, Confidence, ConceptScore, Disambiguated
        // Red (Assertion): Polarity, Negated, Uncertain, Conditional, Generic, Subject, HistoryOf
        rows.add(Arrays.asList(
                "Document","Begin","End","Text",
                "Section","SmokingStatus",
                "Semantic Group","Semantic Type","SemanticsFallback","CUI","TUI","PreferredText","PrefTextFallback","CodingScheme",
                "CandidateCount","Candidates","Confidence","ConceptScore","Disambiguated",
                "DocTimeRel","DegreeOf","LocationOfText","Coref","CorefChainId","CorefRepText",
                "Polarity","Negated","Uncertain","Conditional","Generic","Subject","HistoryOf"
        ));
        if (xmiDir == null || !Files.isDirectory(xmiDir)) return rows;
        Map<String,String> smokingByDoc = detectSmokingStatus(xmiDir);
        int xmiCount = 0;
        try (DirectoryStream<Path> ds = Files.newDirectoryStream(xmiDir, p -> p.toString().endsWith(".xmi") || p.toString().endsWith(".XMI"))) {
            for (Path p : ds) {
                xmiCount++;
                if (xmiCount % 1000 == 0) System.out.println("[report]   joining BSV for XMI #" + xmiCount);
                MentionsFromXmi m = parseXmiMentions(p);
                Map<String, String[]> map = loadDocBsvSpanMap(bsvDir, m.docId);
                for (MentionRow r : m.rows) {
                    // Keep only concept-bearing clinical concepts (skip rows with no candidates/CUI)
                    if ((r.candidateCount <= 0) && (nvl(r.cui).isEmpty())) continue;
                    String key = r.begin + "," + r.end;
                    String section = ""; String sg = ""; String st = ""; boolean semFallback = false;
                    String[] info = map.get(key);
                    if (info != null) { section = info[0]; sg = info[1]; st = info[2]; }
                    // Normalize section label for readability
                    if (section != null && section.equalsIgnoreCase("SIMPLE_SEGMENT")) section = "S";
                    if ((sg==null || sg.isEmpty()) || (st==null || st.isEmpty())) {
                        String[] sem = semFromTui(nvl(r.tui));
                        if (sem != null) { if (sg==null||sg.isEmpty()) sg = sem[0]; if (st==null||st.isEmpty()) st = sem[1]; semFallback = true; }
                    }
                    boolean prefFallback = false;
                    String prefOut = nvl(r.pref);
                    if (prefOut.isEmpty()) { prefOut = nvl(r.text); prefFallback = true; }
                    rows.add(Arrays.asList(
                            m.docId,
                            String.valueOf(r.begin), String.valueOf(r.end), nvl(r.text),
                            section, nvl(smokingByDoc.get(m.docId)),
                            sg, st, semFallback?"true":"", nvl(r.cui), nvl(r.tui), prefOut, prefFallback?"true":"", nvl(r.scheme),
                            String.valueOf(r.candidateCount), nvl(r.candidatesJoined), String.valueOf(r.confidence), String.valueOf(r.conceptScore), String.valueOf(r.disambiguated),
                            nvl(r.docTimeRel), String.valueOf(r.degreeOf), nvl(r.locationOfText), String.valueOf(r.coref), nvl(r.corefChainId), nvl(r.corefRepText),
                            String.valueOf(r.polarity), String.valueOf(r.negated), String.valueOf(r.uncertain), String.valueOf(r.conditional), String.valueOf(r.generic), r.subject, String.valueOf(r.historyOf)
                    ));
                }
            }
        }
        if (xmiCount > 0) System.out.println("[report]   joined BSV for total XMI files: " + xmiCount);
        if (rows.size() == 1) rows.add(Arrays.asList("No data found"));
        return rows;
    }

    // Load section/semantic span map for one document from its corresponding BSV file
    private static Map<String, String[]> loadDocBsvSpanMap(Path bsvDir, String docId) throws IOException {
        Map<String, String[]> bySpan = new HashMap<>();
        if (bsvDir == null || !Files.isDirectory(bsvDir) || docId == null || docId.isEmpty()) return bySpan;
        Path p1 = bsvDir.resolve(docId + "_table.BSV");
        Path p2 = bsvDir.resolve(docId + ".BSV");
        Path file = Files.isRegularFile(p1) ? p1 : (Files.isRegularFile(p2) ? p2 : null);
        if (file == null) return bySpan; // best-effort: skip join if not found
        List<String> lines = Files.readAllLines(file, StandardCharsets.UTF_8);
        if (lines.isEmpty()) return bySpan;
        String header = lines.get(0);
        String[] h = header.split("\\|", -1);
        int idxSpan=-1, idxSection=-1, idxSg=-1, idxSt=-1;
        for (int i=0;i<h.length;i++) {
            String col = h[i].trim();
            if (col.equalsIgnoreCase("Span")) idxSpan = i;
            else if (col.equalsIgnoreCase("Section")) idxSection = i;
            else if (col.equalsIgnoreCase("Semantic Group")) idxSg = i;
            else if (col.equalsIgnoreCase("Semantic Type")) idxSt = i;
        }
        if (idxSpan < 0) return bySpan;
        for (int i=1;i<lines.size();i++) {
            String line = lines.get(i).trim();
            if (line.isEmpty()) continue;
            String[] cells = line.split("\\|", -1);
            if (cells.length <= idxSpan) continue;
            String span = cells[idxSpan].trim();
            String[] be = span.split(",");
            if (be.length != 2) continue;
            String key = be[0].trim()+","+be[1].trim();
            String section = (idxSection>=0 && idxSection<cells.length) ? cells[idxSection].trim() : "";
            String sg = (idxSg>=0 && idxSg<cells.length) ? cells[idxSg].trim() : "";
            String st = (idxSt>=0 && idxSt<cells.length) ? cells[idxSt].trim() : "";
            bySpan.put(key, new String[]{section, sg, st});
        }
        return bySpan;
    }

    // Build a map of Document -> SmokingStatus label (simple heuristic across known namespaces)
    private static Map<String,String> detectSmokingStatus(Path xmiDir) throws IOException {
        Map<String,String> map = new HashMap<>();
        if (xmiDir == null || !Files.isDirectory(xmiDir)) return map;
        try (DirectoryStream<Path> ds = Files.newDirectoryStream(xmiDir, p -> p.toString().endsWith(".xmi") || p.toString().endsWith(".XMI"))) {
            for (Path p : ds) {
                String doc = baseName(p);
                String status = "";
                try {
                    javax.xml.parsers.DocumentBuilderFactory dbf = javax.xml.parsers.DocumentBuilderFactory.newInstance();
                    dbf.setNamespaceAware(true);
                    javax.xml.parsers.DocumentBuilder db = dbf.newDocumentBuilder();
                    org.w3c.dom.Document docXml = db.parse(p.toFile());
                    org.w3c.dom.Element root = docXml.getDocumentElement();
                    String[] smokingNs = new String[]{
                            "http:///org/apache/ctakes/smokingstatus/type.ecore",
                            "http:///org/apache/ctakes/smokingstatus/i2b2/type.ecore",
                            "http:///org/apache/ctakes/smokingstatus/type/libsvm.ecore"
                    };
                    for (String ns : smokingNs) {
                        org.w3c.dom.NodeList all = root.getElementsByTagNameNS(ns, "*");
                        for (int i=0;i<all.getLength();i++) {
                            org.w3c.dom.Element e = (org.w3c.dom.Element) all.item(i);
                            String s = nvl(e.getAttribute("status"));
                            if (s.isEmpty()) s = nvl(e.getAttribute("smokingStatus"));
                            if (s.isEmpty()) s = nvl(e.getAttribute("classification"));
                            if (s.isEmpty()) s = nvl(e.getAttribute("category"));
                            if (s.isEmpty()) s = nvl(e.getAttribute("value"));
                            if (s.isEmpty()) s = e.getLocalName();
                            if (!s.isEmpty()) { status = s; break; }
                        }
                        if (!status.isEmpty()) break;
                    }
                } catch (Exception ignore) {}
                if (!status.isEmpty()) map.put(doc, status);
            }
        }
        return map;
    }

    // =============== Smoking Status (per document) ===============
    private static List<List<String>> buildSmokingStatus(Path xmiDir) throws IOException {
        List<List<String>> rows = new ArrayList<>();
        rows.add(Arrays.asList("Document","Smoking Status","Source"));
        if (xmiDir == null || !Files.isDirectory(xmiDir)) return rows;
        try (DirectoryStream<Path> ds = Files.newDirectoryStream(xmiDir, p -> p.toString().endsWith(".xmi") || p.toString().endsWith(".XMI"))) {
            for (Path p : ds) {
                String doc = baseName(p);
                String status = ""; String source = "";
                try {
                    javax.xml.parsers.DocumentBuilderFactory dbf = javax.xml.parsers.DocumentBuilderFactory.newInstance();
                    dbf.setNamespaceAware(true);
                    javax.xml.parsers.DocumentBuilder db = dbf.newDocumentBuilder();
                    org.w3c.dom.Document docXml = db.parse(p.toFile());
                    org.w3c.dom.Element root = docXml.getDocumentElement();
                    String[] smokingNs = new String[]{
                            "http:///org/apache/ctakes/smokingstatus/type.ecore",
                            "http:///org/apache/ctakes/smokingstatus/i2b2/type.ecore"
                    };
                    for (String ns : smokingNs) {
                        org.w3c.dom.NodeList all = root.getElementsByTagNameNS(ns, "*");
                        for (int i=0;i<all.getLength();i++) {
                            org.w3c.dom.Element e = (org.w3c.dom.Element) all.item(i);
                            // Try common attribute names to capture status/classification
                            String s = nvl(e.getAttribute("status"));
                            if (s.isEmpty()) s = nvl(e.getAttribute("smokingStatus"));
                            if (s.isEmpty()) s = nvl(e.getAttribute("classification"));
                            if (s.isEmpty()) s = nvl(e.getAttribute("category"));
                            if (s.isEmpty()) s = nvl(e.getAttribute("value"));
                            if (s.isEmpty()) s = e.getLocalName();
                            if (!s.isEmpty()) { status = s; source = ns; break; }
                        }
                        if (!status.isEmpty()) break;
                    }
                } catch (Exception ignore) {}
                if (!status.isEmpty()) rows.add(Arrays.asList(doc, status, source));
            }
        }
        if (rows.size() == 1) rows.add(Arrays.asList("No data found","",""));
        return rows;
    }

    private static class MentionsFromXmi {
        String docId = "";
        String sofa = "";
        Map<String, Concept> conceptById = new HashMap<>();
        Map<String, List<String>> fsArrayElems = new HashMap<>();
        List<MentionRow> rows = new ArrayList<>();
        Map<String, String> eventIdToDocTimeRel = new HashMap<>();
        Map<String, MentionSpan> mentionById = new HashMap<>();
        Set<MentionSpan> corefMarkables = new HashSet<>();
    }
    private static class Concept { String cui, tui, pref, scheme; boolean disamb; double score; }
    private static class MentionRow {
        int begin, end; String type; int polarity; double confidence; boolean negated, uncertain, conditional, generic; String subject; int historyOf; int candidateCount; boolean disambiguated; String cui, tui, pref, scheme; double conceptScore; String candidatesJoined; String text; String xmiId;
        String docTimeRel=""; boolean degreeOf=false; String locationOfText=""; boolean coref=false; String corefChainId=""; String corefRepText="";
    }
    private static class MentionSpan {
        int begin, end;
        MentionSpan(int b,int e){begin=b;end=e;}
        public boolean equals(Object o){ if(!(o instanceof MentionSpan)) return false; MentionSpan m=(MentionSpan)o; return begin==m.begin && end==m.end; }
        public int hashCode(){ return begin*31+end; }
    }

    private static MentionsFromXmi parseXmiMentions(Path xmi) throws Exception {
        MentionsFromXmi out = new MentionsFromXmi();
        javax.xml.parsers.DocumentBuilderFactory dbf = javax.xml.parsers.DocumentBuilderFactory.newInstance();
        dbf.setNamespaceAware(true);
        javax.xml.parsers.DocumentBuilder db = dbf.newDocumentBuilder();
        org.w3c.dom.Document doc = db.parse(xmi.toFile());
        org.w3c.dom.Element root = doc.getDocumentElement();

        // Sofa
        org.w3c.dom.NodeList sofas = root.getElementsByTagNameNS("http:///uima/cas.ecore", "Sofa");
        if (sofas.getLength() > 0) {
            org.w3c.dom.Element e = (org.w3c.dom.Element) sofas.item(0);
            out.sofa = optAttr(e, "sofaString");
        }
        // DocumentID
        org.w3c.dom.NodeList docIds = root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/structured.ecore", "DocumentID");
        if (docIds.getLength() > 0) {
            org.w3c.dom.Element e = (org.w3c.dom.Element) docIds.item(0);
            out.docId = optAttr(e, "documentID");
        } else {
            out.docId = baseName(xmi);
        }
        // Concepts
        org.w3c.dom.NodeList umls = root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/refsem.ecore", "UmlsConcept");
        for (int i=0;i<umls.getLength();i++) {
            org.w3c.dom.Element e = (org.w3c.dom.Element) umls.item(i);
            String id = optAttr(e, "xmi:id");
            Concept c = new Concept();
            c.cui = optAttr(e, "cui");
            c.tui = optAttr(e, "tui");
            c.pref = optAttr(e, "preferredText");
            c.scheme = optAttr(e, "codingScheme");
            c.disamb = Boolean.parseBoolean(optAttr(e, "disambiguated"));
            try { c.score = parseDouble(optAttr(e, "score")); } catch (Exception ignore) { c.score = 0.0; }
            out.conceptById.put(id, c);
        }
        // Temporal Event and properties
        org.w3c.dom.NodeList evProps = root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/refsem.ecore", "EventProperties");
        Map<String,String> propsDocTime = new HashMap<>();
        for (int i=0;i<evProps.getLength();i++) {
            org.w3c.dom.Element e = (org.w3c.dom.Element) evProps.item(i);
            String id = optAttr(e, "xmi:id");
            String dtr = optAttr(e, "docTimeRel");
            if (!id.isEmpty()) propsDocTime.put(id, dtr);
        }
        org.w3c.dom.NodeList events = root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/refsem.ecore", "Event");
        for (int i=0;i<events.getLength();i++) {
            org.w3c.dom.Element e = (org.w3c.dom.Element) events.item(i);
            String id = optAttr(e, "xmi:id");
            String props = optAttr(e, "properties");
            if (!id.isEmpty() && !props.isEmpty()) {
                out.eventIdToDocTimeRel.put(id, nvl(propsDocTime.get(props)));
            }
        }
        // FSArray (if present)
        org.w3c.dom.NodeList arrays = root.getElementsByTagNameNS("http:///uima/cas.ecore", "FSArray");
        for (int i=0;i<arrays.getLength();i++) {
            org.w3c.dom.Element e = (org.w3c.dom.Element) arrays.item(i);
            String id = optAttr(e, "xmi:id");
            String members = optAttr(e, "elements").trim();
            if (!members.isEmpty()) {
                out.fsArrayElems.put(id, Arrays.asList(members.split("\\s+")));
            } else {
                out.fsArrayElems.put(id, Collections.emptyList());
            }
        }
        // Mentions: any textsem:*Mention
        org.w3c.dom.NodeList all = root.getChildNodes();
        for (int i=0;i<all.getLength();i++) {
            org.w3c.dom.Node n = all.item(i);
            if (n.getNodeType() != org.w3c.dom.Node.ELEMENT_NODE) continue;
            org.w3c.dom.Element e = (org.w3c.dom.Element) n;
            String ns = e.getNamespaceURI();
            String name = e.getLocalName();
            if (ns != null && ns.endsWith("/textsem.ecore") && name != null && name.endsWith("Mention")) {
                MentionRow r = new MentionRow();
                r.type = name;
                r.begin = parseInt(optAttr(e, "begin"));
                r.end = parseInt(optAttr(e, "end"));
                r.polarity = parseInt(optAttr(e, "polarity"));
                r.confidence = parseDouble(optAttr(e, "confidence"));
                r.negated = r.polarity < 0; // cTAKES: -1 negative, 1 positive
                r.uncertain = parseInt(optAttr(e, "uncertainty")) != 0;
                r.conditional = Boolean.parseBoolean(optAttr(e, "conditional"));
                r.generic = Boolean.parseBoolean(optAttr(e, "generic"));
                r.subject = nvl(optAttr(e, "subject"));
                r.historyOf = parseInt(optAttr(e, "historyOf"));
                r.xmiId = optAttr(e, "xmi:id");
                // concepts
                String arr = optAttr(e, "ontologyConceptArr");
                if (!arr.isEmpty()) {
                    List<String> candIds = new ArrayList<>();
                    // Case 1: inline space-separated ids (common in cTAKES XMI)
                    if (arr.indexOf(' ') >= 0) {
                        String[] ids = arr.trim().split("\\s+");
                        r.candidateCount = ids.length;
                        candIds = Arrays.asList(ids);
                        Concept first = out.conceptById.get(ids[0]);
                        if (first != null) {
                            r.cui = first.cui; r.tui = first.tui; r.pref = first.pref; r.scheme = first.scheme; r.disambiguated = first.disamb; r.conceptScore = first.score;
                        }
                    } else {
                        // Case 2: single id, either concept or FSArray reference
                        Concept best = out.conceptById.get(arr);
                        if (best != null) {
                            r.candidateCount = 1;
                            r.cui = best.cui; r.tui = best.tui; r.pref = best.pref; r.scheme = best.scheme; r.disambiguated = best.disamb; r.conceptScore = best.score;
                            candIds = Collections.singletonList(arr);
                        } else {
                            List<String> ids = out.fsArrayElems.get(arr);
                            if (ids != null && !ids.isEmpty()) {
                                r.candidateCount = ids.size();
                                candIds = ids;
                                Concept first = out.conceptById.get(ids.get(0));
                                if (first != null) {
                                    r.cui = first.cui; r.tui = first.tui; r.pref = first.pref; r.scheme = first.scheme; r.disambiguated = first.disamb; r.conceptScore = first.score;
                                }
                            }
                        }
                    }
                    if (!candIds.isEmpty()) {
                        List<String> cStrs = new ArrayList<>(candIds.size());
                        for (String id : candIds) {
                            Concept c = out.conceptById.get(id);
                            if (c != null) {
                                // Fallbacks to make Candidates more complete/clinician-friendly
                                String cui = nvl(c.cui).isEmpty() ? (nvl(r.cui).isEmpty()?"?":r.cui) : c.cui;
                                String tui = nvl(c.tui).isEmpty() ? (nvl(r.tui).isEmpty()?"?":r.tui) : c.tui;
                                String pref = nvl(c.pref);
                                if (pref.isEmpty()) pref = nvl(r.pref);
                                if (pref.isEmpty()) pref = nvl(r.text); // final fallback: mention text
                                if (pref.isEmpty()) pref = "?";
                                String cs = cui + ":" + tui + ":" + pref;
                                cStrs.add(cs);
                            }
                        }
                        r.candidatesJoined = String.join("; ", cStrs);
                    }
                }
                r.text = safeSub(out.sofa, r.begin, r.end);
                out.rows.add(r);
                if (!nvl(r.xmiId).isEmpty()) out.mentionById.put(r.xmiId, new MentionSpan(r.begin, r.end));
                // Temporal: attach DocTimeRel if Event link present on mention
                String ev = optAttr(e, "event");
                if (!ev.isEmpty()) {
                    String dtr = out.eventIdToDocTimeRel.get(ev);
                    if (dtr != null) r.docTimeRel = dtr;
                }
            }
        }
        // Coref Markables: mark spans present
        org.w3c.dom.NodeList marks = root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/textsem.ecore", "Markable");
        for (int i=0;i<marks.getLength();i++) {
            org.w3c.dom.Element e = (org.w3c.dom.Element) marks.item(i);
            int b = parseInt(optAttr(e, "begin"));
            int en = parseInt(optAttr(e, "end"));
            out.corefMarkables.add(new MentionSpan(b,en));
        }
        // Relations: DegreeOf and LocationOf + CoreferenceRelation (build chains)
        Map<String,String> relArgToMention = new HashMap<>();
        org.w3c.dom.NodeList relArgs = root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/relation.ecore", "RelationArgument");
        for (int i=0;i<relArgs.getLength();i++) {
            org.w3c.dom.Element e = (org.w3c.dom.Element) relArgs.item(i);
            String id = optAttr(e, "xmi:id");
            String arg = optAttr(e, "argument");
            if (!id.isEmpty() && !arg.isEmpty()) relArgToMention.put(id, arg);
        }
        // Helper to mark a mention by id
        java.util.function.BiConsumer<String,String> markLocation = (mIdPartner, holderId) -> {
            // set location text on the holder mention if we can
            if (mIdPartner == null || mIdPartner.isEmpty()) return;
            MentionSpan sp = out.mentionById.get(mIdPartner);
            if (sp == null) return;
            String txt = safeSub(out.sofa, sp.begin, sp.end);
            for (MentionRow mr : out.rows) {
                if (holderId != null && holderId.equals(mr.xmiId)) {
                    mr.locationOfText = txt; break;
                }
            }
        };
        // Disjoint set for coref chains
        Map<String,String> parent = new HashMap<>();
        java.util.function.Function<String,String> find = new java.util.function.Function<String,String>(){
            public String apply(String x){
                String p = parent.getOrDefault(x, x);
                if (!p.equals(x)) parent.put(x, this.apply(p));
                else parent.putIfAbsent(x, x);
                return parent.get(x);
            }
        };
        java.util.function.BiConsumer<String,String> union = (a,b) -> {
            String ra = find.apply(a); String rb = find.apply(b);
            if (!ra.equals(rb)) parent.put(ra, rb);
        };
        // DegreeOfTextRelation
        org.w3c.dom.NodeList degs = root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/relation.ecore", "DegreeOfTextRelation");
        for (int i=0;i<degs.getLength();i++) {
            org.w3c.dom.Element e = (org.w3c.dom.Element) degs.item(i);
            String a1 = optAttr(e, "arg1"); String a2 = optAttr(e, "arg2");
            String m1 = relArgToMention.get(a1); String m2 = relArgToMention.get(a2);
            for (MentionRow mr : out.rows) {
                if (mr.xmiId != null && (mr.xmiId.equals(m1) || mr.xmiId.equals(m2))) mr.degreeOf = true;
            }
        }
        // LocationOfTextRelation
        org.w3c.dom.NodeList locs = root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/relation.ecore", "LocationOfTextRelation");
        for (int i=0;i<locs.getLength();i++) {
            org.w3c.dom.Element e = (org.w3c.dom.Element) locs.item(i);
            String a1 = optAttr(e, "arg1"); String a2 = optAttr(e, "arg2");
            String m1 = relArgToMention.get(a1); String m2 = relArgToMention.get(a2);
            if (m1==null || m2==null) continue;
            // For holder, we pick the clinical concept (arg1) and partner text as arg2
            markLocation.accept(m2, m1);
        }
        // CoreferenceRelation: union mentions into chains
        org.w3c.dom.NodeList corefs = root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/relation.ecore", "CoreferenceRelation");
        for (int i=0;i<corefs.getLength();i++) {
            org.w3c.dom.Element e = (org.w3c.dom.Element) corefs.item(i);
            String a1 = optAttr(e, "arg1"); String a2 = optAttr(e, "arg2");
            String m1 = relArgToMention.get(a1); String m2 = relArgToMention.get(a2);
            if (m1!=null && m2!=null) union.accept(m1, m2);
        }
        // Assign chain ids and representative text per chain
        Map<String,java.util.List<MentionRow>> chainMap = new HashMap<>();
        for (MentionRow mr : out.rows) {
            if (mr.xmiId == null) continue;
            String rootId = find.apply(mr.xmiId);
            if (!rootId.equals(mr.xmiId)) {
                mr.coref = true;
                chainMap.computeIfAbsent(rootId, k -> new java.util.ArrayList<>()).add(mr);
            }
        }
        int cseq = 1;
        for (Map.Entry<String,java.util.List<MentionRow>> e : chainMap.entrySet()) {
            String chainId = "C" + (cseq++);
            java.util.List<MentionRow> list = e.getValue();
            // pick representative as the earliest mention by begin
            MentionRow rep = list.stream().min(java.util.Comparator.comparingInt(m -> m.begin)).orElse(null);
            String repText = rep != null ? rep.text : "";
            for (MentionRow mr : list) { mr.corefChainId = chainId; mr.corefRepText = repText; }
        }
        // Attach coref flag by span overlap
        for (MentionRow mr : out.rows) {
            if (out.corefMarkables.contains(new MentionSpan(mr.begin, mr.end))) mr.coref = true;
        }
        return out;
    }

    private static int parseInt(String s) { try { return Integer.parseInt(s); } catch (Exception e) { return 0; } }
    private static double parseDouble(String s) { try { return Double.parseDouble(s); } catch (Exception e) { return 0.0; } }
    private static String safeSub(String s, int b, int e) {
        if (s == null) return "";
        if (b < 0 || e > s.length() || b > e) return "";
        return s.substring(b, e);
    }
    private static String optAttr(org.w3c.dom.Element e, String a) { String v = e.getAttribute(a); return v == null ? "" : v; }
    private static String nvl(String s) { return s == null ? "" : s; }
// Minimal TUI -> (Semantic Group, Semantic Type) mapping as a safety net when BSV semantics are missing
private static String[] semFromTui(String tui) {
    if (tui == null || tui.isEmpty()) return null;
    switch (tui) {
        case "T184": return new String[]{"Finding","Sign or Symptom"};
        case "T109": return new String[]{"Chemicals & Drugs","Organic Chemical"};
        case "T121": return new String[]{"Chemicals & Drugs","Pharmacologic Substance"};
        case "T033": return new String[]{"Anatomy","Body Location or Region"};
        case "T029": return new String[]{"Anatomy","Body Location or Region"};
        case "T201": return new String[]{"Attribute","Clinical Attribute"};
        default: return null;
    }
}

    // =============== Mentions post-processing ===============
    private static List<List<String>> normalizeMentionsCui(List<List<String>> mentions) {
        if (mentions == null || mentions.size() <= 1) return mentions;
        // Find CUI column by header name
        List<String> header = mentions.get(0);
        int idx = -1;
        for (int i = 0; i < header.size(); i++) if ("CUI".equalsIgnoreCase(header.get(i))) { idx = i; break; }
        if (idx < 0) return mentions;
        for (int r = 1; r < mentions.size(); r++) {
            List<String> row = mentions.get(r);
            if (row.size() <= idx) continue;
            String cui = row.get(idx);
            if (cui != null) {
                String cleaned = cui.replaceAll("(?i)-c$", "").trim();
                if ("null".equalsIgnoreCase(cleaned)) cleaned = "";
                row.set(idx, cleaned);
            }
        }
        return mentions;
    }

    // =============== Summary Sheet ===============
    private static List<List<String>> buildSummary(Path outDir, List<List<String>> mentionsFull, List<List<String>> mentionsBsv, List<List<String>> runInfo, List<List<String>> modules, Path runLog) throws IOException {
        List<List<String>> rows = new ArrayList<>();
        // Run info block first, if available
        if (runInfo != null && runInfo.size() > 1) {
            rows.add(Arrays.asList("Run Info","Value"));
            for (int i = 1; i < runInfo.size(); i++) rows.add(new ArrayList<>(runInfo.get(i)));
            rows.add(Arrays.asList("",""));
        }
        // Metrics
        rows.add(Arrays.asList("Metric","Value"));

        // Documents found from XMI dir
        int docCount = 0;
        try { docCount = countXmiDocsRecursive(outDir); } catch (Exception ignored) {}
        rows.add(Arrays.asList("Documents Found", String.valueOf(docCount)));

        // MentionsFull-based metrics
        if (mentionsFull != null && mentionsFull.size() > 1) {
            List<String> hdr = mentionsFull.get(0);
            int idxDoc = indexOf(hdr, "Document");
            int idxConf = indexOf(hdr, "Confidence");
            int idxNeg = indexOf(hdr, "Negated");
            int idxUnc = indexOf(hdr, "Uncertain");
            int idxGen = indexOf(hdr, "Generic");
            int idxCand = indexOf(hdr, "CandidateCount");
            int idxCui = indexOf(hdr, "CUI");
            int idxDtr = indexOf(hdr, "DocTimeRel");
            int idxDeg = indexOf(hdr, "DegreeOf");
            int idxCoref = indexOf(hdr, "Coref");
            Set<String> docsWithMentions = new HashSet<>();
            int total = 0, negCount = 0, uncCount = 0, genCount = 0; double confSum = 0.0; long confNum = 0; long candSum = 0; int nonEmptyCui = 0; int dtrCount = 0; int degCount = 0; int corefCount = 0;
            Set<String> distinctCuis = new HashSet<>();
            for (int i = 1; i < mentionsFull.size(); i++) {
                List<String> r = mentionsFull.get(i);
                if (idxCui < 0 || idxDoc < 0) continue;
                total++;
                docsWithMentions.add(r.get(idxDoc));
                String neg = (idxNeg>=0 && r.size()>idxNeg) ? r.get(idxNeg) : "";
                String unc = (idxUnc>=0 && r.size()>idxUnc) ? r.get(idxUnc) : "";
                String gen = (idxGen>=0 && r.size()>idxGen) ? r.get(idxGen) : "";
                if ("true".equalsIgnoreCase(neg)) negCount++;
                if ("true".equalsIgnoreCase(unc)) uncCount++;
                if ("true".equalsIgnoreCase(gen)) genCount++;
                try { if (idxConf>=0 && r.size()>idxConf) { double c = Double.parseDouble(r.get(idxConf)); confSum += c; confNum++; } } catch (Exception ignore) {}
                try { if (idxCand>=0 && r.size()>idxCand) { candSum += Long.parseLong(r.get(idxCand)); } } catch (Exception ignore) {}
                if (idxDtr>=0 && r.size()>idxDtr && !nvl(r.get(idxDtr)).isEmpty()) dtrCount++;
                if (idxDeg>=0 && r.size()>idxDeg && "true".equalsIgnoreCase(r.get(idxDeg))) degCount++;
                if (idxCoref>=0 && r.size()>idxCoref && "true".equalsIgnoreCase(r.get(idxCoref))) corefCount++;
                String cui = r.get(idxCui);
                if (cui != null && !cui.isEmpty()) { distinctCuis.add(cui); nonEmptyCui++; }
            }
            rows.add(Arrays.asList("Documents With Clinical Concepts", String.valueOf(docsWithMentions.size())));
            rows.add(Arrays.asList("Clinical Concepts Total", String.valueOf(total)));
            rows.add(Arrays.asList("Clinical Concepts With CUI", String.valueOf(nonEmptyCui)));
            rows.add(Arrays.asList("Distinct CUIs", String.valueOf(distinctCuis.size())));
            rows.add(Arrays.asList("Negated Clinical Concepts", String.valueOf(negCount)));
            rows.add(Arrays.asList("Uncertain Clinical Concepts", String.valueOf(uncCount)));
            rows.add(Arrays.asList("Generic Clinical Concepts", String.valueOf(genCount)));
            rows.add(Arrays.asList("Average Confidence", confNum>0 ? String.format(Locale.ROOT, "%.3f", confSum/confNum) : "0.000"));
            rows.add(Arrays.asList("Average CandidateCount", total>0 ? String.format(Locale.ROOT, "%.2f", (double)candSum/total) : "0.00"));
            rows.add(Arrays.asList("Concepts With DocTimeRel", String.valueOf(dtrCount)));
            rows.add(Arrays.asList("Concepts With DegreeOf", String.valueOf(degCount)));
            rows.add(Arrays.asList("Concepts In Coref", String.valueOf(corefCount)));
            // Per-note durations from log
            if (runLog != null && java.nio.file.Files.isRegularFile(runLog)) {
                List<DocTiming> dts = parseDocTimings(runLog);
                if (!dts.isEmpty()) {
                    long sum = 0, min = Long.MAX_VALUE, max = Long.MIN_VALUE;
                    for (DocTiming dt : dts) {
                        long dur = Math.max(0, dt.endMs - dt.startMs);
                        sum += dur; if (dur < min) min = dur; if (dur > max) max = dur;
                    }
                    double avgSec = (sum/1000.0) / dts.size();
                    rows.add(Arrays.asList("Average Per-Note Duration (s)", String.format(Locale.ROOT, "%.2f", avgSec)));
                    rows.add(Arrays.asList("Min/Max Per-Note Duration (s)", String.format(Locale.ROOT, "%.2f / %.2f", min/1000.0, max/1000.0)));
                }
            }
            // Top concepts by chosen CUI
            rows.add(Arrays.asList("",""));
            rows.add(Arrays.asList("Top Concepts (Chosen by WSD)", "Count"));
            Map<String,Integer> top = new HashMap<>();
            Map<String,String> cuiToPref = new HashMap<>();
            int idxPref = indexOf(hdr, "PreferredText");
            int idxText = indexOf(hdr, "Text");
            for (int i = 1; i < mentionsFull.size(); i++) {
                List<String> r = mentionsFull.get(i);
                if (idxCui < 0) continue;
                String cui = r.get(idxCui);
                String pref = (idxPref>=0 && r.size()>idxPref) ? r.get(idxPref) : "";
                if (pref == null || pref.isEmpty()) {
                    String text = (idxText>=0 && r.size()>idxText) ? r.get(idxText) : "";
                    if (text != null && !text.isEmpty()) pref = text;
                }
                if (cui == null || cui.isEmpty()) continue;
                top.put(cui, top.getOrDefault(cui, 0) + 1);
                if (pref != null && !pref.isEmpty()) cuiToPref.putIfAbsent(cui, pref);
            }
            List<Map.Entry<String,Integer>> es = new ArrayList<>(top.entrySet());
            es.sort((a,b)->Integer.compare(b.getValue(), a.getValue()));
            int limit = Math.min(10, es.size());
            for (int i = 0; i < limit; i++) {
                Map.Entry<String,Integer> e = es.get(i);
                String label = e.getKey() + (cuiToPref.containsKey(e.getKey()) ? (" — " + cuiToPref.get(e.getKey())) : "");
                rows.add(Arrays.asList(label, String.valueOf(e.getValue())));
            }
        }

        // Pipeline AEs used (friendly labels)
        List<String> aeLabels = collectFriendlyAeLabels(modules);
        if (!aeLabels.isEmpty()) {
            rows.add(Arrays.asList("",""));
            rows.add(Arrays.asList("Pipeline AEs Used", String.join(", ", aeLabels)));
        }
        return rows;
    }

    // =============== Fast Summary (no XMI parsing) ===============
    private static List<List<String>> buildSummaryFast(Path outDir, List<List<String>> runInfo, List<List<String>> modules, Path runLog, List<List<String>> cuiCounts) throws IOException {
        List<List<String>> rows = new ArrayList<>();
        // Run info block first, if available
        if (runInfo != null && runInfo.size() > 1) {
            rows.add(Arrays.asList("Run Info","Value"));
            for (int i = 1; i < runInfo.size(); i++) rows.add(new ArrayList<>(runInfo.get(i)));
            rows.add(Arrays.asList("",""));
        }
        // Metrics
        rows.add(Arrays.asList("Metric","Value"));
        int docCount = 0;
        try { docCount = countXmiDocs(outDir.resolve("xmi")); } catch (Exception ignored) {}
        rows.add(Arrays.asList("Documents Found", String.valueOf(docCount)));

        // Derive concept metrics from CuiCounts (Document|CUI|Negated|Count)
        long totalConcepts = 0;
        java.util.Set<String> distinctCuis = new java.util.HashSet<>();
        java.util.Map<String,Long> cuiTotals = new java.util.HashMap<>();
        if (cuiCounts != null && cuiCounts.size() > 1) {
            for (int i=1; i<cuiCounts.size(); i++) {
                List<String> r = cuiCounts.get(i);
                if (r.size() < 4) continue;
                String cui = nvl(r.get(1));
                String cnt = nvl(r.get(3));
                long c = 0; try { c = Long.parseLong(cnt); } catch (Exception ignore) {}
                if (!cui.isEmpty()) {
                    distinctCuis.add(cui);
                    cuiTotals.put(cui, cuiTotals.getOrDefault(cui, 0L) + c);
                }
                totalConcepts += c;
            }
        }
        rows.add(Arrays.asList("Clinical Concepts Total", String.valueOf(totalConcepts)));
        rows.add(Arrays.asList("Clinical Concepts With CUI", String.valueOf(totalConcepts)));
        rows.add(Arrays.asList("Distinct CUIs", String.valueOf(distinctCuis.size())));

        // Per-note durations from log (merge shard logs if needed)
        List<DocTiming> dts = new ArrayList<>();
        if (runLog != null && java.nio.file.Files.isRegularFile(runLog)) {
            dts = parseDocTimings(runLog);
        } else if (outDir != null && java.nio.file.Files.isDirectory(outDir)) {
            try (DirectoryStream<Path> ds = java.nio.file.Files.newDirectoryStream(outDir, p -> java.nio.file.Files.isDirectory(p) && p.getFileName().toString().startsWith("shard_"))) {
                for (Path sh : ds) {
                    Path rl = sh.resolve("run.log");
                    if (java.nio.file.Files.isRegularFile(rl)) dts.addAll(parseDocTimings(rl));
                }
            } catch (IOException ignore) {}
        }
        if (!dts.isEmpty()) {
            long sum = 0, min = Long.MAX_VALUE, max = Long.MIN_VALUE;
            for (DocTiming dt : dts) {
                long dur = Math.max(0, dt.endMs - dt.startMs);
                sum += dur; if (dur < min) min = dur; if (dur > max) max = dur;
            }
            double avgSec = (sum/1000.0) / dts.size();
            rows.add(Arrays.asList("Average Per-Note Duration (s)", String.format(Locale.ROOT, "%.2f", avgSec)));
            rows.add(Arrays.asList("Min/Max Per-Note Duration (s)", String.format(Locale.ROOT, "%.2f / %.2f", min/1000.0, max/1000.0)));
            // Also write a CSV artifact for external consumption
            try { writeTimingCsv(outDir, dts); } catch (Exception ignore) {}
        } else {
            // Timing markers missing: add placeholder lines to avoid blanks
            rows.add(Arrays.asList("Average Per-Note Duration (s)", ""));
            rows.add(Arrays.asList("Min/Max Per-Note Duration (s)", ""));
        }
        
        // Top CUIs by total count
        if (!cuiTotals.isEmpty()) {
            rows.add(Arrays.asList("",""));
            rows.add(Arrays.asList("Top CUIs (by CuiCounts)", "Count"));
            java.util.List<java.util.Map.Entry<String,Long>> es = new java.util.ArrayList<>(cuiTotals.entrySet());
            es.sort((a,b)->Long.compare(b.getValue(), a.getValue()));
            int limit = Math.min(10, es.size());
            for (int i=0;i<limit;i++) {
                java.util.Map.Entry<String,Long> e = es.get(i);
                rows.add(Arrays.asList(e.getKey(), String.valueOf(e.getValue())));
            }
        }

        // Pipeline AEs used (friendly labels)
        List<String> aeLabels = collectFriendlyAeLabels(modules);
        if (!aeLabels.isEmpty()) {
            rows.add(Arrays.asList("",""));
            rows.add(Arrays.asList("Pipeline AEs Used", String.join(", ", aeLabels)));
        }
        return rows;
    }

    private static int countXmiDocsRecursive(Path outDir) throws IOException {
        if (outDir == null || !Files.isDirectory(outDir)) return 0;
        int total = 0;
        Path xmiDir = outDir.resolve("xmi");
        if (Files.isDirectory(xmiDir)) total += countXmiDocs(xmiDir);
        try (DirectoryStream<Path> ds = Files.newDirectoryStream(outDir, p -> Files.isDirectory(p) && p.getFileName().toString().startsWith("shard_"))) {
            for (Path sh : ds) {
                Path sx = sh.resolve("xmi");
                if (Files.isDirectory(sx)) total += countXmiDocs(sx);
            }
        }
        return total;
    }

    private static void writeTimingCsv(Path outDir, java.util.List<DocTiming> dts) throws IOException {
        if (outDir == null || dts == null || dts.isEmpty()) return;
        java.nio.file.Path dir = outDir.resolve("timing_csv");
        java.nio.file.Files.createDirectories(dir);
        java.nio.file.Path csv = dir.resolve("timing.csv");
        try (java.io.BufferedWriter bw = java.nio.file.Files.newBufferedWriter(csv, java.nio.charset.StandardCharsets.UTF_8)) {
            bw.write("Document,StartMillis,EndMillis,DurationMillis,DurationSeconds\n");
            for (DocTiming dt : dts) {
                long dur = Math.max(0L, dt.endMs - dt.startMs);
                String line = String.join(",",
                        escCsv(dt.doc), String.valueOf(dt.startMs), String.valueOf(dt.endMs), String.valueOf(dur), String.format(java.util.Locale.ROOT, "%.3f", dur/1000.0));
                bw.write(line); bw.write("\n");
            }
        }
    }
    private static synchronized void addPipelineTimingRow(Path parentOutDir, String pipeline, int docs, int timedDocs, String avgSec, String initSec, String procSec, String totalSec, String runLog) throws IOException {
        if (parentOutDir == null) return;
        java.nio.file.Path dir = parentOutDir.resolve("timing_csv");
        java.nio.file.Files.createDirectories(dir);
        java.nio.file.Path csv = dir.resolve("pipeline_timing.csv");
        boolean exists = java.nio.file.Files.isRegularFile(csv);
        try (java.io.BufferedWriter bw = java.nio.file.Files.newBufferedWriter(csv, java.nio.charset.StandardCharsets.UTF_8, java.nio.file.StandardOpenOption.CREATE, java.nio.file.StandardOpenOption.APPEND)) {
            if (!exists) {
                bw.write("Pipeline,Documents,TimedDocs,TimingCoverage(%),AvgSecondsPerDoc,InitSeconds,ProcessSeconds,TotalSeconds,RunLog\n");
            }
            double cov = (docs > 0) ? (100.0 * timedDocs / docs) : 0.0;
            String line = String.join(",",
                    escCsv(pipeline), String.valueOf(docs), String.valueOf(timedDocs), String.format(java.util.Locale.ROOT, "%.1f", cov),
                    nvlCsv(avgSec), nvlCsv(initSec), nvlCsv(procSec), nvlCsv(totalSec), escCsv(runLog));
            bw.write(line); bw.write("\n");
        }
    }
    private static String nvlCsv(String s) { return (s==null||s.isEmpty())?"0":s; }
    
    private static void consolidateTimingCsv(Path parentOutDir, Path runDir) throws IOException {
        if (parentOutDir == null || runDir == null) return;
        Path srcCsv = runDir.resolve("timing_csv/timing.csv");
        if (!java.nio.file.Files.isRegularFile(srcCsv)) return;
        
        Path destDir = parentOutDir.resolve("timing_csv");
        java.nio.file.Files.createDirectories(destDir);
        
        // Create a consolidated CSV with all individual document timings
        Path consolidatedCsv = destDir.resolve("all_document_timings.csv");
        boolean exists = java.nio.file.Files.isRegularFile(consolidatedCsv);
        
        String runName = runDir.getFileName().toString();
        try (java.io.BufferedReader br = java.nio.file.Files.newBufferedReader(srcCsv, java.nio.charset.StandardCharsets.UTF_8);
             java.io.BufferedWriter bw = java.nio.file.Files.newBufferedWriter(consolidatedCsv, 
                 java.nio.charset.StandardCharsets.UTF_8, 
                 java.nio.file.StandardOpenOption.CREATE, 
                 java.nio.file.StandardOpenOption.APPEND)) {
            
            String line = br.readLine(); // Read header
            if (!exists && line != null) {
                // Write header with pipeline column
                bw.write("Pipeline," + line);
                bw.newLine();
            }
            
            // Write data lines with pipeline prefix
            while ((line = br.readLine()) != null) {
                if (!line.trim().isEmpty()) {
                    bw.write(runName + "," + line);
                    bw.newLine();
                }
            }
        }
    }
    
    private static String escCsv(String s) {
        if (s == null) return "";
        if (s.contains(",") || s.contains("\"") || s.contains("\n")) return '"' + s.replace("\"","\"\"") + '"';
        return s;
    }

    private static int countXmiDocs(Path xmiDir) throws IOException {
        if (xmiDir == null || !Files.isDirectory(xmiDir)) return 0;
        try (DirectoryStream<Path> ds = Files.newDirectoryStream(xmiDir, p -> p.toString().endsWith(".xmi") || p.toString().endsWith(".XMI"))) {
            int c = 0; for (Path p : ds) c++; return c;
        }
    }

    private static int indexOf(List<String> header, String name) {
        for (int i = 0; i < header.size(); i++) if (name.equalsIgnoreCase(header.get(i))) return i;
        return -1;
    }

    // =============== Run log parsing for doc timings ===============
    private static class DocTiming { String doc; long startMs; long endMs; }
    private static List<DocTiming> parseDocTimings(Path runLog) {
        List<DocTiming> out = new ArrayList<>();
        if (runLog == null || !Files.isRegularFile(runLog)) return out;
        Map<String, DocTiming> byDoc = new LinkedHashMap<>();
        java.util.regex.Pattern tsPat = java.util.regex.Pattern.compile("^(\\d{1,2} \\w{3} \\d{4} \\d{2}:\\d{2}:\\d{2})\\s+INFO\\s+([^\n]+)$");
        java.util.regex.Pattern readPat = java.util.regex.Pattern.compile("Reading .*[/\\\\]([^/\\\\]+)\\.txt");
        // Some pipelines log starts as "Started processing: <doc>"
        java.util.regex.Pattern startPat = java.util.regex.Pattern.compile("Started processing:\\s+([^\\s]+)");
        java.util.regex.Pattern writePat = java.util.regex.Pattern.compile("Writing XMI to .*[/\\\\]([^/\\\\]+)\\.txt\\.xmi");
        java.text.SimpleDateFormat sdf = new java.text.SimpleDateFormat("d MMM yyyy HH:mm:ss", java.util.Locale.ENGLISH);
        // Also parse explicit timing markers from TimingStartAE/TimingEndAE if present
        java.util.regex.Pattern tStart = java.util.regex.Pattern.compile("^\\[timing] START\\t([^\\t]+)\\t(\\d+)$");
        java.util.regex.Pattern tEnd = java.util.regex.Pattern.compile("^\\[timing] END\\t([^\\t]+)\\t(\\d*)\\t(\\d+)\\t(\\-?\\d+)$");
        try (java.io.BufferedReader br = java.nio.file.Files.newBufferedReader(runLog, java.nio.charset.StandardCharsets.UTF_8)) {
            String line;
            while ((line = br.readLine()) != null) {
                java.util.regex.Matcher mts = tStart.matcher(line);
                if (mts.find()) {
                    String doc = stripTxt(mts.group(1));
                    long ms = 0L; try { ms = Long.parseLong(mts.group(2)); } catch (Exception ignore) {}
                    DocTiming dt = byDoc.computeIfAbsent(doc, k -> new DocTiming());
                    dt.doc = doc; if (dt.startMs == 0L) dt.startMs = ms;
                    continue;
                }
                java.util.regex.Matcher mte = tEnd.matcher(line);
                if (mte.find()) {
                    String doc = stripTxt(mte.group(1));
                    long end = 0L; try { end = Long.parseLong(mte.group(3)); } catch (Exception ignore) {}
                    DocTiming dt = byDoc.computeIfAbsent(doc, k -> new DocTiming());
                    dt.doc = doc; dt.endMs = Math.max(dt.endMs, end);
                    continue;
                }
                java.util.regex.Matcher m = tsPat.matcher(line);
                if (!m.find()) continue;
                String ts = m.group(1);
                long ms;
                try { ms = sdf.parse(ts).getTime(); } catch (Exception e) { continue; }
                String after = m.group(2);
                java.util.regex.Matcher mr = readPat.matcher(after);
                if (mr.find()) {
                    String doc = stripTxt(mr.group(1));
                    DocTiming dt = byDoc.computeIfAbsent(doc, k -> new DocTiming());
                    dt.doc = doc; if (dt.startMs == 0L) dt.startMs = ms;
                } else {
                    java.util.regex.Matcher msrt = startPat.matcher(after);
                    if (msrt.find()) {
                        String doc = stripTxt(msrt.group(1));
                        DocTiming dt = byDoc.computeIfAbsent(doc, k -> new DocTiming());
                        dt.doc = doc; if (dt.startMs == 0L) dt.startMs = ms;
                    }
                }
                java.util.regex.Matcher mw = writePat.matcher(after);
                if (mw.find()) {
                    String doc = stripTxt(mw.group(1));
                    DocTiming dt = byDoc.computeIfAbsent(doc, k -> new DocTiming());
                    dt.doc = doc; dt.endMs = Math.max(dt.endMs, ms);
                }
            }
        } catch (Exception ignore) {}
        out.addAll(byDoc.values());
        return out;
    }
    private static String stripTxt(String s) { if (s == null) return ""; if (s.endsWith(".txt")) return s.substring(0, s.length()-4); return s; }

    // Collect friendly AE labels from Modules sheet rows (filtering writers/util)
    private static List<String> collectFriendlyAeLabels(List<List<String>> modules) {
        List<String> labels = new ArrayList<>();
        if (modules == null || modules.size() <= 1) return labels;
        java.util.Set<String> seen = new java.util.LinkedHashSet<>();
        for (int i = 1; i < modules.size(); i++) {
            List<String> r = modules.get(i);
            if (r.size() < 5) continue;
            String ae = r.get(3);      // AE column: Order,Phase,Step,AE,Label
            String label = r.get(4);   // Label column
            if (label == null) label = "";
            String low = ae == null ? "" : ae.toLowerCase(java.util.Locale.ROOT);
            // Exclude utility/loggers/writers when listing pipeline AEs for summary
            if (low.contains("writer") || low.contains("util.log") || low.contains("finishedlogger")) continue;
            if (label.isEmpty()) label = friendlyAeLabel(ae);
            if (label.isEmpty()) continue;
            if (!seen.contains(label)) { seen.add(label); labels.add(label); }
        }
        return labels;
    }

    // =============== Clinician Guide (AE explanations and columns) ===============
    private static List<List<String>> buildClinicianGuide(Path piper, List<List<String>> modules) throws IOException {
        // Columns: AE, What it does, Example, Columns in report
        List<List<String>> rows = new ArrayList<>();
        rows.add(Arrays.asList("AE","What it does","Example","Columns in report"));

        Set<String> present = new LinkedHashSet<>();
        if (modules != null && modules.size() > 1) {
            for (int i = 1; i < modules.size(); i++) {
                List<String> r = modules.get(i);
                String ae = r.size() > 3 ? r.get(3) : ""; // AE column
                if (!ae.isEmpty()) present.add(ae);
            }
        }

        // Known AEs and their info
        class Info { String what, ex, cols; Info(String w,String e,String c){what=w;ex=e;cols=c;} }
        Map<String,Info> info = new LinkedHashMap<>();
        info.put("TsDefaultTokenizerPipeline", new Info(
                "Splits text into sentences and tokens.",
                "Patient took aspirin → tokens: Patient, took, aspirin.",
                "Tokens (Tokens sheet), Span/Document Text support"));
        info.put("ContextDependentTokenizerAnnotator", new Info(
                "Refines tokenization in clinical contexts (e.g., units, abbreviations).",
                "mg, bpm kept together appropriately.",
                "Tokens support"));
        info.put("POSTagger", new Info(
                "Assigns part-of-speech tags used by chunker and downstream AEs.",
                "aspirin/NN, took/VBD",
                "No direct columns"));
        info.put("TsChunkerSubPipe", new Info(
                "Detects phrase chunks (NP/VP) that refine concept boundaries.",
                "[Patient]NP [took]VP [aspirin]NP",
                "No direct columns; improves Clinical Concepts"));
        info.put("TsDictionarySubPipe", new Info(
                "Matches text spans to UMLS concepts using a fast dictionary.",
                "aspirin → C0004057 (Drug)",
                "Clinical Concepts: CUI, Preferred Text, Semantic Group/Type"));
        info.put("tools.wsd.SimpleWsdDisambiguatorAnnotator", new Info(
                "Chooses best concept per mention; sets disambiguated, score, confidence.",
                "aspirin (T109 vs T121) → picks T109 by context overlap.",
                "Clinical Concepts: Confidence, ConceptScore, Disambiguated, CandidateCount, Candidates"));
        info.put("TsAttributeCleartkSubPipe", new Info(
                "Assigns assertion attributes (negation, uncertainty, conditional, generic, subject, history).",
                "No fever → Negated=true; Family history of diabetes → Subject=family, HistoryOf=1.",
                "Clinical Concepts: Polarity/Negated/Uncertain/Conditional/Generic/Subject/HistoryOf"));
        info.put("TsTemporalSubPipe", new Info(
                "Detects clinical events/times and their temporal relations.",
                "Admission yesterday; pain before surgery.",
                "Clinical Concepts: DocTimeRel (temporal relation to document time)"));
        info.put("TsRelationSubPipe", new Info(
                "Extracts non-temporal relations between mentions.",
                "Drug treats condition; severe pain; ulcer in stomach.",
                "Clinical Concepts: DegreeOf (Y/N), LocationOfText (first partner text)"));
        info.put("TsCorefSubPipe", new Info(
                "Links pronouns and mentions that refer to the same entity.",
                "He … the patient … same entity.",
                "Clinical Concepts: Coref (true if mention participates in a chain)"));
        info.put("FileTreeXmiWriter", new Info(
                "Writes full analysis to XMI per document.",
                "xmi/note1.txt.xmi",
                "XMI sheet count in RunInfo"));
        info.put("SemanticTableFileWriter", new Info(
                "Writes per-document clinical concept tables (BSV/CSV/HTML).",
                "See bsv_table/ and csv_table/",
                "Clinical Concepts sheet (aggregated)"));
        info.put("CuiListFileWriter", new Info(
                "Lists concepts per document/mention with basic fields.",
                "cui_list/*.bsv",
                "CuiList sheet (aggregated)"));
        info.put("CuiCountFileWriter", new Info(
                "Counts concepts per document.",
                "cui_count/*.bsv",
                "CuiCounts sheet"));
        info.put("TokenTableFileWriter", new Info(
                "Lists tokens and spans per document.",
                "bsv_tokens/*.bsv",
                "Tokens sheet"));

        // Confidence/Score explainer row first
        rows.add(Arrays.asList(
                "Confidence & ConceptScore",
                "Confidence is mention-level (0..1). ConceptScore is the chosen concept’s score (0..1). In this run both come from WSD’s context-overlap heuristic.",
                "‘aspirin’ in a sentence about medication yields higher scores than in an unrelated sentence.",
                "Clinical Concepts: Confidence, ConceptScore"
        ));

        // Add present AEs in pipeline order
        for (String ae : present) {
            Info i = info.get(ae);
            if (i != null) rows.add(Arrays.asList(ae, i.what, i.ex, i.cols));
        }

        return rows;
    }
}
