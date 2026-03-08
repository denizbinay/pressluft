#!/bin/sh

set -eu

if ! command -v ansible-playbook >/dev/null 2>&1; then
  printf '%s\n' 'ansible-playbook is required for ansible-check' >&2
  exit 1
fi

if [ ! -f bin/pressluft-agent ]; then
  printf '%s\n' 'bin/pressluft-agent is required; run `make agent` first' >&2
  exit 1
fi

if [ "$(id -u)" -ne 0 ]; then
  printf '%s\n' 'ansible-check needs root because configure.yml manages system packages and services; rerun as root or with sudo' >&2
  exit 1
fi

inventory_path="$(mktemp)"
trap 'rm -f "$inventory_path"' EXIT INT TERM

cat >"$inventory_path" <<'EOF'
localhost ansible_connection=local
EOF

ansible-playbook -i "$inventory_path" ops/ansible/playbooks/configure.yml --check --diff \
  -e server_id=1 \
  -e control_plane_url=http://127.0.0.1:8081 \
  -e pressluft_execution_mode=dev \
  -e dev_ws_token=dev-token \
  -e profile_key=nginx-stack \
  -e profile_path=ops/profiles/nginx-stack/profile.yaml \
  -e profile_support_level=supported \
  -e profile_configure_guarantee='advisory check mode' \
  -e agent_binary_path=bin/pressluft-agent
