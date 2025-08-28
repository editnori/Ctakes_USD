#!/usr/bin/env bash
set -euo pipefail

# cTAKES Cluster Runner (sharded, parallel Piper runs)
#
# Usage:
#   export CTAKES_HOME=/workspace/ctakes/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0
#   export IN=/workspace/ctakes/SD5000_1
#   ./scripts/ctakes_cluster_run.sh
#
# Tunables via env:
#   RUNNERS=22            # number of parallel shards
#   XMX_MB=6144           # heap for -Xms/-Xmx (MB)
#   HSQLJAR=...           # optional extra HSQL jar
#   SLF4J=...             # optional extra SLF4J jar
#   APIKEY=...            # UMLS API key (optional)
#   PIPER=...             # .piper path (required)
#   DBTEMPLATE=...        # template XML (umls_full_gui.xml)
#
# Notes:
# - Creates per-runner HSQL stores in /dev/shm
# - Writes outputs and per-runner XMLs under $OUT

pkill -f PiperFileRunner || true

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME=${CTAKES_HOME:?set CTAKES_HOME}
IN=${IN:?set IN}
RUNNERS=${RUNNERS:-22}
XMX_MB=${XMX_MB:-6144}
HSQLJAR=${HSQLJAR:-}
SLF4J=${SLF4J:-}
APIKEY=${APIKEY:-${UMLS_KEY:-6370dcdd-d438-47ab-8749-5a8fb9d013f2}}
# Default to local WSD-enabled TS Fast pipeline; override by exporting PIPER
PIPER=${PIPER:-$BASE_DIR/pipelines/wsd/TsDefaultFastPipeline_WSD.piper}

# Default dictionary XML template: use builder output under cTAKES resources unless overridden
PROPS="$BASE_DIR/docs/builder_full_clinical.properties"
DICT_NAME=$(awk -F= '/^\s*dictionary.name\s*=/{print $2}' "$PROPS" | tr -d '\r' | xargs)
DBTEMPLATE=${DBTEMPLATE:-$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast/${DICT_NAME}.xml}

TS=$(date +%s)
SHARDS="/workspace/ctakes/shards_$TS"; mkdir -p "$SHARDS"
OUT="/workspace/ctakes/output/cluster_$TS"; mkdir -p "$OUT"

echo "Sharding inputs from: $IN"
find "$IN" -type f -name "*.txt" | nl -ba | awk -v N="$RUNNERS" -v S="$SHARDS" '
  { g=$1%N; printf "mkdir -p %s/%03d; ln \"%s\" \"%s/%03d/\"\n",S,g,$2,S,g }' | bash

echo "Launching $RUNNERS runners..."
for i in $(seq -f "%03g" 0 $((RUNNERS-1))); do
  shard="$SHARDS/$i"; [ -d "$shard" ] || continue
  outdir="$OUT/shard_$i"; mkdir -p "$outdir"

  workdb="/dev/shm/${DICT_NAME}_w$i"; rm -rf "$workdb"; mkdir -p "$workdb"
  # copy script/properties if present adjacent to template (optional)
  for ext in script properties; do
    src="${DBTEMPLATE%.xml}.$ext"; [ -f "$src" ] && cp -f "$src" "$workdb/${DICT_NAME}.$ext" || true
  done

  xml="$outdir/${DICT_NAME}_$i.xml"
  sed -e 's|jdbcDriver" value="[^"]*|jdbcDriver" value="org.hsqldb.jdbc.JDBCDriver|g' \
      -e "s|jdbcUrl\" value=\"[^\"]*|jdbcUrl\" value=\"jdbc:hsqldb:file:$workdb/${DICT_NAME}|g" \
      "$DBTEMPLATE" > "$xml"

  CP="$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*:$BASE_DIR/.build_tools"
  [ -n "$HSQLJAR" ] && CP="$CP:$HSQLJAR"
  [ -n "$SLF4J" ] && CP="$CP:$SLF4J"

  (
    stdbuf -oL -eL java ${APIKEY:+-Dctakes.umls_apikey=$APIKEY} \
      -Dorg.slf4j.simpleLogger.defaultLogLevel=info \
      -Xms${XMX_MB}m -Xmx${XMX_MB}m -XX:+UseG1GC -XX:ParallelGCThreads=2 -XX:ConcGCThreads=1 \
      -cp "$CP" \
      org.apache.ctakes.core.pipeline.PiperFileRunner \
      -p "$PIPER" -i "$shard" -o "$outdir" -l "$xml" \
      | sed -u "s/^/[R$i] /" | tee "$outdir/run.log"
  ) &
done

echo "Tailing logs in: $OUT"
tail -F "$OUT"/shard_*/run.log
