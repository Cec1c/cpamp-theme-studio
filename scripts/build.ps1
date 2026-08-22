[CmdletBinding()]
param(
    [string]$Version = '0.1.5-dev',
    [string]$Target = '',
    [string]$OutputDirectory = 'dist',
    [switch]$SkipTests
)

$ErrorActionPreference = 'Stop'
$pluginRoot = Split-Path -Parent $PSScriptRoot
$versionValue = $Version.TrimStart('v')
if ($versionValue -notmatch '^[0-9][0-9A-Za-z.+-]*$') {
    throw "Invalid plugin version: $Version"
}

$hostOS = (go env GOHOSTOS).Trim()
$hostArch = (go env GOHOSTARCH).Trim()
if ([string]::IsNullOrWhiteSpace($Target)) {
    $Target = "$hostOS-$hostArch"
}
$targetParts = $Target.Split('-', 2)
if ($targetParts.Count -ne 2) {
    throw "Target must use <goos>-<goarch>, for example windows-amd64."
}
$targetOS = $targetParts[0]
$targetArch = $targetParts[1]
if ($targetOS -notin @('windows', 'linux', 'darwin') -or $targetArch -notin @('amd64', 'arm64')) {
    throw "Unsupported target: $Target"
}
if ($targetOS -ne $hostOS -or $targetArch -ne $hostArch) {
    throw "CGO shared libraries must be built on a native $Target runner (current host: $hostOS-$hostArch)."
}

$extension = switch ($targetOS) {
    'windows' { '.dll' }
    'darwin' { '.dylib' }
    default { '.so' }
}
$outputPath = if ([IO.Path]::IsPathRooted($OutputDirectory)) {
    [IO.Path]::GetFullPath($OutputDirectory)
} else {
    [IO.Path]::GetFullPath((Join-Path $pluginRoot $OutputDirectory))
}
$libraryPath = Join-Path $outputPath "cpamp-theme-studio$extension"
$headerPath = Join-Path $outputPath 'cpamp-theme-studio.h'

$oldCGO = $env:CGO_ENABLED
$oldGOOS = $env:GOOS
$oldGOARCH = $env:GOARCH
Push-Location $pluginRoot
try {
    if (-not $SkipTests) {
        go test ./...
        if ($LASTEXITCODE -ne 0) { throw "go test failed with exit code $LASTEXITCODE" }
        $node = Get-Command node -ErrorAction SilentlyContinue
        if ($null -ne $node) {
            node --check assets/loader.js
            if ($LASTEXITCODE -ne 0) { throw "loader syntax check failed with exit code $LASTEXITCODE" }
        }
    }
    New-Item -ItemType Directory -Force -Path $outputPath | Out-Null
    $env:CGO_ENABLED = '1'
    $env:GOOS = $targetOS
    $env:GOARCH = $targetArch
    go build -trimpath -buildmode=c-shared -ldflags "-s -w -X=main.pluginVersion=$versionValue" -o $libraryPath .
    if ($LASTEXITCODE -ne 0) { throw "go build failed with exit code $LASTEXITCODE" }
    if (Test-Path -LiteralPath $headerPath) {
        Remove-Item -LiteralPath $headerPath
    }
    Get-Item -LiteralPath $libraryPath
} finally {
    $env:CGO_ENABLED = $oldCGO
    $env:GOOS = $oldGOOS
    $env:GOARCH = $oldGOARCH
    Pop-Location
}
