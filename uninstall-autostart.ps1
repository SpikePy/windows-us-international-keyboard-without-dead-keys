#!/usr/bin/env pwsh
# Stops the background helper if running and removes it from the current
# user's Startup folder. No admin rights needed - this only touches the
# per-user Startup folder.

$ErrorActionPreference = "Stop"

$exeName = "us-international-without-dead-keys.exe"
$startupDir = [Environment]::GetFolderPath("Startup")
$destination = Join-Path $startupDir $exeName

Get-Process -Name ([System.IO.Path]::GetFileNameWithoutExtension($exeName)) -ErrorAction SilentlyContinue | Stop-Process -Force

if (Test-Path $destination) {
    Remove-Item -Path $destination -Force
    Write-Host "Removed $destination"
} else {
    Write-Host "Not installed in the Startup folder ($destination) - nothing to remove."
}
