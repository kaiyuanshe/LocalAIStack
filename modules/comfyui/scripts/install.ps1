. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

$comfyuiHome = Get-ComfyUIHome
$comfyuiRepo = if ($env:COMFYUI_REPO) { $env:COMFYUI_REPO } else { "https://github.com/comfyanonymous/ComfyUI.git" }
$comfyuiRef = if ($env:COMFYUI_REF) { $env:COMFYUI_REF } else { "master" }

Ensure-Directory -Path (Split-Path -Parent $comfyuiHome)

if (-not (Test-Path -LiteralPath (Join-Path $comfyuiHome ".git"))) {
  git clone $comfyuiRepo $comfyuiHome
}

Push-Location $comfyuiHome
try {
  git fetch --tags --prune origin
  git checkout $comfyuiRef
  git pull --ff-only origin $comfyuiRef

  $venvDir = Join-Path $comfyuiHome ".venv"
  if (-not (Test-Path -LiteralPath $venvDir)) {
    Invoke-Python -Arguments @("-m", "venv", ".venv")
  }

  $venvPython = Join-Path $venvDir "Scripts\python.exe"
  & $venvPython -m pip install --upgrade pip
  & $venvPython -m pip install -r requirements.txt
} finally {
  Pop-Location
}

$localBin = Get-LocalBinDir
Ensure-Directory -Path $localBin
$launcherPath = Join-Path $localBin "comfyui-las.cmd"
$launcher = @"
@echo off
set "COMFYUI_HOME=%COMFYUI_HOME%"
if not defined COMFYUI_HOME set "COMFYUI_HOME=$comfyuiHome"
call "%COMFYUI_HOME%\.venv\Scripts\activate.bat"
python "%COMFYUI_HOME%\main.py" %*
"@
Set-Content -LiteralPath $launcherPath -Value $launcher -Encoding ASCII

Write-Output "ComfyUI installed at: $comfyuiHome"
Write-Output "Start command: comfyui-las.cmd --listen 127.0.0.1 --port 8188"
