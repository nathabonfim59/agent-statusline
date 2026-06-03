$ErrorActionPreference = "Stop"

$Repo      = "nathabonfim59/agent-statusline"
$Binary    = "claude-statusline"
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { "$env:USERPROFILE\.local\bin" }

# detect arch
$Arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { "amd64" }
    "ARM64" { "arm64" }
    default { Write-Error "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"; exit 1 }
}

# resolve latest version
if (-not $env:VERSION) {
    $release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $release.tag_name
} else {
    $Version = $env:VERSION
}

if (-not $Version) {
    Write-Error "Could not determine latest release version"
    exit 1
}

$Filename = "$Binary-windows-$Arch.exe"
$Url      = "https://github.com/$Repo/releases/download/$Version/$Filename"
$TmpFile  = "$env:TEMP\$Binary.exe"
$Dest     = "$InstallDir\$Binary.exe"

Write-Host "Downloading $Binary $Version for windows/$Arch..."
Invoke-WebRequest -Uri $Url -OutFile $TmpFile

if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir | Out-Null
}

Move-Item -Force $TmpFile $Dest
Write-Host "Installed $Binary $Version -> $Dest"

# check if install dir is in PATH
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
$inPath   = ($userPath -split ";") -contains $InstallDir

if (-not $inPath) {
    Write-Host ""
    Write-Host "$InstallDir is not in your PATH."
    Write-Host "To add it permanently, run:"
    Write-Host ""
    Write-Host "  [Environment]::SetEnvironmentVariable('PATH', `$env:PATH + ';$InstallDir', 'User')"
    Write-Host ""
    Write-Host "Then restart your terminal."
}
