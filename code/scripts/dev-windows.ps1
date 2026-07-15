$ErrorActionPreference = "Stop"
$CodeRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$ServerRoot = Join-Path $CodeRoot "server"
$ClientRoot = Join-Path $CodeRoot "client"

if (-not (Test-Path (Join-Path $CodeRoot "..\.env"))) {
    throw "Root .env file is missing. Run .\scripts\setup-windows.ps1 first."
}

Start-Process powershell -ArgumentList @(
    "-NoExit",
    "-Command",
    "Set-Location '$ServerRoot'; go run ./cmd/api"
)
Start-Process powershell -ArgumentList @(
    "-NoExit",
    "-Command",
    "Set-Location '$ClientRoot'; npm run dev"
)

Write-Host "Backend and frontend terminals have been opened."
Write-Host "Frontend: http://localhost:3000"
Write-Host "Backend health: http://localhost:8080/api/health"
