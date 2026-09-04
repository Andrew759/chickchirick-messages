package factory

import (
	"chickchirick-messages/cmd/service"
	"chickchirick-messages/internal/controller/c_controller"
	grpcclient "chickchirick-messages/internal/grpc"
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
	authClient *grpcclient.AuthClient,
) {
	err := BuildServer(dbDecorator, redisDecorator, httpClient, authClient)
	if err != nil {
		panic(err)
	}
}

func BuildServer(dbDecorator *service.DBDecorator, redisDecorator *service.RedisDecorator, httpClient *http.Client, authClient *grpcclient.AuthClient) error {
	e := gin.Default()

	config := cors.Config{
		AllowCredentials: true,

		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},

		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
		},

		AllowOrigins: []string{
			viper.GetString(chirik_config.UserApp),
			viper.GetString(chirik_config.AuthApp),
			viper.GetString(chirik_config.FrontendAppUrl),
		},
	}

	e.Use(cors.New(config))

	InitMessageServer(e, &c_controller.DIContainer{
		DBDecorator: dbDecorator, RedisDecorator: redisDecorator, AuthClient: authClient,
	})

	err := e.Run()
	if err != nil {
		return err
	}

	return nil
}
