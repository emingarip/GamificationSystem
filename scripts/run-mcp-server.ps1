[CmdletBinding()]
param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$BinaryArgs
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

function Import-EnvFile {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    if (-not (Test-Path -LiteralPath $Path)) {
        return
    }

    foreach ($line in Get-Content -LiteralPath $Path) {
        $trimmed = $line.Trim()
        if ([string]::IsNullOrWhiteSpace($trimmed) -or $trimmed.StartsWith("#")) {
            continue
        }

        $separatorIndex = $trimmed.IndexOf("=")
        if ($separatorIndex -lt 1) {
            continue
        }

        $name = $trimmed.Substring(0, $separatorIndex).Trim()
        $value = $trimmed.Substring($separatorIndex + 1)
        $value = $value.Trim()

        if (($value.StartsWith('"') -and $value.EndsWith('"')) -or ($value.StartsWith("'") -and $value.EndsWith("'"))) {
            $value = $value.Substring(1, $value.Length - 2)
        }

        # docker compose escape convention in .env.example/.env
        $value = $value -replace "\$\$", "$"
        Set-Item -Path "Env:$name" -Value $value
    }
}

function Set-DefaultEnv {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,
        [Parameter(Mandatory = $true)]
        [string]$Value
    )

    $current = [Environment]::GetEnvironmentVariable($Name)
    if ([string]::IsNullOrWhiteSpace($current)) {
        Set-Item -Path "Env:$Name" -Value $Value
    }
}

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$muscleRoot = Join-Path $repoRoot "internal\muscle"
$binaryPath = Join-Path $muscleRoot "mcp-server.exe"
$envFile = Join-Path $repoRoot ".env"

Import-EnvFile -Path $envFile

# Local docker-compose ports differ from in-container service ports.
Set-DefaultEnv -Name "REDIS_HOST" -Value "localhost"
Set-DefaultEnv -Name "REDIS_PORT" -Value "6379"
Set-DefaultEnv -Name "NEO4J_URI" -Value "bolt://localhost:7688"
Set-DefaultEnv -Name "NEO4J_USERNAME" -Value "neo4j"
Set-DefaultEnv -Name "NEO4J_PASSWORD" -Value "password"
Set-DefaultEnv -Name "NEO4J_DATABASE" -Value "neo4j"
Set-DefaultEnv -Name "KAFKA_BROKERS" -Value "localhost:9092"
Set-DefaultEnv -Name "KAFKA_TOPIC" -Value "match-events"
Set-DefaultEnv -Name "KAFKA_GROUP_ID" -Value "muscle-rule-engine"
Set-DefaultEnv -Name "LLM_HOST" -Value "localhost"
Set-DefaultEnv -Name "LLM_PORT" -Value "8000"
Set-DefaultEnv -Name "MODEL_NAME" -Value "llama-3-8b"
Set-DefaultEnv -Name "LLM_TEMPERATURE" -Value "0.1"
Set-DefaultEnv -Name "LLM_MAX_TOKENS" -Value "2048"
Set-DefaultEnv -Name "APP_PORT" -Value "3000"
Set-DefaultEnv -Name "WS_PORT" -Value "3001"
Set-DefaultEnv -Name "LOG_LEVEL" -Value "info"

if (-not (Test-Path -LiteralPath $binaryPath)) {
    throw "MCP binary not found: $binaryPath. Run scripts\setup-mcp.ps1 first."
}

Push-Location $muscleRoot
try {
    & $binaryPath @BinaryArgs
    exit $LASTEXITCODE
}
finally {
    Pop-Location
}
