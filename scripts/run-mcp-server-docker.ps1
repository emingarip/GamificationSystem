[CmdletBinding()]
param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ServerArgs
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$composeFile = Join-Path $repoRoot "docker-compose-mcp.yml"

if (-not (Test-Path -LiteralPath $composeFile)) {
    throw "docker-compose-mcp.yml bulunamadi: $composeFile"
}

Push-Location $repoRoot
try {
    # MCP stdio stream'ini bozmamak icin bootstrap ciktilarini tamamen sustur.
    $bootstrapCommand = 'docker compose -f "{0}" up -d redis neo4j mcp-server >nul 2>nul' -f $composeFile
    cmd /c $bootstrapCommand | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "Docker MCP stack baslatilamadi."
    }

    if (-not $ServerArgs -or $ServerArgs.Count -eq 0) {
        $ServerArgs = @("-transport", "stdio")
    }

    & docker exec -i gamification-mcp-server /app/mcp-server @ServerArgs
    exit $LASTEXITCODE
}
finally {
    Pop-Location
}
