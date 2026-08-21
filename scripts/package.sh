#!/usr/bin/env bash
set -euo pipefail

version="${1:-0.1.0-dev}"
target="${2:-$(go env GOHOSTOS)-$(go env GOHOSTARCH)}"
version="${version#v}"

if [[ ! "${version}" =~ ^[0-9][0-9A-Za-z.+-]*$ ]]; then
  echo "invalid plugin version: ${version}" >&2
  exit 1
fi
IFS=- read -r target_os target_arch <<<"${target}"
case "${target_os}" in
  windows|linux|darwin) ;;
  *) echo "unsupported target: ${target}" >&2; exit 1 ;;
esac
case "${target_arch}" in
  amd64|arm64) ;;
  *) echo "unsupported target: ${target}" >&2; exit 1 ;;
esac

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist_dir="${repo_root}/dist"
stage_dir="${dist_dir}/.package-${target}"
archive_name="cpamp-theme-studio_${version}_${target_os}_${target_arch}.zip"
archive_path="${dist_dir}/${archive_name}"

case "${target_os}" in
  linux) extension=.so ;;
  darwin) extension=.dylib ;;
  windows) extension=.dll ;;
esac

mkdir -p "${dist_dir}"
rm -rf -- "${stage_dir}"
mkdir -p "${stage_dir}"
trap 'rm -rf -- "${stage_dir}"' EXIT

"${repo_root}/scripts/build.sh" "${version}" "${target}" "${stage_dir}"
cp "${repo_root}/LICENSE" "${repo_root}/README.md" "${repo_root}/README.zh-CN.md" "${repo_root}/THIRD_PARTY_NOTICES.md" "${stage_dir}/"
cp -R "${repo_root}/docs" "${stage_dir}/docs"
rm -f -- "${archive_path}"
(
  cd "${stage_dir}"
  zip -X -q -r "${archive_path}" "cpamp-theme-studio${extension}" LICENSE README.md README.zh-CN.md THIRD_PARTY_NOTICES.md docs
)
if command -v sha256sum >/dev/null 2>&1; then
  checksum="$(sha256sum "${archive_path}" | awk '{print $1}')"
else
  checksum="$(shasum -a 256 "${archive_path}" | awk '{print $1}')"
fi
printf '%s  %s\n' "${checksum}" "${archive_name}" >"${dist_dir}/checksums.txt"
ls -lh "${archive_path}"
