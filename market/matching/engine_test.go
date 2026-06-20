package matching

import (
	"math/rand"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tent-of-trials/market/orderbook"
	"github.com/tent-of-trials/market/types"
)

func setupEngine() *MatchingEngine {
	config := EngineConfig{
		MaxPendingOrders: 1000,
		EnableShorting:   true,
	}
	books := map[types.Symbol]*orderbook.OrderBook{
		"BTC/USD": orderbook.NewOrderBook("BTC/USD", orderbook.Config{MaxDepth: 100}),
	}
	return NewMatchingEngine(config, books)
}

func dec(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

// TestPriceTimePriority verifies that orders are matched in price-time priority.
func TestPriceTimePriority(t *testing.T) {
	engine := setupEngine()

	// Add asks: A1 at 100, A2 at 100 (later), A3 at 101
	engine.PlaceOrder(&types.Order{Side: types.Sell, Price: dec("100"), Quantity: dec("1"), RemainingQty: dec("1"), Symbol: "BTC/USD", ID: "A1"})
	time.Sleep(1 * time.Millisecond)
	engine.PlaceOrder(&types.Order{Side: types.Sell, Price: dec("100"), Quantity: dec("1"), RemainingQty: dec("1"), Symbol: "BTC/USD", ID: "A2"})
	time.Sleep(1 * time.Millisecond)
	engine.PlaceOrder(&types.Order{Side: types.Sell, Price: dec("101"), Quantity: dec("1"), RemainingQty: dec("1"), Symbol: "BTC/USD", ID: "A3"})

	// Add bid to match 2 quantities at 101 (should match A1 then A2)
	trades, err := engine.PlaceOrder(&types.Order{Side: types.Buy, Price: dec("101"), Quantity: dec("2"), RemainingQty: dec("2"), Symbol: "BTC/USD", ID: "B1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(trades))
	}
	if trades[0].SellOrderID != "A1" {
		t.Errorf("expected first trade to match A1, got %s", trades[0].SellOrderID)
	}
	if trades[1].SellOrderID != "A2" {
		t.Errorf("expected second trade to match A2, got %s", trades[1].SellOrderID)
	}
}

// TestPartialFills verifies that partial fills leave correct remaining quantity.
func TestPartialFills(t *testing.T) {
	engine := setupEngine()

	o1 := &types.Order{Side: types.Sell, Price: dec("100"), Quantity: dec("10"), RemainingQty: dec("10"), Symbol: "BTC/USD", ID: "S1"}
	engine.PlaceOrder(o1)

	o2 := &types.Order{Side: types.Buy, Price: dec("100"), Quantity: dec("4"), RemainingQty: dec("4"), Symbol: "BTC/USD", ID: "B1"}
	trades, _ := engine.PlaceOrder(o2)

	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].Quantity.String() != "4" {
		t.Errorf("expected trade quantity 4, got %s", trades[0].Quantity.String())
	}

	if o1.RemainingQty.String() != "6" {
		t.Errorf("expected S1 remaining quantity 6, got %s", o1.RemainingQty.String())
	}
	if o2.RemainingQty.String() != "0" {
		t.Errorf("expected B1 remaining quantity 0, got %s", o2.RemainingQty.String())
	}
}

// TestCanceledOrFilledNotMatched verifies canceled/filled orders cannot be matched again.
func TestCanceledOrFilledNotMatched(t *testing.T) {
	engine := setupEngine()

	engine.PlaceOrder(&types.Order{Side: types.Sell, Price: dec("100"), Quantity: dec("1"), RemainingQty: dec("1"), Symbol: "BTC/USD", ID: "S1"})
	engine.CancelOrder("BTC/USD", "S1")

	trades, _ := engine.PlaceOrder(&types.Order{Side: types.Buy, Price: dec("100"), Quantity: dec("1"), RemainingQty: dec("1"), Symbol: "BTC/USD", ID: "B1"})
	if len(trades) != 0 {
		t.Errorf("expected 0 trades against canceled order S1, got %d", len(trades))
	}
}

// TestRandomizedInvariants runs randomized table-driven tests checking invariants.
func TestRandomizedInvariants(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < 100; i++ {
		engine := setupEngine()
		var orders []*types.Order

		for j := 0; j < 50; j++ {
			side := types.Buy
			if rand.Intn(2) == 0 {
				side = types.Sell
			}
			qtyStr := decimal.NewFromInt(int64(rand.Intn(10) + 1)).String()
			priceStr := decimal.NewFromInt(int64(rand.Intn(10) + 100)).String() // 100-109
			
			order := &types.Order{
				Side:         side,
				Price:        dec(priceStr),
				Quantity:     dec(qtyStr),
				RemainingQty: dec(qtyStr),
				Symbol:       "BTC/USD",
			}
			orders = append(orders, order)
			
			trades, _ := engine.PlaceOrder(order)
			
			// Verify invariant: Trade quantities cannot be negative.
			for _, tr := range trades {
				if tr.Quantity.LessThanOrEqual(decimal.Zero) {
					t.Fatalf("invariant violated: negative or zero trade quantity %s", tr.Quantity.String())
				}
			}

			// Verify invariant: remaining qty should not be negative.
			if order.RemainingQty.LessThan(decimal.Zero) {
				t.Fatalf("invariant violated: negative remaining quantity %s", order.RemainingQty.String())
			}
		}
	}
}
