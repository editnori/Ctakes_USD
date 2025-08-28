package tools.ytex;

import java.io.*;
import java.nio.charset.StandardCharsets;
import java.sql.*;
import java.util.*;

/**
 * Minimal UMLS loader for YTEX-compatible WSD, targeting HSQLDB.
 * Loads MRREL, MRCONSO (ENG only), MRSTY into an HSQL file DB.
 *
 * Usage:
 *   java -cp "...:$CTAKES_HOME/lib/*:.build_tools" tools.ytex.LoadUmlsForYtex \
 *     -m /path/to/umls_loader/META \
 *     -d /path/to/out/ytex_umls
 */
public class LoadUmlsForYtex {
    public static void main(String[] args) throws Exception {
        File metaDir = null; File outDbBase = null; boolean engOnly = true;
        for (int i = 0; i < args.length; i++) {
            switch (args[i]) {
                case "-m": metaDir = new File(args[++i]); break;
                case "-d": outDbBase = new File(args[++i]); break;
                case "--all-langs": engOnly = false; break;
                default: usage("Unknown arg: " + args[i]);
            }
        }
        if (metaDir == null || outDbBase == null) usage("Missing args");
        if (!new File(metaDir, "MRREL.RRF").isFile()) usage("META missing MRREL.RRF");
        if (!new File(metaDir, "MRCONSO.RRF").isFile()) usage("META missing MRCONSO.RRF");
        if (!new File(metaDir, "MRSTY.RRF").isFile()) usage("META missing MRSTY.RRF");

        outDbBase.getParentFile().mkdirs();
        String jdbcUrl = "jdbc:hsqldb:file:" + outDbBase.getAbsolutePath()
                + ";hsqldb.default_table_type=cached;hsqldb.lock_file=false;hsqldb.result_max_memory_rows=50000";
        Class.forName("org.hsqldb.jdbc.JDBCDriver");
        try (Connection conn = DriverManager.getConnection(jdbcUrl, "sa", "")) {
            conn.setAutoCommit(false);
            ddl(conn);
            loadMRREL(conn, new File(metaDir, "MRREL.RRF"));
            loadMRSTY(conn, new File(metaDir, "MRSTY.RRF"));
            loadMRCONSO(conn, new File(metaDir, "MRCONSO.RRF"), engOnly);
            // index helpful for joins
            try (Statement st = conn.createStatement()) {
                st.execute("CREATE INDEX IF NOT EXISTS IDX_MRREL_CUI1 ON MRREL(CUI1)");
                st.execute("CREATE INDEX IF NOT EXISTS IDX_MRREL_CUI2 ON MRREL(CUI2)");
                st.execute("CREATE INDEX IF NOT EXISTS IDX_MRCONSO_CUI ON MRCONSO(CUI)");
                st.execute("CREATE INDEX IF NOT EXISTS IDX_MRSTY_CUI ON MRSTY(CUI)");
            }
            conn.commit();
        }
        System.out.println("Done. DB: " + outDbBase.getAbsolutePath());
    }

    private static void ddl(Connection conn) throws SQLException {
        try (Statement st = conn.createStatement()) {
            st.execute("CREATE TABLE IF NOT EXISTS MRREL (" +
                    "RUI VARCHAR(20), CUI1 VARCHAR(8), CUI2 VARCHAR(8), REL VARCHAR(20), SAB VARCHAR(50), RELA VARCHAR(100))");
            st.execute("CREATE TABLE IF NOT EXISTS MRCONSO (" +
                    "AUI VARCHAR(20) PRIMARY KEY, CUI VARCHAR(8), LAT VARCHAR(8), TS VARCHAR(4), LUI VARCHAR(20), STT VARCHAR(8), " +
                    "SUI VARCHAR(20), ISPREF VARCHAR(4), SAUI LONGVARCHAR, SCUI LONGVARCHAR, SDUI LONGVARCHAR, SAB VARCHAR(50), " +
                    "TTY VARCHAR(20), CODE LONGVARCHAR, STR LONGVARCHAR, SRL VARCHAR(10), SUPPRESS VARCHAR(8), CVF VARCHAR(50))");
            st.execute("CREATE TABLE IF NOT EXISTS MRSTY (" +
                    "CUI VARCHAR(8), TUI VARCHAR(4), STN VARCHAR(50), STY VARCHAR(255), ATUI VARCHAR(20), CVF VARCHAR(50))");
            // ensure STR has sufficient capacity in case table pre-existed
            String[] widenCols = {"STR","SCUI","SDUI","SAUI","CODE"};
            for (String col : widenCols) {
                try (Statement st2 = conn.createStatement()) {
                    st2.execute("ALTER TABLE MRCONSO ALTER COLUMN " + col + " SET DATA TYPE LONGVARCHAR");
                } catch (SQLException ignore) {}
            }
        }
    }

    private static void loadMRREL(Connection conn, File rrf) throws IOException, SQLException {
        System.out.println("Loading MRREL: " + rrf);
        // Skip if already populated
        long existing = countRows(conn, "MRREL");
        if (existing > 0) {
            System.out.println("MRREL already populated (rows=" + existing + ") - skipping load");
            return;
        }
        try (BufferedReader br = reader(rrf);
             PreparedStatement ps = conn.prepareStatement(
                     "INSERT INTO MRREL(RUI,CUI1,CUI2,REL,SAB,RELA) VALUES(?,?,?,?,?,?)")) {
            String line; int batch = 0; long rows = 0;
            while ((line = br.readLine()) != null) {
                String[] f = splitRrf(line, 15);
                // UMLS MRREL order: CUI1|AUI1|STYPE1|REL|CUI2|AUI2|STYPE2|RELA|RUI|SAB|SL|RG|DIR|SUPPRESS|CVF
                String cui1 = f[0]; String rel = f[3]; String cui2 = f[4]; String rela = f[7]; String rui = f[8]; String sab = f[9];
                ps.setString(1, nz(rui));
                ps.setString(2, nz(cui1));
                ps.setString(3, nz(cui2));
                ps.setString(4, nz(rel));
                ps.setString(5, nz(sab));
                ps.setString(6, nz(rela));
                ps.addBatch();
                if (++batch >= 5000) { ps.executeBatch(); conn.commit(); batch = 0; }
                if (++rows % 1_000_000 == 0) System.out.println("  MRREL rows: " + rows);
            }
            if (batch > 0) { ps.executeBatch(); conn.commit(); }
            System.out.println("MRREL rows total: " + rows);
        }
    }

    private static void loadMRSTY(Connection conn, File rrf) throws IOException, SQLException {
        System.out.println("Loading MRSTY: " + rrf);
        // Skip if already populated
        long existing = countRows(conn, "MRSTY");
        if (existing > 0) {
            System.out.println("MRSTY already populated (rows=" + existing + ") - skipping load");
            return;
        }
        try (BufferedReader br = reader(rrf);
             PreparedStatement ps = conn.prepareStatement(
                     "INSERT INTO MRSTY(CUI,TUI,STN,STY,ATUI,CVF) VALUES(?,?,?,?,?,?)")) {
            String line; int batch = 0; long rows = 0;
            while ((line = br.readLine()) != null) {
                String[] f = splitRrf(line, 6);
                // CUI|TUI|STN|STY|ATUI|CVF
                ps.setString(1, nz(f[0]));
                ps.setString(2, nz(f[1]));
                ps.setString(3, nz(f[2]));
                ps.setString(4, nz(f[3]));
                ps.setString(5, nz(f[4]));
                ps.setString(6, nz(f[5]));
                ps.addBatch();
                if (++batch >= 5000) { ps.executeBatch(); conn.commit(); batch = 0; }
                if (++rows % 1_000_000 == 0) System.out.println("  MRSTY rows: " + rows);
            }
            if (batch > 0) { ps.executeBatch(); conn.commit(); }
            System.out.println("MRSTY rows total: " + rows);
        }
    }

    private static void loadMRCONSO(Connection conn, File rrf, boolean engOnly) throws IOException, SQLException {
        System.out.println("Loading MRCONSO (" + (engOnly ? "ENG only" : "all langs") + "): " + rrf);
        // Reset MRCONSO in case of prior partial/failed load
        try (Statement st = conn.createStatement()) {
            st.execute("DELETE FROM MRCONSO");
            conn.commit();
        } catch (SQLException ignore) {}
        // Use MERGE to avoid duplicate AUI violations (some UMLS releases may repeat AUI rows)
        final String mergeSql =
                "MERGE INTO MRCONSO T " +
                "USING (VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)) V(" +
                "AUI,CUI,LAT,TS,LUI,STT,SUI,ISPREF,SAUI,SCUI,SDUI,SAB,TTY,CODE,STR,SRL,SUPPRESS,CVF) " +
                "ON T.AUI = V.AUI " +
                "WHEN NOT MATCHED THEN INSERT (AUI,CUI,LAT,TS,LUI,STT,SUI,ISPREF,SAUI,SCUI,SDUI,SAB,TTY,CODE,STR,SRL,SUPPRESS,CVF) " +
                "VALUES (V.AUI,V.CUI,V.LAT,V.TS,V.LUI,V.STT,V.SUI,V.ISPREF,V.SAUI,V.SCUI,V.SDUI,V.SAB,V.TTY,V.CODE,V.STR,V.SRL,V.SUPPRESS,V.CVF)";
        try (BufferedReader br = reader(rrf);
             PreparedStatement ps = conn.prepareStatement(mergeSql)) {
            String line; int batch = 0; long rows = 0; long kept = 0;
            while ((line = br.readLine()) != null) {
                String[] f = splitRrf(line, 18);
                // Order per UMLS: CUI|LAT|TS|LUI|STT|SUI|ISPREF|AUI|SAUI|SCUI|SDUI|SAB|TTY|CODE|STR|SRL|SUPPRESS|CVF
                String lat = f[1];
                if (engOnly && (lat == null || !lat.equals("ENG"))) continue;
                ps.setString(1, nz(f[7])); // AUI
                ps.setString(2, nz(f[0])); // CUI
                ps.setString(3, nz(f[1])); // LAT
                ps.setString(4, nz(f[2])); // TS
                ps.setString(5, nz(f[3])); // LUI
                ps.setString(6, nz(f[4])); // STT
                ps.setString(7, nz(f[5])); // SUI
                ps.setString(8, nz(f[6])); // ISPREF
                ps.setString(9, nz(f[8])); // SAUI
                ps.setString(10, nz(f[9])); // SCUI
                ps.setString(11, nz(f[10])); // SDUI
                ps.setString(12, nz(f[11])); // SAB
                ps.setString(13, nz(f[12])); // TTY
                ps.setString(14, nz(f[13])); // CODE
                ps.setString(15, nz(f[14])); // STR
                ps.setString(16, nz(f[15])); // SRL
                ps.setString(17, nz(f[16])); // SUPPRESS
                ps.setString(18, nz(f[17])); // CVF
                ps.addBatch();
                kept++;
                if (++batch >= 2000) { ps.executeBatch(); conn.commit(); batch = 0; }
                if (++rows % 1_000_000 == 0) System.out.println("  MRCONSO lines: " + rows + ", kept: " + kept);
            }
            if (batch > 0) { ps.executeBatch(); conn.commit(); }
            System.out.println("MRCONSO kept rows total: " + kept);
        }
    }

    private static long countRows(Connection conn, String table) throws SQLException {
        try (Statement st = conn.createStatement();
             ResultSet rs = st.executeQuery("SELECT COUNT(*) FROM " + table)) {
            if (rs.next()) return rs.getLong(1);
            return 0L;
        }
    }

    private static BufferedReader reader(File f) throws IOException {
        return new BufferedReader(new InputStreamReader(new FileInputStream(f), StandardCharsets.UTF_8), 1 << 20);
    }

    private static String[] splitRrf(String line, int expected) {
        // RRF is pipe-delimited with trailing pipe; simple split is fine here
        String[] a = line.split("\\|", -1);
        if (a.length < expected) {
            String[] b = new String[expected];
            System.arraycopy(a, 0, b, 0, a.length);
            for (int i = a.length; i < expected; i++) b[i] = "";
            return b;
        }
        return a;
    }

    private static String nz(String s) { return (s == null || s.isEmpty()) ? null : s; }

    private static void usage(String m) {
        System.err.println(m);
        System.err.println("Usage: tools.ytex.LoadUmlsForYtex -m <META dir> -d <out db base> [--all-langs]");
        System.exit(1);
    }
}
