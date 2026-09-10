#Requires -RunAsAdministrator
# Installs the compiled US-AltGr-International keyboard layout on this machine.
# Run this from an elevated PowerShell prompt (Run as Administrator), from the
# folder produced by the GitHub Action (it must contain the .dll files and dllname.txt).

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$dllName = (Get-Content (Join-Path $scriptDir "dllname.txt")).Trim()
$layoutText = "US International - AltGr - No Dead Keys"
$baseLanguage = 0x0409  # en-US, from LOCALEID in the .klc

$amd64Dll = Join-Path $scriptDir $dllName
$x86Dll = Join-Path $scriptDir "x86_$dllName"

if (-not (Test-Path $amd64Dll)) {
    throw "Could not find $dllName next to this script."
}

Copy-Item $amd64Dll (Join-Path $env:WINDIR "System32\$dllName") -Force
Write-Host "Copied $dllName to System32."

if (Test-Path $x86Dll) {
    Copy-Item $x86Dll (Join-Path $env:WINDIR "SysWOW64\$dllName") -Force
    Write-Host "Copied $dllName to SysWOW64 (for 32-bit app compatibility)."
}

$layoutsKey = "HKLM:\SYSTEM\CurrentControlSet\Control\Keyboard Layouts"

# Find a free custom layout key in the private range 0xA000xxxx-0xAFFFxxxx
# reserved for non-Microsoft layouts (same scheme MSKLC/community tools use).
$langId = $null
for ($upper = 0xA000; $upper -le 0xAFFF; $upper++) {
    $candidate = "{0:x4}{1:x4}" -f $upper, $baseLanguage
    if (-not (Test-Path (Join-Path $layoutsKey $candidate))) {
        $langId = $candidate
        break
    }
}
if (-not $langId) { throw "No free keyboard layout ID slot found in the custom range." }

# Layout Id is a separate 4-hex-digit value that must be unique across all
# installed layouts; pick the next unused one.
$maxId = 0
Get-ChildItem $layoutsKey | ForEach-Object {
    $v = (Get-ItemProperty $_.PSPath -Name "Layout Id" -ErrorAction SilentlyContinue)."Layout Id"
    if ($v) {
        $n = [Convert]::ToInt32($v, 16)
        if ($n -gt $maxId) { $maxId = $n }
    }
}
$layoutId = "{0:x4}" -f ($maxId + 1)

$keyPath = Join-Path $layoutsKey $langId
New-Item -Path $keyPath -Force | Out-Null
Set-ItemProperty -Path $keyPath -Name "Layout File" -Value $dllName
Set-ItemProperty -Path $keyPath -Name "Layout Text" -Value $layoutText
Set-ItemProperty -Path $keyPath -Name "Layout Id" -Value $layoutId

Write-Host ""
Write-Host "Installed '$layoutText' as layout $langId (Layout Id $layoutId)."
Write-Host ""
Write-Host "Next steps:"
Write-Host "  1. Sign out and back in (or reboot) so Windows picks up the new layout."
Write-Host "  2. Settings > Time & Language > Language & region > English (United States)"
Write-Host "     > Language options > Keyboards > Add a keyboard > '$layoutText'."
