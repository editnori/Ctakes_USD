package tools;

import java.io.BufferedReader;
import java.io.File;
import java.io.FileInputStream;
import java.io.IOException;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.PreparedStatement;
import java.sql.SQLException;
import java.sql.Statement;
import java.util.HashSet;
import java.util.Locale;
import java.util.Set;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * Adds an RXNORM table to a cTAKES fast dictionary HSQLDB.
 * Each row contains the numeric CUI and the associated RxCUI (as provided by MRCONSO.RRF).
 *
 * Usage:
 *   java ... tools.DictionaryRxnormAugmenter -l <dictionary.xml> -u <umls_root>
 *
 * The tool expects MRCONSO.RRF under <umls_root>/META or directly under <umls_root>.
 */
public class DictionaryRxnormAugmenter {

    private static final Pattern JDBC_PATTERN = Pattern.compile("key=\"jdbcUrl\" value=\"([^\"]+)\"");

    public static void main(String[] args) throws Exception {
        File dictXml = null;
        File umlsRoot = null;
        for (int i = 0; i < args.length; i++) {
            switch (args[i]) {
                case "-l":
                    dictXml = new File(args[++i]);
                    break;
                case "-u":
                    umlsRoot = new File(args[++i]);
                    break;
                default:
                    usage();
                    return;
            }
        }
        if (dictXml == null || umlsRoot == null) {
            usage();
            return;
        }
        if (!dictXml.isFile()) {
            throw new IllegalArgumentException("Dictionary XML not found: " + dictXml);
        }
        File mrconso = locateMrconso(umlsRoot);
        if (!mrconso.isFile()) {
            throw new IllegalArgumentException("MRCONSO.RRF not found under " + umlsRoot.getAbsolutePath());
        }

        String jdbcUrl = extractJdbcUrl(dictXml);
        if (jdbcUrl == null || jdbcUrl.isEmpty()) {
            throw new IllegalArgumentException("Could not locate jdbcUrl in xml: " + dictXml);
        }
        if (!jdbcUrl.contains("hsqldb.default_table_type")) {
            jdbcUrl = jdbcUrl + ";hsqldb.default_table_type=cached";
        }
        if (!jdbcUrl.contains("hsqldb.result_max_memory_rows")) {
            jdbcUrl = jdbcUrl + ";hsqldb.result_max_memory_rows=5000";
        }

        System.out.println("Connecting to dictionary: " + jdbcUrl);
        System.out.println("Using MRCONSO: " + mrconso.getAbsolutePath());

        Class.forName("org.hsqldb.jdbcDriver");
        try (Connection conn = DriverManager.getConnection(jdbcUrl, "sa", "")) {
            conn.setAutoCommit(false);
            createTable(conn);
            insertRxnormCodes(conn, mrconso);
            try (Statement shutdown = conn.createStatement()) {
                shutdown.execute("SHUTDOWN");
            }
        }
        System.out.println("Done augmenting RxNorm table.");
    }

    private static void createTable(Connection conn) throws SQLException {
        try (Statement st = conn.createStatement()) {
            st.execute("DROP TABLE IF EXISTS RXNORM_CODES");
            st.execute("DROP TABLE IF EXISTS RXNORMCODES");
            st.execute("DROP TABLE IF EXISTS RXNORM");
            st.execute("CREATE CACHED TABLE RXNORM (CUI BIGINT, RXCUI VARCHAR(32))");
            st.execute("CREATE INDEX IDX_RXNORM ON RXNORM (CUI)");
        }
    }

    private static void insertRxnormCodes(Connection conn, File mrconso) throws IOException, SQLException {
        final String insertSql = "INSERT INTO RXNORM (CUI, RXCUI) VALUES (?, ?)";
        final Set<String> seen = new HashSet<>(256_000);
        long totalLines = 0;
        long inserted = 0;
        try (PreparedStatement ps = conn.prepareStatement(insertSql);
             BufferedReader reader = new BufferedReader(new InputStreamReader(new FileInputStream(mrconso), StandardCharsets.UTF_8), 1 << 20)) {
            String line;
            while ((line = reader.readLine()) != null) {
                totalLines++;
                // Skip comments (unlikely) and empty lines
                if (line.isEmpty() || line.charAt(0) == '#') {
                    continue;
                }
                String[] fields = line.split("[|]", -1);
                if (fields.length < 14) {
                    continue;
                }
                if (!"RXNORM".equals(fields[11])) {
                    continue;
                }
                String cuiStr = fields[0];
                String code = fields[13];
                if (code == null || code.isEmpty()) {
                    continue;
                }
                long cuiLong;
                try {
                    cuiLong = parseCui(cuiStr);
                } catch (NumberFormatException nfe) {
                    System.err.println("[warn] Unable to parse CUI: " + cuiStr);
                    continue;
                }
                String key = cuiLong + "|" + code;
                if (!seen.add(key)) {
                    continue;
                }
                ps.setLong(1, cuiLong);
                ps.setString(2, code);
                ps.addBatch();
                inserted++;
                if ((inserted % 1_000) == 0) {
                    ps.executeBatch();
                    conn.commit();
                }
                if ((totalLines % 500_000) == 0) {
                    System.out.println(String.format(Locale.ROOT, "Processed %,d MRCONSO rows (%,d inserts)", totalLines, inserted));
                }
            }
            ps.executeBatch();
            conn.commit();
        }
        System.out.println(String.format(Locale.ROOT, "Inserted %,d RxNorm codes", inserted));
    }

    private static long parseCui(String cuiStr) {
        if (cuiStr == null || cuiStr.length() < 2) {
            throw new NumberFormatException("Invalid CUI: " + cuiStr);
        }
        if (cuiStr.charAt(0) == 'C' || cuiStr.charAt(0) == 'c') {
            return Long.parseLong(cuiStr.substring(1));
        }
        return Long.parseLong(cuiStr);
    }

    private static File locateMrconso(File umlsRoot) {
        File meta = new File(umlsRoot, "META/MRCONSO.RRF");
        if (meta.isFile()) {
            return meta;
        }
        File direct = new File(umlsRoot, "MRCONSO.RRF");
        if (direct.isFile()) {
            return direct;
        }
        return meta; // will fail later with clear message
    }

    private static String extractJdbcUrl(File xml) throws IOException {
        try (BufferedReader br = new BufferedReader(new InputStreamReader(new FileInputStream(xml), StandardCharsets.UTF_8))) {
            String line;
            while ((line = br.readLine()) != null) {
                Matcher m = JDBC_PATTERN.matcher(line);
                if (m.find()) {
                    return m.group(1);
                }
            }
        }
        return null;
    }

    private static void usage() {
        System.err.println("Usage: tools.DictionaryRxnormAugmenter -l <dictionary.xml> -u <umls_dir>");
    }
}
