package main

import (
	"chickchirick-messages/cmd/config"
	"chickchirick-messages/cmd/factory"
	"chickchirick-messages/cmd/service"
	messageController "chickchirick-messages/internal/controller/service/message"
	"chickchirick-messages/internal/gen/messenger"
	grpcClient "chickchirick-messages/internal/grpc"
	"chickchirick-messages/internal/model/message"
	"chickchirick-messages/internal/ws"
	"chickchirick-messages/pkg/chirik_config"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

func main() {
	factory.InitViper()

	appConfig := config.AppConfiguration{}.NewAppConfiguration()

	dbDecorator := service.InitORM(&appConfig.DatabaseConfig)
	defer dbDecorator.CloseDB()

	//TODO: не должно попасть в коммиты
	dbDecorator.GDB().AutoMigrate(message.Message{}, message.Deleted{}, message.File{}, message.Group{}, message.Meta{}, message.Personal{}, message.Settings{}, message.Status{}, message.UserRelation{})

	redisDecorator := service.InitRedis(appConfig.RedisConfig)
	defer redisDecorator.RedisClose()

	httpClient := factory.InitHttpClient()

	authAddr := viper.GetString(chirik_config.AuthAppGrpc)
	authClient, err := grpcClient.NewAuthClient(authAddr)
	if err != nil {
		log.Fatalf("failed to init auth client: %v", err)
	}
	defer authClient.Close()

	s := grpc.NewServer(
		grpc.StreamInterceptor(grpcClient.StreamAuthInterceptor(authClient)),
	)

	msgController := messageController.NewController(dbDecorator.Gorm, redisDecorator.Client)
	messenger.RegisterMessengerServiceServer(s, msgController)
	go func() {
		lis, err := net.Listen("tcp", ":50052")
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		fmt.Println("gRPC Messenger server running on :50052")
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// WebSocket gateway for frontend
	go func() {
		mux := http.NewServeMux()

		mux.HandleFunc("/ws", ws.HandleWS("127.0.0.1:50052"))

		fmt.Println("WebSocket server running on :8085")
		if err := http.ListenAndServe(":8085", mux); err != nil {
			log.Fatalf("failed to start ws server: %v", err)
		}
	}()

	go func() {
		factory.BuildAndServe(dbDecorator, redisDecorator, httpClient, authClient)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	fmt.Println("shutting down servers...")
	s.GracefulStop()
}
