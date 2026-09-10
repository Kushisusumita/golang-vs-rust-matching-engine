use matching_engine_rust::types::{Order, Side};
use matching_engine_rust::generator::generate_orders;
use matching_engine_rust::orderbook::OrderBook;

#[test]
fn test_matching_logic() {
    let mut ob = OrderBook::with_capacity(100);

    // 1. Place resting Ask: 10 units @ 100
    ob.process_order(Order { id: 1, price: 100, quantity: 10, side: Side::Sell });
    assert_eq!(ob.asks.len(), 1);
    assert_eq!(ob.asks[0].total_qty, 10);

    // 2. Place matching Bid: 6 units @ 100 (partial fill)
    ob.process_order(Order { id: 2, price: 100, quantity: 6, side: Side::Buy });
    assert_eq!(ob.trades_count, 1);
    assert_eq!(ob.matched_volume, 6);
    assert_eq!(ob.asks[0].total_qty, 4);

    // 3. Place aggressive Bid: 10 units @ 101 (exhausts ask @ 100, 6 units rest at 101)
    ob.process_order(Order { id: 3, price: 101, quantity: 10, side: Side::Buy });
    assert_eq!(ob.trades_count, 2);
    assert_eq!(ob.matched_volume, 10);
    assert_eq!(ob.asks.len(), 0);
    assert_eq!(ob.bids.len(), 1);
    assert_eq!(ob.bids[0].price, 101);
    assert_eq!(ob.bids[0].total_qty, 6);
}

#[test]
fn test_determinism_100k() {
    const SEED: u64 = 0xDEADBEEFCAFE1234;
    const N: usize = 100_000;

    let orders = generate_orders(N, SEED);
    let mut ob = OrderBook::with_capacity(N);

    for order in orders {
        ob.process_order(order);
    }

    assert_eq!(ob.trades_count, 77576);
    assert_eq!(ob.matched_volume, 1973216);
    assert_eq!(ob.bids.len(), 55);
    assert_eq!(ob.asks.len(), 50);
}
