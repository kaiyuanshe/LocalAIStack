. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

function Find-OllamaUninstallCommand {
  $roots = @(
    "HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*",
    "HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*",
    "HKLM:\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*"
  )

  foreach ($root in $roots) {
    $entries = Get-ItemProperty -Path $root -ErrorAction SilentlyContinue
    foreach ($entry in $entries) {
      $name = [string]$entry.DisplayName
      if ($name -like "Ollama*") {
        if ($entry.QuietUninstallString) {
          return [string]$entry.QuietUninstallString
        }
        if ($entry.UninstallString) {
          return [string]$entry.UninstallString
        }
      }
    }
  }
  return $null
}

$commandLine = Find-OllamaUninstallCommand
if ($commandLine) {
  $tempCmd = Join-Path ([System.IO.Path]::GetTempPath()) ("ollama-uninstall-" + [guid]::NewGuid().ToString("N") + ".cmd")
  try {
    Set-Content -LiteralPath $tempCmd -Value "@echo off`r`n$commandLine`r`n" -Encoding ASCII
    & cmd.exe /c $tempCmd
  } finally {
    if (Test-Path -LiteralPath $tempCmd) {
      Remove-Item -LiteralPath $tempCmd -Force
    }
  }
}

$candidates = @(
  (Join-Path $env:LOCALAPPDATA "Programs\Ollama"),
  (Join-Path $env:LOCALAPPDATA "Ollama")
)
foreach ($candidate in $candidates) {
  if ($candidate -and (Test-Path -LiteralPath $candidate)) {
    Remove-Item -LiteralPath $candidate -Recurse -Force -ErrorAction SilentlyContinue
  }
}

Write-Output "Ollama uninstalled (data preserved at ~/.ollama)."
