#!/usr/bin/env bash
set -euo pipefail

repository="Cec1c/cpamp-theme-studio"
requested_version="latest"
download_proxy=""
forwarded=()

while (($#)); do
  case "$1" in
    --bootstrap-version)
      [[ $# -ge 2 ]] || { echo "--bootstrap-version requires a value" >&2; exit 2; }
      requested_version="${2#v}"
      shift 2
      ;;
    --download-proxy)
      [[ $# -ge 2 ]] || { echo "--download-proxy requires a value" >&2; exit 2; }
      download_proxy="$2"
      shift 2
      ;;
    *)
      forwarded+=("$1")
      shift
      ;;
  esac
done

if [[ -n "${download_proxy}" ]]; then
  export HTTP_PROXY="${download_proxy}"
  export HTTPS_PROXY="${download_proxy}"
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
bundled="${script_dir}/cpamp-theme-bootstrap"
if [[ -x "${bundled}" ]]; then
  exec "${bundled}" "${forwarded[@]}"
fi

for command_name in curl unzip sha256sum uname mktemp; do
  command -v "${command_name}" >/dev/null 2>&1 || {
    echo "required command is missing: ${command_name}" >&2
    exit 1
  }
done

case "$(uname -m)" in
  x86_64|amd64) architecture="amd64" ;;
  aarch64|arm64) architecture="arm64" ;;
  *) echo "unsupported Linux architecture: $(uname -m)" >&2; exit 1 ;;
esac

temporary="$(mktemp -d "${TMPDIR:-/tmp}/cpamp-theme-bootstrap.XXXXXXXX")"
trap 'rm -rf -- "${temporary}"' EXIT
curl_args=(--fail --silent --show-error --location --proto '=https' --tlsv1.2)
if [[ -n "${download_proxy}" ]]; then
  curl_args+=(--proxy "${download_proxy}")
fi

if [[ "${requested_version}" == "latest" ]]; then
  checksums_url="https://github.com/${repository}/releases/latest/download/checksums.txt"
else
  [[ "${requested_version}" =~ ^[0-9][0-9A-Za-z.+-]*$ ]] || {
    echo "invalid bootstrap version: ${requested_version}" >&2
    exit 2
  }
  checksums_url="https://github.com/${repository}/releases/download/v${requested_version}/checksums.txt"
fi

effective_url="$(curl "${curl_args[@]}" --output "${temporary}/checksums.txt" --write-out '%{url_effective}' "${checksums_url}")"
if [[ "${requested_version}" == "latest" ]]; then
  requested_version="$(sed -nE 's#^.*/download/v([^/]+)/checksums\.txt$#\1#p' <<<"${effective_url}")"
fi
[[ "${requested_version}" =~ ^[0-9][0-9A-Za-z.+-]*$ ]] || {
  echo "could not resolve a valid release version from ${effective_url}" >&2
  exit 1
}

archive="cpamp-theme-studio_${requested_version}_linux_${architecture}.zip"
expected="$(awk -v archive="${archive}" '$2 == archive { print $1 }' "${temporary}/checksums.txt")"
[[ "${expected}" =~ ^[0-9a-fA-F]{64}$ ]] || {
  echo "checksums.txt does not contain ${archive}" >&2
  exit 1
}
curl "${curl_args[@]}" --output "${temporary}/${archive}" \
  "https://github.com/${repository}/releases/download/v${requested_version}/${archive}"
actual="$(sha256sum "${temporary}/${archive}" | awk '{print $1}')"
[[ "${actual,,}" == "${expected,,}" ]] || {
  echo "SHA-256 mismatch for ${archive}" >&2
  exit 1
}

unzip -q "${temporary}/${archive}" cpamp-theme-bootstrap -d "${temporary}"
chmod 0755 "${temporary}/cpamp-theme-bootstrap"
echo "Verified CPAMP Theme Studio bootstrap v${requested_version} (${architecture}, SHA-256 ${actual})."
exec "${temporary}/cpamp-theme-bootstrap" "${forwarded[@]}"
