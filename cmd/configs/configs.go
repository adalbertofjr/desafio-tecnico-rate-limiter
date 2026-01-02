package configs

import "github.com/spf13/viper"

type Config struct {
	RateLimiterMaxRequests      int    `mapstructure:"RATE_LIMITER_MAX_REQUESTS"`
	RateLimiterTimeDelay        string `mapstructure:"RATE_LIMITER_TIME_DELAY"`
	RateLimiterTokenMaxRequests int    `mapstructure:"RATE_LIMITER_TOKEN_MAX_REQUESTS"`
	RateLimiterTokenTimeDelay   string `mapstructure:"RATE_LIMITER_TOKEN_TIME_DELAY"`
}

func LoadConfig(path string) (*Config, error) {
	var config *Config

	viper.SetConfigName("app_name")
	viper.AddConfigPath(path)
	viper.SetConfigType("env")
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	viper.BindEnv("RATE_LIMITER_MAX_REQUESTS")
	viper.BindEnv("RATE_LIMITER_TIME_DELAY")
	viper.BindEnv("RATE_LIMITER_TOKEN_MAX_REQUESTS")
	viper.BindEnv("RATE_LIMITER_TOKEN_TIME_DELAY")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	return config, nil
}
