package edgex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	edgexsdk "github.com/edgex-Tech/edgex-golang-sdk/sdk"

	"arbitrage-bot/internal/config"
	"arbitrage-bot/internal/exchange"
)

type Client struct {
	cfg        config.EdgeXConfig
	httpClient *http.Client
	metadata   *MetadataResponse
	sdkClient  *edgexsdk.Client
}

// EdgeX API Response structures
type EdgeXResponse struct {
	Code         string          `json:"code"`
	Data         json.RawMessage `json:"data"`
	Msg          *string         `json:"msg"`
	ErrorParam   *string         `json:"errorParam"`
	RequestTime  string          `json:"requestTime"`
	ResponseTime string          `json:"responseTime"`
	TraceId      string          `json:"traceId"`
}

type FundingRateData struct {
	ContractId       string `json:"contractId"`
	FundingRate      string `json:"fundingRate"`
	IndexPrice       string `json:"indexPrice"`
	FundingTimestamp string `json:"fundingTimestamp"`
}

type MetadataResponse struct {
	Global       GlobalConfig `json:"global"`
	ContractList []Contract   `json:"contractList"`
}

type GlobalConfig struct {
	AppName string `json:"appName"`
}

type Contract struct {
	ContractId   string `json:"contractId"`
	ContractName string `json:"contractName"`
	TickSize     string `json:"tickSize"`
	StepSize     string `json:"stepSize"`
}

func NewClient(cfg config.EdgeXConfig) *Client {
	client := &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Initialize SDK client if credentials are configured
	if cfg.APIKey != "" && cfg.SecretKey != "" {
		// Parse account ID from API key or config
		// For now, we'll need to add account_id to config
		// accountID, _ := strconv.ParseInt(cfg.AccountID, 10, 64)

		// TODO: Initialize EdgeX SDK client
		// sdkClient, err := edgexsdk.NewClient(
		// 	edgexsdk.WithBaseURL(cfg.BaseURL),
		// 	edgexsdk.WithAccountID(accountID),
		// 	edgexsdk.WithStarkPrivateKey(cfg.SecretKey),
		// )
		// if err != nil {
		// 	fmt.Printf("Warning: Failed to create EdgeX SDK client: %v\n", err)
		// } else {
		// 	client.sdkClient = sdkClient
		// 	fmt.Println("EdgeX SDK client initialized successfully")
		// }
	}

	// Fetch metadata on initialization
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if err := client.fetchMetadata(ctx); err != nil {
			fmt.Printf("Warning: Failed to fetch EdgeX metadata (attempt %d/3): %v\n", i+1, err)
			time.Sleep(time.Second)
			continue
		}
		break
	}

	return client
}

var _ exchange.Exchange = (*Client)(nil)

func (c *Client) fetchMetadata(ctx context.Context) error {
	url := c.cfg.BaseURL + "/api/v1/public/meta/getMetaData"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var apiResp EdgeXResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return err
	}

	if apiResp.Code != "SUCCESS" {
		return fmt.Errorf("API error: %s", apiResp.Code)
	}

	var metadata MetadataResponse
	if err := json.Unmarshal(apiResp.Data, &metadata); err != nil {
		return err
	}

	c.metadata = &metadata
	return nil
}

func (c *Client) getContractId(symbol string) (string, error) {
	if c.metadata == nil {
		return "", fmt.Errorf("metadata not loaded")
	}

	// Normalize: ETH-USD -> ETHUSD, BTC-USD -> BTCUSD
	normalizedSymbol := strings.ReplaceAll(symbol, "-", "")

	// EdgeX uses USD not USDT
	normalizedSymbol = strings.TrimSuffix(normalizedSymbol, "T") // Remove trailing T if present

	for _, contract := range c.metadata.ContractList {
		if contract.ContractName == normalizedSymbol {
			return contract.ContractId, nil
		}
	}

	return "", fmt.Errorf("contract not found for symbol: %s (normalized: %s)", symbol, normalizedSymbol)
}

// addAuthHeaders adds authentication headers to the request if API key is configured
func (c *Client) addAuthHeaders(req *http.Request) {
	if c.cfg.APIKey != "" && c.cfg.SecretKey != "" {
		// EdgeX uses specific auth headers
		// Adjust based on actual EdgeX API documentation
		req.Header.Set("X-API-KEY", c.cfg.APIKey)
		req.Header.Set("X-API-SECRET", c.cfg.SecretKey)
		// Or use Authorization header:
		// req.Header.Set("Authorization", "Bearer " + c.cfg.APIKey)
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

func (c *Client) GetFundingRate(symbol string) (float64, error) {
	contractId, err := c.getContractId(symbol)
	if err != nil {
		return 0, err
	}

	url := fmt.Sprintf("%s/api/v1/public/funding/getLatestFundingRate?contractId=%s",
		c.cfg.BaseURL, contractId)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var apiResp EdgeXResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return 0, err
	}

	if apiResp.Code != "SUCCESS" {
		return 0, fmt.Errorf("API error: %s", apiResp.Code)
	}

	var fundingData []FundingRateData
	if err := json.Unmarshal(apiResp.Data, &fundingData); err != nil {
		return 0, err
	}

	if len(fundingData) == 0 {
		return 0, fmt.Errorf("no funding data returned")
	}

	return strconv.ParseFloat(fundingData[0].FundingRate, 64)
}

func (c *Client) GetPrice(symbol string) (float64, error) {
	contractId, err := c.getContractId(symbol)
	if err != nil {
		return 0, err
	}

	// Use funding rate endpoint to get index price
	url := fmt.Sprintf("%s/api/v1/public/funding/getLatestFundingRate?contractId=%s",
		c.cfg.BaseURL, contractId)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var apiResp EdgeXResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return 0, err
	}

	if apiResp.Code != "SUCCESS" {
		return 0, fmt.Errorf("API error: %s", apiResp.Code)
	}

	var fundingData []FundingRateData
	if err := json.Unmarshal(apiResp.Data, &fundingData); err != nil {
		return 0, err
	}

	if len(fundingData) == 0 {
		return 0, fmt.Errorf("no funding data returned")
	}

	return strconv.ParseFloat(fundingData[0].IndexPrice, 64)
}

func (c *Client) GetBalance(asset string) (float64, error) {
	if c.sdkClient == nil {
		return 0, fmt.Errorf("SDK client not initialized - requires authentication")
	}

	// TODO: Use SDK to get balance
	// assets, err := c.sdkClient.Asset.GetAccountAsset(context.Background())
	return 0, fmt.Errorf("not implemented - requires SDK integration")
}

func (c *Client) GetPosition(symbol string) (*exchange.Position, error) {
	if c.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized - requires authentication")
	}

	// TODO: Use SDK to get position
	// positions, err := c.sdkClient.Account.GetAccountPosition(context.Background())
	return nil, fmt.Errorf("not implemented - requires SDK integration")
}

func (c *Client) PlaceOrder(req *exchange.OrderRequest) (*exchange.OrderResponse, error) {
	if c.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized - check api_key and secret_key configuration")
	}

	// TODO: Implement using EdgeX SDK
	// The SDK provides client.Order.CreateOrder() method
	// Need to:
	// 1. Convert symbol to contract ID
	// 2. Convert order parameters to SDK format
	// 3. Call SDK's CreateOrder method

	return nil, fmt.Errorf("EdgeX下单功能需要配置 account_id 和 stark_private_key,详见文档")
}

func (c *Client) CancelOrder(symbol, orderID string) error {
	if c.sdkClient == nil {
		return fmt.Errorf("SDK client not initialized")
	}

	// TODO: Use SDK to cancel order
	// err := c.sdkClient.Order.CancelOrder(context.Background(), orderID)
	return fmt.Errorf("not implemented - requires SDK integration")
}
