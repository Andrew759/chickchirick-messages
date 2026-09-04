package grpc

import (
	"context"
	"fmt"
	"log"

	"chickchirick-messages/internal/gen/auth"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AuthClient клиент GRPC авторизации
type AuthClient struct {
	Service auth.AuthServiceClient //Интерфейс из сгенерированного кода
	conn    *grpc.ClientConn       //Хранится соединение, чтобы его можно было закрыть
}

// NewAuthClient инициализирует подключение
func NewAuthClient(addr string) (*AuthClient, error) {
	//TODO: для прода нужно secure подключение, сейчас insecure
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("did not connect to auth service: %v", err)
	}

	client := auth.NewAuthServiceClient(conn)

	return &AuthClient{
		Service: client,
		conn:    conn,
	}, nil
}

// Close закрывает соединение
func (ac *AuthClient) Close() {
	if err := ac.conn.Close(); err != nil {
		log.Printf("error closing auth connection: %v", err)
	}
}

func (ac *AuthClient) Validate(ctx context.Context, token string) (string, error) {
	req := &auth.ValidateRequest{
		Token: token,
	}

	resp, err := ac.Service.ValidateToken(ctx, req)
	if err != nil {
		return "", fmt.Errorf("grpc request failed: %v", err)
	}

	if !resp.Valid {
		return "", fmt.Errorf("token is invalid")
	}

	return resp.UserUuid, nil
}
