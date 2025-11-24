package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App        AppConfig        `mapstructure:"app"`
	Exchanges  ExchangesConfig  `mapstructure:"exchanges"`
	Strategies StrategiesConfig `mapstructure:"strategies"`
}

type AppConfig struct {
	LogLevel string `mapstructure:"log_level"`
	Port     int    `mapstructure:"port"`
}

type ExchangesConfig struct {
	Hyperliquid HyperliquidConfig `mapstructure:"hyperliquid"`
	Lighter     LighterConfig     `mapstructure:"lighter"`
	EdgeX       EdgeXConfig       `mapstructure:"edgex"`
}

type HyperliquidConfig struct {
	BaseURL       string `mapstructure:"base_url"`
	APIKey        string `mapstructure:"api_key"`
	SecretKey     string `mapstructure:"secret_key"`
	WalletAddress string `mapstructure:"wallet_address"`
	PrivateKey    string `mapstructure:"private_key"`
}

type LighterConfig struct {
	BaseURL    string `mapstructure:"base_url"`
	APIKey     string `mapstructure:"api_key"`
	PrivateKey string `mapstructure:"private_key"`
}

type EdgeXConfig struct {
	BaseURL         string `mapstructure:"base_url"`
	APIKey          string `mapstructure:"api_key"`
	SecretKey       string `mapstructure:"secret_key"`
	AccountID       string `mapstructure:"account_id"`
	StarkPrivateKey string `mapstructure:"stark_private_key"`
}

type StrategiesConfig struct {
	FundingArb FundingArbConfig `mapstructure:"funding_arb"`
	XPFarming  XPFarmingConfig  `mapstructure:"xp_farming"`
}

type FundingArbConfig struct {
	Enabled         bool     `mapstructure:"enabled"`
	Pairs           []string `mapstructure:"pairs"`
	MinFundingDiff  float64  `mapstructure:"min_funding_diff"`
	Leverage        float64  `mapstructure:"leverage"`
	CheckIntervalMs int      `mapstructure:"check_interval_ms"`
	ExecuteTrades   bool     `mapstructure:"execute_trades"`
}

type XPFarmingConfig struct {
	Enabled           bool    `mapstructure:"enabled"`
	TargetVolumeDaily float64 `mapstructure:"target_volume_daily"`
	MaxSlippage       float64 `mapstructure:"max_slippage"`
}

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
