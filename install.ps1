$ErrorActionPreference = "Stop"

$Repo       = "nathabonfim59/agent-statusline"
$Binary     = "agent-statusline"
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { "$env:USERPROFILE\.local\bin" }
$ConfigDir  = if ($env:CONFIG_DIR) { $env:CONFIG_DIR } else { "$env:APPDATA\agent-statusline" }

# Detect architecture
$Arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { "amd64" }
    "ARM64" { "arm64" }
    default { Write-Error "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"; exit 1 }
}

# Resolve the latest v* release
$Version = if ($env:VERSION) { $env:VERSION } else { $null }

if (-not $Version -and (Get-Command gh -ErrorAction SilentlyContinue)) {
    try {
        $Version = (gh release view --repo $Repo --json tagName -q .tagName 2>$null)
    } catch {
        $Version = $null
    }
}

if (-not $Version) {
    $release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $release.tag_name
}

if (-not $Version) {
    Write-Error "Could not determine latest release version"
    exit 1
}

# Download the Windows archive
$Filename = "$Binary`_$Version`_windows_$Arch.zip"
$Url      = "https://github.com/$Repo/releases/download/$Version/$Filename"
$TmpDir   = "$env:TEMP\$Binary-$Version"
$Archive  = "$TmpDir\$Filename"

if (-not (Test-Path $TmpDir)) {
    New-Item -ItemType Directory -Path $TmpDir | Out-Null
}

Write-Host "Downloading $Binary $Version for windows/$Arch..."
$downloaded = $false
if (Get-Command gh -ErrorAction SilentlyContinue) {
    try {
        gh release download $Version --repo $Repo --pattern $Filename --dir $TmpDir
        if (Test-Path $Archive) {
            $downloaded = $true
        }
    } catch {
        $downloaded = $false
    }
}

if (-not $downloaded) {
    Invoke-WebRequest -Uri $Url -OutFile $Archive
}

if (-not (Test-Path $Archive)) {
    Write-Error "Download failed: $Archive not found"
    exit 1
}

Write-Host "Extracting $Filename..."
Expand-Archive -Path $Archive -DestinationPath $TmpDir -Force

# Install binary
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir | Out-Null
}

$Dest = "$InstallDir\$Binary.exe"
Write-Host "Installing to $Dest..."
Move-Item -Force "$TmpDir\$Binary.exe" $Dest

# Install bundled themes and example config once
if (-not $env:NO_INSTALL_CONFIG -and -not (Test-Path $ConfigDir)) {
    Write-Host "Installing default themes and example config to $ConfigDir..."
    New-Item -ItemType Directory -Path $ConfigDir | Out-Null
    if (Test-Path "$TmpDir\config.example.yaml") {
        Copy-Item "$TmpDir\config.example.yaml" "$ConfigDir\config.yaml"
    }
    if (Test-Path "$TmpDir\themes") {
        Copy-Item -Recurse "$TmpDir\themes" "$ConfigDir\themes"
    }
}

# Clean up
Remove-Item -Recurse -Force $TmpDir

Write-Host "Installed $Binary $Version -> $Dest"

# PATH reminder
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
