#!/bin/bash
# Docs Sync Script - Gamification System
# Bu script Swagger dokümantasyonunu tek kaynaktan diger konumlara senkronize eder.
#
# Tek kaynak: internal/muscle/docs/swagger.json (swag init ile uretilir)
# 
# Senkronize edilen yerler:
#   1. internal/muscle/docs/swagger.json -> (tek kaynak)
#   2. internal/muscle/mcp/resources/swagger.json (MCP embed)
#   NOT: docs-portal/docs/openapi.yaml kullanilmiyor - docs portal canli Swagger UI kullanir
#
# Kullanim:
#   chmod +x scripts/sync-docs.sh
#   ./scripts/sync-docs.sh
# 
# veya:
#   bash scripts/sync-docs.sh

set -e

# Renkli cikti
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m'

echo ""
echo -e "${MAGENTA}============================================================${NC}"
echo -e "${MAGENTA}  Gamification System - Dokumantasyon Senkronizasyonu${NC}"
echo -e "${MAGENTA}============================================================${NC}"
echo ""

# Repo root - script konumundan hesapla
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
GO_MODULE_PATH="$PROJECT_ROOT/internal/muscle"

# Adim 1: Swagger uretimi
echo -e "${CYAN}[STEP] Adim 1: Swagger dokumantasyonu uretiliyor...${NC}"

# swag kurulu mu kontrol et
if ! command -v swag &> /dev/null; then
    echo -e "  ${YELLOW}[WARN] swag kurulu degil, kuruluyor...${NC}"
    go install github.com/swaggo/swag/cmd/swag@latest
    export PATH="$PATH:$(go env GOPATH)/bin"
fi

if command -v swag &> /dev/null; then
    cd "$GO_MODULE_PATH"
    swag init -g main.go -o docs --parseDependency --parseInternal
    
    if [ $? -ne 0 ]; then
        echo -e "${RED}[ERROR] swag init basarisiz oldu!${NC}"
        exit 1
    fi
    echo -e "${GREEN}[OK] Swagger dokumantasyonu uretildi${NC}"
    cd - > /dev/null
else
    echo -e "${YELLOW}[WARN] swag bulunamadi, mevcut swagger.json kullanilacak${NC}"
fi

# Adim 2: MCP resources'a kopyala
echo -e "${CYAN}[STEP] Adim 2: MCP embed kaynagi guncelleniyor...${NC}"
SOURCE_SWAGGER="$GO_MODULE_PATH/docs/swagger.json"
MCP_SWAGGER="$GO_MODULE_PATH/mcp/resources/swagger.json"

if [ -f "$SOURCE_SWAGGER" ]; then
    cp -f "$SOURCE_SWAGGER" "$MCP_SWAGGER"
    echo -e "${GREEN}[OK] MCP resources swagger.json guncellendi${NC}"
else
    echo -e "${RED}[ERROR] Kaynak swagger.json bulunamadi: $SOURCE_SWAGGER${NC}"
    exit 1
fi

# Adim 3: openapi.yaml kullanilmiyor - bilgi mesaji
echo -e "${CYAN}[STEP] Adim 3: openapi.yaml kullanim kontrolu...${NC}"
echo -e "  ${YELLOW}[INFO] docs-portal/docs/openapi.yaml kullanilmamaktadir.${NC}"
echo -e "  ${YELLOW}[INFO] Docs portal /api-reference sayfasi canli Swagger UI'ya yonlendirir.${NC}"

# Adim 4: Degisiklik kontrolu
echo -e "${CYAN}[STEP] Adim 4: Degisiklik kontrolu...${NC}"
cd "$PROJECT_ROOT"
if command -v git &> /dev/null; then
    CHANGES=$(git status --porcelain 2>/dev/null || true)
    if [ -n "$CHANGES" ]; then
        echo -e "${YELLOW}[WARN] Bekleyen degisiklikler:${NC}"
        echo "$CHANGES" | grep -E "internal/muscle/docs|internal/muscle/mcp" | while read -r line; do
            echo -e "  $line"
        done
        echo ""
        echo -e "  ${CYAN}Degisiklikleri commit etmek icin:${NC}"
        echo -e "    git add . && git commit -m 'docs: sync swagger'"
    else
        echo -e "${GREEN}[OK] Tum dokumantasyon senkronize${NC}"
    fi
fi

echo ""
echo -e "${GREEN}============================================================${NC}"
echo -e "${GREEN}  Dokumantasyon senkronizasyonu tamamlandi!${NC}"
echo ""
echo -e "  ${WHITE}Tek kaynak: internal/muscle/docs/swagger.json${NC}"
echo -e "  ${WHITE}Senkronize: internal/muscle/mcp/resources/swagger.json${NC}"
echo ""
echo -e "  ${YELLOW}Dikkat: openapi.yaml kullanimdisi (docs portal redirect kullanir)${NC}"
echo -e "${GREEN}============================================================${NC}"
echo ""