#!/usr/bin/env bash
set -euo pipefail

repo_url="https://github.com/ggml-org/llama.cpp.git"
source_dir="/usr/local/llama.cpp"

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

is_truthy() {
  local value
  value="$(echo "${1:-}" | tr '[:upper:]' '[:lower:]')"
  [[ "$value" == "1" || "$value" == "on" || "$value" == "true" || "$value" == "yes" ]]
}

is_falsy() {
  local value
  value="$(echo "${1:-}" | tr '[:upper:]' '[:lower:]')"
  [[ "$value" == "0" || "$value" == "off" || "$value" == "false" || "$value" == "no" ]]
}

has_nvidia_gpu() {
  command -v nvidia-smi >/dev/null 2>&1 && nvidia-smi -L >/dev/null 2>&1
}

pick_gnu_compiler_pair() {
  local version
  for version in 10 11 12 13; do
    if [[ -x "/usr/bin/gcc-${version}" && -x "/usr/bin/g++-${version}" ]]; then
      echo "/usr/bin/gcc-${version};/usr/bin/g++-${version}"
      return 0
    fi
  done
  if command -v gcc >/dev/null 2>&1 && command -v g++ >/dev/null 2>&1; then
    echo "$(command -v gcc);$(command -v g++)"
    return 0
  fi
  return 1
}

unique_semicolon_list() {
  local input="$1"
  local out="" seen=";" item
  IFS=';' read -r -a items <<< "$input"
  for item in "${items[@]}"; do
    [[ -z "$item" ]] && continue
    if [[ "$seen" != *";$item;"* ]]; then
      out="${out:+$out;}$item"
      seen+="$item;"
    fi
  done
  echo "$out"
}

detect_cuda_archs() {
  local names archs=""
  names="$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | tr '[:upper:]' '[:lower:]' || true)"
  while IFS= read -r name; do
    [[ -z "$name" ]] && continue
    case "$name" in
      *v100*) archs="${archs:+$archs;}70" ;;
      *a100*) archs="${archs:+$archs;}80" ;;
      *h100*|*h200*) archs="${archs:+$archs;}90" ;;
      *a10*|*l4*|*a40*|*rtx\ 30*|*3090*|*3080*|*3070*) archs="${archs:+$archs;}86" ;;
      *rtx\ 40*|*4090*|*4080*|*4070*|*l40*|*l40s*) archs="${archs:+$archs;}89" ;;
    esac
  done <<< "$names"
  unique_semicolon_list "$archs"
}

cuda_requested=0
if [[ -n "${LLAMA_CUDA:-}" ]]; then
  if is_truthy "${LLAMA_CUDA}"; then
    cuda_requested=1
  elif is_falsy "${LLAMA_CUDA}"; then
    cuda_requested=0
  fi
elif has_nvidia_gpu; then
  cuda_requested=1
  echo "Detected NVIDIA GPU; enabling CUDA build automatically."
fi

if [[ "$cuda_requested" -eq 1 && -z "${LLAMA_CUDA_ARCHS:-}" ]]; then
  detected_archs="$(detect_cuda_archs)"
  if [[ -n "$detected_archs" ]]; then
    export LLAMA_CUDA_ARCHS="$detected_archs"
    echo "Auto-detected CUDA architectures: ${LLAMA_CUDA_ARCHS}"
  fi
fi

if [[ "$cuda_requested" -eq 1 ]]; then
  export LLAMA_CUDA=1
else
  export LLAMA_CUDA=0
fi

build_parallel="${LLAMA_BUILD_PARALLEL:-}"
if [[ -z "$build_parallel" ]]; then
  if [[ "$cuda_requested" -eq 1 ]]; then
    # CUDA template instantiation is memory-heavy. Default to serial build for stability.
    build_parallel=1
  else
    build_parallel="$(nproc)"
  fi
fi
if ! [[ "$build_parallel" =~ ^[0-9]+$ ]] || [[ "$build_parallel" -lt 1 ]]; then
  echo "Invalid LLAMA_BUILD_PARALLEL=${build_parallel}; expected positive integer." >&2
  exit 1
fi

if [[ -d "$source_dir/.git" ]]; then
  $SUDO git -C "$source_dir" fetch --tags --prune
  $SUDO git -C "$source_dir" reset --hard origin/master
else
  $SUDO rm -rf "$source_dir"
  $SUDO git clone --depth 1 "$repo_url" "$source_dir"
fi

if [[ "${LLAMA_CUDA:-}" == "1" || "${LLAMA_CUDA:-}" == "ON" ]]; then
  if ! command -v nvcc >/dev/null 2>&1; then
    echo "CUDA build requested but nvcc is not available. Install CUDA toolkit or run with LLAMA_CUDA=0." >&2
    exit 1
  fi
  $SUDO sed -i 's/list(APPEND CUDA_FLAGS -compress-mode=${GGML_CUDA_COMPRESSION_MODE})/# disabled by LocalAIStack: compress-mode not supported on this toolchain/' \
    "$source_dir/ggml/src/ggml-cuda/CMakeLists.txt"
  compiler_pair="$(pick_gnu_compiler_pair || true)"
  if [[ -z "$compiler_pair" ]]; then
    echo "CUDA build requested but no GCC/G++ compiler pair was found in PATH." >&2
    exit 1
  fi
fi

cmake_flags=("-DLLAMA_BUILD_SERVER=ON" "-DCMAKE_BUILD_TYPE=Release" "-DCMAKE_CXX_STANDARD=17")
compiler_pair="$(pick_gnu_compiler_pair || true)"
if [[ -n "$compiler_pair" ]]; then
  c_compiler="${compiler_pair%%;*}"
  cxx_compiler="${compiler_pair##*;}"
  cmake_flags+=("-DCMAKE_C_COMPILER=${c_compiler}" "-DCMAKE_CXX_COMPILER=${cxx_compiler}")
fi
if [[ "${LLAMA_CUDA:-}" == "1" || "${LLAMA_CUDA:-}" == "ON" ]]; then
  cmake_flags+=("-DCMAKE_CUDA_STANDARD=17")
  cmake_flags+=("-DGGML_CUDA=ON")
  cmake_flags+=("-DCMAKE_CUDA_COMPILER=/usr/bin/nvcc")
  if [[ -n "${cxx_compiler:-}" ]]; then
    cmake_flags+=("-DCMAKE_CUDA_HOST_COMPILER=${cxx_compiler}")
  fi
  cmake_flags+=("-DCUDAToolkit_ROOT=/usr")
  cmake_flags+=("-DCUDA_TOOLKIT_ROOT_DIR=/usr")
  cmake_flags+=("-DGGML_CUDA_COMPRESSION_MODE=none")
  if [[ -n "${LLAMA_CUDA_ARCHS:-}" ]]; then
    cmake_flags+=("-DCMAKE_CUDA_ARCHITECTURES=${LLAMA_CUDA_ARCHS}")
  fi

  if [[ -n "${LLAMA_CUDA_FA:-}" ]]; then
    if is_truthy "${LLAMA_CUDA_FA}"; then
      cmake_flags+=("-DGGML_CUDA_FA=ON")
    elif is_falsy "${LLAMA_CUDA_FA}"; then
      cmake_flags+=("-DGGML_CUDA_FA=OFF")
    fi
  elif [[ "${LLAMA_CUDA_ARCHS:-}" == "70" ]]; then
    cmake_flags+=("-DGGML_CUDA_FA=OFF")
    echo "Detected V100 (sm_70); disabling GGML_CUDA_FA by default to reduce compile load."
  fi

  echo "Building llama.cpp with CUDA support (parallel=${build_parallel})."
else
  echo "Building llama.cpp in CPU-only mode (parallel=${build_parallel})."
fi

$SUDO rm -rf "$source_dir/build"
$SUDO cmake -S "$source_dir" -B "$source_dir/build" "${cmake_flags[@]}"
$SUDO cmake --build "$source_dir/build" --config Release --parallel "$build_parallel"

for bin in llama-cli llama-server; do
  if [[ -x "$source_dir/build/bin/$bin" ]]; then
    $SUDO install -m 0755 "$source_dir/build/bin/$bin" "/usr/local/bin/$bin"
  fi
done

if [[ ! -x /usr/local/bin/llama-cli ]]; then
  echo "llama-cli was not built successfully." >&2
  exit 1
fi
