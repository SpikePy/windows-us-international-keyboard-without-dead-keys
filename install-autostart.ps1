#!/usr/bin/env pwsh
# Copies altgrhook.exe into the current user's Startup folder so it launches
# automatically at login. No admin rights needed - this only touches the
# per-user Startup folder, not any system-wide location.

$ErrorActionPreference = "Stop"

$source = Join-Path $PSScriptRoot "altgrhook.exe"
if (-not (Test-Path $source)) {
    Write-Error "altgrhook.exe not found next to this script ($source). Build or download it first."
    exit 1
}

$startupDir = [Environment]::GetFolderPath("Startup")
$destination = Join-Path $startupDir "altgrhook.exe"

if ((Resolve-Path $source).Path -eq $destination) {
    Write-Host "altgrhook.exe is already running from the Startup folder - nothing to do."
    exit 0
}

# Always copy to the same fixed destination filename and overwrite in
# place, so re-running this script (e.g. after an update) replaces the
# existing copy instead of leaving stale/duplicate copies behind.
Copy-Item -Path $source -Destination $destination -Force

Write-Host "Installed: $destination"
Write-Host "altgrhook will now start automatically the next time you log in."
Write-Host "Starting it now..."
Start-Process -FilePath $destination
