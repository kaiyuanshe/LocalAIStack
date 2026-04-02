. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

$comfyuiHome = Get-ComfyUIHome
Remove-IfExists -Path (Join-Path (Get-LocalBinDir) "comfyui-las.cmd")
Remove-IfExists -Path $comfyuiHome
Write-Output "ComfyUI purged from: $comfyuiHome"
