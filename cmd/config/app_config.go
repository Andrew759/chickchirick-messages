package config

import (
	"chickchirick-messages/cmd/config/dto"
	globalConfig "chickchirick-messages/pkg/chirik_config"

	"github.com/spf13/viper"
)

type AppConfigurationInterface interface {
	NewAppConfiguration() AppConfiguration
}

type AppConfiguration struct {
	ServerURL   string
	Environment string
	dto.DatabaseConfig
	RedisConfig
}

type RedisConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Db       int
}

func (c AppConfiguration) NewAppConfiguration() AppConfiguration {
	return AppConfiguration{
		ServerURL:      viper.GetString(globalConfig.ServerUrl),
		Environment:    viper.GetString(globalConfig.Enviroment),
		DatabaseConfig: PrepareDatabaseConfig(),
		RedisConfig: RedisConfig{
			Host:     viper.GetString(globalConfig.RedisHost),
			Port:     viper.GetInt(globalConfig.RedisInternalPort),
			User:     viper.GetString(globalConfig.RedisUser),
			Password: viper.GetString(globalConfig.RedisPassword),
			Db:       viper.GetInt(globalConfig.RedisDB),
		},
	}
}

func PrepareDatabaseConfig() dto.DatabaseConfig {
	dbc := dto.DatabaseConfig{}

	dbc.SetHost(viper.GetString(globalConfig.DbHost))
	dbc.SetPort(viper.GetInt(globalConfig.DbInternalPort))
	dbc.SetName(viper.GetString(globalConfig.DbName))
	dbc.SetUser(viper.GetString(globalConfig.DbUser))
	dbc.SetPassword(viper.GetString(globalConfig.DbPass))
	dbc.SetTimezone(viper.GetString(globalConfig.DbTimezone))

	return dbc
}
