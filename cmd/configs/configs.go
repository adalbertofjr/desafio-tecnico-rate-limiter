package configs

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

var (
	// ErrMissingConfigField is returned when a required config field is missing
	ErrMissingConfigField = "missing required config field: "
)

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

	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	viper.SetDefault("RATE_LIMITER_REDIS_ADDR", "localhost:6379")

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

	if err := validConfigField(); err != nil {
		return nil, err
	}

	return config, nil
}

func validConfigField() error {
	for k, v := range map[string]string{
		"SERVER_PORT":                     viper.GetString("SERVER_PORT"),
		"RATE_LIMITER_MAX_REQUESTS":       viper.GetString("RATE_LIMITER_MAX_REQUESTS"),
		"RATE_LIMITER_TIME_DELAY":         viper.GetString("RATE_LIMITER_TIME_DELAY"),
		"RATE_LIMITER_TOKEN_MAX_REQUESTS": viper.GetString("RATE_LIMITER_TOKEN_MAX_REQUESTS"),
		"RATE_LIMITER_TOKEN_TIME_DELAY":   viper.GetString("RATE_LIMITER_TOKEN_TIME_DELAY"),
		"RATE_LIMITER_CLEANUP_INTERVAL":   viper.GetString("RATE_LIMITER_CLEANUP_INTERVAL"),
		"RATE_LIMITER_TTL":                viper.GetString("RATE_LIMITER_TTL"),
		"RATE_LIMITER_REDIS_ADDR":         viper.GetString("RATE_LIMITER_REDIS_ADDR"),
	} {
		if v == "" {
			err := fmt.Errorf(ErrMissingConfigField + k)
			return err
		}
	}

	return nil
}

func (c *Config) ParseTimerDuration(value string) time.Duration {
	timer, err := time.ParseDuration(value)
	if err != nil {
		panic(err)
	}
	return timer
}
