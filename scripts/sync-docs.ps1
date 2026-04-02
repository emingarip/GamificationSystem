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
#   powershell -ExecutionPolicy Bypass -File .\scripts\sync-docs.ps1

param(
    [switch]$Verbose
)

$ErrorActionPreference = "Stop"

# Repo root - $PSScriptRoot zaten script'in oldugu klasör (ornegin: .../scripts)
# Bu durumda $PSScriptRoot parent alinarak repo root bulunur
if ($PSScriptRoot) {
    $ProjectRoot = Split-Path -Parent $PSScriptRoot
} else {
    # PSScriptRoot bos ise (ornegin interactive modda) current directory kullan
    $ProjectRoot = Get-Location
}

# Dogrulama: Repo root icinde beklenen klasorler var mi?
$expectedPaths = @("internal", "scripts", "README.md")
$missingPaths = @()
foreach ($path in $expectedPaths) {
    $fullPath = Join-Path $ProjectRoot $path
    if (-not (Test-Path $fullPath)) {
        $missingPaths += $path
    }
}
if ($missingPaths.Count -gt 0) {
    Write-Host "[ERROR] Repo root bulunamadi veya beklenen klasorler eksik!" -ForegroundColor Red
    Write-Host "  Bulunulan: $ProjectRoot" -ForegroundColor Yellow
    Write-Host "  Eksik: $($missingPaths -join ', ')" -ForegroundColor Yellow
    Write-Host "  Lutfen script'i proje kokunden calistirin:" -ForegroundColor Yellow
    Write-Host "    powershell -ExecutionPolicy Bypass -File .\scripts\sync-docs.ps1" -ForegroundColor Yellow
    exit 1
}

# Renkli cikti fonksiyonlari
function Write-Step { param($m) Write-Host "[STEP] $m" -ForegroundColor Cyan }
function Write-Success { param($m) Write-Host "[OK] $m" -ForegroundColor Green }
function Write-Warn { param($m) Write-Host "[WARN] $m" -ForegroundColor Yellow }
function Write-Fail { param($m) Write-Host "[ERROR] $m" -ForegroundColor Red }
function Write-Info { param($m) Write-Host "  $m" }

Write-Host ""
Write-Host "============================================================" -ForegroundColor Magenta
Write-Host "  Gamification System - Dokumantasyon Senkronizasyonu" -ForegroundColor Magenta
Write-Host "============================================================" -ForegroundColor Magenta
Write-Host ""
Write-Host "  Repo root: $ProjectRoot" -ForegroundColor Gray

$GoModulePath = Join-Path $ProjectRoot "internal\muscle"

# Adim 1: Swagger uretimi
Write-Step "Adim 1: Swagger dokumantasyonu uretiliyor..."

# swag kurulu mu kontrol et
$swagPath = (Get-Command swag -ErrorAction SilentlyContinue).Source
if (-not $swagPath) {
    Write-Info "swag kurulu degil, kuruluyor..."
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
    go install github.com/swaggo/swag/cmd/swag@latest
    $swagPath = (Get-Command swag -ErrorAction SilentlyContinue).Source
}

if ($swagPath) {
    Push-Location $GoModulePath
    try {
        if ($Verbose) { Write-Info "swag init -g main.go -o docs" }
        & swag init -g main.go -o docs --parseDependency --parseInternal
        
        if ($LASTEXITCODE -ne 0) {
            Write-Fail "swag init basarisiz oldu!"
            exit 1
        }
        Write-Success "Swagger dokumantasyonu uretildi"
    }
    finally {
        Pop-Location
    }
} else {
    Write-Warn "swag bulunamadi, mevcut swagger.json kullanilacak"
}

# Adim 2: MCP resources'a kopyala
Write-Step "Adim 2: MCP embed kaynagi guncelleniyor..."
$sourceSwagger = Join-Path $GoModulePath "docs\swagger.json"
$mcpSwagger = Join-Path $GoModulePath "mcp\resources\swagger.json"

if (Test-Path $sourceSwagger) {
    Copy-Item -Path $sourceSwagger -Destination $mcpSwagger -Force
    Write-Success "MCP resources swagger.json guncellendi"
} else {
    Write-Fail "Kaynak swagger.json bulunamadi: $sourceSwagger"
    exit 1
}

# Adim 3: openapi.yaml kullanilmiyor - bilgi mesaji
Write-Step "Adim 3: openapi.yaml kullanim kontrolu..."
Write-Info "docs-portal/docs/openapi.yaml kullanilmamaktadir."
Write-Info "Docs portal /api-reference sayfasi canli Swagger UI'ya (/swagger/index.html) yonlendirir."
Write-Info "Bkz: docs-portal/src/pages/api-reference.tsx"

# Adim 4: Degisiklik kontrolu
Write-Step "Adim 4: Degisiklik kontrolu..."
Push-Location $ProjectRoot
try {
    $changes = git status --porcelain 2>$null
    if ($changes) {
        Write-Warn "Bekleyen degisiklikler:"
        $changes | Where-Object { $_ -match "internal/muscle/docs|internal/muscle/mcp" } | ForEach-Object { Write-Info $_ }
        Write-Host ""
        Write-Info "Degisiklikleri commit etmek icin:"
        Write-Info "  git add . && git commit -m 'docs: sync swagger'"
    } else {
        Write-Success "Tum dokumantasyon senkronize"
    }
}
catch {
    Write-Info "Git kontrolu atlandi"
}
finally {
    Pop-Location
}

Write-Host ""
Write-Host "============================================================" -ForegroundColor Green
Write-Host "  Dokumantasyon senkronizasyonu tamamlandi!" -ForegroundColor Green
Write-Host ""
Write-Host "  Tek kaynak: internal/muscle/docs/swagger.json" -ForegroundColor White
Write-Host "  Senkronize: internal/muscle/mcp/resources/swagger.json" -ForegroundColor White
Write-Host ""
Write-Host "  Dikkat: openapi.yaml kullanimdisi (docs portal redirect kullanir)" -ForegroundColor Yellow
Write-Host "============================================================" -ForegroundColor Green
Write-Host ""