#!/usr/bin/env bash
set -euo pipefail

# 仓库目录（与 systemd 工作目录、二进制部署目录分离）
REPO_DIR="/root/TrackMe"
# 运行目录：须含 config.json、certs 等
DEPLOY_DIR="/root/local/apps/api"
BINARY_NAME="trackme"
SERVICE_NAME="trackme-api"
GIT_USER="ec2-user"

log() { printf '[%s] %s\n' "$(date -u +'%Y-%m-%dT%H:%M:%SZ')" "$*"; }

if [[ "$(id -u)" -ne 0 ]]; then
	log "请使用 root 执行：sudo bash $0"
	exit 1
fi

if ! command -v go >/dev/null 2>&1; then
	log "未找到 go，请先安装并加入 PATH"
	exit 1
fi

if [[ ! -d "$REPO_DIR/.git" ]]; then
	log "不是 git 仓库：$REPO_DIR"
	exit 1
fi

mkdir -p "$DEPLOY_DIR"

log "git pull ($REPO_DIR)…"
# /root 下仓库 ec2-user 无法进入，须用 root 执行 git；属主为 ec2-user 时仍用该用户拉取以免改属主
REPO_OWNER="$(stat -c '%U' "$REPO_DIR")"
if [[ "$REPO_OWNER" == "$GIT_USER" ]] && id "$GIT_USER" &>/dev/null; then
	sudo -u "$GIT_USER" git -C "$REPO_DIR" pull --ff-only
else
	git -C "$REPO_DIR" pull --ff-only
fi

log "go build…"
# gopacket/pcap 需要 CGO + 系统已安装 libpcap-devel
export CGO_ENABLED=1
(
	cd "$REPO_DIR"
	go build -ldflags="-s -w" -o "$DEPLOY_DIR/${BINARY_NAME}.new" ./cmd
)

chmod +x "$DEPLOY_DIR/${BINARY_NAME}.new"
if [[ -f "$DEPLOY_DIR/$BINARY_NAME" ]]; then
	BACKUP="$DEPLOY_DIR/${BINARY_NAME}.bak.$(date +%Y%m%d%H%M%S)"
	cp -a "$DEPLOY_DIR/$BINARY_NAME" "$BACKUP"
	log "已备份当前二进制 -> $BACKUP"
fi
mv -f "$DEPLOY_DIR/${BINARY_NAME}.new" "$DEPLOY_DIR/$BINARY_NAME"

log "重启 $SERVICE_NAME…"
systemctl restart "$SERVICE_NAME"
systemctl --no-pager -l status "$SERVICE_NAME" || true
log "二进制已更新。以下为 journal 最近 30 条并持续跟随（Ctrl+C 退出）…"
journalctl -u "$SERVICE_NAME" -n 30 -f
