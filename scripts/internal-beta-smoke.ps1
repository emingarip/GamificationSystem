[CmdletBinding()]
param(
    [string]$ApiBaseUrl = "http://localhost:3000/api/v1",
    [string]$HealthUrl = "http://localhost:3000/health",
    [string]$AdminUrl = "http://localhost:5173",
    [string]$AdminEmail,
    [string]$AdminPassword = "admin123",
    [int]$TimeoutSeconds = 180,
    [switch]$StartStack,
    [switch]$SkipCleanup
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

function Write-Step {
    param([string]$Message)
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Get-RepoRoot {
    return Split-Path -Parent $PSScriptRoot
}

function Get-EnvValue {
    param(
        [string]$Path,
        [string]$Key
    )

    if (-not (Test-Path $Path)) {
        return $null
    }

    $line = Get-Content $Path | Where-Object { $_ -match "^$Key=" } | Select-Object -First 1
    if (-not $line) {
        return $null
    }

    return ($line -split "=", 2)[1].Trim()
}

function Wait-Until {
    param(
        [scriptblock]$Condition,
        [string]$Description,
        [int]$TimeoutSeconds = 180
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        try {
            if (& $Condition) {
                return
            }
        } catch {
            Start-Sleep -Seconds 2
            continue
        }

        Start-Sleep -Seconds 2
    }

    throw "Timed out waiting for: $Description"
}

function Invoke-Json {
    param(
        [ValidateSet("GET", "POST", "PUT", "DELETE")]
        [string]$Method,
        [string]$Url,
        [object]$Body,
        [hashtable]$Headers
    )

    $params = @{
        Method      = $Method
        Uri         = $Url
        ErrorAction = "Stop"
    }

    if ($Headers) {
        $params.Headers = $Headers
    }

    if ($null -ne $Body) {
        $params.ContentType = "application/json"
        $params.Body = ($Body | ConvertTo-Json -Depth 10)
    }

    return Invoke-RestMethod @params
}

function Assert-True {
    param(
        [bool]$Condition,
        [string]$Message
    )

    if (-not $Condition) {
        throw $Message
    }
}

$repoRoot = Get-RepoRoot
$envPath = Join-Path $repoRoot ".env"

if (-not $AdminEmail) {
    $AdminEmail = Get-EnvValue -Path $envPath -Key "ADMIN_USERNAME"
}

if (-not $AdminEmail) {
    $AdminEmail = "admin"
}

$headers = @{}
$createdRuleId = $null
$createdBadgeId = $null
$selectedUserId = $null
$originalPoints = $null
$runId = Get-Date -Format "yyyyMMddHHmmss"

try {
    if ($StartStack) {
        Write-Step "Starting docker compose stack"
        Push-Location $repoRoot
        try {
            docker compose up -d neo4j redis zookeeper kafka muscle admin | Out-Host
        } finally {
            Pop-Location
        }
    }

    Write-Step "Waiting for API health endpoint"
    Wait-Until -Description "API health" -TimeoutSeconds $TimeoutSeconds -Condition {
        $health = Invoke-RestMethod -Method GET -Uri $HealthUrl -ErrorAction Stop
        return $health.status -eq "healthy" -or $health.status -eq "degraded"
    }

    Write-Step "Waiting for admin UI"
    Wait-Until -Description "admin UI" -TimeoutSeconds $TimeoutSeconds -Condition {
        $response = Invoke-WebRequest -UseBasicParsing -Uri $AdminUrl -Method GET -ErrorAction Stop
        return $response.StatusCode -eq 200
    }

    Write-Step "Authenticating admin user"
    $loginResponse = Invoke-Json -Method POST -Url "$ApiBaseUrl/auth/login" -Body @{
        email    = $AdminEmail
        password = $AdminPassword
    }
    Assert-True -Condition (-not [string]::IsNullOrWhiteSpace($loginResponse.token)) -Message "Login did not return a token"
    $headers = @{ Authorization = "Bearer $($loginResponse.token)" }

    Write-Step "Fetching users"
    $usersResponse = Invoke-Json -Method GET -Url "$ApiBaseUrl/users" -Body $null -Headers $headers
    Assert-True -Condition ($usersResponse.users.Count -gt 0) -Message "Users list is empty"
    $selectedUserId = if ($usersResponse.users.id -contains "user_1") { "user_1" } else { $usersResponse.users[0].id }

    Write-Step "Reading user profile"
    $beforeProfile = Invoke-Json -Method GET -Url "$ApiBaseUrl/users/$selectedUserId" -Body $null -Headers $headers
    $originalPoints = [int]$beforeProfile.points
    $originalBadgeCount = @($beforeProfile.rich_badge_info).Count

    Write-Step "Creating smoke badge"
    $badgeResponse = Invoke-Json -Method POST -Url "$ApiBaseUrl/badges" -Body @{
        name        = "Smoke Badge $runId"
        description = "Temporary badge for internal smoke validation"
        icon        = "flame"
        points      = 25
        criteria    = "Created by smoke test"
        rarity      = "rare"
    } -Headers $headers
    $createdBadgeId = $badgeResponse.id
    Assert-True -Condition (-not [string]::IsNullOrWhiteSpace($createdBadgeId)) -Message "Badge creation did not return an id"

    Write-Step "Creating smoke rule"
    $createdRuleId = "smoke-rule-$runId"
    $ruleResponse = Invoke-Json -Method POST -Url "$ApiBaseUrl/rules" -Body @{
        id          = $createdRuleId
        name        = "Smoke Rule $runId"
        description = "Temporary rule for internal smoke validation"
        event_type  = "goal"
        points      = 15
        enabled     = $true
        cooldown    = 0
        conditions  = @()
        rewards     = @{
            badge_id = $createdBadgeId
        }
    } -Headers $headers
    Assert-True -Condition ($ruleResponse.id -eq $createdRuleId) -Message "Rule creation failed or returned a different id"

    Write-Step "Running dry-run event test"
    $dryRunResponse = Invoke-Json -Method POST -Url "$ApiBaseUrl/events/test" -Body @{
        event = @{
            event_id   = "smoke-dry-$runId"
            event_type = "goal"
            match_id   = "match_1"
            team_id    = "team_galatasaray"
            player_id  = $selectedUserId
            minute     = 45
            timestamp  = (Get-Date).ToUniversalTime().ToString("o")
            metadata   = @{}
        }
        dry_run = $true
    } -Headers $headers
    Assert-True -Condition ($dryRunResponse.executed -eq $false) -Message "Dry-run executed actions unexpectedly"
    Assert-True -Condition (@($dryRunResponse.actions).Count -gt 0) -Message "Dry-run did not produce any actions"
    Assert-True -Condition (@($dryRunResponse.affected_users) -contains $selectedUserId) -Message "Dry-run did not target the expected user"

    $afterDryRunProfile = Invoke-Json -Method GET -Url "$ApiBaseUrl/users/$selectedUserId" -Body $null -Headers $headers
    Assert-True -Condition ([int]$afterDryRunProfile.points -eq $originalPoints) -Message "Dry-run changed user points"
    Assert-True -Condition (@($afterDryRunProfile.rich_badge_info).Count -eq $originalBadgeCount) -Message "Dry-run changed user badges"

    Write-Step "Running execute event test"
    $executeResponse = Invoke-Json -Method POST -Url "$ApiBaseUrl/events/test" -Body @{
        event = @{
            event_id   = "smoke-exec-$runId"
            event_type = "goal"
            match_id   = "match_1"
            team_id    = "team_galatasaray"
            player_id  = $selectedUserId
            minute     = 46
            timestamp  = (Get-Date).ToUniversalTime().ToString("o")
            metadata   = @{}
        }
        dry_run = $false
    } -Headers $headers
    Assert-True -Condition ($executeResponse.executed -eq $true) -Message "Execute path did not run actions"

    $afterExecuteProfile = Invoke-Json -Method GET -Url "$ApiBaseUrl/users/$selectedUserId" -Body $null -Headers $headers
    Assert-True -Condition ([int]$afterExecuteProfile.points -ge ($originalPoints + 15)) -Message "Execute path did not add expected points"
    $assignedBadge = @($afterExecuteProfile.rich_badge_info) | Where-Object { $_.id -eq $createdBadgeId } | Select-Object -First 1
    Assert-True -Condition ($null -ne $assignedBadge) -Message "Execute path did not assign the smoke badge"

    Write-Step "Verifying analytics endpoints"
    $summary = Invoke-Json -Method GET -Url "$ApiBaseUrl/analytics/summary" -Body $null -Headers $headers
    Assert-True -Condition ($summary.total_users -ge 1) -Message "Analytics summary returned invalid user count"
    $activity = Invoke-Json -Method GET -Url "$ApiBaseUrl/analytics/activity?limit=10" -Body $null -Headers $headers
    Assert-True -Condition (@($activity.activities).Count -ge 1) -Message "Analytics activity returned no entries"

    Write-Host ""
    Write-Host "Smoke test completed successfully." -ForegroundColor Green
    Write-Host "Verified user: $selectedUserId"
    Write-Host "Created badge: $createdBadgeId"
    Write-Host "Created rule: $createdRuleId"
}
catch {
    Write-Host ""
    Write-Host "Smoke test failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
finally {
    if ($SkipCleanup) {
        Write-Step "Skipping cleanup by request"
    } elseif ($headers.Count -gt 0) {
        try {
            if ($null -ne $selectedUserId -and $null -ne $originalPoints) {
                Write-Step "Restoring original user points"
                $currentProfile = Invoke-Json -Method GET -Url "$ApiBaseUrl/users/$selectedUserId" -Body $null -Headers $headers
                $currentPoints = [int]$currentProfile.points
                $delta = $originalPoints - $currentPoints

                if ($delta -gt 0) {
                    Invoke-Json -Method PUT -Url "$ApiBaseUrl/users/$selectedUserId/points" -Body @{
                        points    = [int]$delta
                        operation = "add"
                    } -Headers $headers | Out-Null
                } elseif ($delta -lt 0) {
                    Invoke-Json -Method PUT -Url "$ApiBaseUrl/users/$selectedUserId/points" -Body @{
                        points    = [int](-1 * $delta)
                        operation = "subtract"
                    } -Headers $headers | Out-Null
                }
            }
        } catch {
            Write-Warning "Failed to restore user points: $($_.Exception.Message)"
        }

        try {
            if ($createdRuleId) {
                Write-Step "Deleting smoke rule"
                Invoke-Json -Method DELETE -Url "$ApiBaseUrl/rules/$createdRuleId" -Body $null -Headers $headers | Out-Null
            }
        } catch {
            Write-Warning "Failed to delete smoke rule: $($_.Exception.Message)"
        }

        try {
            if ($createdBadgeId) {
                Write-Step "Deleting smoke badge"
                Invoke-Json -Method DELETE -Url "$ApiBaseUrl/badges/$createdBadgeId" -Body $null -Headers $headers | Out-Null
            }
        } catch {
            Write-Warning "Failed to delete smoke badge: $($_.Exception.Message)"
        }
    }
}
