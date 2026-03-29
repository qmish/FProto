#!/usr/bin/env bash
# Profile FProto Gateway during load test.
#
# Prerequisites:
#   - Gateway running with pprof on :9090 (default metrics addr)
#   - k6 installed
#   - go tool pprof available
#
# Usage:
#   ./tests/load/profile.sh [gateway_host] [duration_seconds]

set -euo pipefail

GATEWAY_HOST="${1:-localhost}"
GATEWAY_PPROF="http://${GATEWAY_HOST}:9090"
DURATION="${2:-30}"
OUTPUT_DIR="tests/load/profiles/$(date +%Y%m%d-%H%M%S)"

mkdir -p "$OUTPUT_DIR"

echo "=== FProto Profiling ==="
echo "Gateway pprof: $GATEWAY_PPROF"
echo "Duration: ${DURATION}s"
echo "Output: $OUTPUT_DIR"
echo ""

echo "[1/4] CPU profile (${DURATION}s)..."
curl -sS "${GATEWAY_PPROF}/debug/pprof/profile?seconds=${DURATION}" \
  -o "${OUTPUT_DIR}/cpu.prof" &
CPU_PID=$!

echo "[2/4] Starting k6 load test in parallel..."
k6 run --duration "${DURATION}s" --vus 100 tests/load/k6-throughput.js \
  --summary-export "${OUTPUT_DIR}/k6-summary.json" 2>&1 | tee "${OUTPUT_DIR}/k6-output.txt" &
K6_PID=$!

wait $CPU_PID
echo "  CPU profile saved."

echo "[3/4] Heap profile..."
curl -sS "${GATEWAY_PPROF}/debug/pprof/heap" -o "${OUTPUT_DIR}/heap.prof"
echo "  Heap profile saved."

echo "[4/4] Goroutine profile..."
curl -sS "${GATEWAY_PPROF}/debug/pprof/goroutine" -o "${OUTPUT_DIR}/goroutine.prof"
echo "  Goroutine profile saved."

wait $K6_PID 2>/dev/null || true

echo ""
echo "=== Profiling complete ==="
echo "Analyze with:"
echo "  go tool pprof -http=:6060 ${OUTPUT_DIR}/cpu.prof"
echo "  go tool pprof -http=:6061 ${OUTPUT_DIR}/heap.prof"
echo "  go tool pprof -top ${OUTPUT_DIR}/goroutine.prof"
