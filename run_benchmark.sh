#!/bin/bash
# In-process сравнение движков Go и Rust, без сети. Порядок как у автора:
# сначала Go, потом Rust. Протокол — ровно авторский с обеих сторон:
# его golang/main.go как есть (прогрев 10k, 5 итераций, среднее) и
# rust/src/bin/author_protocol.rs с тем же протоколом. Движок Go не изменён.
#
# WARM=1 — сначала прогнать оба бинарника вхолостую, без вывода. Первый запуск
# после простоя или сборки на Mac идёт вдвое медленнее (5.3 мс вместо 2.6 у Go),
# а 5 итераций автора целиком в него попадают; уже второй запуск подряд —
# нормальный. ROUNDS=N — повторить замер N раз подряд, чтобы видеть разброс.
set -e
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
export PATH="/opt/homebrew/opt/go/bin:$PATH"
source "$HOME/.cargo/env" 2>/dev/null || true
ROUNDS="${ROUNDS:-1}"

echo "=========================================================="
echo "    HIGH-PERFORMANCE MATCHING ENGINE BENCHMARK            "
echo "                   Go vs. Rust Evaluation                 "
echo "=========================================================="
echo "Host:         $(uname -m) $(uname -s) $(sw_vers -productVersion 2>/dev/null) | $(sysctl -n machdep.cpu.brand_string 2>/dev/null || echo unknown)"
echo "Cores:        $(sysctl -n hw.ncpu) logical = $(sysctl -n hw.perflevel0.logicalcpu 2>/dev/null || echo '?') performance + $(sysctl -n hw.perflevel1.logicalcpu 2>/dev/null || echo '?') efficiency | RAM $(( $(sysctl -n hw.memsize) / 1073741824 )) GB"
echo "Load avg:     $(uptime | sed 's/.*load averages: //')   (замер достоверен при < ~4)"
echo "Go Version:   $(go version)"
echo "Rust Version: $(rustc --version)"
echo "Protocol:     автора — прогрев 10k, 5 итераций по 100 000 ордеров, среднее; in-process, один поток"
[ "${WARM:-0}" = "1" ] && echo "Warm-up:      холостой прогон обоих бинарников перед замером (WARM=1)"
echo "=========================================================="
echo ""

# Собираем ВНЕ golang/, чтобы не перезаписать бинарник, который автор закоммитил.
mkdir -p "$DIR/.build"
echo ">>> Building Go engine (author's code, unmodified)..."
( cd "$DIR/golang" && go build -ldflags="-s -w" -o "$DIR/.build/engine_go" . )
echo ">>> Building Rust engine (release, target-cpu=native)..."
( cd "$DIR/rust" && RUSTFLAGS="-C target-cpu=native" cargo build --release -q )

if [ "${WARM:-0}" = "1" ]; then
  echo ">>> Warm-up: холостой прогон Go и Rust..."
  "$DIR/.build/engine_go" >/dev/null; "$DIR/rust/target/release/author_protocol" >/dev/null
fi

for ((r=1; r<=ROUNDS; r++)); do
  [ "$ROUNDS" -gt 1 ] && echo "################ раунд $r/$ROUNDS ################"
  echo "----------------------------------------------------------"
  echo "1. GO MATCHING ENGINE"
  echo "----------------------------------------------------------"
  "$DIR/.build/engine_go"
  echo "----------------------------------------------------------"
  echo "2. RUST MATCHING ENGINE"
  echo "----------------------------------------------------------"
  "$DIR/rust/target/release/author_protocol"
done

echo "=========================================================="
echo "                   BENCHMARK COMPLETE                     "
echo "=========================================================="
