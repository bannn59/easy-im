#!/usr/bin/env bash
# easy-im load test: single-node vs multi-node (2-3) via nginx LB.
#
# Usage:
#   ./scripts/loadtest.sh [duration] [connections] [threads]
#
# Defaults: duration=30s connections=200 threads=8. Same wrk params are used
# for single-node and multi-node so results are comparable.
#
# Prereqs: wrk, nginx installed; docker compose (postgres+kafka) up;
# backend migrations applied.
set -euo pipefail

cd "$(dirname "$0")/.."

DURATION="${1:-30s}"
CONNECTIONS="${2:-200}"
THREADS="${3:-8}"
BASE="${BASE:-http://localhost}"
API_PORT="${API_PORT:-8081}"      # first node port (single-node baseline)
NODES="${NODES:-3}"               # node count for multi-node phase
METRICS_PORT_START="${METRICS_PORT_START:-9090}"
TOKENS="${TOKENS:-/tmp/loadtest_tokens.json}"
REPORT_DIR="research"
API_BIN="${API_BIN:-/tmp/easyim-api-bin}"
mkdir -p "$REPORT_DIR"
log() { echo -e "\n\033[1;34m[$1]\033[0m $2"; }

# --- prebuild api binary ----------------------------------------------------

log "build" "building api binary → $API_BIN"
go build -o "$API_BIN" ./cmd/api

# --- environment checks -----------------------------------------------------

: "${DATABASE_URL:?set DATABASE_URL (postgres://easyim:easyim@localhost:5433/easyim?sslmode=disable)}"
: "${AUTH_JWT_SECRET:?set AUTH_JWT_SECRET}"
: "${KAFKA_BROKERS:=localhost:19092}"

# --- helpers ---------------------------------------------------------------

# run_wrk <label> <url> <method> <body>
run_wrk() {
  local label="$1" url="$2" method="$3" body="${4:-}"
  local outfile="$REPORT_DIR/wrk_${label}.txt"
  log "wrk" "$label — $method $url (${THREADS}t/${CONNECTIONS}c/${DURATION})"
  env WRK_METHOD="$method" WRK_BODY="$body" WRK_COOKIE="$COOKIE" \
    wrk -t"$THREADS" -c"$CONNECTIONS" -d"$DURATION" --latency \
    -s scripts/wrk_auth.lua "$url" | tee "$outfile"
}

# cookie_for <n> — nth token's session cookie from the token file.
cookie_for() {
  local n="$1"
  python3 -c "
import json,sys
d=json.load(open('$TOKENS'))
print(d[$n]['cookie'])
"
}

# first_conversation_id — the hub's first conversation id (for history wrk).
first_conversation_id() {
  curl -s -H "Cookie: $COOKIE" "$BASE:$API_PORT/v1/conversations" \
    | python3 -c "
import json,sys
d=json.load(sys.stdin)
print(d['conversations'][0]['id'])
"
}

# metrics_snapshot <node_index> <label>
metrics_snapshot() {
  local idx="$1" label="$2"
  local port=$((METRICS_PORT_START + idx))
  log "metrics" "$label :127.0.0.1:$port"
  curl -s "http://127.0.0.1:$port/metrics" \
    | grep -E "easyim_(http_requests_total|ws_online_conns|messages_sent_total|kafka_publish_total|fanout_events_total)" \
    | head -30 | tee "$REPORT_DIR/metrics_${label}.txt"
}

# start_api <port> — launch one api node (compiled binary) with its own
# metrics port. Uses a prebuilt binary so $! is the real server PID (go run
# spawns a child process, breaking kill-by-pid).
start_api() {
  local port="$1"
  local mport=$((port - 8081 + METRICS_PORT_START))  # 8081→9090, 8082→9091, ...
  DATABASE_URL="$DATABASE_URL" \
  AUTH_JWT_SECRET="$AUTH_JWT_SECRET" \
  AUTH_DEV_INSECURE=1 \
  KAFKA_BROKERS="$KAFKA_BROKERS" \
  METRICS_ADDR=":$mport" \
  PORT="$port" "$API_BIN" >/tmp/easyim-api-$port.log 2>&1 &
  echo $!
}

# wait_port <port> — poll until the port accepts connections.
wait_port() {
  local port="$1"
  for i in $(seq 1 30); do
    if curl -s -o /dev/null "http://127.0.0.1:$port/healthz"; then return 0; fi
    sleep 0.5
  done
  log "error" "port $port never became ready"
  return 1
}

# --- phase 0: start node 0 + prepare fixtures --------------------------------

log "phase" "starting node 0 on :$API_PORT"
NODE_PIDS=""
NODE_PIDS="$NODE_PIDS $(start_api "$API_PORT")"
wait_port "$API_PORT"
log "nodes" "node 0 ready"

if [ ! -f "$TOKENS" ]; then
  log "setup" "seeding load-test fixtures (users/friends/conversations/messages)"
  go run ./scripts -base "$BASE:$API_PORT" -users 10 -out "$TOKENS"
else
  log "setup" "using existing token file $TOKENS"
fi
COOKIE=$(cookie_for 0)
CONV_ID=$(first_conversation_id)
log "setup" "hub cookie + conversation $CONV_ID ready"

# --- phase 1: single-node baseline ------------------------------------------

log "phase" "SINGLE-NODE baseline (port $API_PORT)"

SINGLE="$BASE:$API_PORT"
run_wrk "single_conversations" "$SINGLE/v1/conversations" GET
run_wrk "single_friends"       "$SINGLE/v1/friends"       GET
run_wrk "single_login"         "$SINGLE/v1/auth/login"    POST '{"email":"lt_000@test.local","password":"pass1234"}'
run_wrk "single_send"          "$SINGLE/v1/conversations/$CONV_ID/messages" POST '{"client_msg_id":"wrk-__SEQ__","body":"load test message"}'
run_wrk "single_history"       "$SINGLE/v1/conversations/$CONV_ID/messages" GET
metrics_snapshot 0 "single"

# --- phase 2: multi-node via nginx LB ---------------------------------------

log "phase" "MULTI-NODE via nginx :8080 (${NODES} nodes)"
# Start nodes 1..N-1 (node 0 from phase 1 keeps running).
for i in $(seq 1 $((NODES-1))); do
  port=$((API_PORT + i))
  pid=$(start_api "$port")
  NODE_PIDS="$NODE_PIDS $pid"
  log "nodes" "started node $i on :$port (pid $pid)"
done
for i in $(seq 1 $((NODES-1))); do
  wait_port "$((API_PORT + i))"
done

# nginx LB config with as many upstream servers as NODES.
python3 - "$NODES" "$API_PORT" <<'PY'
import sys, re
n, base = int(sys.argv[1]), int(sys.argv[2])
ups = "\n".join(f"    server 127.0.0.1:{base+i};" for i in range(n))
conf = open('../deploy/nginx-loadtest.conf').read()
conf = re.sub(r"upstream easyim_api \{.*?\}", f"upstream easyim_api {{\n{ups}\n}}", conf, flags=re.S)
open('/tmp/nginx-loadtest.conf', 'w').write(conf)
PY
nginx -c /tmp/nginx-loadtest.conf -g "pid /tmp/nginx.pid;" -p /tmp/ 2>/dev/null
log "lb" "nginx LB started on :8080 -> $NODES nodes"

MULTI="$BASE:8080"
run_wrk "multi_conversations" "$MULTI/v1/conversations" GET
run_wrk "multi_friends"       "$MULTI/v1/friends"       GET
run_wrk "multi_login"         "$MULTI/v1/auth/login"    POST '{"email":"lt_000@test.local","password":"pass1234"}'
run_wrk "multi_send"          "$MULTI/v1/conversations/$CONV_ID/messages" POST '{"client_msg_id":"wrk-__SEQ__","body":"load test message"}'
run_wrk "multi_history"       "$MULTI/v1/conversations/$CONV_ID/messages" GET
for i in $(seq 0 $((NODES-1))); do
  metrics_snapshot "$i" "multi_node$i"
done

# --- cleanup ----------------------------------------------------------------

log "cleanup" "stopping nginx and extra nodes"
nginx -c /tmp/nginx-loadtest.conf -g "pid /tmp/nginx.pid;" -s quit -p /tmp/ 2>/dev/null || true
for pid in $NODE_PIDS; do kill "$pid" 2>/dev/null || true; done
sleep 1

# --- summary ---------------------------------------------------------------

log "summary" "wrk outputs in $REPORT_DIR/wrk_*.txt, metrics in $REPORT_DIR/metrics_*.txt"
echo "Single vs multi RPS:"
for f in conversations friends login send history; do
  echo "  $f:"
  grep -E "Requests/sec" "$REPORT_DIR"/wrk_single_$f.txt | sed 's/^/    single /'
  grep -E "Requests/sec" "$REPORT_DIR"/wrk_multi_$f.txt | sed 's/^/    multi  /'
done
