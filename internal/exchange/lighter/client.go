package lighter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/elliottech/lighter-go/client"
	lighterhttp "github.com/elliottech/lighter-go/client/http"
	"github.com/elliottech/lighter-go/types"
	"github.com/elliottech/lighter-go/types/txtypes"

	"arbitrage-bot/internal/config"
	"arbitrage-bot/internal/exchange"
)

type Client struct {
	cfg        config.LighterConfig
	httpClient *http.Client
	txClient   *client.TxClient
}

// Lighter API Response structures
type FundingRatesResponse struct {
	Code         int           `json:"code"`
	FundingRates []FundingRate `json:"funding_rates"`
}

type FundingRate struct {
	MarketId int     `json:"market_id"`
	Exchange string  `json:"exchange"`
	Symbol   string  `json:"symbol"`
	Rate     float64 `json:"rate"`
}

const (
	LighterChainId = 1 // Mainnet chain ID, adjust if needed
)

func NewClient(cfg config.LighterConfig) *Client {
	c := &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Initialize TxClient if private key is configured
	if cfg.PrivateKey != "" && cfg.APIKey != "" {
		httpCli := lighterhttp.NewClient(cfg.BaseURL)

		// CreateClient(httpClient, privateKey, chainId, apiKeyIndex, accountIndex)
		// Using default values: apiKeyIndex=0, accountIndex=1
		txClient, err := client.CreateClient(httpCli, cfg.PrivateKey, LighterChainId, 0, 1)
		if err != nil {
			fmt.Printf("Warning: Failed to create Lighter TxClient: %v\n", err)
		} else {
			c.txClient = txClient
			// Verify the client
			if err := txClient.Check(); err != nil {
				fmt.Printf("Warning: Lighter TxClient check failed: %v\n", err)
			} else {
				fmt.Println("Lighter TxClient initialized successfully")
			}
		}
	}

	return c
}

var _ exchange.Exchange = (*Client)(nil)

func (c *Client) GetFundingRate(symbol string) (float64, error) {
	// Normalize: ETH-USD -> ETH
	normalizedSymbol := strings.TrimSuffix(symbol, "-USD")
	normalizedSymbol = strings.TrimSuffix(normalizedSymbol, "USDT")

	url := c.cfg.BaseURL + "/api/v1/funding-rates"

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var fundingResp FundingRatesResponse
	if err := json.Unmarshal(body, &fundingResp); err != nil {
		return 0, err
	}

	if fundingResp.Code != 200 {
		return 0, fmt.Errorf("API error code: %d", fundingResp.Code)
	}

	// Find the funding rate for the symbol
	for _, fr := range fundingResp.FundingRates {
		if strings.EqualFold(fr.Symbol, normalizedSymbol) {
			return fr.Rate, nil
		}
	}

	return 0, fmt.Errorf("funding rate not found for symbol: %s", normalizedSymbol)
}

// addAuthHeaders adds authentication headers to the request if API key is configured
func (c *Client) addAuthHeaders(req *http.Request) {
	if c.cfg.APIKey != "" {
		req.Header.Set("X-API-KEY", c.cfg.APIKey)
		// Lighter may use different header names, adjust as needed
		// Common alternatives: "Authorization", "api-key", etc.
	}
}

// makeAuthenticatedRequest creates and executes an authenticated HTTP request
func (c *Client) makeAuthenticatedRequest(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	c.addAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

func (c *Client) GetPrice(symbol string) (float64, error) {
	// TODO: Implement using orderbook or ticker endpoint
	return 0, fmt.Errorf("not implemented - use orderbook endpoint")
}

func (c *Client) GetBalance(asset string) (float64, error) {
	return 0, fmt.Errorf("not implemented - requires authentication")
}

func (c *Client) GetPosition(symbol string) (*exchange.Position, error) {
	return nil, fmt.Errorf("not implemented - requires authentication")
}

func (c *Client) PlaceOrder(req *exchange.OrderRequest) (*exchange.OrderResponse, error) {
	if c.txClient == nil {
		return nil, fmt.Errorf("txClient not initialized - check private_key and api_key configuration")
	}

	// Convert symbol to market index
	marketIndex, err := c.getMarketIndex(req.Symbol)
	if err != nil {
		return nil, err
	}

	// Convert order parameters
	isAsk := uint8(0)
	if req.Side == "sell" {
		isAsk = 1
	}

	// Convert price and size to Lighter's format
	// Lighter uses fixed-point integers
	// TODO: Adjust precision based on market specs
	priceInt := uint32(req.Price * 100) // Example: 2 decimal places
	sizeInt := int64(req.Size * 1e18)   // Example: 18 decimal places

	// Determine order type
	orderType := uint8(txtypes.LimitOrder)
	if req.Type == "market" {
		orderType = uint8(txtypes.MarketOrder)
	}

	// Time in force
	timeInForce := uint8(txtypes.GoodTillTime)
	if req.Type == "market" {
		timeInForce = uint8(txtypes.ImmediateOrCancel)
	}

	reduceOnly := uint8(0)
	if req.ReduceOnly {
		reduceOnly = 1
	}

	// Create order request
	orderReq := &types.CreateOrderTxReq{
		MarketIndex:      uint8(marketIndex),
		ClientOrderIndex: time.Now().UnixNano(),
		BaseAmount:       sizeInt,
		Price:            priceInt,
		IsAsk:            isAsk,
		Type:             orderType,
		TimeInForce:      timeInForce,
		ReduceOnly:       reduceOnly,
		TriggerPrice:     txtypes.NilOrderTriggerPrice,
		OrderExpiry:      time.Now().Add(24 * time.Hour).Unix(),
	}

	// Get signed transaction
	txInfo, err := c.txClient.GetCreateOrderTransaction(orderReq, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create order transaction: %w", err)
	}

	// Convert to JSON
	txJSON, err := txInfo.GetTxInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %w", err)
	}

	// Send the signed order to the exchange
	orderURL := c.cfg.BaseURL + "/api/v1/orders"
	resp, err := c.makeAuthenticatedRequest("POST", orderURL, bytes.NewBufferString(txJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to send order: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("order failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse order response
	var orderResp map[string]interface{}
	if err := json.Unmarshal(respBody, &orderResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &exchange.OrderResponse{
		Status:  "submitted",
		OrderID: fmt.Sprintf("%v", orderResp["order_id"]),
	}, nil
}

func (c *Client) CancelOrder(symbol, orderID string) error {
	if c.txClient == nil {
		return fmt.Errorf("txClient not initialized")
	}

	// TODO: Implement cancel order using SDK
	return fmt.Errorf("not implemented")
}

// getMarketIndex converts symbol to Lighter market index
// This is a simplified version - you should fetch this from the API
func (c *Client) getMarketIndex(symbol string) (uint16, error) {
	// Normalize symbol
	normalizedSymbol := strings.TrimSuffix(symbol, "-USD")

	// Hardcoded mapping for common pairs
	// In production, fetch this from /api/v1/markets endpoint
	marketMap := map[string]uint16{
		"ETH":  1,
		"BTC":  2,
		"SOL":  3,
		"AVAX": 4,
		// Add more as needed
	}

	if marketIndex, ok := marketMap[normalizedSymbol]; ok {
		return marketIndex, nil
	}

	return 0, fmt.Errorf("unknown market: %s", symbol)
}
