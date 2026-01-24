#!/bin/bash
#
# MindBalancer Installer
# https://www.mindbalancer.org
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/mindbalancer/mindbalancer-labs/main/scripts/install.sh | bash
#

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

REPO="https://github.com/mindbalancer/mindbalancer-labs.git"
INSTALL_DIR="/usr/local/bin"
TMP_DIR=$(mktemp -d)

cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT

echo -e "${PURPLE}"
cat << 'BANNER'
  __  __ _           _ ____        _                           
 |  \/  (_)_ __   __| | __ )  __ _| | __ _ _ __   ___ ___ _ __ 
 | |\/| | | '_ \ / _` |  _ \ / _` | |/ _` | '_ \ / __/ _ \ '__|
 | |  | | | | | | (_| | |_) | (_| | | (_| | | | | (_|  __/ |   
 |_|  |_|_|_| |_|\__,_|____/ \__,_|_|\__,_|_| |_|\___\___|_|   
BANNER
echo -e "${NC}"
echo -e "${CYAN}The ProxySQL for AI${NC}"
echo ""

step() { echo -e "${BLUE}==>${NC} $1"; }
success() { echo -e "${GREEN}✓${NC} $1"; }
error() { echo -e "${RED}✗${NC} $1"; exit 1; }

step "Checking requirements..."
command -v git &>/dev/null || error "Git required"
success "Git found"
command -v go &>/dev/null || error "Go 1.20+ required (https://go.dev/dl/)"
success "Go found"

step "Cloning repository..."
git clone --depth 1 "$REPO" "$TMP_DIR/mb" 2>/dev/null
success "Cloned"

step "Building..."
cd "$TMP_DIR/mb"
if command -v make &>/dev/null; then
    make build 2>/dev/null
else
    go build -o bin/mindbalancer ./cmd/mindbalancer
    go build -o bin/mindsql ./cmd/mindsql
fi
success "Built"

step "Installing to $INSTALL_DIR..."
SUDO=""; [ -w "$INSTALL_DIR" ] || SUDO="sudo"
$SUDO cp "$TMP_DIR/mb/bin/mindbalancer" "$INSTALL_DIR/"
$SUDO cp "$TMP_DIR/mb/bin/mindsql" "$INSTALL_DIR/"
$SUDO chmod +x "$INSTALL_DIR/mindbalancer" "$INSTALL_DIR/mindsql"
success "Installed"

step "Creating config..."
mkdir -p ~/.mindbalancer
[ -f ~/.mindbalancer/mindbalancer.cnf ] || cat > ~/.mindbalancer/mindbalancer.cnf << 'EOF'
[mindbalancer]
admin_bind_address = 127.0.0.1
admin_port = 6032
proxy_bind_address = 127.0.0.1
proxy_port = 6034
admin_http_port = 6033
data_dir = ~/.mindbalancer/data
log_level = info
health_check_enabled = true
prometheus_enabled = true
prometheus_port = 9090
EOF
success "Config ready"

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  MindBalancer installed! 🎉${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "  Start:     ${PURPLE}mindbalancer -config ~/.mindbalancer/mindbalancer.cnf${NC}"
echo -e "  Admin:     ${PURPLE}mindsql${NC}"
echo -e "  Dashboard: ${PURPLE}http://localhost:6033${NC}"
echo ""
echo -e "  Docs:    https://www.mindbalancer.org/docs"
echo -e "  Support: burak1607@gmail.com"
echo ""
