package hyperliquid

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"arbitrage-bot/internal/config"
	"arbitrage-bot/internal/exchange"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sonirico/go-hyperliquid"
)

type Client struct {
	cfg      config.HyperliquidConfig
	info     *hyperliquid.Info
	exchange *hyperliquid.Exchange
	meta     *hyperliquid.Meta
}

func NewClient(cfg config.HyperliquidConfig) *Client {
	ctx := context.Background()

	// Initialize Info client
	// NewInfo(ctx, baseURL, skipWS, meta, spotMeta, opts...)
	info := hyperliquid.NewInfo(ctx, cfg.BaseURL, true, nil, nil)

	// Fetch Meta (needed for Exchange and symbol lookup)
	meta, err := info.Meta(ctx)
	if err != nil {
		log.Printf("Failed to fetch meta: %v", err)
	}

	var exc *hyperliquid.Exchange
	if cfg.PrivateKey != "" && meta != nil {
		pk, err := crypto.HexToECDSA(cfg.PrivateKey)
		if err != nil {
			log.Printf("Failed to parse private key: %v", err)
		} else {
			// Derive address if not provided
			walletAddr := cfg.WalletAddress
			if walletAddr == "" {
				walletAddr = crypto.PubkeyToAddress(pk.PublicKey).Hex()
			}

			// NewExchange(ctx, pk, baseURL, meta, vaultAddress, accountAddress, spotMeta, opts...)
			exc = hyperliquid.NewExchange(ctx, pk, cfg.BaseURL, meta, "", walletAddr, nil)
		}
	}

	return &Client{
		cfg:      cfg,
		info:     info,
		exchange: exc,
		meta:     meta,
	}
}

// Implement Exchange interface
var _ exchange.Exchange = (*Client)(nil)

func (c *Client) GetFundingRate(symbol string) (float64, error) {
	// Normalize symbol: ETH-USD -> ETH
	normalizedSymbol := strings.TrimSuffix(symbol, "-USD")

	// Use SDK to get MetaAndAssetCtxs
	state, err := c.info.MetaAndAssetCtxs(context.Background())
	if err != nil {
		return 0, err
	}

	// Find the asset index
	assetIndex := -1
	for i, asset := range state.Universe {
		if asset.Name == normalizedSymbol {
			assetIndex = i
			break
		}
	}

	if assetIndex == -1 {
		return 0, fmt.Errorf("symbol %s not found in universe", normalizedSymbol)
	}

	if assetIndex >= len(state.Ctxs) {
		return 0, fmt.Errorf("asset context not found for index %d", assetIndex)
	}

	fundingStr := state.Ctxs[assetIndex].Funding
	fundingRate, err := strconv.ParseFloat(fundingStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse funding rate: %w", err)
	}

	return fundingRate, nil
}

func (c *Client) GetPrice(symbol string) (float64, error) {
	// Normalize symbol
	normalizedSymbol := strings.TrimSuffix(symbol, "-USD")

	state, err := c.info.MetaAndAssetCtxs(context.Background())
	if err != nil {
		return 0, err
	}

	for i, asset := range state.Universe {
		if asset.Name == normalizedSymbol {
			if i < len(state.Ctxs) {
				return strconv.ParseFloat(state.Ctxs[i].MidPx, 64)
			}
		}
	}
	return 0, fmt.Errorf("symbol not found")
}

func (c *Client) GetBalance(asset string) (float64, error) {
	// TODO: Implement using c.info.UserState(address)
	return 0, fmt.Errorf("not implemented")
}

func (c *Client) GetPosition(symbol string) (*exchange.Position, error) {
	// TODO: Implement using c.info.UserState(address)
	return nil, fmt.Errorf("not implemented")
}

func (c *Client) PlaceOrder(req *exchange.OrderRequest) (*exchange.OrderResponse, error) {
	if c.exchange == nil {
		return nil, fmt.Errorf("exchange client not initialized (check private key)")
	}

	normalizedSymbol := strings.TrimSuffix(req.Symbol, "-USD")

	// Find asset index from meta
	assetIndex := -1
	for i, asset := range c.meta.Universe {
		if asset.Name == normalizedSymbol {
			assetIndex = i
			break
		}
	}
	if assetIndex == -1 {
		return nil, fmt.Errorf("symbol %s not found", normalizedSymbol)
	}

	isBuy := req.Side == "buy"

	// Construct Order Request
	orderReq := hyperliquid.CreateOrderRequest{
		Coin:  normalizedSymbol,
		IsBuy: isBuy,
		Size:  req.Size,
		Price: req.Price,
		OrderType: hyperliquid.OrderType{
			Limit: &hyperliquid.LimitOrderType{
				Tif: hyperliquid.TifGtc,
			},
		},
		ReduceOnly: req.ReduceOnly,
	}

	// Pass nil for builder info
	res, err := c.exchange.Order(context.Background(), orderReq, nil)
	if err != nil {
		return nil, err
	}

	// Parse response
	status := "unknown"
	var orderID string

	if res.Error != nil {
		return nil, fmt.Errorf("order failed: %s", *res.Error)
	}

	if res.Resting != nil {
		status = "open"
		orderID = strconv.FormatInt(res.Resting.Oid, 10)
	} else if res.Filled != nil {
		status = "filled"
		orderID = strconv.Itoa(res.Filled.Oid)
	}

	return &exchange.OrderResponse{
		Status:  status,
		OrderID: orderID,
	}, nil
}

func (c *Client) CancelOrder(symbol, orderID string) error {
	return fmt.Errorf("not implemented")
}
