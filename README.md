# Arbitrage Bot

## 简介
这是一个针对 Hyperliquid, Lighter, EdgeX 等 DEX 的永续合约套利与 XP 刷量机器人。
目前实现了基础架构、配置加载、Hyperliquid 行情获取以及 Funding Rate 套利监控逻辑。

## 功能特性
- **多交易所支持**: 
  - Hyperliquid (已实现 Funding Rate 获取)
  - Lighter (骨架已建立)
  - EdgeX (骨架已建立)
- **策略引擎**:
  - Funding Rate 套利: 自动监控多交易所资金费率差，触发套利机会。
- **配置化**: 支持 `config.yaml` 热配置。

## 快速开始

### 1. 配置
修改 `config/config.yaml`，填入你的 API Key 和钱包私钥。

```yaml
exchanges:
  hyperliquid:
    wallet_address: "YOUR_WALLET_ADDRESS"
    # ...
```

### 2. 运行
```bash
go run cmd/main.go
```

## 开发进度
- [x] 项目结构初始化
- [x] 配置系统 (Viper)
- [x] Exchange 接口定义
- [x] Hyperliquid Client (GetFundingRate & PlaceOrder with L1 Signing)
- [x] Lighter Client (Real API - GetFundingRate)
- [x] EdgeX Client (Real API - GetFundingRate & GetPrice)
- [x] EdgeX WebSocket (实时 ticker 订阅)
- [x] Funding Arb 策略逻辑 (监控与差价计算 + 自动下单)
- [x] XP 刷量策略 (随机间隔 + Wash Trade)
- [ ] Lighter/EdgeX 下单功能 (需要复杂签名,见文档)
- [ ] 持久化与监控

## 测试 WebSocket

运行 WebSocket 测试程序:
```bash
go run cmd/test_ws/main.go
```

## 注意事项
- **Lighter 和 EdgeX**: 已对接真实 API,可获取实时 Funding Rate。
- **EdgeX WebSocket**: 已实现,可实时订阅 ticker 数据,降低延迟。
- **API Key 配置**: 
  - ✅ 配置文件已预留 Lighter 和 EdgeX 的 API Key 字段
  - ✅ 代码已实现鉴权方法 (`addAuthHeaders`)
  - ℹ️ 当前公开 API 无需 API Key,私有 API(下单、查询账户)需要配置
  - 📖 详见 `docs/APIKey配置说明.md`
- **下单功能**: 
  - ✅ Hyperliquid: 完整实现,可通过 `config.yaml` 中的 `execute_trades` 开关控制。
  - ✅ Lighter: **已完成!** 使用官方 SDK,需配置 `api_key` 和 `private_key`。详见 `docs/Lighter下单功能说明.md`
  - ⚠️ EdgeX: 需要 StarkEx L2 签名,复杂度极高,建议手动对冲。
- **推荐方案**: 
  - 方案 A: Hyperliquid + Lighter 双交易所自动化
  - 方案 B: 仅 Hyperliquid 自动化,其他手动对冲

## 相关文档
- `docs/API对接总结.md` - API 对接详细说明
- `docs/APIKey配置说明.md` - API Key 配置指南 ⭐
- `docs/WebSocket与下单实现计划.md` - WebSocket 和下单功能实现计划
- `docs/项目完成总结.md` - 完整功能说明
- `docs/快速使用指南.md` - 5分钟快速上手
- `docs/可行性分析与补充.md` - 项目可行性分析
