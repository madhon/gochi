package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

type AppConfig struct {
	ServeAddress string `mapstructure:"SERVE_ADDRESS"`
}

func LoadAppConfig(path string) (AppConfig, error) {
	if path == "" {
		return AppConfig{}, fmt.Errorf("config path is empty")
	}

	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			return AppConfig{}, fmt.Errorf("config file not found: %s", path)
		}
		return AppConfig{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AppConfig
	if err := viper.Unmarshal(&config); err != nil {
		return AppConfig{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}
