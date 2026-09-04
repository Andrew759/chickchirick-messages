package c_controller

import (
	"chickchirick-messages/cmd/service"
	grpcclient "chickchirick-messages/internal/grpc"

	"github.com/gin-gonic/gin"
)

type DIContainer struct {
	DBDecorator    *service.DBDecorator
	RedisDecorator *service.RedisDecorator
	AuthClient     *grpcclient.AuthClient
}

type Controller struct {
	E  *gin.Engine
	DI *DIContainer
}

type RequestHandler interface {
	RegisterRoutes()
}

type ControllerInterface interface {
	RequestHandler
}
