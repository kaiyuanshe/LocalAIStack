. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

Invoke-Python -Arguments @("-m", "pip", "uninstall", "-y", "huggingface_hub")
