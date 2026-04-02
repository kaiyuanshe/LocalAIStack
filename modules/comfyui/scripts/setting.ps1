. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

param(
  [Parameter(Mandatory = $true, Position = 0)]
  [string]$ModelBundle
)

$comfyuiHome = Get-ComfyUIHome
$srcBase = Get-ModelsHome
$src = Join-Path $srcBase "$ModelBundle\split_files"

if (-not (Test-Path -LiteralPath $src)) {
  throw "Model split_files directory not found: $src"
}

$diffusionDir = Join-Path $comfyuiHome "models\diffusion_models"
$encoderDir = Join-Path $comfyuiHome "models\text_encoders"
$vaeDir = Join-Path $comfyuiHome "models\vae"
Ensure-Directory -Path $diffusionDir
Ensure-Directory -Path $encoderDir
Ensure-Directory -Path $vaeDir

switch ($ModelBundle) {
  "Comfy-Org_z_image_turbo" {
    New-FileLinkOrCopy -Target (Join-Path $src "diffusion_models\z_image_turbo_bf16.safetensors") -LinkPath (Join-Path $diffusionDir "z_image_turbo_bf16.safetensors")
    New-FileLinkOrCopy -Target (Join-Path $src "diffusion_models\z_image_turbo_nvfp4.safetensors") -LinkPath (Join-Path $diffusionDir "z_image_turbo_nvfp4.safetensors")
    New-FileLinkOrCopy -Target (Join-Path $src "text_encoders\qwen_3_4b.safetensors") -LinkPath (Join-Path $encoderDir "qwen_3_4b.safetensors")
    New-FileLinkOrCopy -Target (Join-Path $src "text_encoders\qwen_3_4b_fp4_mixed.safetensors") -LinkPath (Join-Path $encoderDir "qwen_3_4b_fp4_mixed.safetensors")
    New-FileLinkOrCopy -Target (Join-Path $src "text_encoders\qwen_3_4b_fp8_mixed.safetensors") -LinkPath (Join-Path $encoderDir "qwen_3_4b_fp8_mixed.safetensors")
    New-FileLinkOrCopy -Target (Join-Path $src "vae\ae.safetensors") -LinkPath (Join-Path $vaeDir "ae.safetensors")
  }
  default {
    throw "Unsupported model bundle: $ModelBundle. Currently supported: Comfy-Org_z_image_turbo"
  }
}

Write-Output "ComfyUI model links configured for: $ModelBundle"
Write-Output "ComfyUI home: $comfyuiHome"
