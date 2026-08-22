#!/usr/bin/env bash
set -euo pipefail

version="${1:-0.1.5-dev}"
target="${2:-$(go env GOHOSTOS)-$(go env GOHOSTARCH)}"
output_dir="${3:-dist}"
version="${version#v}"

if [[ ! "${version}" =~ ^[0-9][0-9A-Za-z.+-]*$ ]]; then
  echo "invalid plugin version: ${version}" >&2
  exit 1
fi
IFS=- read -r target_os target_arch <<<"${target}"
host_os="$(go env GOHOSTOS)"
host_arch="$(go env GOHOSTARCH)"
if [[ "${target_os}" != "${host_os}" || "${target_arch}" != "${host_arch}" ]]; then
  echo "CGO shared libraries must be built on a native ${target} runner (current host: ${host_os}-${host_arch})" >&2
  exit 1
fi
case "${target_os}" in
  linux) extension=.so ;;
  darwin) extension=.dylib ;;
  windows) extension=.dll ;;
  *) echo "unsupported target: ${target}" >&2; exit 1 ;;
esac
case "${target_arch}" in
  amd64|arm64) ;;
  *) echo "unsupported architecture: ${target_arch}" >&2; exit 1 ;;
esac

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ "${output_dir}" != /* ]]; then
  output_dir="${repo_root}/${output_dir}"
fi
mkdir -p "${output_dir}"
cd "${repo_root}"

go test ./...
if command -v node >/dev/null 2>&1; then
  node --check assets/loader.js
fi
CGO_ENABLED=1 GOOS="${target_os}" GOARCH="${target_arch}" \
  go build -trimpath -buildmode=c-shared \
  -ldflags "-s -w -X=main.pluginVersion=${version}" \
  -o "${output_dir}/cpamp-theme-studio${extension}" .
rm -f "${output_dir}/cpamp-theme-studio.h"
ls -lh "${output_dir}/cpamp-theme-studio${extension}"
