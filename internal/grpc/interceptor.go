package grpc

import (
	"context"
	"net/http" // Добавили для парсинга кук

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	cookieHeader    = "cookie"
	accessTokenName = "access_token"
)

func StreamAuthInterceptor(ac *AuthClient) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return status.Error(codes.Unauthenticated, "metadata is not found")
		}

		values := md.Get(cookieHeader)
		if len(values) == 0 {
			return status.Error(codes.Unauthenticated, "cookies not transferred")
		}

		rawCookies := values[0]
		header := http.Header{"Cookie": []string{rawCookies}}
		req := &http.Request{Header: header}

		cookie, err := req.Cookie(accessTokenName)
		if err != nil {
			return status.Error(codes.Unauthenticated, "access_token cookie not found")
		}

		token := cookie.Value

		userUUID, err := ac.Validate(ss.Context(), token)
		if err != nil {
			return status.Error(codes.Unauthenticated, "authorization failed: "+err.Error())
		}

		newCtx := context.WithValue(ss.Context(), "user_uuid", userUUID)

		wrapped := &wrappedStream{
			ServerStream: ss,
			ctx:          newCtx,
		}

		return handler(srv, wrapped)
	}
}

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}
