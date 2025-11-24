# Lighter SDK 使用说明

## SDK 定位

`github.com/elliottech/lighter-go` SDK 的主要功能是:
- ✅ **签名交易**: 处理所有需要签名的操作(下单、取消订单等)
- ✅ **API Key 管理**: 创建和验证 API Key
- ✅ **Auth Token**: 生成鉴权 Token
- ❌ **不提供完整的 HTTP 客户端**: 需要自己实现 HTTP 调用

## 为什么使用 REST API 获取 Funding Rate?

### 原因
1. **公开数据**: Funding Rate 是公开 API,无需签名
2. **SDK 设计**: lighter-go 专注于签名,不提供数据查询封装
3. **简单高效**: 直接 HTTP GET 请求即可

### 对比

| 功能 | 使用方式 | 原因 |
|------|---------|------|
| 获取 Funding Rate | ✅ REST API | 公开数据,无需签名 |
| 获取价格 | ✅ REST API | 公开数据,无需签名 |
| 下单 | ⚠️ SDK + REST | 需要 SDK 签名,然后发送 HTTP 请求 |
| 取消订单 | ⚠️ SDK + REST | 需要 SDK 签名,然后发送 HTTP 请求 |
| 查询余额 | ⚠️ REST + Auth Token | 需要 SDK 生成 Auth Token |

## 正确的使用方式

### 1. 公开 API (当前实现)
```go
// 直接使用 HTTP GET
url := "https://mainnet.zklighter.elliot.ai/api/v1/funding-rates"
resp, err := http.Get(url)
// 解析 JSON
```

**优点**:
- ✅ 简单直接
- ✅ 无需签名
- ✅ 性能好

### 2. 私有 API (需要 Auth Token)
```go
import "github.com/elliottech/lighter-go/client"

// 1. 创建客户端
txClient, err := client.CreateClient(privateKey, apiKey, baseURL)

// 2. 生成 Auth Token
authToken, err := txClient.CreateAuthToken(0) // 0 = 7小时有效期

// 3. 使用 Auth Token 调用 API
req, _ := http.NewRequest("GET", url, nil)
req.Header.Set("Authorization", "Bearer " + authToken)
resp, err := http.DefaultClient.Do(req)
```

### 3. 下单 (需要签名)
```go
import (
    "github.com/elliottech/lighter-go/client"
    "github.com/elliottech/lighter-go/types/txtypes"
)

// 1. 创建客户端
txClient, err := client.CreateClient(privateKey, apiKey, baseURL)

// 2. 构造订单
order := txtypes.CreateOrderRequest{
    Symbol: "ETH-USDC",
    Side: "buy",
    Size: "0.1",
    Price: "3000",
    // ...
}

// 3. 签名订单
signedOrder, err := txClient.SignCreateOrder(order, -1, 255, 0)

// 4. 发送到交易所
// 需要自己实现 HTTP POST
req, _ := http.NewRequest("POST", baseURL + "/api/v1/orders", bytes.NewBuffer(signedOrder))
req.Header.Set("Content-Type", "application/json")
resp, err := http.DefaultClient.Do(req)
```

## 当前实现状态

### ✅ 已实现 (REST API)
```go
// internal/exchange/lighter/client.go

func (c *Client) GetFundingRate(symbol string) (float64, error) {
    // 直接 HTTP GET,无需 SDK
    url := c.cfg.BaseURL + "/api/v1/funding-rates"
    resp, err := c.httpClient.Get(url)
    // ...
}
```

**为什么这样做**:
- Funding Rate 是公开数据
- 无需签名或鉴权
- REST API 更简单高效

### ⚠️ 待实现 (SDK + REST)
```go
func (c *Client) PlaceOrder(req *exchange.OrderRequest) (*exchange.OrderResponse, error) {
    // 1. 使用 SDK 签名
    // 2. 发送 HTTP 请求
    return nil, fmt.Errorf("not implemented - requires SDK integration")
}
```

**需要做的**:
1. 集成 lighter-go SDK
2. 实现签名逻辑
3. 发送签名后的请求

## 总结

### 当前方案 ✅
- **公开 API**: 使用 REST API (正确)
- **私有 API**: 预留了 SDK 集成接口

### SDK 的作用
- **不是**: 完整的 API 客户端
- **是**: 签名工具 + 基础 HTTP 辅助

### 为什么看起来"没用 SDK"
因为当前只实现了公开 API,这些 API **本来就不需要 SDK**。

当需要下单时,会这样使用:
```
公开数据 (Funding Rate) → REST API ✅
私有数据 (余额) → SDK (Auth Token) + REST API
交易操作 (下单) → SDK (签名) + REST API
```

所以当前的实现是**完全正确**的!🎯
