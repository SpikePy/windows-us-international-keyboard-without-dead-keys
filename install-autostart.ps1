#!/usr/bin/env pwsh
# Downloads the latest altgrhook.exe release from GitHub and installs it
# into the current user's Startup folder so it launches automatically at
# login. No admin rights needed - this only touches the per-user Startup
# folder, not any system-wide location.

$ErrorActionPreference = "Stop"
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$repo = "SpikePy/altgrhook"
$startupDir = [Environment]::GetFolderPath("Startup")
$destination = Join-Path $startupDir "altgrhook.exe"

Write-Host "Checking latest release of $repo..."
$release = Invoke-RestMethod -UseBasicParsing -Uri "https://api.github.com/repos/$repo/releases/latest"
$asset = $release.assets | Where-Object { $_.name -eq "altgrhook.exe" } | Select-Object -First 1
if (-not $asset) {
    Write-Error "No altgrhook.exe asset found in release $($release.tag_name)."
    exit 1
}

# Download to a temp file first so a failed/interrupted download never
# corrupts an already-installed copy.
$tempFile = Join-Path ([System.IO.Path]::GetTempPath()) "altgrhook-$($release.tag_name).exe"
Write-Host "Downloading $($release.tag_name) ($($asset.browser_download_url))..."
Invoke-WebRequest -UseBasicParsing -Uri $asset.browser_download_url -OutFile $tempFile

# Stop any already-running copy so the file below isn't locked and so we
# don't end up with two instances racing over the same hotkeys.
Get-Process -Name "altgrhook" -ErrorAction SilentlyContinue | Stop-Process -Force

# Always move to the same fixed destination filename, replacing whatever
# is already there, so re-running this script never leaves duplicate
# copies behind in the Startup folder.
Move-Item -Path $tempFile -Destination $destination -Force

Write-Host "Installed $($release.tag_name): $destination"
Write-Host "altgrhook will now start automatically the next time you log in."
Write-Host "Starting it now..."
Start-Process -FilePath $destination
