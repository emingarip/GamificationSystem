[CmdletBinding()]
param(
    [switch]$SkipDeps,
    [switch]$SkipBuild,
    [switch]$UseDocker
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

function Write-Step {
    param([string]$Message)
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Ensure-EnvFile {
    param(
        [Parameter(Mandatory = $true)]
        [string]$EnvPath,
        [Parameter(Mandatory = $true)]
        [string]$ExamplePath
    )

    if (-not (Test-Path -LiteralPath $EnvPath)) {
        if (-not (Test-Path -LiteralPath $ExamplePath)) {
            throw ".env.example bulunamadı: $ExamplePath"
        }
        Copy-Item -LiteralPath $ExamplePath -Destination $EnvPath
        Write-Host ".env dosyası .env.example üzerinden oluşturuldu." -ForegroundColor Yellow
    }
}

function Wait-ForContainerHealthy {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ContainerName,
        [int]$TimeoutSeconds = 120
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)

    while ((Get-Date) -lt $deadline) {
        $state = docker inspect --format "{{.State.Health.Status}}" $ContainerName 2>$null
        if ($LASTEXITCODE -eq 0 -and $state -eq "healthy") {
            return
        }
        Start-Sleep -Seconds 2
    }

    throw "Container healthy duruma gelmedi: $ContainerName"
}

function Build-McpBinary {
    param(
        [Parameter(Mandatory = $true)]
        [string]$RepoRoot
    )

    $muscleRoot = Join-Path $RepoRoot "internal\muscle"
    $binaryPath = Join-Path $muscleRoot "mcp-server.exe"
    $go = Get-Command go -ErrorAction SilentlyContinue

    if ($go) {
        Push-Location $muscleRoot
        try {
            & $go.Source build -o $binaryPath .\cmd\mcp-server
        }
        finally {
            Pop-Location
        }
        return $binaryPath
    }

    docker run --rm -e GOOS=windows -e GOARCH=amd64 -v "${RepoRoot}:/src" -w /src/internal/muscle golang:1.25 go build -o /src/internal/muscle/mcp-server.exe ./cmd/mcp-server
    if ($LASTEXITCODE -ne 0) {
        throw "Docker ile MCP binary build başarısız oldu."
    }

    return $binaryPath
}

function Write-ClientConfigs {
    param(
        [Parameter(Mandatory = $true)]
        [string]$RepoRoot
    )

    $configDir = Join-Path $RepoRoot "scripts\mcp-configs"
    New-Item -ItemType Directory -Force -Path $configDir | Out-Null

    $launcherPath = (Resolve-Path (Join-Path $RepoRoot "scripts\run-mcp-server.cmd")).Path
    $payload = @{
        mcpServers = @{
            gamification = @{
                command = $launcherPath
                args    = @()
            }
        }
    } | ConvertTo-Json -Depth 10

    Set-Content -LiteralPath (Join-Path $configDir "cursor.mcp.json") -Value $payload -Encoding UTF8
    Set-Content -LiteralPath (Join-Path $configDir "claude-desktop.mcp.json") -Value $payload -Encoding UTF8
    Set-Content -LiteralPath (Join-Path $configDir "generic-mcp.json") -Value $payload -Encoding UTF8

    return $configDir
}

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$envPath = Join-Path $repoRoot ".env"
$envExamplePath = Join-Path $repoRoot ".env.example"

Write-Step "MCP setup başlıyor"
Ensure-EnvFile -EnvPath $envPath -ExamplePath $envExamplePath

if (-not $SkipDeps) {
    Write-Step "Redis ve Neo4j başlatılıyor"
    docker compose up -d redis neo4j
    if ($LASTEXITCODE -ne 0) {
        throw "docker compose up başarısız oldu."
    }

    Write-Step "Container health kontrol ediliyor"
    Wait-ForContainerHealthy -ContainerName "gamification-redis"
    Wait-ForContainerHealthy -ContainerName "gamification-neo4j"
}

$binaryPath = Join-Path $repoRoot "internal\muscle\mcp-server.exe"
if (-not $SkipBuild) {
    Write-Step "MCP binary build ediliyor"
    $binaryPath = Build-McpBinary -RepoRoot $repoRoot
}

Write-Step "Agent config snippet'leri üretiliyor"
$configDir = Write-ClientConfigs -RepoRoot $repoRoot

if ($UseDocker) {
    Write-Step "Docker tabanlı MCP kurulumu"
    
    # Build Docker image
    Write-Host "MCP Docker image building..." -ForegroundColor Yellow
    docker build -f Dockerfile.mcp -t gamification-mcp:latest .
    if ($LASTEXITCODE -ne 0) {
        throw "Docker build failed."
    }
    
    # Start Docker Compose services
    docker compose -f docker-compose-mcp.yml up -d
    if ($LASTEXITCODE -ne 0) {
        throw "docker compose up failed."
    }
    
    Write-Host "MCP Docker services started." -ForegroundColor Green
}

Write-Host ""
Write-Host "Hazır." -ForegroundColor Green
if ($UseDocker) {
    Write-Host "Mode    : Docker (docker-compose-mcp.yml)" -ForegroundColor Cyan
    Write-Host "Services: redis, neo4j, mcp-server" -ForegroundColor Cyan
} else {
    Write-Host "Launcher: $(Join-Path $repoRoot 'scripts\run-mcp-server.cmd')" -ForegroundColor Cyan
}
Write-Host "Binary  : $binaryPath"
Write-Host "Config  : $configDir"
Write-Host ""
Write-Host "Agent config'ine doğrudan şu dosyayı verebilirsin:" -ForegroundColor Cyan
Write-Host "  $configDir\generic-mcp.json"
