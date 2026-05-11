#!/usr/bin/env bash
# ============================================================
#  EmlakPro — Deploy Script
#  Sunucuda çalışır: bash /opt/emlakpro/deploy.sh
#  Ya da: make deploy HOST=root@192.168.55.45
# ============================================================

set -euo pipefail

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; BOLD='\033[1m'; NC='\033[0m'
ok()   { echo -e "${GREEN}[OK]${NC}    $*"; }
info() { echo -e "${YELLOW}[INFO]${NC}  $*"; }
err()  { echo -e "${RED}[HATA]${NC}  $*"; exit 1; }

APP_DIR="/opt/emlakpro"
SRC_DIR="$APP_DIR/src"
BIN_DIR="$APP_DIR/bin"
CFG     ="$APP_DIR/config/config.json"
SERVICE="emlakpro"
GO_BIN="/usr/local/go/bin/go"

[[ ! -f "$GO_BIN" ]] && err "Go bulunamadı: $GO_BIN"
[[ ! -f "$CFG"    ]] && err "Config bulunamadı: $CFG"
[[ ! -d "$SRC_DIR" ]] && err "Kaynak kod bulunamadı: $SRC_DIR"

echo -e "\n${BOLD}━━━ EmlakPro Deploy ━━━${NC}"
info "Dizin: $SRC_DIR"
info "Versiyon: $(cd $SRC_DIR && git rev-parse --short HEAD 2>/dev/null || echo dev)"

# Bağımlılıklar
info "Go bağımlılıkları indiriliyor..."
cd "$SRC_DIR"
$GO_BIN mod download
ok "Bağımlılıklar hazır."

# Derleme
info "Binary derleniyor..."
mkdir -p "$BIN_DIR"
VERSION=$(git rev-parse --short HEAD 2>/dev/null || echo "dev")
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  $GO_BIN build \
  -ldflags="-s -w -X main.version=$VERSION" \
  -o "$BIN_DIR/emlakpro" \
  ./cmd/server/

chown emlakpro:emlakpro "$BIN_DIR/emlakpro" 2>/dev/null || true
chmod 750 "$BIN_DIR/emlakpro"
ok "Binary derlendi: $BIN_DIR/emlakpro ($(du -sh $BIN_DIR/emlakpro | cut -f1))"

# Servisi yeniden başlat
info "Servis yeniden başlatılıyor..."
systemctl restart "$SERVICE"
sleep 2

if systemctl is-active --quiet "$SERVICE"; then
  ok "Servis çalışıyor: $SERVICE"
else
  err "Servis başlatılamadı! Log: journalctl -u $SERVICE -n 30"
fi

# Sağlık kontrolü
sleep 1
PORT=$(python3 -c "import json; d=json.load(open('$CFG')); print(d['app']['port'])" 2>/dev/null || echo "8080")
if curl -sf "http://localhost:$PORT/api/health" | grep -q "ok"; then
  ok "Sağlık kontrolü geçti: http://localhost:$PORT/api/health"
else
  err "Sağlık kontrolü başarısız!"
fi

echo ""
echo -e "${GREEN}${BOLD}✓ Deploy tamamlandı! Versiyon: $VERSION${NC}"
echo ""
