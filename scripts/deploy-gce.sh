#!/usr/bin/env bash
# Runs on the Compute Engine VM. Uploaded by .github/workflows/deploy-gce.yml.
set -euo pipefail

RELEASE_FILE="${1:?Usage: deploy-gce.sh <release.tar.gz>}"
SERVICE_USER="litloop"
ENV_FILE="/etc/litloop/litloop.env"
RUN_MIGRATIONS="${GCP_RUN_MIGRATIONS:-true}"

if [[ ! -f "$RELEASE_FILE" ]]; then
  echo "Release archive not found: $RELEASE_FILE" >&2
  exit 1
fi

if ! sudo test -f "$ENV_FILE"; then
  echo "Missing $ENV_FILE. Create it from deploy/litloop.env.example first." >&2
  exit 1
fi

if ! command -v ffmpeg >/dev/null 2>&1; then
  echo "WARNING: ffmpeg is not installed; media processing will not work." >&2
fi

if ! id -u "$SERVICE_USER" >/dev/null 2>&1; then
  echo "Creating system user $SERVICE_USER"
  sudo useradd \
    --system \
    --home-dir /var/lib/litloop \
    --shell /usr/sbin/nologin \
    "$SERVICE_USER"
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

tar -xzf "$RELEASE_FILE" -C "$TMP_DIR"

echo "Installing binaries"
sudo install -m 0755 "$TMP_DIR"/bin/* /usr/local/bin/

echo "Installing systemd units"
sudo install -m 0644 "$TMP_DIR"/systemd/*.service /etc/systemd/system/

echo "Preparing application directories"
sudo install -d -m 0755 -o "$SERVICE_USER" -g "$SERVICE_USER" /var/lib/litloop/media

sudo systemctl daemon-reload
sudo systemctl enable litloop-api.service litloop-worker.service >/dev/null

if [[ "$RUN_MIGRATIONS" != "false" ]]; then
  echo "Running database migrations"
  sudo systemctl start litloop-migrate.service
fi

echo "Restarting services"
sudo systemctl restart litloop-api.service litloop-worker.service

if command -v curl >/dev/null 2>&1; then
  for _ in {1..30}; do
    if curl -fsS http://127.0.0.1:8000/healthz >/dev/null 2>&1; then
      echo "Health check passed"
      rm -f "$RELEASE_FILE"
      exit 0
    fi
    sleep 2
  done

  echo "Health check failed" >&2
  sudo systemctl status litloop-api.service litloop-worker.service --no-pager >&2 || true
  sudo journalctl -u litloop-api.service -n 50 --no-pager >&2 || true
  exit 1
fi

echo "curl is not installed; skipping HTTP health check"
rm -f "$RELEASE_FILE"
