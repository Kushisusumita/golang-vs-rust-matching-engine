use crate::types::{Order, Side};
use std::ptr;

pub struct OrderNode {
    pub id: u64,
    pub price: u64,
    pub quantity: u64,
    pub side: Side,
    pub next: *mut OrderNode,
}

pub struct OrderPool {
    _nodes: Vec<OrderNode>,
    free: Vec<*mut OrderNode>,
}

impl OrderPool {
    pub fn with_capacity(cap: usize) -> Self {
        let mut nodes = Vec::with_capacity(cap);
        for _ in 0..cap {
            nodes.push(OrderNode {
                id: 0,
                price: 0,
                quantity: 0,
                side: Side::Buy,
                next: ptr::null_mut(),
            });
        }

        let mut free = Vec::with_capacity(cap);
        for node in nodes.iter_mut() {
            free.push(node as *mut OrderNode);
        }

        Self { _nodes: nodes, free }
    }

    #[inline(always)]
    pub fn alloc(&mut self, id: u64, price: u64, qty: u64, side: Side) -> *mut OrderNode {
        if let Some(ptr) = self.free.pop() {
            unsafe {
                (*ptr).id = id;
                (*ptr).price = price;
                (*ptr).quantity = qty;
                (*ptr).side = side;
                (*ptr).next = ptr::null_mut();
            }
            ptr
        } else {
            let b = Box::new(OrderNode {
                id,
                price,
                quantity: qty,
                side,
                next: ptr::null_mut(),
            });
            Box::into_raw(b)
        }
    }

    #[inline(always)]
    pub fn free(&mut self, ptr: *mut OrderNode) {
        unsafe {
            (*ptr).next = ptr::null_mut();
        }
        self.free.push(ptr);
    }
}

pub struct PriceLevel {
    pub price: u64,
    pub total_qty: u64,
    pub head: *mut OrderNode,
    pub tail: *mut OrderNode,
    pub order_count: u32,
}

impl PriceLevel {
    #[inline(always)]
    pub fn new(price: u64) -> Self {
        Self {
            price,
            total_qty: 0,
            head: ptr::null_mut(),
            tail: ptr::null_mut(),
            order_count: 0,
        }
    }

    #[inline(always)]
    pub fn append(&mut self, node: *mut OrderNode, qty: u64) {
        unsafe {
            (*node).next = ptr::null_mut();
            if self.tail.is_null() {
                self.head = node;
                self.tail = node;
            } else {
                (*self.tail).next = node;
                self.tail = node;
            }
        }
        self.total_qty += qty;
        self.order_count += 1;
    }

    #[inline(always)]
    pub fn pop(&mut self) -> *mut OrderNode {
        if self.head.is_null() {
            return ptr::null_mut();
        }
        let node = self.head;
        unsafe {
            self.head = (*node).next;
            if self.head.is_null() {
                self.tail = ptr::null_mut();
            }
            (*node).next = ptr::null_mut();
        }
        self.order_count -= 1;
        node
    }

    #[inline(always)]
    pub fn is_empty(&self) -> bool {
        self.head.is_null()
    }
}

pub struct OrderBook {
    pub bids: Vec<Box<PriceLevel>>,
    pub asks: Vec<Box<PriceLevel>>,
    pub level_pool: Vec<Box<PriceLevel>>,
    pub pool: OrderPool,
    pub trades_count: u64,
    pub matched_volume: u64,
}

impl OrderBook {
    pub fn with_capacity(cap: usize) -> Self {
        let mut level_pool = Vec::with_capacity(512);
        for _ in 0..512 {
            level_pool.push(Box::new(PriceLevel::new(0)));
        }

        Self {
            bids: Vec::with_capacity(512),
            asks: Vec::with_capacity(512),
            level_pool,
            pool: OrderPool::with_capacity(cap),
            trades_count: 0,
            matched_volume: 0,
        }
    }

    #[inline(always)]
    fn acquire_level(&mut self, price: u64) -> Box<PriceLevel> {
        if let Some(mut pl) = self.level_pool.pop() {
            pl.price = price;
            pl.total_qty = 0;
            pl.head = ptr::null_mut();
            pl.tail = ptr::null_mut();
            pl.order_count = 0;
            pl
        } else {
            Box::new(PriceLevel::new(price))
        }
    }

    #[inline(always)]
    fn release_level(&mut self, mut pl: Box<PriceLevel>) {
        pl.head = ptr::null_mut();
        pl.tail = ptr::null_mut();
        self.level_pool.push(pl);
    }

    #[inline(always)]
    pub fn process_order(&mut self, mut order: Order) {
        match order.side {
            Side::Buy => {
                while order.quantity > 0 && !self.asks.is_empty() {
                    let best_ask = &mut self.asks[0];
                    if best_ask.price > order.price {
                        break;
                    }

                    while order.quantity > 0 && !best_ask.head.is_null() {
                        let maker = unsafe { &mut *best_ask.head };

                        let match_qty = std::cmp::min(order.quantity, maker.quantity);
                        self.trades_count += 1;
                        self.matched_volume += match_qty;

                        order.quantity -= match_qty;
                        maker.quantity -= match_qty;
                        best_ask.total_qty -= match_qty;

                        if maker.quantity == 0 {
                            let popped = best_ask.pop();
                            self.pool.free(popped);
                        }
                    }

                    if best_ask.is_empty() {
                        let pl = self.asks.remove(0);
                        self.release_level(pl);
                    }
                }

                if order.quantity > 0 {
                    let node = self.pool.alloc(order.id, order.price, order.quantity, order.side);
                    self.insert_bid(node);
                }
            }
            Side::Sell => {
                while order.quantity > 0 && !self.bids.is_empty() {
                    let best_bid = &mut self.bids[0];
                    if best_bid.price < order.price {
                        break;
                    }

                    while order.quantity > 0 && !best_bid.head.is_null() {
                        let maker = unsafe { &mut *best_bid.head };

                        let match_qty = std::cmp::min(order.quantity, maker.quantity);
                        self.trades_count += 1;
                        self.matched_volume += match_qty;

                        order.quantity -= match_qty;
                        maker.quantity -= match_qty;
                        best_bid.total_qty -= match_qty;

                        if maker.quantity == 0 {
                            let popped = best_bid.pop();
                            self.pool.free(popped);
                        }
                    }

                    if best_bid.is_empty() {
                        let pl = self.bids.remove(0);
                        self.release_level(pl);
                    }
                }

                if order.quantity > 0 {
                    let node = self.pool.alloc(order.id, order.price, order.quantity, order.side);
                    self.insert_ask(node);
                }
            }
        }
    }

    #[inline(always)]
    fn insert_bid(&mut self, node: *mut OrderNode) {
        let (price, qty) = unsafe { ((*node).price, (*node).quantity) };
        let idx = self.bids.partition_point(|l| l.price > price);
        if idx < self.bids.len() && self.bids[idx].price == price {
            self.bids[idx].append(node, qty);
        } else {
            let mut pl = self.acquire_level(price);
            pl.append(node, qty);
            self.bids.insert(idx, pl);
        }
    }

    #[inline(always)]
    fn insert_ask(&mut self, node: *mut OrderNode) {
        let (price, qty) = unsafe { ((*node).price, (*node).quantity) };
        let idx = self.asks.partition_point(|l| l.price < price);
        if idx < self.asks.len() && self.asks[idx].price == price {
            self.asks[idx].append(node, qty);
        } else {
            let mut pl = self.acquire_level(price);
            pl.append(node, qty);
            self.asks.insert(idx, pl);
        }
    }
}
