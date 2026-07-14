#!/usr/bin/env bash
set -euo pipefail

# 版本：优先 git tag，其次环境变量，最后 dev
if [[ -n "${VERSION:-}" ]]; then
  version="${VERSION}"
elif git describe --tags --always --dirty >/dev/null 2>&1; then
  version="$(git describe --tags --always --dirty)"
else
  version="dev"
fi

output_dir="${OUTPUT_DIR:-dist}"

mkdir -p "${output_dir}"
echo "版本: ${version}"
echo "输出: ${output_dir}/"

while read -r GOOS GOARCH EXT; do
  [[ -z "${GOOS}" ]] && continue
  name="sniproxy_${GOOS}_${GOARCH}"
  outdir="${output_dir}/${name}"
  mkdir -p "${outdir}"

  bin="${outdir}/sniproxy"
  if [[ "${GOOS}" == "windows" ]]; then
    bin="${bin}.exe"
  fi

  echo "打包 ${GOOS}/${GOARCH} ..."
  CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build -trimpath \
    -ldflags "-s -w -X main.version=${version}" \
    -o "${bin}" .

  cp config.yaml "${outdir}/"
  [[ -f deploy/sniproxy.service ]] && cp deploy/sniproxy.service "${outdir}/"
  [[ -f LICENSE ]] && cp LICENSE "${outdir}/"

  if [[ "${EXT}" == "tar.gz" ]]; then
    tar -czf "${output_dir}/${name}.tar.gz" -C "${outdir}" .
  else
    (cd "${output_dir}" && zip -qr "${name}.zip" "${name}")
  fi
  rm -rf "${outdir}"
  echo "  -> ${output_dir}/${name}.${EXT}"
done <<EOF
linux amd64 tar.gz
linux arm64 tar.gz
windows amd64 zip
darwin amd64 zip
darwin arm64 zip
EOF

(
  cd "${output_dir}"
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 *.{tar.gz,zip} > checksums.txt 2>/dev/null || true
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum *.{tar.gz,zip} > checksums.txt 2>/dev/null || true
  fi
)

echo "完成。产物在 ${output_dir}/"
ls -lah "${output_dir}/"
