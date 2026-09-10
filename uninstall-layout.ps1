#Requires -RunAsAdministrator
# Removes the US-AltGr-International keyboard layout installed by install-layout.ps1.
# Run this from an elevated PowerShell prompt (Run as Administrator).

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$dllName = (Get-Content (Join-Path $scriptDir "dllname.txt")).Trim()
$layoutsKey = "HKLM:\SYSTEM\CurrentControlSet\Control\Keyboard Layouts"

$found = Get-ChildItem $layoutsKey | Where-Object {
    (Get-ItemProperty $_.PSPath -Name "Layout File" -ErrorAction SilentlyContinue)."Layout File" -eq $dllName
}

if (-not $found) {
    Write-Host "No registered layout found for $dllName."
} else {
    foreach ($key in $found) {
        Write-Host "Removing registry key $($key.PSChildName)"
        Remove-Item $key.PSPath -Force
    }
}

Remove-Item (Join-Path $env:WINDIR "System32\$dllName") -Force -ErrorAction SilentlyContinue
Remove-Item (Join-Path $env:WINDIR "SysWOW64\$dllName") -Force -ErrorAction SilentlyContinue

Write-Host "Uninstalled. Sign out/in or reboot for Windows to fully forget the layout."
