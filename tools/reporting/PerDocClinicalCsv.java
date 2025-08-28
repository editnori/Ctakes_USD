package tools.reporting;

import java.io.*;
import java.nio.charset.StandardCharsets;
import java.nio.file.*;
import java.util.*;

/**
 * Generate per-document CSV files (one per XMI) that match the columns used
 * in the "Clinical Concepts" sheet of the ExcelXmlReport. This enables
 * reviewing individual notes without building the workbook.
 *
 * Usage:
 *   java tools.reporting.PerDocClinicalCsv -r <run_parent_dir> [-t <target_subdir>]
 *
 * Defaults:
 *   - XMI dir:    <run_parent_dir>/xmi (falls back to shard_* / xmi)
 *   - BSV dir:    <run_parent_dir>/bsv_table (falls back to shard_* / bsv_table)
 *   - CSV target: <run_parent_dir>/<target_subdir> (default: csv_table_concepts)
 */
public class PerDocClinicalCsv {

    public static void main(String[] args) throws Exception {
        Map<String,String> cli = parseArgs(args);
        if (!cli.containsKey("-r") && !cli.containsKey("--run")) {
            System.err.println("Usage: java tools.reporting.PerDocClinicalCsv -r <run_parent_dir> [-t <target_subdir>]");
            System.exit(2);
        }
        Path parent = Paths.get(nvl(cli.getOrDefault("-r", cli.getOrDefault("--run", "")))).toAbsolutePath().normalize();
        String targetSubdir = cli.getOrDefault("-t", cli.getOrDefault("--target", "csv_table_concepts"));

        if (!Files.isDirectory(parent)) {
            System.err.println("Run parent dir not found: " + parent);
            System.exit(2);
        }

        // Resolve XMI and BSV dirs (prefer top-level; fallback to shard_*)
        List<Path> xmiDirs = new ArrayList<>();
        List<Path> bsvDirs = new ArrayList<>();
        Path xmiTop = parent.resolve("xmi");
        Path bsvTop = parent.resolve("bsv_table");
        if (Files.isDirectory(xmiTop)) xmiDirs.add(xmiTop);
        if (Files.isDirectory(bsvTop)) bsvDirs.add(bsvTop);
        try (DirectoryStream<Path> ds = Files.newDirectoryStream(parent, p -> Files.isDirectory(p) && p.getFileName().toString().startsWith("shard_"))) {
            for (Path sh : ds) {
                Path x = sh.resolve("xmi");
                Path b = sh.resolve("bsv_table");
                if (Files.isDirectory(x)) xmiDirs.add(x);
                if (Files.isDirectory(b)) bsvDirs.add(b);
            }
        }
        if (xmiDirs.isEmpty()) {
            System.err.println("No XMI directories found under: " + parent);
            System.exit(2);
        }
        // Build BSV span maps per doc across all BSV dirs
        Map<String, Map<String,String[]>> bsvByDoc = new HashMap<>();
        for (Path b : bsvDirs) mergeInto(bsvByDoc, buildBsvSpanMap(b));

        // Smoking status per doc (from any XMI dir)
        Map<String,String> smokingByDoc = detectSmokingStatus(xmiDirs.get(0));

        Path outDir = parent.resolve(targetSubdir);
        Files.createDirectories(outDir);

        int written = 0; int docs = 0;
        for (Path xd : xmiDirs) {
            try (DirectoryStream<Path> ds = Files.newDirectoryStream(xd, p -> p.toString().endsWith(".xmi") || p.toString().endsWith(".XMI"))) {
                for (Path xmi : ds) {
                    docs++;
                    MentionsFromXmi m = parseXmiMentions(xmi);
                    Map<String,String[]> spanMap = bsvByDoc.getOrDefault(m.docId, Collections.emptyMap());
                    Path csv = outDir.resolve(m.docId + ".CSV");
                    try (BufferedWriter bw = Files.newBufferedWriter(csv, StandardCharsets.UTF_8)) {
                        // Header (matches ExcelXmlReport Clinical Concepts)
                        bw.write(String.join(",", Arrays.asList(
                                "Document","Begin","End","Text",
                                "Section","SmokingStatus",
                                "Semantic Group","Semantic Type","SemanticsFallback","CUI","TUI","PreferredText","PrefTextFallback","CodingScheme",
                                "CandidateCount","Candidates","Confidence","ConceptScore","Disambiguated",
                                "DocTimeRel","DegreeOf","LocationOfText","Coref","CorefChainId","CorefRepText",
                                "Polarity","Negated","Uncertain","Conditional","Generic","Subject","HistoryOf"
                        )));
                        bw.write("\n");
                        String smoking = nvl(smokingByDoc.get(m.docId));
                        for (MentionRow r : m.rows) {
                            if ((r.candidateCount <= 0) && (nvl(r.cui).isEmpty())) continue; // concept-bearing only
                            String key = r.begin + "," + r.end;
                            String section = ""; String sg = ""; String st = ""; boolean semFallback = false;
                            String[] info = spanMap.get(key);
                            if (info != null) { section = nvl(info[0]); sg = nvl(info[1]); st = nvl(info[2]); }
                            if (section.equalsIgnoreCase("SIMPLE_SEGMENT")) section = "S";
                            if ((sg.isEmpty()) || (st.isEmpty())) {
                                String[] sem = semFromTui(nvl(r.tui));
                                if (sem != null) { if (sg.isEmpty()) sg = sem[0]; if (st.isEmpty()) st = sem[1]; semFallback = true; }
                            }
                            boolean prefFallback = false;
                            String prefOut = nvl(r.pref);
                            if (prefOut.isEmpty()) { prefOut = nvl(r.text); prefFallback = true; }
                            List<String> row = Arrays.asList(
                                    m.docId,
                                    String.valueOf(r.begin), String.valueOf(r.end), csvEscape(nvl(r.text)),
                                    csvEscape(section), csvEscape(smoking),
                                    csvEscape(sg), csvEscape(st), semFallback?"true":"", nvl(r.cui), nvl(r.tui), csvEscape(prefOut), prefFallback?"true":"", csvEscape(nvl(r.scheme)),
                                    String.valueOf(r.candidateCount), csvEscape(nvl(r.candidatesJoined)), String.valueOf(r.confidence), String.valueOf(r.conceptScore), String.valueOf(r.disambiguated),
                                    csvEscape(nvl(r.docTimeRel)), String.valueOf(r.degreeOf), csvEscape(nvl(r.locationOfText)), String.valueOf(r.coref), csvEscape(nvl(r.corefChainId)), csvEscape(nvl(r.corefRepText)),
                                    String.valueOf(r.polarity), String.valueOf(r.negated), String.valueOf(r.uncertain), String.valueOf(r.conditional), String.valueOf(r.generic), csvEscape(nvl(r.subject)), String.valueOf(r.historyOf)
                            );
                            bw.write(String.join(",", row));
                            bw.write("\n");
                        }
                    }
                    written++;
                }
            }
        }
        System.out.println("[csv] Wrote " + written + " per-document CSV files from " + docs + " XMI docs into: " + outDir);
    }

    // --------------- Helpers (adapted from ExcelXmlReport) ---------------
    private static Map<String,String> parseArgs(String[] args) {
        Map<String,String> m = new HashMap<>();
        for (int i=0; i<args.length; i++) {
            String a = args[i];
            if (a.startsWith("-")) {
                String v = (i+1 < args.length && !args[i+1].startsWith("-")) ? args[i+1] : "";
                m.put(a, v);
                if (!v.isEmpty()) i++;
            }
        }
        return m;
    }
    private static String nvl(String s) { return s == null ? "" : s; }
    private static String baseName(Path p) {
        String n = p.getFileName().toString();
        int i = n.lastIndexOf('.');
        return i > 0 ? n.substring(0, i) : n;
    }
    private static String csvEscape(String s) {
        if (s == null) return "";
        boolean need = s.contains(",") || s.contains("\"") || s.contains("\n") || s.contains("\r");
        if (!need) return s;
        String v = s.replace("\"", "\"\"");
        return "\"" + v + "\"";
    }

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
                    String key = span.replaceAll("\\s+", "");
                    bySpan.putIfAbsent(key, new String[]{section, sg, st});
                }
            }
        }
        return byDoc;
    }

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

    private static class MentionsFromXmi {
        String docId; String sofa;
        Map<String, Concept> conceptById = new HashMap<>();
        Map<String, List<String>> fsArrayElems = new HashMap<>();
        List<MentionRow> rows = new ArrayList<>();
        Map<String, MentionSpan> mentionById = new HashMap<>();
        Set<MentionSpan> corefMarkables = new HashSet<>();
        Map<String,String> eventIdToDocTimeRel = new HashMap<>();
    }
    private static class Concept { String cui, tui, pref, scheme; boolean disamb; double score; }
    private static class MentionRow {
        int begin, end, polarity, historyOf, candidateCount; double confidence, conceptScore; boolean negated, uncertain, conditional, generic, disambiguated, degreeOf, coref; String subject;
        String cui, tui, pref, scheme, candidatesJoined, text, xmiId, docTimeRel, locationOfText, corefChainId, corefRepText;
    }
    private static class MentionSpan { int begin, end; MentionSpan(int b,int e){begin=b;end=e;} public boolean equals(Object o){ if(!(o instanceof MentionSpan)) return false; MentionSpan m=(MentionSpan)o; return begin==m.begin && end==m.end; } public int hashCode(){ return Objects.hash(begin,end);} }

    private static MentionsFromXmi parseXmiMentions(Path xmi) throws Exception {
        javax.xml.parsers.DocumentBuilderFactory dbf = javax.xml.parsers.DocumentBuilderFactory.newInstance();
        dbf.setNamespaceAware(true);
        javax.xml.parsers.DocumentBuilder db = dbf.newDocumentBuilder();
        org.w3c.dom.Document doc = db.parse(xmi.toFile());
        org.w3c.dom.Element root = doc.getDocumentElement();
        MentionsFromXmi out = new MentionsFromXmi();
        // Sofa + docId
        org.w3c.dom.NodeList sofas = root.getElementsByTagNameNS("http:///uima/cas.ecore", "Sofa");
        if (sofas.getLength() > 0) { out.sofa = ((org.w3c.dom.Element)sofas.item(0)).getAttribute("sofaString"); }
        org.w3c.dom.NodeList docInfos = root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/textspan.ecore", "DocumentID");
        if (docInfos.getLength()>0) {
            out.docId = ((org.w3c.dom.Element)docInfos.item(0)).getAttribute("documentID");
        }
        if (out.docId==null || out.docId.isEmpty()) out.docId = baseName(xmi);
        // Concepts
        org.w3c.dom.NodeList umls = root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/refsem.ecore", "UmlsConcept");
        for (int i=0;i<umls.getLength();i++) {
            org.w3c.dom.Element e = (org.w3c.dom.Element) umls.item(i);
            String id = e.getAttribute("xmi:id");
            Concept c = new Concept();
            c.cui = e.getAttribute("cui");
            c.tui = e.getAttribute("tui");
            c.pref = e.getAttribute("preferredText");
            c.scheme = e.getAttribute("codingScheme");
            c.disamb = Boolean.parseBoolean(e.getAttribute("disambiguated"));
            try { c.score = Double.parseDouble(nvl(e.getAttribute("score"))); } catch (Exception ignore) { c.score = 0.0; }
            out.conceptById.put(id, c);
        }
        // Event mentions (for DocTimeRel mapping)
        org.w3c.dom.NodeList evs = root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/textsem.ecore", "EventMention");
        for (int i=0;i<evs.getLength();i++) {
            org.w3c.dom.Element e = (org.w3c.dom.Element) evs.item(i);
            String id = e.getAttribute("xmi:id");
            String dtr = e.getAttribute("docTimeRel");
            if (!nvl(id).isEmpty() && !nvl(dtr).isEmpty()) out.eventIdToDocTimeRel.put(id, dtr);
        }
        // FSArray elements
        org.w3c.dom.NodeList fsa = root.getElementsByTagNameNS("http:///uima/cas.ecore", "FSArray");
        for (int i=0;i<fsa.getLength();i++) {
            org.w3c.dom.Element e = (org.w3c.dom.Element) fsa.item(i);
            String id = e.getAttribute("xmi:id");
            String members = nvl(e.getAttribute("elements")).trim();
            if (!members.isEmpty()) out.fsArrayElems.put(id, Arrays.asList(members.split("\\s+")));
            else out.fsArrayElems.put(id, Collections.emptyList());
        }
        // Mentions
        org.w3c.dom.NodeList all = root.getChildNodes();
        for (int i=0;i<all.getLength();i++) {
            org.w3c.dom.Node n = all.item(i);
            if (n.getNodeType() != org.w3c.dom.Node.ELEMENT_NODE) continue;
            org.w3c.dom.Element e = (org.w3c.dom.Element) n;
            String ns = e.getNamespaceURI();
            String name = e.getLocalName();
            if (ns != null && ns.endsWith("/textsem.ecore") && name != null && name.endsWith("Mention")) {
                MentionRow r = new MentionRow();
                r.begin = parseInt(e.getAttribute("begin"));
                r.end = parseInt(e.getAttribute("end"));
                r.polarity = parseInt(e.getAttribute("polarity"));
                r.confidence = parseDouble(e.getAttribute("confidence"));
                r.negated = r.polarity < 0;
                r.uncertain = parseInt(e.getAttribute("uncertainty")) != 0;
                r.conditional = Boolean.parseBoolean(e.getAttribute("conditional"));
                r.generic = Boolean.parseBoolean(e.getAttribute("generic"));
                r.subject = nvl(e.getAttribute("subject"));
                r.historyOf = parseInt(e.getAttribute("historyOf"));
                r.xmiId = e.getAttribute("xmi:id");
                String arr = nvl(e.getAttribute("ontologyConceptArr"));
                if (!arr.isEmpty()) {
                    List<String> candIds = new ArrayList<>();
                    if (arr.indexOf(' ') >= 0) {
                        String[] ids = arr.trim().split("\\s+");
                        r.candidateCount = ids.length;
                        candIds = Arrays.asList(ids);
                        Concept first = out.conceptById.get(ids[0]);
                        if (first != null) { r.cui = first.cui; r.tui = first.tui; r.pref = first.pref; r.scheme = first.scheme; r.disambiguated = first.disamb; r.conceptScore = first.score; }
                    } else {
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
                                if (first != null) { r.cui = first.cui; r.tui = first.tui; r.pref = first.pref; r.scheme = first.scheme; r.disambiguated = first.disamb; r.conceptScore = first.score; }
                            }
                        }
                    }
                    if (!candIds.isEmpty()) {
                        List<String> cStrs = new ArrayList<>(candIds.size());
                        for (String id : candIds) {
                            Concept c = out.conceptById.get(id);
                            if (c != null) {
                                String cui = nvl(c.cui).isEmpty() ? (nvl(r.cui).isEmpty()?"?":r.cui) : c.cui;
                                String tui = nvl(c.tui).isEmpty() ? (nvl(r.tui).isEmpty()?"?":r.tui) : c.tui;
                                String pref = nvl(c.pref);
                                if (pref.isEmpty()) pref = nvl(r.pref);
                                if (pref.isEmpty()) pref = nvl(r.text);
                                if (pref.isEmpty()) pref = "?";
                                cStrs.add(cui + ":" + tui + ":" + pref);
                            }
                        }
                        r.candidatesJoined = String.join("; ", cStrs);
                    }
                }
                r.text = safeSub(out.sofa, r.begin, r.end);
                out.rows.add(r);
                if (!nvl(r.xmiId).isEmpty()) out.mentionById.put(r.xmiId, new MentionSpan(r.begin, r.end));
                String ev = nvl(e.getAttribute("event"));
                if (!ev.isEmpty()) {
                    String dtr = out.eventIdToDocTimeRel.get(ev);
                    if (dtr != null) r.docTimeRel = dtr;
                }
            }
        }
        // Coref Markables
        org.w3c.dom.NodeList marks = root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/textsem.ecore", "Markable");
        for (int i=0;i<marks.getLength();i++) {
            org.w3c.dom.Element e = (org.w3c.dom.Element) marks.item(i);
            int b = parseInt(e.getAttribute("begin"));
            int en = parseInt(e.getAttribute("end"));
            out.corefMarkables.add(new MentionSpan(b,en));
        }
        // Relations
        Map<String,String> relArgToMention = new HashMap<>();
        org.w3c.dom.NodeList relArgs = root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/relation.ecore", "RelationArgument");
        for (int i=0;i<relArgs.getLength();i++) {
            org.w3c.dom.Element e = (org.w3c.dom.Element) relArgs.item(i);
            String id = nvl(e.getAttribute("xmi:id"));
            String arg = nvl(e.getAttribute("argument"));
            if (!id.isEmpty() && !arg.isEmpty()) relArgToMention.put(id, arg);
        }
        // Helper to mark a mention by id
        java.util.function.BiConsumer<String,String> markLocation = (mIdPartner, holderId) -> {
            if (mIdPartner == null || mIdPartner.isEmpty()) return;
            MentionSpan sp = out.mentionById.get(mIdPartner);
            if (sp == null) return;
            String txt = safeSub(out.sofa, sp.begin, sp.end);
            for (MentionRow mr : out.rows) { if (holderId != null && holderId.equals(mr.xmiId)) { mr.locationOfText = txt; break; } }
        };
        // Disjoint set for coref chains
        Map<String,String> parent = new HashMap<>();
        java.util.function.Function<String,String> find = new java.util.function.Function<String,String>(){
            public String apply(String x){ String p = parent.getOrDefault(x, x); if (!p.equals(x)) parent.put(x, this.apply(p)); else parent.putIfAbsent(x, x); return parent.get(x);} };
        java.util.function.BiConsumer<String,String> union = (a,b) -> { String ra = find.apply(a); String rb = find.apply(b); if (!ra.equals(rb)) parent.put(ra, rb); };
        // DegreeOfTextRelation
        org.w3c.dom.NodeList degs = root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/relation.ecore", "DegreeOfTextRelation");
        for (int i=0;i<degs.getLength();i++) {
            org.w3c.dom.Element e = (org.w3c.dom.Element) degs.item(i);
            String a1 = nvl(e.getAttribute("arg1")); String a2 = nvl(e.getAttribute("arg2"));
            String m1 = relArgToMention.get(a1); String m2 = relArgToMention.get(a2);
            for (MentionRow mr : out.rows) { if (mr.xmiId != null && (mr.xmiId.equals(m1) || mr.xmiId.equals(m2))) mr.degreeOf = true; }
        }
        // LocationOfTextRelation
        org.w3c.dom.NodeList locs = root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/relation.ecore", "LocationOfTextRelation");
        for (int i=0;i<locs.getLength();i++) {
            org.w3c.dom.Element e = (org.w3c.dom.Element) locs.item(i);
            String a1 = nvl(e.getAttribute("arg1")); String a2 = nvl(e.getAttribute("arg2"));
            String m1 = relArgToMention.get(a1); String m2 = relArgToMention.get(a2);
            if (m1==null || m2==null) continue; markLocation.accept(m2, m1);
        }
        // CoreferenceRelation: union mentions into chains
        org.w3c.dom.NodeList corefs = root.getElementsByTagNameNS("http:///org/apache/ctakes/typesystem/type/relation.ecore", "CoreferenceRelation");
        for (int i=0;i<corefs.getLength();i++) {
            org.w3c.dom.Element e = (org.w3c.dom.Element) corefs.item(i);
            String a1 = nvl(e.getAttribute("arg1")); String a2 = nvl(e.getAttribute("arg2"));
            String m1 = relArgToMention.get(a1); String m2 = relArgToMention.get(a2);
            if (m1!=null && m2!=null) union.accept(m1, m2);
        }
        // Assign chain ids and representative text
        Map<String,List<MentionRow>> chainMap = new HashMap<>();
        for (MentionRow mr : out.rows) {
            if (mr.xmiId == null) continue;
            String rootId = find.apply(mr.xmiId);
            if (!rootId.equals(mr.xmiId)) { mr.coref = true; chainMap.computeIfAbsent(rootId, k -> new ArrayList<>()).add(mr); }
        }
        int cseq = 1;
        for (Map.Entry<String,List<MentionRow>> e : chainMap.entrySet()) {
            String chainId = "C" + (cseq++);
            List<MentionRow> list = e.getValue();
            MentionRow rep = list.stream().min(java.util.Comparator.comparingInt(m -> m.begin)).orElse(null);
            String repText = rep != null ? rep.text : "";
            for (MentionRow mr : list) { mr.corefChainId = chainId; mr.corefRepText = repText; }
        }
        for (MentionRow mr : out.rows) { if (out.corefMarkables.contains(new MentionSpan(mr.begin, mr.end))) mr.coref = true; }
        return out;
    }

    private static String safeSub(String s, int b, int e) { if (s == null) return ""; if (b < 0 || e > s.length() || b > e) return ""; return s.substring(b, e); }
    private static int parseInt(String s) { try { return Integer.parseInt(s); } catch (Exception e) { return 0; } }
    private static double parseDouble(String s) { try { return Double.parseDouble(s); } catch (Exception e) { return 0.0; } }

    // Minimal TUI -> (Semantic Group, Semantic Type) mapping as a safety net
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
    private static void mergeInto(Map<String, Map<String,String[]>> target, Map<String, Map<String,String[]>> src) {
        for (Map.Entry<String, Map<String,String[]>> e : src.entrySet()) {
            Map<String,String[]> m = target.computeIfAbsent(e.getKey(), k -> new HashMap<>());
            m.putAll(e.getValue());
        }
    }
}
