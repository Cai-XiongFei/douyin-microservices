[CmdletBinding()]
param(
    [switch]$StopInfrastructure
)

$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
$binDirectory = Join-Path $projectRoot "bin"
$runDirectory = Join-Path $projectRoot ".run"

$serviceNames = @(
    "api",
    "message",
    "relation",
    "comment",
    "favorite",
    "video",
    "user"
)

function Stop-ManagedService {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $pidFile = Join-Path $runDirectory "$Name.pid"
    if (-not (Test-Path -LiteralPath $pidFile)) {
        Write-Host "${Name}: no managed PID file."
        return
    }

    $processID = 0
    $pidText = (Get-Content -LiteralPath $pidFile -Raw).Trim()
    if (-not [int]::TryParse($pidText, [ref]$processID)) {
        Write-Warning "$Name has an invalid PID file; removing the stale file."
        Remove-Item -LiteralPath $pidFile -Force
        return
    }

    $process = Get-Process -Id $processID -ErrorAction SilentlyContinue
    if ($null -eq $process) {
        Write-Host "$Name is already stopped."
        Remove-Item -LiteralPath $pidFile -Force
        return
    }

    $expectedPath = [System.IO.Path]::GetFullPath(
        (Join-Path $binDirectory "$Name.exe")
    )

    $actualPath = $null
    try {
        $actualPath = [System.IO.Path]::GetFullPath($process.Path)
    }
    catch {
        Write-Warning "Cannot verify process $processID for $Name; it will not be stopped."
        return
    }

    if (-not $actualPath.Equals(
        $expectedPath,
        [System.StringComparison]::OrdinalIgnoreCase
    )) {
        Write-Warning "PID $processID does not belong to $expectedPath; it will not be stopped."
        Remove-Item -LiteralPath $pidFile -Force
        return
    }

    Stop-Process -Id $processID -ErrorAction SilentlyContinue

    $deadline = (Get-Date).AddSeconds(5)
    while ((Get-Date) -lt $deadline) {
        if ($null -eq (Get-Process -Id $processID -ErrorAction SilentlyContinue)) {
            break
        }
        Start-Sleep -Milliseconds 200
    }

    if ($null -ne (Get-Process -Id $processID -ErrorAction SilentlyContinue)) {
        Stop-Process -Id $processID -Force
    }

    Remove-Item -LiteralPath $pidFile -Force
    Write-Host "Stopped $Name (PID $processID)."
}

foreach ($name in $serviceNames) {
    Stop-ManagedService -Name $name
}

if ($StopInfrastructure) {
    Push-Location $projectRoot
    try {
        & docker compose stop
        if ($LASTEXITCODE -ne 0) {
            throw "docker compose stop failed."
        }
        Write-Host "Stopped Docker infrastructure."
    }
    finally {
        Pop-Location
    }
}
else {
    Write-Host "Docker infrastructure remains running."
    Write-Host "Use -StopInfrastructure to stop MySQL, Etcd, MinIO, Redis and RabbitMQ too."
}