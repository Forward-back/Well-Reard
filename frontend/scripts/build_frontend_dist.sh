#!/usr/bin/env bash
# Build frontend static assets for Lite / desktop packaging (host-side).
# Docker UI images use the multi-stage frontend/Dockerfile instead — no host npm.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FRONTEND_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$FRONTEND_DIR/.." && pwd)"

if [ -z "${VITE_FRONTEND_COMMIT:-}" ]; then
	VITE_FRONTEND_COMMIT="$(git -C "$REPO_ROOT" rev-parse --short HEAD 2>/dev/null || echo unknown)"
	export VITE_FRONTEND_COMMIT
fi

export VITE_IS_DOCKER="${VITE_IS_DOCKER:-true}"

cd "$FRONTEND_DIR"
npm ci
npm run build
