package main

// OrderNode represents a pooled order with direct pointers (avoiding slice bounds checks in Go)
type OrderNode struct {
	ID       uint64
	Price    uint64
	Quantity uint64
	Side     Side
	Next     *OrderNode
}

type OrderPool struct {
	nodes []OrderNode
	free  []*OrderNode
}

func NewOrderPool(capacity int) *OrderPool {
	p := &OrderPool{
		nodes: make([]OrderNode, capacity),
		free:  make([]*OrderNode, capacity),
	}
	for i := 0; i < capacity; i++ {
		p.free[i] = &p.nodes[i]
	}
	return p
}

func (p *OrderPool) Get(id, price, qty uint64, side Side) *OrderNode {
	n := len(p.free)
	if n == 0 {
		return &OrderNode{ID: id, Price: price, Quantity: qty, Side: side}
	}
	o := p.free[n-1]
	p.free = p.free[:n-1]
	o.ID = id
	o.Price = price
	o.Quantity = qty
	o.Side = side
	o.Next = nil
	return o
}

func (p *OrderPool) Put(o *OrderNode) {
	o.Next = nil
	p.free = append(p.free, o)
}

type PriceLevel struct {
	Price      uint64
	TotalQty   uint64
	Head       *OrderNode
	Tail       *OrderNode
	OrderCount uint32
}

func (pl *PriceLevel) Append(o *OrderNode) {
	o.Next = nil
	if pl.Tail == nil {
		pl.Head = o
		pl.Tail = o
	} else {
		pl.Tail.Next = o
		pl.Tail = o
	}
	pl.TotalQty += o.Quantity
	pl.OrderCount++
}

func (pl *PriceLevel) Pop() *OrderNode {
	if pl.Head == nil {
		return nil
	}
	o := pl.Head
	pl.Head = o.Next
	if pl.Head == nil {
		pl.Tail = nil
	}
	pl.OrderCount--
	o.Next = nil
	return o
}

func (pl *PriceLevel) IsEmpty() bool {
	return pl.Head == nil
}

type OrderBook struct {
	Bids          []*PriceLevel
	Asks          []*PriceLevel
	orderPool     *OrderPool
	levelPool     []*PriceLevel
	TradesCount   uint64
	MatchedVolume uint64
}

func NewOrderBook(initialCap int) *OrderBook {
	ob := &OrderBook{
		Bids:      make([]*PriceLevel, 0, 512),
		Asks:      make([]*PriceLevel, 0, 512),
		orderPool: NewOrderPool(initialCap),
		levelPool: make([]*PriceLevel, 0, 512),
	}
	for i := 0; i < 512; i++ {
		ob.levelPool = append(ob.levelPool, &PriceLevel{})
	}
	return ob
}

func (ob *OrderBook) acquireLevel(price uint64) *PriceLevel {
	n := len(ob.levelPool)
	var pl *PriceLevel
	if n > 0 {
		pl = ob.levelPool[n-1]
		ob.levelPool = ob.levelPool[:n-1]
		pl.Price = price
		pl.Head = nil
		pl.Tail = nil
		pl.TotalQty = 0
		pl.OrderCount = 0
	} else {
		pl = &PriceLevel{Price: price}
	}
	return pl
}

func (ob *OrderBook) releaseLevel(pl *PriceLevel) {
	pl.Head = nil
	pl.Tail = nil
	ob.levelPool = append(ob.levelPool, pl)
}

func (ob *OrderBook) ProcessOrder(id, price, quantity uint64, side Side) {
	if side == Buy {
		for quantity > 0 && len(ob.Asks) > 0 {
			bestAsk := ob.Asks[0]
			if bestAsk.Price > price {
				break
			}

			for quantity > 0 && bestAsk.Head != nil {
				maker := bestAsk.Head
				matchQty := quantity
				if maker.Quantity < matchQty {
					matchQty = maker.Quantity
				}

				ob.TradesCount++
				ob.MatchedVolume += matchQty

				quantity -= matchQty
				maker.Quantity -= matchQty
				bestAsk.TotalQty -= matchQty

				if maker.Quantity == 0 {
					bestAsk.Pop()
					ob.orderPool.Put(maker)
				}
			}

			if bestAsk.IsEmpty() {
				ob.releaseLevel(bestAsk)
				copy(ob.Asks, ob.Asks[1:])
				ob.Asks = ob.Asks[:len(ob.Asks)-1]
			}
		}

		if quantity > 0 {
			node := ob.orderPool.Get(id, price, quantity, side)
			ob.insertBid(node)
		}
	} else {
		for quantity > 0 && len(ob.Bids) > 0 {
			bestBid := ob.Bids[0]
			if bestBid.Price < price {
				break
			}

			for quantity > 0 && bestBid.Head != nil {
				maker := bestBid.Head
				matchQty := quantity
				if maker.Quantity < matchQty {
					matchQty = maker.Quantity
				}

				ob.TradesCount++
				ob.MatchedVolume += matchQty

				quantity -= matchQty
				maker.Quantity -= matchQty
				bestBid.TotalQty -= matchQty

				if maker.Quantity == 0 {
					bestBid.Pop()
					ob.orderPool.Put(maker)
				}
			}

			if bestBid.IsEmpty() {
				ob.releaseLevel(bestBid)
				copy(ob.Bids, ob.Bids[1:])
				ob.Bids = ob.Bids[:len(ob.Bids)-1]
			}
		}

		if quantity > 0 {
			node := ob.orderPool.Get(id, price, quantity, side)
			ob.insertAsk(node)
		}
	}
}

func (ob *OrderBook) insertBid(order *OrderNode) {
	low, high := 0, len(ob.Bids)
	for low < high {
		mid := int(uint(low+high) >> 1)
		if ob.Bids[mid].Price <= order.Price {
			high = mid
		} else {
			low = mid + 1
		}
	}

	if low < len(ob.Bids) && ob.Bids[low].Price == order.Price {
		ob.Bids[low].Append(order)
		return
	}

	pl := ob.acquireLevel(order.Price)
	pl.Append(order)

	ob.Bids = append(ob.Bids, nil)
	copy(ob.Bids[low+1:], ob.Bids[low:])
	ob.Bids[low] = pl
}

func (ob *OrderBook) insertAsk(order *OrderNode) {
	low, high := 0, len(ob.Asks)
	for low < high {
		mid := int(uint(low+high) >> 1)
		if ob.Asks[mid].Price >= order.Price {
			high = mid
		} else {
			low = mid + 1
		}
	}

	if low < len(ob.Asks) && ob.Asks[low].Price == order.Price {
		ob.Asks[low].Append(order)
		return
	}

	pl := ob.acquireLevel(order.Price)
	pl.Append(order)

	ob.Asks = append(ob.Asks, nil)
	copy(ob.Asks[low+1:], ob.Asks[low:])
	ob.Asks[low] = pl
}
