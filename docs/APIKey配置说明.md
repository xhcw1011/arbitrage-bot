# API Key 配置说明

## 概述

本项目已经为 Lighter 和 EdgeX 预留了 API Key 配置字段,可以在需要时填入相应的凭证。

## 配置文件位置

`config/config.yaml`

## 配置结构

### Hyperliquid
```yaml
exchanges:
  hyperliquid:
    base_url: "https://api.hyperliquid.xyz"
    api_key: ""          # 可选,目前未使用
    secret_key: ""       # 可选,目前未使用
    wallet_address: ""   # 可选,会从 private_key 自动推导
    private_key: ""      # 必需(如果要下单) - EVM 私钥,不带 0x 前缀
```

**说明**:
- Hyperliquid 使用 EVM 私钥进行签名
- 不需要单独的 API Key
- `private_key` 用于签名所有交易

### Lighter
```yaml
exchanges:
  lighter:
    base_url: "https://mainnet.zklighter.elliot.ai"
    api_key: ""        # Lighter API Key (用于鉴权)
    private_key: ""    # 用于签名交易
```

**说明**:
- `api_key`: 用于 HTTP 请求鉴权
- `private_key`: 用于签名交易(下单时需要)
- 当前仅公开 API 不需要 API Key
- 下单功能需要两者都配置

**如何获取**:
1. 访问 Lighter 官网
2. 创建账户并生成 API Key
3. 导出钱包私钥

### EdgeX
```yaml
exchanges:
  edgex:
    base_url: "https://pro.edgex.exchange"
    api_key: ""      # EdgeX API Key
    secret_key: ""   # EdgeX Secret Key
```

**说明**:
- `api_key` 和 `secret_key` 成对使用
- 用于私有 API 鉴权
- 当前仅公开 API 不需要

**如何获取**:
1. 访问 EdgeX 官网
2. 登录账户
3. 在 API 设置中生成 Key Pair

## 代码实现

### 1. 配置结构 (`internal/config/config.go`)

```go
type LighterConfig struct {
    BaseURL    string `mapstructure:"base_url"`
    APIKey     string `mapstructure:"api_key"`
    PrivateKey string `mapstructure:"private_key"`
}

type EdgeXConfig struct {
    BaseURL   string `mapstructure:"base_url"`
    APIKey    string `mapstructure:"api_key"`
    SecretKey string `mapstructure:"secret_key"`
}
```

### 2. 鉴权方法

#### Lighter (`internal/exchange/lighter/client.go`)
```go
// addAuthHeaders adds authentication headers to the request
func (c *Client) addAuthHeaders(req *http.Request) {
    if c.cfg.APIKey != "" {
        req.Header.Set("X-API-KEY", c.cfg.APIKey)
    }
}

// makeAuthenticatedRequest creates an authenticated HTTP request
func (c *Client) makeAuthenticatedRequest(method, url string, body io.Reader) (*http.Response, error) {
    req, err := http.NewRequest(method, url, body)
    if err != nil {
        return nil, err
    }
    
    c.addAuthHeaders(req)
    req.Header.Set("Content-Type", "application/json")
    
    return c.httpClient.Do(req)
}
```

#### EdgeX (`internal/exchange/edgex/client.go`)
```go
// addAuthHeaders adds authentication headers to the request
func (c *Client) addAuthHeaders(req *http.Request) {
    if c.cfg.APIKey != "" && c.cfg.SecretKey != "" {
        req.Header.Set("X-API-KEY", c.cfg.APIKey)
        req.Header.Set("X-API-SECRET", c.cfg.SecretKey)
    }
}

// makeAuthenticatedRequest creates an authenticated HTTP request
func (c *Client) makeAuthenticatedRequest(method, url string, body io.Reader) (*http.Response, error) {
    req, err := http.NewRequest(method, url, body)
    if err != nil {
        return nil, err
    }
    
    c.addAuthHeaders(req)
    req.Header.Set("Content-Type", "application/json")
    
    return c.httpClient.Do(req)
}
```

## 使用场景

### 当前状态 (无需 API Key)
✅ **可用功能**:
- 获取 Funding Rate (所有交易所)
- 获取价格 (Hyperliquid, EdgeX)
- Hyperliquid 下单 (仅需 private_key)

❌ **不可用**:
- Lighter 下单 (需要 api_key + private_key)
- EdgeX 下单 (需要 api_key + secret_key)
- 查询余额/仓位 (需要鉴权)

### 配置 API Key 后
✅ **额外可用**:
- 查询账户余额
- 查询持仓信息
- 查询订单历史
- 下单 (需要额外实现签名逻辑)

## 安全建议

### 1. 不要提交到 Git
```bash
# 确保 config.yaml 在 .gitignore 中
echo "config/config.yaml" >> .gitignore
```

### 2. 使用环境变量 (推荐)
```bash
# 设置环境变量
export LIGHTER_API_KEY="your_api_key"
export LIGHTER_PRIVATE_KEY="your_private_key"
export EDGEX_API_KEY="your_api_key"
export EDGEX_SECRET_KEY="your_secret_key"
```

代码会自动读取环境变量(通过 Viper 的 `AutomaticEnv()`)。

### 3. API Key 权限设置
- ✅ 仅开启交易权限
- ❌ 禁用提现权限
- ✅ 设置 IP 白名单
- ✅ 定期轮换 Key

### 4. 加密存储 (高级)
```go
// 可以实现配置加密
// 例如使用 age 或 sops
```

## 测试 API Key

### 测试 Lighter API Key
```bash
curl -H "X-API-KEY: your_api_key" \
  https://mainnet.zklighter.elliot.ai/api/v1/account/balance
```

### 测试 EdgeX API Key
```bash
curl -H "X-API-KEY: your_api_key" \
     -H "X-API-SECRET: your_secret_key" \
  https://pro.edgex.exchange/api/v1/private/account/getAccountInfo
```

## 常见问题

### Q1: 公开 API 需要 API Key 吗?
**A**: 不需要。当前实现的功能(获取 Funding Rate、价格)都是公开 API,无需鉴权。

### Q2: 什么时候需要配置 API Key?
**A**: 当你需要:
- 查询账户信息
- 下单交易 (Lighter/EdgeX)
- 查询私有数据

### Q3: API Key 配置错误会怎样?
**A**: 
- 公开 API 不受影响
- 私有 API 会返回 401/403 错误
- 程序会记录错误日志但不会崩溃

### Q4: 如何验证 API Key 是否正确?
**A**: 
1. 查看日志中是否有鉴权错误
2. 使用 curl 测试 API
3. 检查交易所后台 API Key 状态

## Header 名称参考

不同交易所可能使用不同的 Header 名称:

| 交易所 | API Key Header | Secret Header | 备注 |
|--------|---------------|---------------|------|
| Lighter | `X-API-KEY` | - | 可能需要调整 |
| EdgeX | `X-API-KEY` | `X-API-SECRET` | 可能需要调整 |
| 通用 | `Authorization` | - | Bearer Token |

**注意**: 实际使用时需要参考各交易所的官方文档,调整 `addAuthHeaders` 方法中的 Header 名称。

## 下一步

如果需要实现完整的下单功能,需要:

1. **查阅官方文档**: 确认正确的 Header 名称和格式
2. **实现签名逻辑**: 
   - Lighter: 使用 C 库签名
   - EdgeX: 实现 StarkEx L2 签名
3. **测试鉴权**: 先测试简单的查询 API
4. **实现下单**: 在鉴权成功后实现下单逻辑

## 总结

✅ **已完成**:
- 配置文件预留了所有必要字段
- 代码中实现了鉴权方法
- 支持环境变量配置

⚠️ **待完善**:
- 根据实际 API 文档调整 Header 名称
- 实现完整的签名逻辑
- 添加 API Key 验证测试

当前配置已经足够支持未来的扩展,你可以随时填入 API Key 并使用鉴权功能!
