#!/usr/bin/env bash
# Build the three artifacts the browser playground needs:
#   docs/assets/playground/capy.wasm      (Go compiler compiled to wasm)
#   docs/assets/playground/wasm_exec.js   (the wasm runtime — comes with Go)
#   docs/assets/playground/samples.json   (bundled curated sample sources)
#
# CI runs this on every docs deploy. Run locally to preview the playground
# at docs/assets/playground/index.html via:
#   python3 -m http.server -d docs/assets/playground/ 8000
set -euo pipefail
cd "$(dirname "$0")/.."

OUT=docs/assets/playground

echo "[playground] building capy.wasm…"
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o "$OUT/capy.wasm" ./cmd/capy-wasm

echo "[playground] copying wasm_exec.js…"
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$OUT/wasm_exec.js"

echo "[playground] bundling samples.json…"
go run ./cmd/playground-bundle > "$OUT/samples.json"

echo "[playground] done:"
ls -lh "$OUT/" | tail -n +2
