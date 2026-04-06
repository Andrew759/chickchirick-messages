package factory

import (
	"chickchirick-messages/cmd/service"
	"chickchirick-messages/internal/controller/c_controller"
	"chickchirick-messages/pkg/chirik_config"
	"net/http"

	_ "net/http/pprof"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func BuildAndServe(
	dbDecorator *service.DBDecorator,
	redisDecorator *service.RedisDecorator,
	httpClient *http.Client,
) {
	err := BuildServer(dbDecorator, redisDecorator, httpClient)
	if err != nil {
		panic(err)
	}
}

func BuildServer(dbDecorator *service.DBDecorator, redisDecorator *service.RedisDecorator, httpClient *http.Client) error {
	e := gin.Default()

	//TODO: доработать CORS
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{
		viper.GetString(chirik_config.UserApp),
		viper.GetString(chirik_config.AuthApp),
	}

	e.Use(cors.New(config))

	InitMessageServer(e, &c_controller.DIContainer{DBDecorator: dbDecorator, RedisDecorator: redisDecorator})

	err := e.Run()
	if err != nil {
		return err
	}

	return nil
}
