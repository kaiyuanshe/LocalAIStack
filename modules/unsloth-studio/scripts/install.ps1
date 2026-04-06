. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

$installerUrl = if ($env:UNSLOTH_STUDIO_INSTALL_URL) { $env:UNSLOTH_STUDIO_INSTALL_URL } else { "https://unsloth.ai/install.ps1" }
$tmpScript = Join-Path ([System.IO.Path]::GetTempPath()) ("unsloth-studio-install-" + [System.Guid]::NewGuid().ToString("N") + ".ps1")
$arguments = @()
$pythonVersion = if ($env:UNSLOTH_STUDIO_PYTHON_VERSION) { $env:UNSLOTH_STUDIO_PYTHON_VERSION } else { "3.12" }

Invoke-PythonCode -Code @"
import importlib.util
import sys

if importlib.util.find_spec("_bz2") is None:
    raise SystemExit(
        "Python is missing the standard-library _bz2 extension. "
        "Unsloth Studio depends on bz2 via the datasets package. "
        "Reinstall Python with bzip2 support, then retry."
    )
"@

if ($env:UNSLOTH_STUDIO_NO_TORCH -eq "1") {
  $arguments += "--no-torch"
}
if ($env:UNSLOTH_STUDIO_VERBOSE -eq "1") {
  $arguments += "--verbose"
}
if ($pythonVersion) {
  $arguments += @("--python", $pythonVersion)
}

try {
  Invoke-WebRequest -Uri $installerUrl -OutFile $tmpScript
  & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $tmpScript @arguments
  if ($LASTEXITCODE -ne 0) {
    throw "Unsloth Studio installer failed with exit code $LASTEXITCODE."
  }
} finally {
  Remove-IfExists -Path $tmpScript
}
