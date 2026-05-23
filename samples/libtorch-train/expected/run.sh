#!/usr/bin/env bash
# Build + run.
set -euo pipefail
: "${LIBTORCH:?LIBTORCH must point at the libtorch install root}"
cmake -S . -B build -DCMAKE_PREFIX_PATH="$LIBTORCH"
cmake --build build -j
./build/m_n_i_s_t_classifier
