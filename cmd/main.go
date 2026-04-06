package main

import (
	"chickchirick-messages/cmd/config"
	"chickchirick-messages/cmd/factory"
	"chickchirick-messages/cmd/service"
)

func main() {
	factory.InitViper()

	appConfig := config.AppConfiguration{}.NewAppConfiguration()

	dbDecorator := service.InitORM(&appConfig.DatabaseConfig)
	defer dbDecorator.CloseDB()

	redisDecorator := service.InitRedis(appConfig.RedisConfig)
	defer redisDecorator.RedisClose()

	httpClient := factory.InitHttpClient()
	factory.BuildAndServe(dbDecorator, redisDecorator, httpClient)
}
