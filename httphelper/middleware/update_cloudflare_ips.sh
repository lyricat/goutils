#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
data_dir="${script_dir}/ban_data"

mkdir -p "${data_dir}"
curl -fsSL "https://www.cloudflare.com/ips-v4/" -o "${data_dir}/cloudflare_ipv4.txt"
curl -fsSL "https://www.cloudflare.com/ips-v6/" -o "${data_dir}/cloudflare_ipv6.txt"
