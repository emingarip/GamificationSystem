[CmdletBinding()]
param(
    [switch]$Build,
    [switch]$Start,
    [switch]$Stop,
    [switch]$Logs,
    [switch]$Interactive,
    [switch]$Clean
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$composeFile = "docker-compose-mcp.yml"

function Write-Step {
    param([string]$Message)
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Build-McpDocker {
    Write-Step "MCP Docker image building..."
    docker build -f Dockerfile.mcp -t gamification-mcp:latest .
    if ($LASTEXITCODE -ne 0) {
        throw "Docker build failed."
    }
    Write-Host "Image built: gamification-mcp:latest" -ForegroundColor Green
}

function Start-McpServices {
    Write-Step "Starting MCP services..."
    docker compose -f $composeFile up -d
    if ($LASTEXITCODE -ne 0) {
        throw "docker compose up failed."
    }

    Write-Host "Waiting for services to be healthy..." -ForegroundColor Yellow
    Start-Sleep -Seconds 5

    # Check service status
    docker compose -f $composeFile ps
}

function Stop-McpServices {
    Write-Step "Stopping MCP services..."
    docker compose -f $composeFile down
}

function Show-Logs {
    Write-Step "MCP Server logs..."
    docker compose -f $composeFile logs -f mcp-server
}

function Run-Interactive {
    Write-Step "Running MCP server interactively..."
    docker compose -f $composeFile run --rm --service-ports mcp-server
}

function Clean-Resources {
    Write-Step "Cleaning up MCP resources..."
    docker compose -f $composeFile down -v
    docker rmi gamification-mcp:latest 2>$null
    Write-Host "Cleanup complete." -ForegroundColor Green
}

# Main execution
Push-Location $repoRoot
try {
    if ($Clean) {
        Clean-Resources
        return
    }

    if ($Build) {
        Build-McpDocker
    }

    if ($Stop) {
        Stop-McpServices
        return
    }

    if ($Logs) {
        Show-Logs
        return
    }

    if ($Interactive) {
        Run-Interactive
        return
    }

    if ($Start) {
        Start-McpServices
        return
    }

    # Default: Build and start
    Build-McpDocker
    Start-McpServices

    Write-Host ""
    Write-Host "MCP Server Docker setup complete!" -ForegroundColor Green
    Write-Host ""
    Write-Host "To view logs:   .\scripts\run-mcp-docker.ps1 -Logs" -ForegroundColor Cyan
    Write-Host "To stop:       .\scripts\run-mcp-docker.ps1 -Stop" -ForegroundColor Cyan
    Write-Host "To clean:      .\scripts\run-mcp-docker.ps1 -Clean" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "To use with MCP clients, configure them to connect to:" -ForegroundColor Yellow
    Write-Host "  - Redis: localhost:6379" -ForegroundColor White
    Write-Host "  - Neo4j: localhost:7475 (HTTP), localhost:7688 (Bolt)" -ForegroundColor White
    Write-Host ""
    Write-Host "Note: The MCP server uses stdio transport (not HTTP)." -ForegroundColor Yellow
    Write-Host "For direct testing, use: .\scripts\run-mcp-docker.ps1 -Interactive" -ForegroundColor White
}
finally {
    Pop-Location
}