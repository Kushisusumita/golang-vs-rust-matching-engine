package main

type Xorshift64 struct {
	state uint64
}

func NewXorshift64(seed uint64) *Xorshift64 {
	if seed == 0 {
		seed = 0xDEADBEEFCAFE1234
	}
	return &Xorshift64{state: seed}
}

func (x *Xorshift64) Next() uint64 {
	v := x.state
	v ^= v << 13
	v ^= v >> 7
	v ^= v << 17
	x.state = v
	return v
}

type OrderInput struct {
	ID       uint64
	Price    uint64
	Quantity uint64
	Side     Side
}

func GenerateOrders(n int, seed uint64) []OrderInput {
	rng := NewXorshift64(seed)
	orders := make([]OrderInput, n)
	for i := 0; i < n; i++ {
		r1 := rng.Next()
		r2 := rng.Next()
		r3 := rng.Next()

		side := Side(r1 & 1)
		// Price around 10,000 with +/- 100 tick variance
		priceOffset := int64(r2%200) - 100
		price := uint64(10000 + priceOffset)
		// Quantity between 1 and 100
		qty := 1 + (r3 % 100)

		orders[i] = OrderInput{
			ID:       uint64(i + 1),
			Price:    price,
			Quantity: qty,
			Side:     side,
		}
	}
	return orders
}
