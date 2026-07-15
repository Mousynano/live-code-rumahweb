[CmdletBinding()]
param(
    [string]$AdminPassword,
    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"
$CodeRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$RepoRoot = (Resolve-Path (Join-Path $CodeRoot "..")).Path
$EnvPath = Join-Path $RepoRoot ".env"
$EnvExamplePath = Join-Path $RepoRoot ".env.example"

function Require-Command {
    param([string]$Name, [string]$InstallHint)
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "$Name is not installed. $InstallHint"
    }
}

function Set-EnvLine {
    param([string]$Path, [string]$Key, [string]$Value)
    $lines = Get-Content $Path
    $updated = $false
    $lines = $lines | ForEach-Object {
        if ($_ -match "^$([regex]::Escape($Key))=") {
            $updated = $true
            "$Key=$Value"
        } else {
            $_
        }
    }
    if (-not $updated) {
        $lines += "$Key=$Value"
    }
    Set-Content -Path $Path -Value $lines -Encoding utf8
}

Require-Command "go" "Install the current stable Windows x64 MSI from https://go.dev/dl/."
Require-Command "node" "Install Node.js 24 LTS or newer."
Require-Command "npm" "npm is normally installed together with Node.js."

$GoVersion = (& go version)
Write-Host "Using $GoVersion"
Write-Host "Using Node $(& node --version) and npm $(& npm --version)"

if (-not (Test-Path $EnvPath)) {
    Copy-Item $EnvExamplePath $EnvPath
    Write-Host "Created $EnvPath from .env.example"
}

$randomBytes = New-Object byte[] 48
[System.Security.Cryptography.RandomNumberGenerator]::Fill($randomBytes)
$jwtSecret = [Convert]::ToBase64String($randomBytes)
Set-EnvLine -Path $EnvPath -Key "JWT_SECRET" -Value $jwtSecret

if ($AdminPassword) {
    Push-Location (Join-Path $CodeRoot "server")
    try {
        $passwordHash = (& go run ./cmd/hash-password $AdminPassword).Trim()
    } finally {
        Pop-Location
    }
    Set-EnvLine -Path $EnvPath -Key "ADMIN_PASSWORD_HASH" -Value $passwordHash
    Write-Host "ADMIN_PASSWORD_HASH has been generated."
} else {
    Write-Warning "ADMIN_PASSWORD_HASH is still a placeholder. Re-run with: .\scripts\setup-windows.ps1 -AdminPassword 'your-password'"
}

Push-Location (Join-Path $CodeRoot "server")
try {
    & go mod download
    if (-not $SkipTests) {
        & go test ./...
    }
} finally {
    Pop-Location
}

Push-Location (Join-Path $CodeRoot "client")
try {
    & npm ci
    if (-not $SkipTests) {
        & npm test
    }
} finally {
    Pop-Location
}

Write-Host ""
Write-Host "Setup complete. Put your real WHOISFREAKS_API_KEY in $EnvPath, then run:"
Write-Host "  .\scripts\dev-windows.ps1"
