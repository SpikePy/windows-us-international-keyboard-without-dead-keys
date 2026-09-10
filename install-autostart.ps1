#!/usr/bin/env pwsh
# Downloads the latest release from GitHub and installs it into the
# current user's Startup folder so it launches automatically at login. No
# admin rights needed - this only touches the per-user Startup folder,
# not any system-wide location.

$ErrorActionPreference = "Stop"
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$exeName = "us-international-without-dead-keys.exe"
$repo = "SpikePy/windows-us-international-keyboard-without-dead-keys"
$startupDir = [Environment]::GetFolderPath("Startup")
$destination = Join-Path $startupDir $exeName

Write-Host "Checking latest release of $repo..."
$release = Invoke-RestMethod -UseBasicParsing -Uri "https://api.github.com/repos/$repo/releases/latest"
$asset = $release.assets | Where-Object { $_.name -eq $exeName } | Select-Object -First 1
if (-not $asset) {
    Write-Error "No $exeName asset found in release $($release.tag_name)."
    exit 1
}

# Download to a temp file first so a failed/interrupted download never
# corrupts an already-installed copy.
$tempFile = Join-Path ([System.IO.Path]::GetTempPath()) "$([System.IO.Path]::GetFileNameWithoutExtension($exeName))-$($release.tag_name).exe"
Write-Host "Downloading $($release.tag_name) ($($asset.browser_download_url))..."
Invoke-WebRequest -UseBasicParsing -Uri $asset.browser_download_url -OutFile $tempFile

# Stop any already-running copy so the file below isn't locked and so we
# don't end up with two instances racing over the same hotkeys. Wait for
# it to fully exit - Stop-Process can return before Windows has finished
# tearing down the killed process's kernel objects (including the named
# mutex the exe uses as a single-instance guard), and starting the new
# copy too soon after can make it see that mutex as still held and exit
# immediately.
$running = Get-Process -Name ([System.IO.Path]::GetFileNameWithoutExtension($exeName)) -ErrorAction SilentlyContinue
if ($running) {
    $running | Stop-Process -Force
    $running | Wait-Process -ErrorAction SilentlyContinue
}

# Always move to the same fixed destination filename, replacing whatever
# is already there, so re-running this script never leaves duplicate
# copies behind in the Startup folder.
Move-Item -Path $tempFile -Destination $destination -Force

Write-Host "Installed $($release.tag_name): $destination"
Write-Host "It will now start automatically the next time you log in."
Write-Host "Starting it now..."
Start-Process -FilePath $destination
