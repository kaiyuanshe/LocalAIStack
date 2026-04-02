. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

$comfyuiHome = Get-ComfyUIHome
if (-not (Test-Path -LiteralPath $comfyuiHome)) {
  throw "ComfyUI home not found: $comfyuiHome"
}

$venvPython = Join-Path $comfyuiHome ".venv\Scripts\python.exe"
if (-not (Test-Path -LiteralPath $venvPython)) {
  throw "ComfyUI virtualenv not found: $(Join-Path $comfyuiHome '.venv')"
}

& $venvPython -c "import importlib.util; raise SystemExit('torch is not installed in ComfyUI environment') if importlib.util.find_spec('torch') is None else print('ok')"

$launcher = Join-Path (Get-LocalBinDir) "comfyui-las.cmd"
if (-not (Test-Path -LiteralPath $launcher)) {
  throw "launcher not found: $launcher"
}

Write-Output "ComfyUI verify passed"
