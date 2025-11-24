package lighter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"arbitrage-bot/internal/config"
	"arbitrage-bot/internal/exchange"
)

type Client struct {
	cfg        config.LighterConfig
	httpClient *http.Client
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

func NewClient(cfg config.LighterConfig) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
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
	// Lighter doesn't have a simple price endpoint in the public API
	// We would need to use orderbook or other endpoints
	// For now, return error
	return 0, fmt.Errorf("not implemented - use orderbook endpoint")
}

func (c *Client) GetBalance(asset string) (float64, error) {
	return 0, fmt.Errorf("not implemented - requires authentication")
}

func (c *Client) GetPosition(symbol string) (*exchange.Position, error) {
	return nil, fmt.Errorf("not implemented - requires authentication")
}

func (c *Client) PlaceOrder(req *exchange.OrderRequest) (*exchange.OrderResponse, error) {
	return nil, fmt.Errorf("not implemented - requires authentication")
}

func (c *Client) CancelOrder(symbol, orderID string) error {
	return fmt.Errorf("not implemented - requires authentication")
}
