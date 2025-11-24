# API 对接完成总结

## 已完成的工作

### 1. Lighter API 对接
- **Base URL**: `https://mainnet.zklighter.elliot.ai`
- **实现功能**:
  - ✅ `GetFundingRate`: 获取实时资金费率
  - ❌ `GetPrice`: 需要使用 orderbook 端点
  - ❌ `PlaceOrder`: 需要实现签名逻辑(使用 lighter-go SDK)

**API 示例**:
```bash
curl "https://mainnet.zklighter.elliot.ai/api/v1/funding-rates"
```

**返回格式**:
```json
{
  "code": 200,
  "funding_rates": [
    {
      "market_id": 1,
      "exchange": "binance",
      "symbol": "ETH",
      "rate": 0.00005
    }
  ]
}
```

### 2. EdgeX API 对接
- **Base URL**: `https://pro.edgex.exchange`
- **实现功能**:
  - ✅ `GetFundingRate`: 获取实时资金费率
  - ✅ `GetPrice`: 从 funding rate 端点获取 indexPrice
  - ✅ Metadata 缓存: 启动时加载合约列表
  - ❌ `PlaceOrder`: 需要实现 L2 签名和鉴权

**API 示例**:
```bash
# 获取元数据
curl "https://pro.edgex.exchange/api/v1/public/meta/getMetaData"

# 获取资金费率
curl "https://pro.edgex.exchange/api/v1/public/funding/getLatestFundingRate?contractId=10000001"
```

**合约映射**:
- ETH-USD → ETHUSD (contractId: 10000002)
- BTC-USD → BTCUSD (contractId: 10000001)
- SOL-USD → SOLUSD (contractId: 10000003)

### 3. Hyperliquid (已完成)
- **Base URL**: `https://api.hyperliquid.xyz`
- **实现功能**:
  - ✅ `GetFundingRate`: 使用 SDK
  - ✅ `GetPrice`: 使用 SDK
  - ✅ `PlaceOrder`: 完整的 L1 签名和下单逻辑

## 运行效果

程序现在可以:
1. 实时监控 Hyperliquid、Lighter、EdgeX 三个交易所的 Funding Rate
2. 计算最大价差
3. 当价差超过阈值时,触发套利机会提醒
4. (可选) 在 Hyperliquid 上自动执行套利交易

**示例输出**:
```
2025/11/24 14:20:01 Checking funding opportunities...
2025/11/24 14:20:01 [ETH-USD] Best Diff: 0.000087 (Threshold: 0.001000) - No Opportunity
2025/11/24 14:20:01 [BTC-USD] Best Diff: 0.000058 (Threshold: 0.001000) - No Opportunity
```

## 下一步建议

### 短期优化
1. **实现 Lighter GetPrice**: 对接 orderbook API 获取实时价格
2. **优化符号映射**: 创建统一的符号映射表
3. **错误重试**: 为 API 调用添加指数退避重试机制

### 中期目标
1. **Lighter 下单**: 使用 `lighter-go` SDK 实现签名和下单
2. **EdgeX 下单**: 实现 L2 签名逻辑
3. **持久化**: 记录交易历史到 SQLite
4. **监控面板**: 添加 Prometheus metrics

### 长期规划
1. **WebSocket 支持**: 使用 WebSocket 替代轮询,降低延迟
2. **多策略并行**: 支持同时运行多个套利策略
3. **风控系统**: 实现仓位管理、止损等风控机制
4. **回测系统**: 基于历史数据进行策略回测

## 配置说明

在 `config/config.yaml` 中:
```yaml
exchanges:
  hyperliquid:
    base_url: "https://api.hyperliquid.xyz"
    private_key: "YOUR_PRIVATE_KEY"  # 用于下单
  lighter:
    base_url: "https://mainnet.zklighter.elliot.ai"
  edgex:
    base_url: "https://pro.edgex.exchange"

strategies:
  funding_arb:
    enabled: true
    pairs: ["ETH-USD", "BTC-USD"]
    min_funding_diff: 0.001  # 0.1%
    execute_trades: false    # 设为 true 启用自动交易
```

## 注意事项

⚠️ **安全提醒**:
- Private Key 请妥善保管,不要提交到 Git
- 建议使用环境变量或加密存储
- 测试时建议使用小额资金

⚠️ **API 限制**:
- Lighter 和 EdgeX 的公开 API 可能有 Rate Limit
- 建议添加请求频率控制
- 生产环境建议使用 WebSocket

⚠️ **数据准确性**:
- Funding Rate 数据来自各交易所公开 API
- 实际交易前请验证数据准确性
- 注意时区和时间戳格式差异
