package main

import (
	"context"
	"log"

	"arbitrage-bot/internal/config"
	"arbitrage-bot/internal/exchange"
	"arbitrage-bot/internal/exchange/edgex"
	"arbitrage-bot/internal/exchange/hyperliquid"
	"arbitrage-bot/internal/exchange/lighter"
	"arbitrage-bot/internal/strategy"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Config loaded successfully")
	log.Printf("App Port: %d", cfg.App.Port)
	log.Printf("Funding Arb Enabled: %v", cfg.Strategies.FundingArb.Enabled)
	log.Printf("Hyperliquid Wallet: %s", cfg.Exchanges.Hyperliquid.WalletAddress)

	// Initialize Exchanges
	exchanges := make(map[string]exchange.Exchange)
	exchanges["hyperliquid"] = hyperliquid.NewClient(cfg.Exchanges.Hyperliquid)
	exchanges["lighter"] = lighter.NewClient(cfg.Exchanges.Lighter)
	exchanges["edgex"] = edgex.NewClient(cfg.Exchanges.EdgeX)

	// Initialize and Start Strategy
	if cfg.Strategies.FundingArb.Enabled {
		arbStrategy := strategy.NewFundingArbStrategy(cfg.Strategies.FundingArb, exchanges)

		// Run in background
		ctx := context.Background()
		go arbStrategy.Start(ctx)
	}

	if cfg.Strategies.XPFarming.Enabled {
		xpStrategy := strategy.NewXPFarmingStrategy(cfg.Strategies.XPFarming, exchanges)

		// Run in background
		ctx := context.Background()
		go xpStrategy.Start(ctx)
	}

	// Keep main alive
	select {}
}
