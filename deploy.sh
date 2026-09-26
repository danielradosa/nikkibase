#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

echo "==> engine"
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o web/src/generated/nikkibase.wasm ./wasm

echo "==> checks"
go vet ./...
go test ./... >/dev/null
npm --prefix web ci --silent
npm --prefix web test >/dev/null

echo "==> site"
npm --prefix web run build >/dev/null

for required in web/dist/index.html web/dist/assets/nikkibase-*.wasm web/dist/assets/nikkibase-*.wasm.br web/dist/third-party-notices.txt web/dist/privacy.txt web/dist/data/index.json web/dist/keystream.bin; do
  [ -e "$required" ] || { echo "missing $required -- see .ai/research/sources/README.md" >&2; exit 1; }
done
version=$(python3 -c "import json;print(json.load(open('web/dist/data/index.json'))['version'])")
[ -d "web/dist/data/$version" ] || { echo "index.json points at $version, which is not in the build" >&2; exit 1; }
echo "==> data version $version"

go run ./cmd/bundle -check "web/dist/data/$version/provenance.json"

go test -tags dataset ./golden/ -args -data ../web/dist/data

service="${1:-}"
echo "==> railway up${service:+ --service $service}"
railway up ${service:+--service "$service"}
