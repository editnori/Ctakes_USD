package tools.reporting.uima;

import org.apache.ctakes.typesystem.type.relation.CoreferenceRelation;
import org.apache.ctakes.typesystem.type.relation.DegreeOfTextRelation;
import org.apache.ctakes.typesystem.type.relation.LocationOfTextRelation;
import org.apache.ctakes.typesystem.type.relation.RelationArgument;
import org.apache.ctakes.typesystem.type.textsem.IdentifiedAnnotation;
import org.apache.ctakes.typesystem.type.textsem.Markable;
import org.apache.ctakes.typesystem.type.textsem.MedicationMention;
import org.apache.ctakes.typesystem.type.refsem.UmlsConcept;
import org.apache.uima.UimaContext;
import org.apache.uima.analysis_component.JCasAnnotator_ImplBase;
import org.apache.uima.analysis_engine.AnalysisEngineProcessException;
import org.apache.uima.fit.descriptor.ConfigurationParameter;
import org.apache.uima.fit.util.JCasUtil;
import org.apache.uima.jcas.JCas;
import tools.reporting.uima.DocIdUtil;

import java.io.BufferedWriter;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.StandardOpenOption;
import java.util.ArrayDeque;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.Deque;
import java.util.HashMap;
import java.util.HashSet;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Set;

/**
 * Emits a self-contained HTML overview of the source note with interactive annotation layers.
 * Layers include core concepts, WSD-selected concepts, drug mentions, relation members and coreference chains.
 */
public class HtmlAnnotationOverviewWriter extends JCasAnnotator_ImplBase {

    public static final String PARAM_SUBDIR = "SubDirectory";

    @ConfigurationParameter(name = PARAM_SUBDIR, mandatory = false)
    private String subDir = "html";

    private String outputBase;

    @Override
    public void initialize(UimaContext context) {
        Object od = context.getConfigParameterValue("OutputDirectory");
        if (od == null) {
            od = System.getProperty("ctakes.output.dir");
        }
        if (od == null) {
            od = System.getProperty("OUTPUT_DIR");
        }
        outputBase = (od == null) ? "." : od.toString();
        Object sd = context.getConfigParameterValue(PARAM_SUBDIR);
        if (sd != null && !sd.toString().trim().isEmpty()) {
            subDir = sd.toString().trim();
        }
    }

    @Override
    public void process(JCas jCas) throws AnalysisEngineProcessException {
        final String docId = DocIdUtil.resolveDocId(jCas);
        final String text = jCas.getDocumentText() == null ? "" : jCas.getDocumentText();

        Map<IdentifiedAnnotation, AnnotationLayers> layerMap = buildLayerMetadata(jCas);
        List<AnnotationView> annotations = buildAnnotationViews(jCas, text, layerMap);
        annotations.sort(Comparator.comparingInt((AnnotationView a) -> a.begin)
                                   .thenComparingInt(a -> a.end - a.begin));

        String html = renderHtml(docId, text, annotations);
        writeHtml(docId, html);
    }

    private Map<IdentifiedAnnotation, AnnotationLayers> buildLayerMetadata(JCas jCas) {
        Map<IdentifiedAnnotation, AnnotationLayers> layerData = new HashMap<>();

        Set<IdentifiedAnnotation> relationMembers = new HashSet<>();
        for (DegreeOfTextRelation relation : JCasUtil.select(jCas, DegreeOfTextRelation.class)) {
            addRelationParticipant(relation, relationMembers);
        }
        for (LocationOfTextRelation relation : JCasUtil.select(jCas, LocationOfTextRelation.class)) {
            addRelationParticipant(relation, relationMembers);
        }

        Set<IdentifiedAnnotation> corefMembers = new HashSet<>();
        for (CoreferenceRelation relation : JCasUtil.select(jCas, CoreferenceRelation.class)) {
            IdentifiedAnnotation arg1 = asMention(relation.getArg1());
            IdentifiedAnnotation arg2 = asMention(relation.getArg2());
            if (arg1 != null) {
                corefMembers.add(arg1);
            }
            if (arg2 != null) {
                corefMembers.add(arg2);
            }
        }

        for (IdentifiedAnnotation ia : JCasUtil.select(jCas, IdentifiedAnnotation.class)) {
            AnnotationLayers layers = new AnnotationLayers();
            layers.layers.add("core");
            if (ia instanceof MedicationMention) {
                layers.layers.add("drug");
            }
            if (relationMembers.contains(ia)) {
                layers.layers.add("relations");
            }
            if (corefMembers.contains(ia)) {
                layers.layers.add("coref");
            }
            UmlsConcept concept = firstConcept(ia);
            if (concept != null && concept.getDisambiguated()) {
                layers.layers.add("wsd");
            }
            layers.concept = concept;
            layerData.put(ia, layers);
        }
        return layerData;
    }

    private static void addRelationParticipant(org.apache.ctakes.typesystem.type.relation.Relation relation,
                                               Set<IdentifiedAnnotation> sink) {
        if (relation == null) {
            return;
        }
        RelationArgument arg1 = relation.getArg1();
        RelationArgument arg2 = relation.getArg2();
        IdentifiedAnnotation mention1 = asMention(arg1);
        IdentifiedAnnotation mention2 = asMention(arg2);
        if (mention1 != null) {
            sink.add(mention1);
        }
        if (mention2 != null) {
            sink.add(mention2);
        }
    }

    private static IdentifiedAnnotation asMention(RelationArgument argument) {
        if (argument == null || argument.getArgument() == null) {
            return null;
        }
        if (argument.getArgument() instanceof IdentifiedAnnotation) {
            return (IdentifiedAnnotation) argument.getArgument();
        }
        return null;
    }

    private List<AnnotationView> buildAnnotationViews(JCas jCas, String text,
                                                      Map<IdentifiedAnnotation, AnnotationLayers> layerMap) {
        List<AnnotationView> views = new ArrayList<>();
        for (IdentifiedAnnotation ia : layerMap.keySet()) {
            if (ia.getBegin() < 0 || ia.getEnd() > text.length() || ia.getBegin() >= ia.getEnd()) {
                continue;
            }
            AnnotationLayers layers = layerMap.get(ia);
            String covered = escapeHtml(text.substring(ia.getBegin(), ia.getEnd()));
            String section = findSection(jCas, ia);
            String tooltip = buildTooltip(ia, layers.concept, section);
            String rxCodes = collectRxnormCodes(ia);
            AnnotationView view = new AnnotationView();
            view.begin = ia.getBegin();
            view.end = ia.getEnd();
            view.layers = new ArrayList<>(layers.layers);
            view.coveredText = covered;
            view.tooltip = tooltip;
            view.typeName = ia.getType().getShortName();
            view.cui = layers.concept == null ? "" : nvl(layers.concept.getCui());
            view.rxCui = rxCodes;
            view.section = section;
            view.preferredText = layers.concept == null ? "" : nvl(layers.concept.getPreferredText());
            view.polarity = ia.getPolarity();

            view.discoveryTechnique = ia.getDiscoveryTechnique();
            views.add(view);
        }
        return views;
    }

    private String renderHtml(String docId, String text, List<AnnotationView> annotations)
            throws AnalysisEngineProcessException {
        StringBuilder sb = new StringBuilder();
        sb.append("<!DOCTYPE html><html><head><meta charset=\"utf-8\"/>");
        sb.append("<title>").append(escapeHtml(docId)).append(" – cTAKES Annotations</title>");
        sb.append("<style>");
        sb.append(BASE_CSS);
        sb.append("</style><script>");
        sb.append(BASE_JS);
        sb.append("</script></head><body>");
        sb.append("<header><h1>").append(escapeHtml(docId)).append("</h1>");
        sb.append(renderLegend());
        sb.append("</header>");
        sb.append("<main><section class=\"note\" id=\"annotated-note\">");
        sb.append(renderAnnotatedText(text, annotations));
        sb.append("</section>");
        sb.append(renderSummaryTable(annotations));
        sb.append("</main></body></html>");
        return sb.toString();
    }

    private String renderLegend() {
        return "<div class=\"controls\">" +
                "<strong>Layers:</strong>" +
                toggle("core", "Core Concepts", true) +
                toggle("wsd", "WSD", true) +
                toggle("drug", "Drug", true) +
                toggle("relations", "Relations", true) +
                toggle("coref", "Coref", true) +
                "<span class=\"hint\">Toggle layers to hide/show annotations.</span>" +
                "</div>";
    }

    private static String toggle(String layer, String label, boolean checked) {
        String id = "toggle-" + layer;
        return "<label><input type=\"checkbox\" id=\"" + id + "\" data-layer=\"" + layer +
                "\" onclick=\"updateLayers()\"" + (checked ? " checked" : "") +
                "/>" + escapeHtml(label) + "</label>";
    }

    private String renderAnnotatedText(String text, List<AnnotationView> annotations) {
        List<Event> events = new ArrayList<>();
        for (AnnotationView view : annotations) {
            events.add(Event.start(view.begin, view));
            events.add(Event.end(view.end, view));
        }
        events.sort(Event::compare);

        StringBuilder sb = new StringBuilder();
        int cursor = 0;
        Deque<AnnotationView> open = new ArrayDeque<>();
        for (Event event : events) {
            int pos = event.offset;
            if (pos > cursor) {
                sb.append(formatTextSegment(text.substring(cursor, pos)));
                cursor = pos;
            }
            if (event.isEnd) {
                closeAnnotation(sb, open, event.view);
            } else {
                open.push(event.view);
                sb.append(startTag(event.view));
            }
        }
        if (cursor < text.length()) {
            sb.append(formatTextSegment(text.substring(cursor)));
        }
        while (!open.isEmpty()) {
            sb.append("</span>");
            open.pop();
        }
        return sb.toString();
    }

    private static void closeAnnotation(StringBuilder sb, Deque<AnnotationView> stack, AnnotationView target) {
        if (!stack.contains(target)) {
            return;
        }
        Deque<AnnotationView> buffer = new ArrayDeque<>();
        while (!stack.isEmpty()) {
            AnnotationView current = stack.pop();
            sb.append("</span>");
            if (current == target) {
                break;
            }
            buffer.push(current);
        }
        while (!buffer.isEmpty()) {
            AnnotationView view = buffer.pop();
            sb.append(startTag(view));
            stack.push(view);
        }
    }

    private static String startTag(AnnotationView view) {
        StringBuilder cls = new StringBuilder("anno");
        for (String layer : view.layers) {
            cls.append(' ').append("layer-").append(layer);
        }
        String dataLayers = String.join(" ", view.layers);
        StringBuilder tag = new StringBuilder();
        tag.append("<span class=\"").append(cls).append("\" data-layers=\"")
           .append(escapeHtml(dataLayers)).append("\" data-type=\"")
           .append(escapeHtml(view.typeName)).append("\" data-cui=\"")
           .append(escapeHtml(view.cui)).append("\" data-rxcui=\"")
           .append(escapeHtml(view.rxCui)).append("\" title=\"")
           .append(escapeHtml(view.tooltip)).append("\">");
        return tag.toString();
    }

    private String renderSummaryTable(List<AnnotationView> views) {
        StringBuilder sb = new StringBuilder();
        sb.append("<section class=\"summary\"><h2>Annotation Details</h2>");
        sb.append("<table><thead><tr><th>Begin</th><th>End</th><th>Text</th><th>Type</th><th>Layers</th><th>CUI</th><th>RxCUI</th><th>Preferred</th><th>Polarity</th><th>Discovery</th></tr></thead><tbody>");
        views.sort(Comparator.comparingInt((AnnotationView v) -> v.begin)
                              .thenComparingInt(v -> v.end));
        for (AnnotationView view : views) {
            sb.append("<tr>")
              .append(cell(Integer.toString(view.begin)))
              .append(cell(Integer.toString(view.end)))
              .append(cell(view.coveredText))
              .append(cell(escapeHtml(view.typeName)))
              .append(cell(escapeHtml(String.join(", ", view.layers))))
              .append(cell(escapeHtml(view.cui)))
              .append(cell(escapeHtml(view.rxCui)))
              .append(cell(escapeHtml(view.preferredText)))
              .append(cell(view.polarity == -1 ? "NEG" : "POS"))
              .append(cell(Integer.toString(view.discoveryTechnique)))
              .append("</tr>");
        }
        sb.append("</tbody></table></section>");
        return sb.toString();
    }

    private static String cell(String value) {
        return "<td>" + value + "</td>";
    }

    private void writeHtml(String docId, String html) throws AnalysisEngineProcessException {
        Path outDir = Paths.get(outputBase, subDir);
        try {
            Files.createDirectories(outDir);
            Path out = outDir.resolve(docId + ".html");
            try (BufferedWriter writer = Files.newBufferedWriter(out, StandardCharsets.UTF_8,
                    StandardOpenOption.CREATE, StandardOpenOption.TRUNCATE_EXISTING)) {
                writer.write(html);
            }
        } catch (IOException e) {
            throw new AnalysisEngineProcessException(e);
        }
    }


    private static String findSection(JCas jCas, IdentifiedAnnotation ia) {
        List<org.apache.ctakes.typesystem.type.textspan.Segment> segments = new ArrayList<>(JCasUtil.select(jCas, org.apache.ctakes.typesystem.type.textspan.Segment.class));
        for (org.apache.ctakes.typesystem.type.textspan.Segment segment : segments) {
            if (ia.getBegin() >= segment.getBegin() && ia.getEnd() <= segment.getEnd()) {
                if (segment.getPreferredText() != null && !segment.getPreferredText().isEmpty()) {
                    return segment.getPreferredText();
                }
                if (segment.getId() != null) {
                    return segment.getId();
                }
            }
        }
        return "";
    }

    private static UmlsConcept firstConcept(IdentifiedAnnotation ia) {
        if (ia == null || ia.getOntologyConceptArr() == null) {
            return null;
        }
        for (int i = 0; i < ia.getOntologyConceptArr().size(); i++) {
            if (ia.getOntologyConceptArr().get(i) instanceof UmlsConcept) {
                return (UmlsConcept) ia.getOntologyConceptArr().get(i);
            }
        }
        return null;
    }

    private static String collectRxnormCodes(IdentifiedAnnotation ia) {
        if (ia == null || ia.getOntologyConceptArr() == null) {
            return "";
        }
        LinkedHashSet<String> codes = new LinkedHashSet<>();
        for (int i = 0; i < ia.getOntologyConceptArr().size(); i++) {
            if (ia.getOntologyConceptArr().get(i) instanceof UmlsConcept) {
                UmlsConcept c = (UmlsConcept) ia.getOntologyConceptArr().get(i);
                String scheme = nvl(c.getCodingScheme()).toUpperCase(Locale.ROOT);
                if (!"RXNORM".equals(scheme)) {
                    continue;
                }
                String code = nvl(c.getCode());
                if (!code.isEmpty()) {
                    codes.add(code);
                }
            }
        }
        return String.join("|", codes);
    }

    private static String buildTooltip(IdentifiedAnnotation ia, UmlsConcept concept, String section) {
        StringBuilder sb = new StringBuilder();
        sb.append(ia.getType().getShortName());
        if (concept != null) {
            sb.append(" | CUI: ").append(nvl(concept.getCui()));
            if (concept.getPreferredText() != null) {
                sb.append(" | ").append(concept.getPreferredText());
            }
            if (concept.getTui() != null) {
                sb.append(" | TUI: ").append(concept.getTui());
            }
        }
        if (section != null && !section.isEmpty()) {
            sb.append(" | Section: ").append(section);
        }
        sb.append(" | Polarity: ").append(ia.getPolarity() == -1 ? "NEG" : "POS");
        return sb.toString();
    }

    private static String formatTextSegment(String raw) {
        if (raw.isEmpty()) {
            return "";
        }
        String escaped = escapeHtml(raw);
        return escaped.replace("\n", "<br/>\n");
    }

    private static String escapeHtml(String value) {
        if (value == null) {
            return "";
        }
        String result = value.replace("&", "&amp;")
                             .replace("<", "&lt;")
                             .replace(">", "&gt;")
                             .replace("\"", "&quot;");
        return result;
    }

    private static String nvl(String value) {
        return value == null ? "" : value;
    }

    private static final String BASE_CSS = "body{font-family:Segoe UI,Helvetica,Arial,sans-serif;margin:0;padding:0;background:#f8fafc;color:#0f172a;}" +
            "header{background:#1e293b;color:#f8fafc;padding:1.5rem;}" +
            "header h1{margin:0 0 .75rem;font-size:1.5rem;}" +
            ".controls{display:flex;flex-wrap:wrap;gap:.75rem;align-items:center;}" +
            ".controls label{display:flex;align-items:center;gap:.25rem;background:#334155;padding:.25rem .5rem;border-radius:.5rem;}" +
            ".controls input{width:1rem;height:1rem;}" +
            ".controls .hint{margin-left:auto;font-size:.85rem;opacity:.75;}" +
            "main{padding:1.5rem;}" +
            ".note{background:#ffffff;border:1px solid #cbd5f5;border-radius:.75rem;padding:1rem;line-height:1.6;box-shadow:0 10px 40px rgba(15,23,42,.08);}" +
            ".note span.anno{border-radius:.35rem;padding:.1rem .25rem;margin:0 .05rem;position:relative;cursor:pointer;box-shadow:0 1px 3px rgba(15,23,42,.15);}" +
            ".anno.layer-core{background:#fde68a;}" +
            ".anno.layer-wsd{background:#bfdbfe;}" +
            ".anno.layer-drug{background:#fbcfe8;}" +
            ".anno.layer-relations{background:#c7f9cc;}" +
            ".anno.layer-coref{background:#ddd6fe;}" +
            "body.hide-core span[data-layers~='core']," +
            "body.hide-wsd span[data-layers~='wsd']," +
            "body.hide-drug span[data-layers~='drug']," +
            "body.hide-relations span[data-layers~='relations']," +
            "body.hide-coref span[data-layers~='coref']{display:none !important;}" +
            ".summary{margin-top:2rem;}" +
            ".summary table{width:100%;border-collapse:collapse;font-size:.9rem;background:#ffffff;border:1px solid #cbd5e1;}" +
            ".summary th{background:#1e293b;color:#f8fafc;text-align:left;padding:.5rem;}" +
            ".summary td{padding:.5rem;border-top:1px solid #e2e8f0;vertical-align:top;}" +
            ".summary tr:nth-child(even){background:#f1f5f9;}" +
            "@media (prefers-color-scheme:dark){body{background:#0f172a;color:#e2e8f0;}header{background:#0f172a;}" +
            ".controls label{background:#1e293b;} .note{background:#1e293b;border-color:#334155;color:#e2e8f0;}" +
            ".summary table{background:#1e293b;color:#e2e8f0;border-color:#334155;}" +
            ".summary td{border-top:1px solid #334155;} .summary tr:nth-child(even){background:#273449;}}");

    private static final String BASE_JS = "function updateLayers(){const active=[];document.querySelectorAll('input[data-layer]').forEach(cb=>{" +
            "document.body.classList.toggle('hide-'+cb.dataset.layer,!cb.checked);});}" +
            "document.addEventListener('DOMContentLoaded',()=>{updateLayers();});";

    private static final class AnnotationLayers {
        final Set<String> layers = new LinkedHashSet<>();
        UmlsConcept concept;
    }

    private static final class AnnotationView {
        int begin;
        int end;
        List<String> layers;
        String coveredText;
        String tooltip;
        String typeName;
        String cui;
        String rxCui;
        String preferredText;
        String section;
        int polarity;
        int discoveryTechnique;

    }

    private static final class Event {
        final int offset;
        final boolean isEnd;
        final AnnotationView view;

        private Event(int offset, boolean isEnd, AnnotationView view) {
            this.offset = offset;
            this.isEnd = isEnd;
            this.view = view;
        }

        static Event start(int offset, AnnotationView view) {
            return new Event(offset, false, view);
        }

        static Event end(int offset, AnnotationView view) {
            return new Event(offset, true, view);
        }

        static int compare(Event a, Event b) {
            if (a.offset != b.offset) {
                return Integer.compare(a.offset, b.offset);
            }
            if (a.isEnd == b.isEnd) {
                return 0;
            }
            // end events before start events at the same position
            return a.isEnd ? -1 : 1;
        }
    }
}

