. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

Invoke-Python -Arguments @("-m", "pip", "install", "--upgrade", "--user", "modelscope")
