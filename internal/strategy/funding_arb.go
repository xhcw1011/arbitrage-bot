package strategy

import (
	"context"
	"log"
	"time"

	"arbitrage-bot/internal/config"
	"arbitrage-bot/internal/exchange"
)

type FundingArbStrategy struct {
	cfg       config.FundingArbConfig
	exchanges map[string]exchange.Exchange
	stopCh    chan struct{}
}

func NewFundingArbStrategy(cfg config.FundingArbConfig, exchanges map[string]exchange.Exchange) *FundingArbStrategy {
	return &FundingArbStrategy{
		cfg:       cfg,
		exchanges: exchanges,
		stopCh:    make(chan struct{}),
	}
}

func (s *FundingArbStrategy) Start(ctx context.Context) {
	log.Println("Starting Funding Arb Strategy...")
	ticker := time.NewTicker(time.Duration(s.cfg.CheckIntervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping Funding Arb Strategy...")
			return
		case <-ticker.C:
			s.checkOpportunities()
		}
	}
}

func (s *FundingArbStrategy) checkOpportunities() {
	// Logic to check funding rates across exchanges
	// For now, just print that we are checking
	log.Println("Checking funding opportunities...")

	// Iterate pairs and get funding rates
	for _, pair := range s.cfg.Pairs {
		rates := make(map[string]float64)
		for name, exc := range s.exchanges {
			rate, err := exc.GetFundingRate(pair)
			if err != nil {
				log.Printf("Error getting funding rate from %s for %s: %v", name, pair, err)
				continue
			}
			rates[name] = rate
			// log.Printf("[%s] %s Funding Rate: %f", name, pair, rate)
		}

		// Calculate max difference
		if len(rates) < 2 {
			continue
		}

		var maxRate, minRate float64
		var maxName, minName string
		first := true

		for name, rate := range rates {
			if first {
				maxRate = rate
				minRate = rate
				maxName = name
				minName = name
				first = false
				continue
			}
			if rate > maxRate {
				maxRate = rate
				maxName = name
			}
			if rate < minRate {
				minRate = rate
				minName = name
			}
		}

		diff := maxRate - minRate
		if diff >= s.cfg.MinFundingDiff {
			log.Printf("OPPORTUNITY FOUND [%s]: Buy %s on %s (Rate: %f) / Sell on %s (Rate: %f) | Diff: %f",
				pair, pair, minName, minRate, maxName, maxRate, diff)

			if s.cfg.ExecuteTrades {
				s.executeArbitrage(pair, minName, maxName)
			}
		} else {
			log.Printf("[%s] Best Diff: %f (Threshold: %f) - No Opportunity", pair, diff, s.cfg.MinFundingDiff)
		}
	}
}

func (s *FundingArbStrategy) executeArbitrage(symbol, longExchange, shortExchange string) {
	// Fixed size for testing - TODO: Make configurable or dynamic
	size := 0.01 // e.g. 0.01 ETH

	log.Printf("Executing Arbitrage: Long %f %s on %s, Short %f %s on %s",
		size, symbol, longExchange, size, symbol, shortExchange)

	// Execute Long
	go func() {
		price, err := s.exchanges[longExchange].GetPrice(symbol)
		if err != nil {
			log.Printf("Failed to get price from %s: %v", longExchange, err)
			return
		}
		// Buy with 1% slippage
		limitPrice := price * 1.01

		_, err = s.exchanges[longExchange].PlaceOrder(&exchange.OrderRequest{
			Symbol: symbol,
			Side:   "buy",
			Size:   size,
			Type:   "limit",
			Price:  limitPrice,
		})
		if err != nil {
			log.Printf("Failed to place Long on %s: %v", longExchange, err)
		} else {
			log.Printf("Placed Long on %s at %f", longExchange, limitPrice)
		}
	}()

	// Execute Short
	go func() {
		price, err := s.exchanges[shortExchange].GetPrice(symbol)
		if err != nil {
			log.Printf("Failed to get price from %s: %v", shortExchange, err)
			return
		}
		// Sell with 1% slippage
		limitPrice := price * 0.99

		_, err = s.exchanges[shortExchange].PlaceOrder(&exchange.OrderRequest{
			Symbol: symbol,
			Side:   "sell",
			Size:   size,
			Type:   "limit",
			Price:  limitPrice,
		})
		if err != nil {
			log.Printf("Failed to place Short on %s: %v", shortExchange, err)
		} else {
			log.Printf("Placed Short on %s at %f", shortExchange, limitPrice)
		}
	}()
}
