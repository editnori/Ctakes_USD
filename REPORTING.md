Reporting Tools (ExcelXmlReport, PerDocClinicalCsv)

Quick start
- Preflight: `bash scripts/report_preflight.sh`
- Build + run: `bash scripts/build_report.sh -o <out_dir> -w <workbook.xlsx> [-M summary|full|csv]`

Environment
- `CTAKES_HOME`: Optional. Defaults to `apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0` under the repo if not set. Not required to compile these tools.
- Java 11+: Verified with `javac -version` and `java -version`.

Notes
- Outputs are XLSX-only. Legacy Excel 2003 XML writer has been removed to reduce confusion.
- Workbook header fill color is auto-selected per pipeline (based on output folder name).

Gradle usage (optional)
- Build: `./gradlew classes`
- Run: `./gradlew run --args="-o <out_dir> -w <workbook.xlsx> -M summary"`
- Jar: `./gradlew reportJar` (produces `build/libs/excel-report-1.0.0.jar`)



