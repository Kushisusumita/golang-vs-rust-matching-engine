#!/bin/bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
source "$HOME/.cargo/env" 2>/dev/null || true

echo "=========================================================="
echo "    HIGH-PERFORMANCE MATCHING ENGINE BENCHMARK (100k)     "
echo "                   Go vs. Rust Evaluation                 "
echo "=========================================================="
echo "Host Machine: $(uname -m) - $(uname -s)"
echo "Go Version:   $(go version)"
echo "Rust Version: $(rustc --version)"
echo "=========================================================="
echo ""

echo ">>> Building Go Matching Engine (optimized)..."
cd "$DIR/golang"
go build -ldflags="-s -w" -o engine_go .

echo ">>> Building Rust Matching Engine (release mode)..."
cd "$DIR/rust"
RUSTFLAGS="-C target-cpu=native" cargo build --release -q

echo ""
echo "----------------------------------------------------------"
echo "1. RUNNING GO MATCHING ENGINE"
echo "----------------------------------------------------------"
cd "$DIR/golang"
./engine_go

echo "----------------------------------------------------------"
echo "2. RUNNING RUST MATCHING ENGINE"
echo "----------------------------------------------------------"
cd "$DIR/rust"
./target/release/matching-engine-rust

echo "=========================================================="
echo "                   BENCHMARK COMPLETE                     "
echo "=========================================================="
