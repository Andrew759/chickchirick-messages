package factory

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

func InitViper() {
	viper.SetConfigFile("/app/.env")
	readConfig()
}

func readConfig() {
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			panic(fmt.Errorf("chirik_config file not found: %w", err))
		}
		panic(fmt.Errorf("viper fatal error: %w", err))
	}
}

func MergeConfigByFile(fileName string) {
	viper.SetConfigFile(fileName)
	if err := viper.MergeInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			panic(fmt.Errorf("chirik_config file not found: %w", err))
		}
		panic(fmt.Errorf("viper fatal error: %w", err))
	}
}
