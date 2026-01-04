package configs

import "github.com/spf13/viper"

type Config struct {
	ServerPort                  string `mapstructure:"SERVER_PORT"`
	RateLimiterMaxRequests      int    `mapstructure:"RATE_LIMITER_MAX_REQUESTS"`
	RateLimiterTimeDelay        string `mapstructure:"RATE_LIMITER_TIME_DELAY"`
	RateLimiterTokenMaxRequests int    `mapstructure:"RATE_LIMITER_TOKEN_MAX_REQUESTS"`
	RateLimiterTokenTimeDelay   string `mapstructure:"RATE_LIMITER_TOKEN_TIME_DELAY"`
	RateLimiterCleanupInterval  string `mapstructure:"RATE_LIMITER_CLEANUP_INTERVAL"`
	RateLimiterTTL              string `mapstructure:"RATE_LIMITER_TTL"`
	RateLimiterRedisAddr        string `mapstructure:"RATE_LIMITER_REDIS_ADDR"`
}

func LoadConfig(path string) (*Config, error) {
	var config *Config

	viper.SetConfigName("app_name")
	viper.AddConfigPath(path)
	viper.SetConfigType("env")
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	viper.BindEnv("SERVER_PORT")
	viper.BindEnv("RATE_LIMITER_MAX_REQUESTS")
	viper.BindEnv("RATE_LIMITER_TIME_DELAY")
	viper.BindEnv("RATE_LIMITER_TOKEN_MAX_REQUESTS")
	viper.BindEnv("RATE_LIMITER_TOKEN_TIME_DELAY")
	viper.BindEnv("RATE_LIMITER_CLEANUP_INTERVAL")
	viper.BindEnv("RATE_LIMITER_TTL")
	viper.BindEnv("RATE_LIMITER_REDIS_ADDR")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	return config, nil
}
