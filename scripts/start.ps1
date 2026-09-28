[CmdletBinding()]
param(
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
$binDirectory = Join-Path $projectRoot "bin"
$logDirectory = Join-Path $projectRoot "logs"
$runDirectory = Join-Path $projectRoot ".run"
$runStamp = Get-Date -Format "yyyyMMdd-HHmmss"

$rpcServices = @(
    [PSCustomObject]@{ Name = "user";     Package = "./cmd/user";     Port = 18085 },
    [PSCustomObject]@{ Name = "video";    Package = "./cmd/video";    Port = 18086 },
    [PSCustomObject]@{ Name = "favorite"; Package = "./cmd/favorite"; Port = 18087 },
    [PSCustomObject]@{ Name = "comment";  Package = "./cmd/comment";  Port = 18088 },
    [PSCustomObject]@{ Name = "relation"; Package = "./cmd/relation"; Port = 18090 },
    [PSCustomObject]@{ Name = "message";  Package = "./cmd/message";  Port = 18091 }
)

$apiService = [PSCustomObject]@{
    Name = "api"
    Package = "./cmd/api"
    Port = 18089
}

$allServices = @($rpcServices) + @($apiService)

function Test-TcpPort {
    param(
        [Parameter(Mandatory = $true)]
        [int]$Port
    )

    $client = New-Object System.Net.Sockets.TcpClient
    try {
        $task = $client.ConnectAsync("127.0.0.1", $Port)
        if (-not $task.Wait(300)) {
            return $false
        }
        return $client.Connected
    }
    catch {
        return $false
    }
    finally {
        $client.Dispose()
    }
}

function Wait-TcpPort {
    param(
        [Parameter(Mandatory = $true)]
        [int]$Port,

        [int]$TimeoutSeconds = 30
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        if (Test-TcpPort -Port $Port) {
            return $true
        }
        Start-Sleep -Milliseconds 500
    }

    return $false
}

function Test-ManagedServiceRunning {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $pidFile = Join-Path $runDirectory "$Name.pid"
    if (-not (Test-Path -LiteralPath $pidFile)) {
        return $false
    }

    $processID = 0
    $pidText = (Get-Content -LiteralPath $pidFile -Raw).Trim()
    if (-not [int]::TryParse($pidText, [ref]$processID)) {
        return $false
    }

    return $null -ne (Get-Process -Id $processID -ErrorAction SilentlyContinue)
}

function Build-Service {
    param(
        [Parameter(Mandatory = $true)]
        $Service
    )

    $outputPath = Join-Path $binDirectory "$($Service.Name).exe"
    Write-Host "Building $($Service.Name)..."

    & $script:goExecutable build -o $outputPath $Service.Package
    if ($LASTEXITCODE -ne 0) {
        throw "Build failed for $($Service.Name)."
    }
}

function Start-ServiceProcess {
    param(
        [Parameter(Mandatory = $true)]
        $Service
    )

    if (Test-TcpPort -Port $Service.Port) {
        Write-Warning "$($Service.Name) was not started because port $($Service.Port) is already in use."
        return
    }

    $executable = Join-Path $binDirectory "$($Service.Name).exe"
    if (-not (Test-Path -LiteralPath $executable)) {
        throw "Executable not found: $executable"
    }

    $stdoutLog = Join-Path $logDirectory "$($Service.Name)-$runStamp.out.log"
    $stderrLog = Join-Path $logDirectory "$($Service.Name)-$runStamp.err.log"

    $process = Start-Process `
        -FilePath $executable `
        -WorkingDirectory $projectRoot `
        -RedirectStandardOutput $stdoutLog `
        -RedirectStandardError $stderrLog `
        -WindowStyle Hidden `
        -PassThru

    $pidFile = Join-Path $runDirectory "$($Service.Name).pid"
    Set-Content -LiteralPath $pidFile -Value $process.Id -Encoding ascii

    Write-Host "Started $($Service.Name) (PID $($process.Id), port $($Service.Port))."
}

New-Item -ItemType Directory -Path $binDirectory -Force | Out-Null
New-Item -ItemType Directory -Path $logDirectory -Force | Out-Null
New-Item -ItemType Directory -Path $runDirectory -Force | Out-Null

Push-Location $projectRoot
try {
    $dockerCommand = Get-Command docker -ErrorAction SilentlyContinue
    if ($null -eq $dockerCommand) {
        throw "docker was not found. Open a new terminal or fix Docker PATH."
    }

    & docker info *> $null
    if ($LASTEXITCODE -ne 0) {
        throw "Docker Desktop is not running. Start Docker Desktop and try again."
    }

    Write-Host "Starting MySQL, Etcd, MinIO, Redis and RabbitMQ..."
    & docker compose up -d
    if ($LASTEXITCODE -ne 0) {
        throw "docker compose up failed."
    }

    foreach ($port in @(3309, 12379, 19000, 16379, 15672)) {
        if (-not (Wait-TcpPort -Port $port -TimeoutSeconds 60)) {
            throw "Infrastructure port $port did not become ready."
        }
    }

    $goCommand = Get-Command go -ErrorAction SilentlyContinue
    if ($null -eq $goCommand) {
        throw "go was not found in PATH."
    }
    $script:goExecutable = $goCommand.Source

    if (-not $SkipBuild) {
        $managedRunning = @(
            $allServices | Where-Object {
                Test-ManagedServiceRunning -Name $_.Name
            }
        )

        if ($managedRunning.Count -gt 0) {
            $names = ($managedRunning.Name -join ", ")
            throw "Managed services are already running: $names. Run scripts/stop.ps1 before rebuilding."
        }

        foreach ($service in $allServices) {
            Build-Service -Service $service
        }
    }

    foreach ($service in $rpcServices) {
        Start-ServiceProcess -Service $service
    }

    foreach ($service in $rpcServices) {
        if (-not (Wait-TcpPort -Port $service.Port -TimeoutSeconds 30)) {
            throw "$($service.Name) did not start on port $($service.Port). Check logs/."
        }
    }

    Start-ServiceProcess -Service $apiService
    if (-not (Wait-TcpPort -Port $apiService.Port -TimeoutSeconds 30)) {
        throw "api did not start on port $($apiService.Port). Check logs/."
    }

    Write-Host ""
    Write-Host "All services are ready."
    Write-Host "API:    http://127.0.0.1:18089"
    Write-Host "Health: http://127.0.0.1:18089/healthz"
    Write-Host "Logs:   $logDirectory"
}
finally {
    Pop-Location
}