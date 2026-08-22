[CmdletBinding()]
param(
    [string]$Version = '0.1.5-dev',
    [string]$Target = ''
)

$ErrorActionPreference = 'Stop'
$pluginRoot = Split-Path -Parent $PSScriptRoot
$distPath = Join-Path $pluginRoot 'dist'
$versionValue = $Version.TrimStart('v')
if ($versionValue -notmatch '^[0-9][0-9A-Za-z.+-]*$') {
    throw "Invalid plugin version: $Version"
}
if ([string]::IsNullOrWhiteSpace($Target)) {
    $Target = "$(go env GOHOSTOS)-$(go env GOHOSTARCH)"
}
$targetParts = $Target.Split('-', 2)
if ($targetParts.Count -ne 2) { throw "Invalid target: $Target" }
$targetOS = $targetParts[0]
$targetArch = $targetParts[1]
if ($targetOS -notin @('windows', 'linux', 'darwin') -or $targetArch -notin @('amd64', 'arm64')) {
    throw "Unsupported target: $Target"
}
$extension = switch ($targetOS) {
    'windows' { '.dll' }
    'darwin' { '.dylib' }
    default { '.so' }
}
$stagePath = Join-Path $distPath ".package-$Target"
$archiveName = "cpamp-theme-studio_${versionValue}_${targetOS}_${targetArch}.zip"
$archivePath = Join-Path $distPath $archiveName
$distFull = [IO.Path]::GetFullPath($distPath)
$stageFull = [IO.Path]::GetFullPath($stagePath)
if (-not $stageFull.StartsWith($distFull + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Unsafe package staging path: $stageFull"
}

if (Test-Path -LiteralPath $stageFull) {
    Remove-Item -LiteralPath $stageFull -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $stageFull | Out-Null
try {
    & (Join-Path $PSScriptRoot 'build.ps1') -Version $versionValue -Target $Target -OutputDirectory $stageFull
    Copy-Item -LiteralPath (Join-Path $pluginRoot 'LICENSE') -Destination $stageFull
    Copy-Item -LiteralPath (Join-Path $pluginRoot 'README.md') -Destination $stageFull
    Copy-Item -LiteralPath (Join-Path $pluginRoot 'README.zh-CN.md') -Destination $stageFull
    Copy-Item -LiteralPath (Join-Path $pluginRoot 'THIRD_PARTY_NOTICES.md') -Destination $stageFull
    Copy-Item -LiteralPath (Join-Path $pluginRoot 'assets\fonts\OFL.txt') -Destination (Join-Path $stageFull 'JETBRAINS_MONO_OFL.txt')
    Copy-Item -LiteralPath (Join-Path $pluginRoot 'docs') -Destination $stageFull -Recurse
    if (Test-Path -LiteralPath $archivePath) {
        Remove-Item -LiteralPath $archivePath
    }
    Compress-Archive -LiteralPath (Get-ChildItem -LiteralPath $stageFull | Select-Object -ExpandProperty FullName) -DestinationPath $archivePath -CompressionLevel Optimal
    $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $archivePath).Hash.ToLowerInvariant()
    "$hash  $archiveName" | Set-Content -LiteralPath (Join-Path $distPath 'checksums.txt') -Encoding ascii
    Get-Item -LiteralPath $archivePath
} finally {
    if (Test-Path -LiteralPath $stageFull) {
        Remove-Item -LiteralPath $stageFull -Recurse -Force
    }
}
