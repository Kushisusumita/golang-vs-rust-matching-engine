#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u8)]
pub enum Side {
    Buy = 0,
    Sell = 1,
}

#[derive(Debug, Clone, Copy)]
pub struct Order {
    pub id: u64,
    pub price: u64,
    pub quantity: u64,
    pub side: Side,
}
