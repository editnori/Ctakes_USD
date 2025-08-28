package tools;

import java.io.*;
import java.nio.charset.StandardCharsets;
import java.sql.*;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * Export a BSV dictionary (CUI|TUI|term) from an HSQL rare-word DB built by cTAKES.
 *
 * Usage:
 *   java -cp "...:$CTAKES_HOME/lib/*:.build_tools" tools.ExportBsvFromHsql \
 *     -l $CTAKES_HOME/resources/.../FullClinical_AllTUIs.xml \
 *     -o dictionaries/FullClinical_AllTUIs/terms.bsv
 */
public class ExportBsvFromHsql {
    public static void main(String[] args) throws Exception {
        File dictXml = null;
        File outFile = null;
        for (int i=0; i<args.length; i++) {
            switch (args[i]) {
                case "-l": dictXml = new File(args[++i]); break;
                case "-o": outFile = new File(args[++i]); break;
                default:
                    System.err.println("Unknown arg: " + args[i]);
                    usage();
            }
        }
        if (dictXml == null || outFile == null) {
            usage();
            return;
        }

        String jdbcUrl = extractJdbcUrl(dictXml);
        if (jdbcUrl == null) {
            throw new IllegalArgumentException("Could not find jdbcUrl in xml: " + dictXml);
        }
        // Encourage HSQL to page large results to disk instead of RAM
        if (!jdbcUrl.contains("hsqldb.result_max_memory_rows")) {
            jdbcUrl = jdbcUrl + (jdbcUrl.contains(";") ? "" : ";") + "hsqldb.result_max_memory_rows=5000";
        }
        if (!jdbcUrl.contains("hsqldb.default_table_type")) {
            jdbcUrl = jdbcUrl + ";hsqldb.default_table_type=cached";
        }
        System.out.println("Connecting: " + jdbcUrl);

        // HSQL 2.x driver is present in cTAKES 6.0.0 lib
        Class.forName("org.hsqldb.jdbc.JDBCDriver");
        try (Connection conn = DriverManager.getConnection(jdbcUrl, "sa", "");
             BufferedWriter bw = writer(outFile)) {
            // Memory-savvy export: iterate TUIs by CUI (ordered), then fetch terms per CUI.
            // This avoids a massive JOIN result kept in memory by HSQL.
            final String tSql = "SELECT CUI, TUI FROM TUI ORDER BY CUI";
            final String cSql = "SELECT TEXT FROM CUI_TERMS WHERE CUI = ?";
            try (PreparedStatement tStmt = conn.prepareStatement(tSql, ResultSet.TYPE_FORWARD_ONLY, ResultSet.CONCUR_READ_ONLY);
                 PreparedStatement cStmt = conn.prepareStatement(cSql, ResultSet.TYPE_FORWARD_ONLY, ResultSet.CONCUR_READ_ONLY)) {
                tStmt.setFetchSize(10000);
                long rows = 0;
                long lastCui = Long.MIN_VALUE;
                java.util.ArrayList<String> tuis = new java.util.ArrayList<>(4);
                try (ResultSet tr = tStmt.executeQuery()) {
                    while (true) {
                        Long cui = null; String tui = null;
                        if (tr.next()) {
                            cui = tr.getLong(1);
                            tui = tr.getString(2);
                        }
                        if (cui == null || (lastCui != Long.MIN_VALUE && cui.longValue() != lastCui)) {
                            // Flush previous CUI
                            if (lastCui != Long.MIN_VALUE && !tuis.isEmpty()) {
                                cStmt.setLong(1, lastCui);
                                try (ResultSet cr = cStmt.executeQuery()) {
                                    while (cr.next()) {
                                        String text = cr.getString(1);
                                        if (text == null || text.isEmpty()) continue;
                                        for (int i = 0; i < tuis.size(); i++) {
                                            String rtui = tuis.get(i);
                                            bw.write(Long.toString(lastCui));
                                            bw.write('|');
                                            bw.write(rtui != null ? rtui : "");
                                            bw.write('|');
                                            bw.write(text);
                                            bw.newLine();
                                            rows++;
                                            if ((rows % 1_000_000) == 0) {
                                                System.out.println("Written BSV rows: " + rows);
                                            }
                                        }
                                    }
                                }
                            }
                            tuis.clear();
                            if (cui == null) break; // done
                        }
                        // accumulate
                        lastCui = cui;
                        tuis.add(tui);
                    }
                }
                System.out.println("Done. BSV rows: " + rows);
            }
        }
    }

    private static BufferedWriter writer(File f) throws IOException {
        File parent = f.getAbsoluteFile().getParentFile();
        if (parent != null) parent.mkdirs();
        return new BufferedWriter(new OutputStreamWriter(new FileOutputStream(f), StandardCharsets.UTF_8), 1 << 20);
    }

    private static String extractJdbcUrl(File xml) throws IOException {
        Pattern p = Pattern.compile("jdbcUrl\\\" value=\\\"([^\\\"]+)\\\"");
        try (BufferedReader br = new BufferedReader(new FileReader(xml))) {
            String line;
            while ((line = br.readLine()) != null) {
                Matcher m = p.matcher(line);
                if (m.find()) {
                    return m.group(1);
                }
            }
        }
        return null;
    }

    private static void usage() {
        System.err.println("Usage: tools.ExportBsvFromHsql -l <dictionary.xml> -o <out.bsv>");
        System.exit(1);
    }
}
