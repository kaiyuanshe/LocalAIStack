#!/usr/bin/env bash
set -euo pipefail

repo_url="https://github.com/ggerganov/llama.cpp.git"
source_dir="/usr/local/llama.cpp"

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

if [[ -d "$source_dir/.git" ]]; then
  $SUDO git -C "$source_dir" fetch --tags --prune
  $SUDO git -C "$source_dir" reset --hard origin/master
else
  $SUDO rm -rf "$source_dir"
  $SUDO git clone --depth 1 "$repo_url" "$source_dir"
fi

if [[ "${LLAMA_CUDA:-}" == "1" || "${LLAMA_CUDA:-}" == "ON" ]]; then
  $SUDO sed -i 's/list(APPEND CUDA_FLAGS -compress-mode=${GGML_CUDA_COMPRESSION_MODE})/# disabled by LocalAIStack: compress-mode not supported on this toolchain/' \
    "$source_dir/ggml/src/ggml-cuda/CMakeLists.txt"
  if [[ ! -x /usr/bin/gcc-10 ]]; then
    echo "gcc-10 is required for CUDA 11.x builds. Install it with: sudo apt install -y gcc-10 g++-10" >&2
    exit 1
  fi
fi

cmake_flags=("-DLLAMA_BUILD_SERVER=ON" "-DCMAKE_BUILD_TYPE=Release")
cmake_flags+=("-DCMAKE_CXX_STANDARD=17" "-DCMAKE_CUDA_STANDARD=17")
if [[ -x /usr/bin/gcc-10 && -x /usr/bin/g++-10 ]]; then
  cmake_flags+=("-DCMAKE_C_COMPILER=/usr/bin/gcc-10" "-DCMAKE_CXX_COMPILER=/usr/bin/g++-10")
fi
if [[ "${LLAMA_CUDA:-}" == "1" || "${LLAMA_CUDA:-}" == "ON" ]]; then
  cmake_flags+=("-DGGML_CUDA=ON")
  cmake_flags+=("-DCMAKE_CUDA_COMPILER=/usr/bin/nvcc")
  if [[ -x /usr/bin/gcc-10 ]]; then
    cmake_flags+=("-DCMAKE_CUDA_HOST_COMPILER=/usr/bin/gcc-10")
  else
    cmake_flags+=("-DCMAKE_CUDA_HOST_COMPILER=/usr/bin/gcc-11")
  fi
  cmake_flags+=("-DCUDAToolkit_ROOT=/usr")
  cmake_flags+=("-DCUDA_TOOLKIT_ROOT_DIR=/usr")
  cmake_flags+=("-DCUDAToolkit_VERSION=11.5")
  cmake_flags+=("-DGGML_CUDA_COMPRESSION_MODE=none")
  if [[ -n "${LLAMA_CUDA_ARCHS:-}" ]]; then
    cmake_flags+=("-DCMAKE_CUDA_ARCHITECTURES=${LLAMA_CUDA_ARCHS}")
  fi
fi

$SUDO rm -rf "$source_dir/build"
$SUDO cmake -S "$source_dir" -B "$source_dir/build" "${cmake_flags[@]}"
$SUDO cmake --build "$source_dir/build" --config Release

for bin in llama-cli llama-server; do
  if [[ -x "$source_dir/build/bin/$bin" ]]; then
    $SUDO install -m 0755 "$source_dir/build/bin/$bin" "/usr/local/bin/$bin"
  fi
done

if [[ ! -x /usr/local/bin/llama-cli ]]; then
  echo "llama-cli was not built successfully." >&2
  exit 1
fi
