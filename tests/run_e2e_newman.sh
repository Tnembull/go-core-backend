#!/usr/bin/env bash
# ==============================================================================
# Automated E2E Test Suite Runner using Newman (Postman / Apidog Compatible)
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="${SCRIPT_DIR}/.."

TEST_PORT=8089
SERVER_BIN="${ROOT_DIR}/bin/test-server"

echo "[*] Compiling Go Core Backend server..."
mkdir -p "${ROOT_DIR}/bin"
(cd "${ROOT_DIR}" && go build -o "${SERVER_BIN}" cmd/server/main.go)

echo "[*] Starting test server instance on port ${TEST_PORT}..."
export PORT="${TEST_PORT}"
export APP_ENV="development"
export INTERNAL_API_KEY="core-secret-api-key-2026"
export JWT_SECRET="ci-cd-test-super-secret-jwt-key"

"${SERVER_BIN}" > "${ROOT_DIR}/test-server.log" 2>&1 &
SERVER_PID=$!

cleanup() {
  echo "[*] Cleaning up test server (PID: ${SERVER_PID})..."
  kill -TERM "${SERVER_PID}" 2>/dev/null || true
  wait "${SERVER_PID}" 2>/dev/null || true
  rm -f "${SERVER_BIN}"
}
trap cleanup EXIT

echo "[*] Waiting for server health probe..."
for i in {1..30}; do
  if curl -s "http://127.0.0.1:${TEST_PORT}/healthz" >/dev/null 2>&1; then
    echo "[✓] Server is live and healthy!"
    break
  fi
  sleep 0.2
done

echo "================================================================="
echo "[*] Executing Postman / Apidog Test Collection with Newman..."
echo "================================================================="

newman run "${SCRIPT_DIR}/postman/go-core-backend.postman_collection.json" \
  -e "${SCRIPT_DIR}/postman/go-core-backend.postman_environment.json" \
  --env-var "base_url=http://127.0.0.1:${TEST_PORT}" \
  --reporters cli

echo "================================================================="
echo "[✓] ALL POSTMAN / APIDOG ASSERIONS PASSED SUCCESSFULLY!"
echo "================================================================="
