#!/usr/bin/env sh
set -eu

root_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

echo "[CODENET] Menguji Management API"
go_cache="${TMPDIR:-/tmp}/codenet-go-cache"
mkdir -p "$go_cache"
(cd "$root_dir/apps/api" && GOCACHE="$go_cache" go test ./...)

echo "[CODENET] Menguji WebFig"
(cd "$root_dir/apps/webfig" && npm test)

echo "[CODENET] Membangun WebFig"
(cd "$root_dir/apps/webfig" && npm run build)
