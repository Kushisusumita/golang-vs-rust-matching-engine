mod types;
mod generator;
mod orderbook;

use std::time::{Duration, Instant};
use generator::generate_orders;
use orderbook::OrderBook;
use types::Order;

struct BenchResult {
    elapsed: Duration,
    throughput: f64,
    avg_latency_ns: f64,
}

fn run_single_benchmark(n: usize, orders: &[Order]) -> (BenchResult, u64, u64, usize, usize, usize, u64, u64) {
    let mut ob = OrderBook::with_capacity(n);

    let start = Instant::now();
    for &order in orders {
        ob.process_order(order);
    }
    let elapsed = start.elapsed();

    let total_orders = n as f64;
    let seconds = elapsed.as_secs_f64();
    let ops = total_orders / seconds;
    let avg_latency_ns = (elapsed.as_nanos() as f64) / total_orders;

    let best_bid = ob.bids.first().map(|l| l.price).unwrap_or(0);
    let best_ask = ob.asks.first().map(|l| l.price).unwrap_or(0);

    let resting_bids: usize = ob.bids.iter().map(|l| l.order_count as usize).sum();
    let resting_asks: usize = ob.asks.iter().map(|l| l.order_count as usize).sum();
    let total_levels = ob.bids.len() + ob.asks.len();

    (
        BenchResult {
            elapsed,
            throughput: ops,
            avg_latency_ns,
        },
        ob.trades_count,
        ob.matched_volume,
        resting_bids,
        resting_asks,
        total_levels,
        best_bid,
        best_ask,
    )
}

fn main() {
    const SEED: u64 = 0xDEADBEEFCAFE1234;
    const N: usize = 100_000;
    const ITERATIONS: usize = 5;

    println!("========================================================");
    println!("            RUST MATCHING ENGINE BENCHMARK              ");
    println!("========================================================");
    println!("Workload: {} orders per run | {} iterations", N, ITERATIONS);
    println!("PRNG: Deterministic Xorshift64 (seed: 0x{:X})", SEED);
    println!("--------------------------------------------------------");

    // Warmup run
    let warmup_orders = generate_orders(10_000, SEED);
    run_single_benchmark(10_000, &warmup_orders);

    let orders = generate_orders(N, SEED);

    let mut total_elapsed = Duration::ZERO;
    let mut min_elapsed = Duration::MAX;
    let mut max_elapsed = Duration::ZERO;

    let mut trades = 0;
    let mut volume = 0;
    let mut resting_b = 0;
    let mut resting_a = 0;
    let mut total_levels = 0;
    let mut best_bid = 0;
    let mut best_ask = 0;

    for i in 1..=ITERATIONS {
        let (res, t, v, rb, ra, lvls, bb, ba) = run_single_benchmark(N, &orders);
        total_elapsed += res.elapsed;
        if res.elapsed < min_elapsed {
            min_elapsed = res.elapsed;
        }
        if res.elapsed > max_elapsed {
            max_elapsed = res.elapsed;
        }
        trades = t;
        volume = v;
        resting_b = rb;
        resting_a = ra;
        total_levels = lvls;
        best_bid = bb;
        best_ask = ba;

        println!(
            "  Iteration {}: {:8.3} ms | {:10.2} M ops/s | {:6.2} ns/order",
            i,
            res.elapsed.as_secs_f64() * 1000.0,
            res.throughput / 1_000_000.0,
            res.avg_latency_ns
        );
    }

    let avg_elapsed = total_elapsed / (ITERATIONS as u32);
    let avg_ops = (N as f64) / avg_elapsed.as_secs_f64();
    let avg_latency = (avg_elapsed.as_nanos() as f64) / (N as f64);
    let best_ops = (N as f64) / min_elapsed.as_secs_f64();

    println!("--------------------------------------------------------");
    println!("SUMMARY (RUST):");
    println!("  Best Time:         {:.3} ms ({:.2} M ops/s)", min_elapsed.as_secs_f64() * 1000.0, best_ops / 1_000_000.0);
    println!("  Average Time:      {:.3} ms ({:.2} M ops/s)", avg_elapsed.as_secs_f64() * 1000.0, avg_ops / 1_000_000.0);
    println!("  Average Latency:   {:.2} ns/order", avg_latency);
    println!("  Trades Executed:   {}", trades);
    println!("  Volume Matched:    {}", volume);
    println!("  Resting in Book:   {} (Bids: {}, Asks: {})", resting_b + resting_a, resting_b, resting_a);
    println!("  Active Levels:     {} (Best Bid: {}, Best Ask: {})", total_levels, best_bid, best_ask);
    println!("========================================================\n");
}
