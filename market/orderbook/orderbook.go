package orderbook

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tent-of-trials/market/types"
)

type Config struct {
	MaxDepth       int
	PriceDecimals  int32
	VolumeDecimals int32
}

type PriceLevel struct {
	Price  decimal.Decimal
	Orders []*types.Order
}

type OrderBook struct {
	mu        sync.RWMutex
	symbol    types.Symbol
	config    Config
	bids      []*PriceLevel // sorted desc
	asks      []*PriceLevel // sorted asc
	orders    map[string]*types.Order
	sequence  uint64
	updatedAt time.Time
	closed    bool
}

func NewOrderBook(symbol types.Symbol, config Config) *OrderBook {
	return &OrderBook{
		symbol:   symbol,
		config:   config,
		bids:     make([]*PriceLevel, 0),
		asks:     make([]*PriceLevel, 0),
		orders:   make(map[string]*types.Order),
		sequence: 0,
	}
}

func minDecimal(a, b decimal.Decimal) decimal.Decimal {
	if a.LessThan(b) {
		return a
	}
	return b
}

func (ob *OrderBook) AddOrder(order *types.Order) ([]*types.Trade, error) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	if ob.closed {
		return nil, ErrBookClosed
	}

	if order.ID == "" {
		order.ID = uuid.New().String()
	}

	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
	order.Status = types.New
	if order.RemainingQty.IsZero() {
		order.RemainingQty = order.Quantity
	}

	var trades []*types.Trade

	// Match order against opposite side
	if order.Side == types.Buy {
		for len(ob.asks) > 0 && order.RemainingQty.GreaterThan(decimal.Zero) {
			bestAsk := ob.asks[0]
			if order.Price.LessThan(bestAsk.Price) {
				break
			}
			
			// Match with orders at this price level
			trades = append(trades, ob.matchAtLevel(bestAsk, order)...)
			
			if len(bestAsk.Orders) == 0 {
				ob.asks = ob.asks[1:] // remove empty level
			}
		}
	} else {
		for len(ob.bids) > 0 && order.RemainingQty.GreaterThan(decimal.Zero) {
			bestBid := ob.bids[0]
			if order.Price.GreaterThan(bestBid.Price) {
				break
			}
			
			// Match with orders at this price level
			trades = append(trades, ob.matchAtLevel(bestBid, order)...)
			
			if len(bestBid.Orders) == 0 {
				ob.bids = ob.bids[1:] // remove empty level
			}
		}
	}

	// If there's remaining quantity, add to book
	if order.RemainingQty.GreaterThan(decimal.Zero) {
		ob.orders[order.ID] = order
		if order.Side == types.Buy {
			ob.addBid(order)
		} else {
			ob.addAsk(order)
		}
	}

	ob.sequence++
	ob.updatedAt = time.Now()
	return trades, nil
}

func (ob *OrderBook) matchAtLevel(level *PriceLevel, taker *types.Order) []*types.Trade {
	var trades []*types.Trade
	
	for i := 0; i < len(level.Orders) && taker.RemainingQty.GreaterThan(decimal.Zero); i++ {
		maker := level.Orders[i]
		
		tradeQty := minDecimal(taker.RemainingQty, maker.RemainingQty)
		tradePrice := maker.Price
		
		taker.RemainingQty = taker.RemainingQty.Sub(tradeQty)
		maker.RemainingQty = maker.RemainingQty.Sub(tradeQty)
		
		trade := &types.Trade{
			Symbol:    ob.symbol,
			Price:     tradePrice,
			Quantity:  tradeQty,
			QuoteQty:  tradePrice.Mul(tradeQty),
			TakerSide: taker.Side,
		}
		
		if taker.Side == types.Buy {
			trade.BuyOrderID = taker.ID
			trade.SellOrderID = maker.ID
			trade.IsBuyerMaker = false
		} else {
			trade.BuyOrderID = maker.ID
			trade.SellOrderID = taker.ID
			trade.IsBuyerMaker = true
		}
		
		trades = append(trades, trade)
		
		if maker.RemainingQty.IsZero() {
			delete(ob.orders, maker.ID)
			// Remove maker from level.Orders
			level.Orders = append(level.Orders[:i], level.Orders[i+1:]...)
			i-- // adjust index
		}
	}
	
	return trades
}

func (ob *OrderBook) addBid(order *types.Order) {
	for i, level := range ob.bids {
		if level.Price.Equal(order.Price) {
			level.Orders = append(level.Orders, order)
			return
		}
		if order.Price.GreaterThan(level.Price) {
			newLevel := &PriceLevel{Price: order.Price, Orders: []*types.Order{order}}
			ob.bids = append(ob.bids[:i], append([]*PriceLevel{newLevel}, ob.bids[i:]...)...)
			return
		}
	}
	ob.bids = append(ob.bids, &PriceLevel{Price: order.Price, Orders: []*types.Order{order}})
}

func (ob *OrderBook) addAsk(order *types.Order) {
	for i, level := range ob.asks {
		if level.Price.Equal(order.Price) {
			level.Orders = append(level.Orders, order)
			return
		}
		if order.Price.LessThan(level.Price) {
			newLevel := &PriceLevel{Price: order.Price, Orders: []*types.Order{order}}
			ob.asks = append(ob.asks[:i], append([]*PriceLevel{newLevel}, ob.asks[i:]...)...)
			return
		}
	}
	ob.asks = append(ob.asks, &PriceLevel{Price: order.Price, Orders: []*types.Order{order}})
}

func (ob *OrderBook) CancelOrder(orderID string) error {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	if ob.closed {
		return ErrBookClosed
	}

	order, exists := ob.orders[orderID]
	if !exists {
		return ErrOrderNotFound
	}

	order.Status = types.Cancelled
	order.UpdatedAt = time.Now()
	delete(ob.orders, orderID)

	// Remove from levels
	if order.Side == types.Buy {
		ob.removeOrderFromLevels(&ob.bids, order)
	} else {
		ob.removeOrderFromLevels(&ob.asks, order)
	}

	ob.updatedAt = time.Now()
	return nil
}

func (ob *OrderBook) removeOrderFromLevels(levels *[]*PriceLevel, order *types.Order) {
	for i, level := range *levels {
		if level.Price.Equal(order.Price) {
			for j, o := range level.Orders {
				if o.ID == order.ID {
					level.Orders = append(level.Orders[:j], level.Orders[j+1:]...)
					if len(level.Orders) == 0 {
						*levels = append((*levels)[:i], (*levels)[i+1:]...)
					}
					return
				}
			}
		}
	}
}

func (ob *OrderBook) GetBids() []*types.Level {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	result := make([]*types.Level, 0, len(ob.bids))
	for _, level := range ob.bids {
		result = append(result, levelToType(level))
	}
	return result
}

func (ob *OrderBook) GetAsks() []*types.Level {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	result := make([]*types.Level, 0, len(ob.asks))
	for _, level := range ob.asks {
		result = append(result, levelToType(level))
	}
	return result
}

func (ob *OrderBook) GetSnapshot() *types.DepthUpdate {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	bids := make([]types.Level, 0, len(ob.bids))
	for _, l := range ob.bids {
		bids = append(bids, *levelToType(l))
	}

	asks := make([]types.Level, 0, len(ob.asks))
	for _, l := range ob.asks {
		asks = append(asks, *levelToType(l))
	}

	return &types.DepthUpdate{
		Symbol:    ob.symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now().UnixMilli(),
	}
}

func levelToType(p *PriceLevel) *types.Level {
	qty := decimal.Zero
	count := int64(len(p.Orders))
	for _, o := range p.Orders {
		qty = qty.Add(o.RemainingQty)
	}
	return &types.Level{
		Price:    p.Price,
		Quantity: qty,
		Count:    count,
	}
}

func (ob *OrderBook) Close() {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	ob.closed = true
	ob.bids = nil
	ob.asks = nil
	ob.orders = nil
}

var (
	ErrBookClosed    = &BookError{"order book is closed"}
	ErrOrderNotFound = &BookError{"order not found"}
)

type BookError struct {
	message string
}

func (e *BookError) Error() string {
	return e.message
}
