package main

import (
	"testing"
)

func TestMatchingLogic(t *testing.T) {
	ob := NewOrderBook(100)

	// 1. Place resting Ask: 10 units @ 100
	ob.ProcessOrder(1, 100, 10, Sell)
	if len(ob.Asks) != 1 || ob.Asks[0].TotalQty != 10 {
		t.Fatalf("Expected 1 Ask level with 10 qty, got %+v", ob.Asks)
	}

	// 2. Place matching Bid: 6 units @ 100 (partial fill of resting ask)
	ob.ProcessOrder(2, 100, 6, Buy)
	if ob.TradesCount != 1 {
		t.Fatalf("Expected 1 trade, got %d", ob.TradesCount)
	}
	if ob.MatchedVolume != 6 {
		t.Fatalf("Expected 6 volume matched, got %d", ob.MatchedVolume)
	}
	if ob.Asks[0].TotalQty != 4 {
		t.Fatalf("Expected remaining ask qty 4, got %d", ob.Asks[0].TotalQty)
	}

	// 3. Place aggressive Bid: 10 units @ 101 (exhausts ask @ 100, 6 units rest at 101)
	ob.ProcessOrder(3, 101, 10, Buy)
	if ob.TradesCount != 2 {
		t.Fatalf("Expected 2 trades, got %d", ob.TradesCount)
	}
	if ob.MatchedVolume != 10 {
		t.Fatalf("Expected 10 total volume, got %d", ob.MatchedVolume)
	}
	if len(ob.Asks) != 0 {
		t.Fatalf("Expected ask book empty, got %d levels", len(ob.Asks))
	}
	if len(ob.Bids) != 1 || ob.Bids[0].Price != 101 || ob.Bids[0].TotalQty != 6 {
		t.Fatalf("Expected 1 bid level @ 101 with 6 qty, got %+v", ob.Bids)
	}
}

func TestDeterminism100k(t *testing.T) {
	const seed uint64 = 0xDEADBEEFCAFE1234
	const n = 100000

	orders := GenerateOrders(n, seed)
	ob := NewOrderBook(n)

	for i := 0; i < n; i++ {
		o := &orders[i]
		ob.ProcessOrder(o.ID, o.Price, o.Quantity, o.Side)
	}

	if ob.TradesCount != 77576 {
		t.Fatalf("Expected 77576 trades, got %d", ob.TradesCount)
	}
	if ob.MatchedVolume != 1973216 {
		t.Fatalf("Expected 1973216 volume, got %d", ob.MatchedVolume)
	}
	if len(ob.Bids) != 55 || len(ob.Asks) != 50 {
		t.Fatalf("Expected 55 bid levels and 50 ask levels, got %d and %d", len(ob.Bids), len(ob.Asks))
	}
}

func BenchmarkGoMatchingEngine100k(b *testing.B) {
	const seed uint64 = 0xDEADBEEFCAFE1234
	const n = 100000
	orders := GenerateOrders(n, seed)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ob := NewOrderBook(n)
		for j := 0; j < n; j++ {
			o := &orders[j]
			ob.ProcessOrder(o.ID, o.Price, o.Quantity, o.Side)
		}
	}
}
