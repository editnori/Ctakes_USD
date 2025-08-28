#!/usr/bin/env bash
set -euo pipefail

# Build a local HSQL UMLS database for YTEX WSD from UMLS META RRFs.
# Usage: scripts/build_ytex_umls_db.sh [META_DIR] [OUT_DB_BASE]

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-$BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0}"

META_DIR="${1:-$BASE_DIR/umls_loader/META}"
OUT_DB_BASE="${2:-/tmp/ctakes_ytex_umls/ytex_umls}"

mkdir -p "$(dirname "$OUT_DB_BASE")"
mkdir -p "$BASE_DIR/.build_tools"

echo "META_DIR:    $META_DIR"
echo "OUT_DB_BASE: $OUT_DB_BASE"

if [[ ! -f "$META_DIR/MRREL.RRF" || ! -f "$META_DIR/MRCONSO.RRF" || ! -f "$META_DIR/MRSTY.RRF" ]]; then
  echo "Missing MRREL.RRF / MRCONSO.RRF / MRSTY.RRF in META_DIR" >&2
  exit 1
fi

# Compile loader
find "$BASE_DIR/tools" -name "*.java" -print0 | xargs -0 javac -cp "$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*" -d "$BASE_DIR/.build_tools"

JAVA_CP="$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*:$BASE_DIR/.build_tools"

echo "Loading UMLS RRF into HSQL (ENG only) …"
java -Xms1g -Xmx4g -cp "$JAVA_CP" tools.ytex.LoadUmlsForYtex -m "$META_DIR" -d "$OUT_DB_BASE"

# Write ytex.properties to point at our DB
YPROPS="$CTAKES_HOME/resources/org/apache/ctakes/ytex/ytex.properties"
mkdir -p "$(dirname "$YPROPS")"
cat > "$YPROPS" <<EOF
# YTEX configuration for local HSQL
db.schema=PUBLIC
db.username=sa
db.password=
db.url=jdbc:hsqldb:file:${OUT_DB_BASE}
db.driver=org.hsqldb.jdbc.JDBCDriver
hibernate.dialect=org.hibernate.dialect.HSQLDialect
db.isolationLevel=READ_UNCOMMITTED
db.type=hsql
ytex.conceptGraphName=sct-rxnorm
ytex.conceptPreload=true
# Use default beanRefContext from the JAR
# ytex.beanRefContext not overridden; defaults to classpath* jar config
EOF

echo "Wrote YTEX props: $YPROPS"
echo "Done. You can now run the WSD pipeline."

