#!/usr/bin/env bash
set -euo pipefail

ROLLBACK_HINT="Rollback hint: stop pressluft services, remove /opt/pressluft/current and partial /opt/pressluft/releases/<version>, then rerun install.sh on a clean host checkpoint."

on_error() {
  local exit_code=$?
  printf 'install failed (exit=%s)\n%s\n' "$exit_code" "$ROLLBACK_HINT" >&2
  exit "$exit_code"
}

trap on_error ERR

if [[ ${EUID:-0} -ne 0 ]]; then
  printf 'install must run as root (try: curl ... | sudo bash)\n' >&2
  exit 1
fi

if [[ -f /etc/os-release ]]; then
  # shellcheck source=/dev/null
  source /etc/os-release
  if [[ "${ID:-}" != "ubuntu" || "${VERSION_ID:-}" != "24.04" ]]; then
    printf 'unsupported OS: expected Ubuntu 24.04, got %s %s\n' "${ID:-unknown}" "${VERSION_ID:-unknown}" >&2
    exit 1
  fi
else
  printf '/etc/os-release missing; cannot validate OS\n' >&2
  exit 1
fi

DB_PATH="${PRESSLUFT_DB_PATH:-/var/lib/pressluft/pressluft.db}"

REPO="https://github.com/denizbinay/pressluft"
VERSION="${PRESSLUFT_VERSION:-latest}"

arch_raw="$(uname -m)"
case "$arch_raw" in
  x86_64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) printf 'unsupported architecture: %s\n' "$arch_raw" >&2; exit 1 ;;
esac

tmpdir="$(mktemp -d)"
cleanup() { rm -rf "$tmpdir"; }
trap cleanup EXIT

printf 'installing prerequisites (apt)\n'
export DEBIAN_FRONTEND=noninteractive
apt-get update -y
apt-get install -y --no-install-recommends \
  ca-certificates \
  curl \
  openssh-client \
  python3 \
  sudo \
  tar \
  ansible

asset="pressluft_linux_${arch}.tar.gz"
if [[ "$VERSION" == "latest" ]]; then
  asset_url="$REPO/releases/latest/download/$asset"
  sums_url="$REPO/releases/latest/download/SHA256SUMS"
else
  asset_url="$REPO/releases/download/$VERSION/$asset"
  sums_url="$REPO/releases/download/$VERSION/SHA256SUMS"
fi

printf 'downloading %s (%s)\n' "$asset" "$VERSION"
effective_url="$(curl -fsSL -o "$tmpdir/$asset" -w '%{url_effective}' "$asset_url")"
curl -fsSL -o "$tmpdir/SHA256SUMS" "$sums_url"

printf 'verifying sha256\n'
( cd "$tmpdir" && grep " ${asset}$" SHA256SUMS | sha256sum -c - )

resolved_version="$VERSION"
if [[ "$VERSION" == "latest" ]]; then
  resolved_version="$(printf '%s' "$effective_url" | sed -n 's#.*\/releases\/download\/\([^/]*\)\/.*#\1#p')"
  if [[ -z "$resolved_version" ]]; then
    resolved_version="latest-$(date -u +%Y%m%d%H%M%S)"
  fi
fi

install_root="/opt/pressluft"
release_dir="$install_root/releases/$resolved_version"

mkdir -p "$install_root/releases"
if [[ ! -d "$release_dir" ]]; then
  staging="$tmpdir/staging"
  mkdir -p "$staging"
  tar -xzf "$tmpdir/$asset" -C "$staging"

  root="$staging"
  shopt -s nullglob
  entries=("$staging"/*)
  shopt -u nullglob
  if [[ ${#entries[@]} -eq 1 && -d "${entries[0]}" ]]; then
    root="${entries[0]}"
  fi

  mkdir -p "$release_dir"
  cp -a "$root"/. "$release_dir"/
fi

ln -sfn "$release_dir" "$install_root/current"

if ! id -u pressluft >/dev/null 2>&1; then
  useradd --system --home /var/lib/pressluft --create-home --shell /usr/sbin/nologin pressluft
fi

install -d -m 0755 /etc/pressluft
install -d -o pressluft -g pressluft -m 0750 /var/lib/pressluft
install -d -o pressluft -g pressluft -m 0700 /var/lib/pressluft/secrets
install -d -o pressluft -g pressluft -m 0750 /var/log/pressluft

env_file="/etc/pressluft/pressluft.env"
if [[ ! -f "$env_file" ]]; then
  cat >"$env_file" <<EOF
PRESSLUFT_LISTEN_ADDR=:8080
PRESSLUFT_DB_PATH=/var/lib/pressluft/pressluft.db
PRESSLUFT_SECRETS_DIR=/var/lib/pressluft/secrets
PRESSLUFT_WEB_DIST_DIR=/opt/pressluft/current/web/.output/public
ANSIBLE_CONFIG=/opt/pressluft/current/ansible/ansible.cfg
EOF
  chmod 0640 "$env_file"
  chown root:root "$env_file"
fi

sudoers_file="/etc/sudoers.d/pressluft"
if [[ ! -f "$sudoers_file" ]]; then
  printf 'pressluft ALL=(ALL) NOPASSWD:ALL\n' >"$sudoers_file"
  chmod 0440 "$sudoers_file"
fi

if [[ ! -f /opt/pressluft/current/packaging/systemd/pressluft-api.service ]]; then
  printf 'missing systemd unit files in release artifact\n' >&2
  exit 1
fi

install -m 0644 /opt/pressluft/current/packaging/systemd/pressluft-api.service /etc/systemd/system/pressluft-api.service
install -m 0644 /opt/pressluft/current/packaging/systemd/pressluft-worker.service /etc/systemd/system/pressluft-worker.service
systemctl daemon-reload

printf 'migrating database\n'
sudo -u pressluft /opt/pressluft/current/pressluft -db "$DB_PATH" migrate up

printf 'bootstrapping local node\n'
sudo -u pressluft /opt/pressluft/current/pressluft -db "$DB_PATH" bootstrap

printf 'initializing admin user\n'
admin_email="${PRESSLUFT_ADMIN_EMAIL:-admin@local}"
admin_display_name="${PRESSLUFT_ADMIN_DISPLAY_NAME:-Admin}"
admin_password="${PRESSLUFT_ADMIN_PASSWORD:-}"
sudo -u pressluft \
  PRESSLUFT_ADMIN_PASSWORD="$admin_password" \
  /opt/pressluft/current/pressluft -db "$DB_PATH" admin init -email "$admin_email" -display-name "$admin_display_name"

systemctl enable --now pressluft-api.service pressluft-worker.service

printf 'install completed successfully (version=%s)\n' "$resolved_version"
