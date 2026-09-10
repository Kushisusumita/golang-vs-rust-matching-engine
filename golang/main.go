package main

import (
	"fmt"
	"runtime"
	"time"
)

type BenchResult struct {
	Elapsed      time.Duration
	Throughput   float64
	AvgLatencyNs float64
}

func runSingleBenchmark(n int, seed uint64, orders []OrderInput) (BenchResult, uint64, uint64, int, int, int, uint64, uint64) {
	ob := NewOrderBook(n)

	runtime.GC()

	start := time.Now()
	for i := 0; i < n; i++ {
		o := &orders[i]
		ob.ProcessOrder(o.ID, o.Price, o.Quantity, o.Side)
	}
	elapsed := time.Since(start)

	totalOrders := float64(n)
	seconds := elapsed.Seconds()
	ops := totalOrders / seconds
	avgLatencyNs := float64(elapsed.Nanoseconds()) / totalOrders

	var bestBid, bestAsk uint64
	if len(ob.Bids) > 0 {
		bestBid = ob.Bids[0].Price
	}
	if len(ob.Asks) > 0 {
		bestAsk = ob.Asks[0].Price
	}

	restingBids := 0
	for _, l := range ob.Bids {
		restingBids += int(l.OrderCount)
	}
	restingAsks := 0
	for _, l := range ob.Asks {
		restingAsks += int(l.OrderCount)
	}

	return BenchResult{
		Elapsed:      elapsed,
		Throughput:   ops,
		AvgLatencyNs: avgLatencyNs,
	}, ob.TradesCount, ob.MatchedVolume, restingBids, restingAsks, len(ob.Bids) + len(ob.Asks), bestBid, bestAsk
}

func main() {
	const seed uint64 = 0xDEADBEEFCAFE1234
	const n = 100000
	const iterations = 5

	fmt.Printf("========================================================\n")
	fmt.Printf("             GO MATCHING ENGINE BENCHMARK               \n")
	fmt.Printf("========================================================\n")
	fmt.Printf("Workload: %d orders per run | %d iterations\n", n, iterations)
	fmt.Printf("PRNG: Deterministic Xorshift64 (seed: 0x%X)\n", uint64(seed))
	fmt.Printf("--------------------------------------------------------\n")

	// Pre-generate orders
	warmupOrders := GenerateOrders(10000, seed)
	runSingleBenchmark(10000, seed, warmupOrders)

	orders := GenerateOrders(n, seed)

	var results []BenchResult
	var totalElapsed time.Duration
	var minElapsed = time.Duration(1<<63 - 1)
	var maxElapsed time.Duration

	var trades, volume uint64
	var restingB, restingA, totalLevels int
	var bestBid, bestAsk uint64

	for i := 1; i <= iterations; i++ {
		res, t, v, rb, ra, lvls, bb, ba := runSingleBenchmark(n, seed, orders)
		results = append(results, res)
		totalElapsed += res.Elapsed
		if res.Elapsed < minElapsed {
			minElapsed = res.Elapsed
		}
		if res.Elapsed > maxElapsed {
			maxElapsed = res.Elapsed
		}
		trades = t
		volume = v
		restingB = rb
		restingA = ra
		totalLevels = lvls
		bestBid = bb
		bestAsk = ba

		fmt.Printf("  Iteration %d: %8.3f ms | %10.2f M ops/s | %6.2f ns/order\n",
			i,
			float64(res.Elapsed.Microseconds())/1000.0,
			res.Throughput/1_000_000.0,
			res.AvgLatencyNs,
		)
	}

	avgElapsed := totalElapsed / iterations
	avgOps := float64(n) / avgElapsed.Seconds()
	avgLatency := float64(avgElapsed.Nanoseconds()) / float64(n)
	bestOps := float64(n) / minElapsed.Seconds()

	fmt.Printf("--------------------------------------------------------\n")
	fmt.Printf("SUMMARY (GO):\n")
	fmt.Printf("  Best Time:         %.3f ms (%.2f M ops/s)\n", float64(minElapsed.Microseconds())/1000.0, bestOps/1_000_000.0)
	fmt.Printf("  Average Time:      %.3f ms (%.2f M ops/s)\n", float64(avgElapsed.Microseconds())/1000.0, avgOps/1_000_000.0)
	fmt.Printf("  Average Latency:   %.2f ns/order\n", avgLatency)
	fmt.Printf("  Trades Executed:   %d\n", trades)
	fmt.Printf("  Volume Matched:    %d\n", volume)
	fmt.Printf("  Resting in Book:   %d (Bids: %d, Asks: %d)\n", restingB+restingA, restingB, restingA)
	fmt.Printf("  Active Levels:     %d (Best Bid: %d, Best Ask: %d)\n", totalLevels, bestBid, bestAsk)
	fmt.Printf("========================================================\n\n")
}
