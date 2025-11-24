package strategy

import (
	"context"
	"log"
	"math/rand"
	"time"

	"arbitrage-bot/internal/config"
	"arbitrage-bot/internal/exchange"
)

type XPFarmingStrategy struct {
	cfg       config.XPFarmingConfig
	exchanges map[string]exchange.Exchange
}

func NewXPFarmingStrategy(cfg config.XPFarmingConfig, exchanges map[string]exchange.Exchange) *XPFarmingStrategy {
	return &XPFarmingStrategy{
		cfg:       cfg,
		exchanges: exchanges,
	}
}

func (s *XPFarmingStrategy) Start(ctx context.Context) {
	log.Println("Starting XP Farming Strategy...")

	// Random interval to avoid detection (e.g., between 5 to 15 minutes)
	// For testing, we use shorter intervals
	minInterval := 30 * time.Second
	maxInterval := 2 * time.Minute

	timer := time.NewTimer(s.randomDuration(minInterval, maxInterval))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping XP Farming Strategy...")
			return
		case <-timer.C:
			s.executeFarming()
			timer.Reset(s.randomDuration(minInterval, maxInterval))
		}
	}
}

func (s *XPFarmingStrategy) executeFarming() {
	// Logic:
	// 1. Select a random exchange (from enabled ones)
	// 2. Check current volume (if API supports) or just execute a trade
	// 3. Place a small order (Long/Short) and immediately close it (or reverse) to generate volume
	// 4. Ensure slippage is within limits

	// For MVP, we'll just pick Hyperliquid and execute a wash trade (Buy then Sell)
	// WARNING: This incurs fees. Ensure config allows this.

	targetExchange := "hyperliquid" // TODO: Randomize or config
	exc, ok := s.exchanges[targetExchange]
	if !ok {
		log.Printf("XP Farming: Exchange %s not found", targetExchange)
		return
	}

	symbol := "ETH" // TODO: Configurable
	size := 0.01    // TODO: Configurable based on target volume

	log.Printf("XP Farming: Executing wash trade on %s for %f %s", targetExchange, size, symbol)

	// 1. Get Price
	price, err := exc.GetPrice(symbol)
	if err != nil {
		log.Printf("XP Farming: Failed to get price: %v", err)
		return
	}

	// 2. Place Buy Order
	buyPrice := price * (1 + s.cfg.MaxSlippage)
	buyReq := &exchange.OrderRequest{
		Symbol:     symbol,
		Side:       "buy",
		Size:       size,
		Type:       "limit",
		Price:      buyPrice,
		ReduceOnly: false,
	}

	buyRes, err := exc.PlaceOrder(buyReq)
	if err != nil {
		log.Printf("XP Farming: Buy failed: %v", err)
		return
	}
	log.Printf("XP Farming: Buy placed (Status: %s, ID: %s)", buyRes.Status, buyRes.OrderID)

	// Wait a bit to ensure fill (if using limit) or just small delay
	time.Sleep(2 * time.Second)

	// 3. Place Sell Order (Close position)
	// Note: In a real scenario, we should check if Buy was filled.
	sellPrice := price * (1 - s.cfg.MaxSlippage)
	sellReq := &exchange.OrderRequest{
		Symbol:     symbol,
		Side:       "sell",
		Size:       size,
		Type:       "limit",
		Price:      sellPrice,
		ReduceOnly: true, // Ensure we are closing
	}

	sellRes, err := exc.PlaceOrder(sellReq)
	if err != nil {
		log.Printf("XP Farming: Sell failed: %v", err)
		return
	}
	log.Printf("XP Farming: Sell placed (Status: %s, ID: %s)", sellRes.Status, sellRes.OrderID)
}

func (s *XPFarmingStrategy) randomDuration(min, max time.Duration) time.Duration {
	delta := max - min
	return min + time.Duration(rand.Int63n(int64(delta)))
}
