#!/usr/bin/env bash
# Freecat Linux CLI installer. Run as root: sudo ./install.sh
set -euo pipefail

PREFIX=${PREFIX:-/opt/freecat}
HERE=$(cd "$(dirname "$0")" && pwd)

if [ "$(id -u)" -ne 0 ]; then
  echo "请用 root 运行: sudo ./install.sh" >&2
  exit 1
fi

install -d -m 0755 "$PREFIX" "$PREFIX/geo"
install -m 0755 "$HERE/freecat" "$PREFIX/freecat"
install -m 0755 "$HERE/mihomo" "$PREFIX/mihomo"
install -m 0644 "$HERE"/geo/* "$PREFIX/geo/"
ln -sf "$PREFIX/freecat" /usr/local/bin/freecat

if command -v systemctl >/dev/null 2>&1; then
  install -m 0644 "$HERE/freecat.service" /etc/systemd/system/freecat.service
  systemctl daemon-reload || true
fi

echo
echo "安装完成: $PREFIX"
echo "下一步:"
echo "  1) sudo freecat login -email 你的邮箱 -password 你的密码"
echo "  2) sudo freecat update"
echo "  3) 仅本地代理:      freecat run"
echo "     全局 TUN 代理:   sudo freecat dns on && sudo freecat run --tun"
echo "     systemd 常驻:    sudo env FREECAT_HOME=/etc/freecat freecat login ... && sudo systemctl enable --now freecat"
