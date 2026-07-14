#!/usr/bin/env bash
# 在 Linux 上安装 SNIProxy 为 systemd 服务（需 root）
set -euo pipefail

PREFIX="${PREFIX:-/opt/sniproxy}"
LOG_DIR="${LOG_DIR:-/var/log/sniproxy}"
SERVICE_DST="/etc/systemd/system/sniproxy.service"
USER_NAME="${USER_NAME:-sniproxy}"
GROUP_NAME="${GROUP_NAME:-sniproxy}"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "请使用 root 运行: sudo $0" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

BIN_SRC="${BIN_SRC:-}"
if [[ -z "${BIN_SRC}" ]]; then
  if [[ -x "${REPO_ROOT}/sniproxy" ]]; then
    BIN_SRC="${REPO_ROOT}/sniproxy"
  elif command -v sniproxy >/dev/null 2>&1; then
    BIN_SRC="$(command -v sniproxy)"
  else
    echo "未找到 sniproxy 二进制，请先编译或设置 BIN_SRC=/path/to/sniproxy" >&2
    exit 1
  fi
fi

if ! id -u "${USER_NAME}" >/dev/null 2>&1; then
  useradd --system --no-create-home --shell /usr/sbin/nologin "${USER_NAME}"
fi

install -d -o root -g root -m 0755 "${PREFIX}"
install -d -o "${USER_NAME}" -g "${GROUP_NAME}" -m 0755 "${LOG_DIR}"

install -o root -g root -m 0755 "${BIN_SRC}" "${PREFIX}/sniproxy"

if [[ ! -f "${PREFIX}/config.yaml" ]]; then
  if [[ -f "${REPO_ROOT}/config.yaml" ]]; then
    install -o root -g "${GROUP_NAME}" -m 0640 "${REPO_ROOT}/config.yaml" "${PREFIX}/config.yaml"
  else
    echo "警告: 未找到 config.yaml，请手动创建 ${PREFIX}/config.yaml" >&2
  fi
else
  echo "保留已有配置: ${PREFIX}/config.yaml"
fi

install -o root -g root -m 0644 "${SCRIPT_DIR}/sniproxy.service" "${SERVICE_DST}"

systemctl daemon-reload
systemctl enable sniproxy.service

echo
echo "安装完成。"
echo "  二进制: ${PREFIX}/sniproxy"
echo "  配置:   ${PREFIX}/config.yaml"
echo "  日志:   ${LOG_DIR}/sni.log"
echo
echo "下一步："
echo "  1. 编辑配置: nano ${PREFIX}/config.yaml"
echo "  2. 启动服务: systemctl start sniproxy"
echo "  3. 查看状态: systemctl status sniproxy"
echo "  4. 热加载:   systemctl reload sniproxy"
