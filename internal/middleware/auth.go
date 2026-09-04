package middleware

import (
	"net/http"

	grpcclient "chickchirick-messages/internal/grpc"

	"github.com/gin-gonic/gin"
)

func Auth(authClient *grpcclient.AuthClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("access_token")
		if err != nil || token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "access token missing"})
			c.Abort()
			return
		}

		uuid, err := authClient.Validate(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired access token"})
			c.Abort()
			return
		}

		c.Set("user_uuid", uuid)
		c.Next()
	}
}
