package main

type Side uint8

const (
	Buy  Side = 0
	Sell Side = 1
)

func (s Side) String() string {
	if s == Buy {
		return "BUY"
	}
	return "SELL"
}

type Order struct {
	ID       uint64
	Price    uint64
	Quantity uint64
	Side     Side
}

type Trade struct {
	MakerOrderID uint64
	TakerOrderID uint64
	Price        uint64
	Quantity     uint64
}
