. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

Remove-IfExists -Path (Join-Path (Get-LocalBinDir) "comfyui-las.cmd")
Write-Output "ComfyUI launcher removed"
