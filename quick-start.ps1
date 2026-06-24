param(
    [int]$BackendPort = 8080,
    [int]$FrontendPort = 5173,
    [ValidateSet("mock", "eino")]
    [string]$Planner = "mock",
    [switch]$SkipInstall,
    [switch]$OpenBrowser,
    [switch]$Stop
)

$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$WebDir = Join-Path $Root "web"
$RunDir = Join-Path $Root ".tmp\dev"
$BackendPidFile = Join-Path $RunDir "backend.pid"
$FrontendPidFile = Join-Path $RunDir "frontend.pid"
$BackendLog = Join-Path $RunDir "backend.log"
$FrontendLog = Join-Path $RunDir "frontend.log"
$BackendErr = Join-Path $RunDir "backend.err.log"
$FrontendErr = Join-Path $RunDir "frontend.err.log"

function Write-Step($Message) {
    Write-Host "==> $Message"
}

function Resolve-Go {
    $fromPath = Get-Command "go" -ErrorAction SilentlyContinue
    if ($fromPath) {
        return $fromPath.Source
    }

    $defaultGo = "C:\Program Files\Go\bin\go.exe"
    if (Test-Path -LiteralPath $defaultGo) {
        return $defaultGo
    }

    throw "Go was not found. Install Go or add go.exe to PATH."
}

function Resolve-Npm {
    $npmCmd = Get-Command "npm.cmd" -ErrorAction SilentlyContinue
    if ($npmCmd) {
        return $npmCmd.Source
    }

    $npm = Get-Command "npm" -ErrorAction SilentlyContinue
    if ($npm) {
        return $npm.Source
    }

    throw "npm was not found. Install Node.js or add npm to PATH."
}

function Test-PortInUse([int]$Port) {
    return [bool](Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue)
}

function Stop-FromPidFile($PidFile, $Name) {
    if (-not (Test-Path -LiteralPath $PidFile)) {
        return
    }

    $rawPid = (Get-Content -LiteralPath $PidFile -ErrorAction SilentlyContinue | Select-Object -First 1)
    if (-not $rawPid) {
        Remove-Item -LiteralPath $PidFile -Force -ErrorAction SilentlyContinue
        return
    }

    $processId = [int]$rawPid
    $process = Get-Process -Id $processId -ErrorAction SilentlyContinue
    if ($process) {
        Write-Step "Stopping $Name process tree pid=$processId"
        & taskkill.exe /PID $processId /T /F | Out-Null
    }

    Remove-Item -LiteralPath $PidFile -Force -ErrorAction SilentlyContinue
}

function Wait-Http($Url, [int]$Seconds) {
    $deadline = (Get-Date).AddSeconds($Seconds)
    while ((Get-Date) -lt $deadline) {
        try {
            $request = [System.Net.WebRequest]::Create($Url)
            $request.Timeout = 1500
            $response = $request.GetResponse()
            $response.Close()
            return $true
        } catch [System.Net.WebException] {
            if ($_.Exception.Response) {
                $_.Exception.Response.Close()
                return $true
            }
        } catch {
            Start-Sleep -Milliseconds 500
        }
        Start-Sleep -Milliseconds 500
    }
    return $false
}

if ($Stop) {
    Stop-FromPidFile $FrontendPidFile "frontend"
    Stop-FromPidFile $BackendPidFile "backend"
    Write-Step "Stopped Travel Agent dev services"
    exit 0
}

New-Item -ItemType Directory -Force -Path $RunDir | Out-Null

$GoExe = Resolve-Go
$NpmExe = Resolve-Npm

if (-not (Test-Path -LiteralPath $WebDir)) {
    throw "web directory was not found: $WebDir"
}

if (Test-PortInUse $BackendPort) {
    throw "Backend port $BackendPort is already in use. Pass -BackendPort <port> or stop the existing process."
}

if (Test-PortInUse $FrontendPort) {
    throw "Frontend port $FrontendPort is already in use. Pass -FrontendPort <port> or stop the existing process."
}

if (-not $SkipInstall -and -not (Test-Path -LiteralPath (Join-Path $WebDir "node_modules"))) {
    Write-Step "Installing frontend dependencies"
    Push-Location $WebDir
    try {
        & $NpmExe install
    } finally {
        Pop-Location
    }
}

$BackendRunner = Join-Path $RunDir "backend-run.ps1"
$FrontendRunner = Join-Path $RunDir "frontend-run.ps1"

@"
`$ErrorActionPreference = "Stop"
Set-Location '$Root'
`$env:TRAVEL_AGENT_HTTP_ADDR = ':$BackendPort'
`$env:TRAVEL_AGENT_PLANNER = '$Planner'
& '$GoExe' run ./cmd/server
"@ | Set-Content -LiteralPath $BackendRunner -Encoding UTF8

@"
`$ErrorActionPreference = "Stop"
Set-Location '$WebDir'
`$env:VITE_API_BASE_URL = 'http://localhost:$BackendPort'
& '$NpmExe' run dev -- --host 127.0.0.1 --port $FrontendPort --strictPort
"@ | Set-Content -LiteralPath $FrontendRunner -Encoding UTF8

Write-Step "Starting backend on http://localhost:$BackendPort planner=$Planner"
$backend = Start-Process -FilePath "powershell.exe" `
    -ArgumentList "-NoProfile -ExecutionPolicy Bypass -File `"$BackendRunner`"" `
    -WorkingDirectory $Root `
    -WindowStyle Hidden `
    -RedirectStandardOutput $BackendLog `
    -RedirectStandardError $BackendErr `
    -PassThru
$backend.Id | Set-Content -LiteralPath $BackendPidFile -Encoding ASCII

if (-not (Wait-Http "http://localhost:$BackendPort/api/v1/travel/plans/__health_probe__" 20)) {
    Write-Warning "Backend did not respond within 20 seconds. Check $BackendLog and $BackendErr"
}

Write-Step "Starting frontend on http://127.0.0.1:$FrontendPort"
$frontend = Start-Process -FilePath "powershell.exe" `
    -ArgumentList "-NoProfile -ExecutionPolicy Bypass -File `"$FrontendRunner`"" `
    -WorkingDirectory $WebDir `
    -WindowStyle Hidden `
    -RedirectStandardOutput $FrontendLog `
    -RedirectStandardError $FrontendErr `
    -PassThru
$frontend.Id | Set-Content -LiteralPath $FrontendPidFile -Encoding ASCII

if (-not (Wait-Http "http://127.0.0.1:$FrontendPort/" 20)) {
    Write-Warning "Frontend did not respond within 20 seconds. Check $FrontendLog and $FrontendErr"
}

Write-Host ""
Write-Host "Travel Agent is running:"
Write-Host "  Frontend: http://127.0.0.1:$FrontendPort"
Write-Host "  Backend:  http://localhost:$BackendPort"
Write-Host ""
Write-Host "Logs:"
Write-Host "  Backend:  $BackendLog"
Write-Host "  Frontend: $FrontendLog"
Write-Host ""
Write-Host "Stop:"
Write-Host "  .\quick-start.ps1 -Stop"

if ($OpenBrowser) {
    Start-Process "http://127.0.0.1:$FrontendPort"
}
